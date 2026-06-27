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
	"github.com/mira4sol/aegis/internal/agent"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/internal/lifecycle"
	"github.com/mira4sol/aegis/internal/storage"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
	"go.uber.org/zap"
)

type recoveryContext struct {
	TransactionID  aegis.TransactionID
	SubmissionKind aegis.SubmissionKind
	BundleID       string
	Signatures     []string
	Blockhash      string
	RetryAttempt   int32
	OpsMemo        string
	OpsLamports    uint64
	PolicyMode     aegis.PolicyMode
}

func (cp *ControlPlane) HandleFailureRecovery(ctx context.Context, txID aegis.TransactionID, kind aegis.FailureKind, title, action string, rc recoveryContext) {
	if cp.agent == nil {
		return
	}

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
	}

	decision, err := cp.agent.Decide(ctx, facts)
	if err != nil {
		cp.logger.Warn("agent decision failed", zap.Error(err))
		return
	}
	decisionID, _ := cp.persistAgentDecision(ctx, txID, decision)
	cp.persistRecoveryAction(ctx, txID, decisionID, decision.Title, "queued")

	if !isOpsTransaction(rc.OpsMemo) {
		cp.logger.Info("client tx failure; AI advisory recorded",
			zap.String("transaction_id", string(txID)), zap.String("summary", decision.Summary))
		return
	}

	kindAction := actionKind(decision)
	if kindAction == "abort" {
		cp.persistRecoveryAction(ctx, txID, decisionID, "Abort recovery", "done")
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

	if err := cp.resubmitOps(ctx, txID, rc, decision); err != nil {
		cp.logger.Warn("autonomous ops resubmit failed", zap.Error(err))
		cp.persistRecoveryAction(ctx, txID, decisionID, "Resubmit failed: "+err.Error(), "failed")
		return
	}
	cp.persistRecoveryAction(ctx, txID, decisionID, decision.Summary, "running")
}

func (cp *ControlPlane) resubmitOps(ctx context.Context, parentID aegis.TransactionID, rc recoveryContext, decision agent.Decision) error {
	if cp.factory == nil {
		return fmt.Errorf("server signer not configured")
	}

	newTxID := aegis.TransactionID("tx_" + uuid.NewString())
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
		memo = string(aegis.OpsMemoPrefix) + "retry"
	}

	tipAccount, err := cp.jito.PickTipAccount()
	if err != nil {
		return err
	}
	opsTx, err := cp.factory.BuildSelfTransferWithTip(ctx, lamports, memo, tipAccount, plan.Lamports, blockhash)
	if err != nil {
		return err
	}
	opsEncoded, err := tx.EncodeTransaction(opsTx, aegis.EncodingBase64)
	if err != nil {
		return err
	}

	sigs, err := tx.ExtractSignaturesFromBundle([]string{opsEncoded}, aegis.EncodingBase64)
	if err != nil {
		return err
	}
	sigsJSON, _ := json.Marshal(sigs)

	retryAttempt := rc.RetryAttempt + 1
	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(newTxID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: memo, Valid: true},
		RetryAttempt: retryAttempt, SubmissionKind: string(aegis.SubmissionBundle),
		Encoding: string(aegis.EncodingBase64), Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return err
	}
	cp.commitTipDecision(ctx, newTxID, &plan)

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: newTxID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
		Metadata: map[string]any{"retry_of": string(parentID), "agent_action": decision.Action},
	})

	parentRow, err := cp.q.GetTransaction(ctx, string(parentID))
	if err != nil {
		return err
	}
	_, err = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(parentID), Status: parentRow.Status, Stage: parentRow.Stage, RetryAttempt: retryAttempt,
	})
	storage.LogDB(cp.logger, "UpdateTransactionStatus.retry_attempt", err)

	bundleID, err := cp.forwardOpsTransaction(ctx, newTxID, opsEncoded, aegis.EncodingBase64, sigs[0], blockhash.String(), plan)
	if err != nil {
		return err
	}
	cp.logger.Info("autonomous ops resubmit",
		zap.String("transaction_id", string(newTxID)), zap.String("bundle_id", bundleID))
	return nil
}

func isOpsTransaction(memo string) bool {
	return strings.HasPrefix(memo, string(aegis.OpsMemoPrefix))
}

func expiredBlockhashForInjection() solana.Hash {
	return solana.Hash{}
}
