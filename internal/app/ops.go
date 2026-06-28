package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/lifecycle"
	"github.com/mira4sol/tx-pilot/internal/scheduler"
	"github.com/mira4sol/tx-pilot/internal/storage"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/internal/tx"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"go.uber.org/zap"
)

func (cp *ControlPlane) SubmitOps(ctx context.Context, req txpilot.SubmitOpsRequest) (txpilot.SubmitResponse, error) {
	if cp.factory == nil {
		return txpilot.SubmitResponse{}, fmt.Errorf("server signer not configured")
	}

	txID := txpilot.TransactionID("tx_" + uuid.NewString())
	policyMode := req.PolicyMode
	if policyMode == "" {
		policyMode = cp.cfg.PolicyMode
	}

	timing := scheduler.Decision{SubmitNow: true, Reason: "immediate submit"}
	if cp.scheduler != nil {
		timing = cp.scheduler.Evaluate(cp.slotState, policyMode)
		if !timing.SubmitNow {
			time.Sleep(cp.scheduler.WaitDuration(timing.DelaySlots))
		}
	}

	plan, err := cp.planTip(ctx, txID, policyMode, req.TipLamports)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}

	blockhash, err := cp.fetchProcessedBlockhash(ctx)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}
	if req.InjectExpiredBlockhash {
		blockhash = expiredBlockhashForInjection()
	}

	lamports := req.Lamports
	if lamports == 0 {
		lamports = 1
	}
	memo := req.Memo
	if memo == "" {
		memo = string(txpilot.OpsMemoPrefix) + "submit"
	} else if !isOpsTransaction(memo) {
		memo = string(txpilot.OpsMemoPrefix) + memo
	}

	tipAccount, err := cp.jito.PickTipAccount()
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}
	// Build a single transaction that carries the ops payload AND the Jito tip.
	// Jito's sendTransaction endpoint auto-bundles it and lands it reliably,
	// whereas a separate-tip sendBundle is dropped by the public block engine.
	opsTx, err := cp.factory.BuildSelfTransferWithTip(ctx, lamports, memo, tipAccount, plan.Lamports, blockhash)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}
	opsEncoded, err := tx.EncodeTransaction(opsTx, txpilot.EncodingBase64)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}

	sigs, err := tx.ExtractSignaturesFromBundle([]string{opsEncoded}, txpilot.EncodingBase64)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(txpilot.StatusPending), Stage: string(txpilot.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: memo, Valid: true},
		RetryAttempt: 0, SubmissionKind: string(txpilot.SubmissionBundle),
		Encoding: string(txpilot.EncodingBase64), Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}
	storage.LogDBResult(cp.logger, "CreateTransaction", string(txID), nil)
	cp.logger.Info("transaction created",
		zap.String("transaction_id", string(txID)),
		zap.String("submission_kind", string(txpilot.SubmissionBundle)),
		zap.Strings("signatures", sigs),
	)
	cp.commitTipDecision(ctx, txID, &plan)

	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: txpilot.StageCreated, Timestamp: time.Now().UTC(),
		Metadata: map[string]any{"timing_reason": timing.Reason, "inject_expired": req.InjectExpiredBlockhash},
	})

	bundleID, err := cp.forwardOpsTransaction(ctx, txID, opsEncoded, txpilot.EncodingBase64, sigs[0], blockhash.String(), plan)
	if err != nil {
		return txpilot.SubmitResponse{}, err
	}

	return txpilot.SubmitResponse{
		TransactionID: txID, SubmissionKind: txpilot.SubmissionBundle,
		Result: bundleID, BundleID: txpilot.BundleID(bundleID),
		Signatures: sigs, Signature: txpilot.Signature(sigs[0]),
		Status: string(txpilot.StatusSubmitted), Encoding: string(txpilot.EncodingBase64),
		TipFloorLamports: plan.FloorLamports, TipLamports: plan.Lamports, TipSource: plan.Source,
	}, nil
}
