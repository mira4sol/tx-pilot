package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/aegis/internal/lifecycle"
	"github.com/mira4sol/aegis/internal/scheduler"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func (cp *ControlPlane) SubmitOps(ctx context.Context, req aegis.SubmitOpsRequest) (aegis.SubmitResponse, error) {
	if cp.factory == nil {
		return aegis.SubmitResponse{}, fmt.Errorf("server signer not configured")
	}

	txID := aegis.TransactionID("tx_" + uuid.NewString())
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
		return aegis.SubmitResponse{}, err
	}

	blockhash, err := cp.fetchProcessedBlockhash(ctx)
	if err != nil {
		return aegis.SubmitResponse{}, err
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
		memo = string(aegis.OpsMemoPrefix) + "submit"
	} else if !isOpsTransaction(memo) {
		memo = string(aegis.OpsMemoPrefix) + memo
	}

	tipAccount, err := cp.jito.PickTipAccount()
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	// Build a single transaction that carries the ops payload AND the Jito tip.
	// Jito's sendTransaction endpoint auto-bundles it and lands it reliably,
	// whereas a separate-tip sendBundle is dropped by the public block engine.
	opsTx, err := cp.factory.BuildSelfTransferWithTip(ctx, lamports, memo, tipAccount, plan.Lamports, blockhash)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	opsEncoded, err := tx.EncodeTransaction(opsTx, aegis.EncodingBase64)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	sigs, err := tx.ExtractSignaturesFromBundle([]string{opsEncoded}, aegis.EncodingBase64)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: memo, Valid: true},
		RetryAttempt: 0, SubmissionKind: string(aegis.SubmissionBundle),
		Encoding: string(aegis.EncodingBase64), Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	cp.commitTipDecision(ctx, txID, &plan)

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
		Metadata: map[string]any{"timing_reason": timing.Reason, "inject_expired": req.InjectExpiredBlockhash},
	})

	bundleID, err := cp.forwardOpsTransaction(ctx, txID, opsEncoded, aegis.EncodingBase64, sigs[0], blockhash.String(), plan)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	return aegis.SubmitResponse{
		TransactionID: txID, SubmissionKind: aegis.SubmissionBundle,
		Result: bundleID, BundleID: aegis.BundleID(bundleID),
		Signatures: sigs, Signature: aegis.Signature(sigs[0]),
		Status: string(aegis.StatusSubmitted), Encoding: string(aegis.EncodingBase64),
		TipFloorLamports: plan.FloorLamports, TipLamports: plan.Lamports, TipSource: plan.Source,
	}, nil
}
