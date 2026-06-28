package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/agent"
	"github.com/mira4sol/tx-pilot/internal/failure"
	"github.com/mira4sol/tx-pilot/internal/lifecycle"
	"github.com/mira4sol/tx-pilot/internal/storage"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/internal/tx"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"go.uber.org/zap"
)

type recoveryContext struct {
	TransactionID  txpilot.TransactionID
	SubmissionKind txpilot.SubmissionKind
	BundleID       string
	Signatures     []string
	Blockhash      string
	RetryAttempt   int32
	OpsMemo        string
	OpsLamports    uint64
	PolicyMode     txpilot.PolicyMode
}

func (cp *ControlPlane) HandleFailureRecovery(ctx context.Context, txID txpilot.TransactionID, kind txpilot.FailureKind, title, action string, rc recoveryContext) {
	if cp.agent == nil {
		return
	}

	cp.logger.Info("failure recovery started",
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(kind)),
		zap.String("title", title),
	)

	facts := agent.DecisionFacts{
		TransactionID: string(txID),
		Failure: failure.Classification{
			Kind: kind, Title: title, RecommendedAction: action,
		},
		RetryAttempt:  int(rc.RetryAttempt),
		CongestionPct: float64(cp.slotState.CongestionPct()),
		CurrentTip:    0,
		FloorTip:      cp.advisoryTipFloor(ctx),
		CurrentSlot:   cp.slotState.CurrentSlot(),
		Leader:        cp.slotState.LeaderAt(cp.slotState.CurrentSlot()),
	}
	if row, err := cp.q.GetTransaction(ctx, string(txID)); err == nil {
		facts.CurrentTip = uint64(row.TipLamports)
		facts.RetryAttempt = int(row.RetryAttempt)
		rc.RetryAttempt = row.RetryAttempt
		if rc.OpsMemo == "" && row.Memo.Valid {
			rc.OpsMemo = row.Memo.String
		}
		if rc.PolicyMode == "" {
			rc.PolicyMode = txpilot.PolicyMode(row.PolicyMode)
		}
		if rc.SubmissionKind == "" {
			rc.SubmissionKind = txpilot.SubmissionKind(row.SubmissionKind)
		}
	}

	decision, err := cp.agent.Decide(ctx, facts)
	if err != nil {
		cp.logger.Warn("agent decision failed", zap.String("transaction_id", string(txID)), zap.Error(err))
		return
	}
	decisionID, err := cp.persistAgentDecision(ctx, txID, decision)
	if err != nil {
		return
	}
	recoveryID := cp.persistRecoveryAction(ctx, txID, decisionID, decision.Title, "queued")

	if !isOpsTransaction(rc.OpsMemo) {
		cp.updateRecoveryActionStatus(ctx, recoveryID, "done")
		cp.logger.Info("client tx failure; AI advisory recorded",
			zap.String("transaction_id", string(txID)), zap.String("summary", decision.Summary))
		return
	}

	kindAction := actionKind(decision)
	if kindAction == "abort" {
		cp.updateRecoveryActionStatus(ctx, recoveryID, "done")
		return
	}

	delaySlots := uint64(0)
	if v, ok := decision.Action["delay_slots"]; ok {
		switch n := v.(type) {
		case float64:
			delaySlots = uint64(n)
		case int:
			delaySlots = uint64(n)
		}
	}
	if delaySlots > 0 {
		time.Sleep(time.Duration(delaySlots) * 400 * time.Millisecond)
	}

	cp.updateRecoveryActionStatus(ctx, recoveryID, "running")
	if err := cp.resubmitOps(ctx, txID, rc, decision); err != nil {
		cp.logger.Warn("autonomous ops resubmit failed", zap.Error(err))
		cp.updateRecoveryActionStatus(ctx, recoveryID, "failed")
		return
	}
	cp.updateRecoveryActionStatus(ctx, recoveryID, "done")
}

func (cp *ControlPlane) resubmitOps(ctx context.Context, parentID txpilot.TransactionID, rc recoveryContext, decision agent.Decision) error {
	if cp.factory == nil {
		return fmt.Errorf("server signer not configured")
	}

	newTxID := txpilot.TransactionID("tx_" + uuid.NewString())
	policyMode := rc.PolicyMode
	if policyMode == "" {
		policyMode = cp.cfg.PolicyMode
	}

	var requestedTip *uint64
	if v, ok := decision.Action["tip_delta_pct"]; ok {
		base := cp.advisoryTipFloor(ctx)
		switch n := v.(type) {
		case float64:
			boosted := uint64(float64(base) * (1 + n/100))
			requestedTip = &boosted
		}
	}
	if tipLam, ok := decision.Action["tip_lamports"]; ok {
		switch n := tipLam.(type) {
		case float64:
			v := uint64(n)
			requestedTip = &v
		}
	}

	plan, err := cp.planTip(ctx, newTxID, policyMode, requestedTip)
	if err != nil {
		return err
	}

	blockhash, err := cp.fetchProcessedBlockhash(ctx)
	if err != nil {
		return err
	}

	lamports := rc.OpsLamports
	if lamports == 0 {
		lamports = 1
	}
	memo := rc.OpsMemo
	if memo == "" {
		memo = string(txpilot.OpsMemoPrefix) + "retry"
	}

	tipAccount, err := cp.jito.PickTipAccount()
	if err != nil {
		return err
	}
	opsTx, err := cp.factory.BuildSelfTransferWithTip(ctx, lamports, memo, tipAccount, plan.Lamports, blockhash)
	if err != nil {
		return err
	}
	opsEncoded, err := tx.EncodeTransaction(opsTx, txpilot.EncodingBase64)
	if err != nil {
		return err
	}

	sigs, err := tx.ExtractSignaturesFromBundle([]string{opsEncoded}, txpilot.EncodingBase64)
	if err != nil {
		return err
	}
	sigsJSON, _ := json.Marshal(sigs)

	retryAttempt := rc.RetryAttempt + 1
	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(newTxID), Status: string(txpilot.StatusPending), Stage: string(txpilot.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: memo, Valid: true},
		RetryAttempt: retryAttempt, SubmissionKind: string(txpilot.SubmissionBundle),
		Encoding: string(txpilot.EncodingBase64), Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return err
	}
	storage.LogDBResult(cp.logger, "CreateTransaction", string(newTxID), nil)
	cp.commitTipDecision(ctx, newTxID, &plan)

	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: newTxID, Stage: txpilot.StageCreated, Timestamp: time.Now().UTC(),
		Metadata: map[string]any{"retry_of": string(parentID), "agent_action": decision.Action},
	})

	parentRow, err := cp.q.GetTransaction(ctx, string(parentID))
	if err != nil {
		return err
	}
	_, err = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(parentID), Status: parentRow.Status, Stage: parentRow.Stage, RetryAttempt: retryAttempt,
	})
	storage.LogDBOp(cp.logger, "UpdateTransactionStatus.retry_attempt", err,
		zap.String("transaction_id", string(parentID)),
		zap.Int32("retry_attempt", retryAttempt),
	)

	bundleID, err := cp.forwardOpsTransaction(ctx, newTxID, opsEncoded, txpilot.EncodingBase64, sigs[0], blockhash.String(), plan)
	if err != nil {
		return err
	}
	cp.logger.Info("autonomous ops resubmit",
		zap.String("transaction_id", string(newTxID)), zap.String("bundle_id", bundleID))
	return nil
}

func isOpsTransaction(memo string) bool {
	return strings.HasPrefix(memo, string(txpilot.OpsMemoPrefix))
}

func expiredBlockhashForInjection() solana.Hash {
	return solana.Hash{}
}
