package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/aegis/internal/agent"
	"github.com/mira4sol/aegis/internal/bundle"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/internal/lifecycle"
	"github.com/mira4sol/aegis/internal/notify"
	"github.com/mira4sol/aegis/internal/queue"
	aegisrpc "github.com/mira4sol/aegis/internal/rpc"
	"github.com/mira4sol/aegis/internal/scheduler"
	"github.com/mira4sol/aegis/internal/storage"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/stream"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type ControlPlane struct {
	cfg        *config.Config
	q          *dbgen.Queries
	rpc        *aegisrpc.Client
	jito       *bundle.JitoClient
	tip        *bundle.TipResolver
	tracker    *lifecycle.Tracker
	slotState  *stream.SlotState
	sigTracker *stream.SignatureTracker
	hub        *notify.Hub
	notify     *notify.Dispatcher
	river      *river.Client[pgx.Tx]
	logger     *zap.Logger
	factory    *tx.Factory
	agent      agent.Provider
	scheduler  *scheduler.Scheduler
	recovery   *RecoveryBridge
}

type Dependencies struct {
	Config     *config.Config
	Queries    *dbgen.Queries
	RPC        *aegisrpc.Client
	Jito       *bundle.JitoClient
	Tip        *bundle.TipResolver
	Tracker    *lifecycle.Tracker
	SlotState  *stream.SlotState
	SigTracker *stream.SignatureTracker
	Hub        *notify.Hub
	Notify     *notify.Dispatcher
	River      *river.Client[pgx.Tx]
	Logger     *zap.Logger
	Factory    *tx.Factory
	Agent      agent.Provider
	Scheduler  *scheduler.Scheduler
}

func NewControlPlane(deps Dependencies) *ControlPlane {
	cp := &ControlPlane{
		cfg: deps.Config, q: deps.Queries, rpc: deps.RPC, jito: deps.Jito, tip: deps.Tip,
		tracker: deps.Tracker, slotState: deps.SlotState, sigTracker: deps.SigTracker,
		hub: deps.Hub, notify: deps.Notify, river: deps.River, logger: deps.Logger,
		factory: deps.Factory, agent: deps.Agent,
		scheduler: deps.Scheduler,
	}
	cp.recovery = &RecoveryBridge{cp: cp}
	return cp
}

func (cp *ControlPlane) SetRiver(client *river.Client[pgx.Tx]) {
	cp.river = client
}

func (cp *ControlPlane) RecoveryBridge() *RecoveryBridge { return cp.recovery }

func (cp *ControlPlane) emitLifecycle(ctx context.Context, ev lifecycle.StageEvent) {
	if err := cp.tracker.Emit(ctx, ev); err != nil {
		storage.LogDBOp(cp.logger, "EmitLifecycle", err,
			zap.String("transaction_id", string(ev.TransactionID)),
			zap.String("stage", string(ev.Stage)),
		)
	}
}

func (cp *ControlPlane) advisoryTipFloor(ctx context.Context) uint64 {
	congestion := cp.slotState.CongestionScore()
	tipRes, err := cp.tip.ResolveTip(ctx, bundle.ResolveInput{
		PolicyMode: cp.cfg.PolicyMode,
		Congestion: congestion,
	})
	if err != nil {
		return cp.cfg.JitoMinTipLamports
	}
	return uint64(tipRes.FloorLamports)
}

func (cp *ControlPlane) SubmitTransaction(ctx context.Context, req aegis.SubmitTransactionRequest) (aegis.SubmitResponse, error) {
	if req.Transaction == "" {
		return aegis.SubmitResponse{}, fmt.Errorf("transaction is required")
	}
	enc := tx.NormalizeEncoding(req.Encoding)
	if enc == "" {
		return aegis.SubmitResponse{}, fmt.Errorf("encoding must be base64 or base58")
	}

	sigs, err := tx.ExtractSignatures(req.Transaction, enc)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	txID := aegis.TransactionID("tx_" + uuid.NewString())
	policyMode := req.PolicyMode
	if policyMode == "" {
		policyMode = cp.cfg.PolicyMode
	}

	plan, err := cp.planTip(ctx, txID, policyMode, req.TipLamports)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	if len(sigs) == 0 {
		return aegis.SubmitResponse{}, fmt.Errorf("transaction has no signature")
	}

	// A pre-signed client transaction cannot have a server tip added (that would
	// invalidate its signature). It is submitted as a single transaction via
	// Jito sendTransaction (auto-bundled for MEV) and mirrored through the RPC so
	// it lands reliably; the planned tip is reported as the advisory amount.
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: req.Memo, Valid: req.Memo != ""},
		RetryAttempt: 0, SubmissionKind: string(aegis.SubmissionBundle),
		Encoding: string(enc), Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: true},
	})
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	storage.LogDBResult(cp.logger, "CreateTransaction", string(txID), nil)
	cp.logger.Info("transaction created",
		zap.String("transaction_id", string(txID)),
		zap.String("submission_kind", string(aegis.SubmissionBundle)),
		zap.Strings("signatures", sigs),
	)
	cp.commitTipDecision(ctx, txID, &plan)

	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
	})

	clientBlockhash, _ := tx.ExtractBlockhash(req.Transaction, enc)
	bundleID, err := cp.forwardClientTransaction(ctx, txID, req.Transaction, enc, sigs[0], clientBlockhash, plan)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	result := bundleID
	if result == "" {
		result = sigs[0]
	}

	return aegis.SubmitResponse{
		TransactionID: txID, SubmissionKind: aegis.SubmissionBundle,
		Result: result, BundleID: aegis.BundleID(bundleID),
		Signatures: sigs, Signature: aegis.Signature(sigs[0]),
		Status: string(aegis.StatusSubmitted), Encoding: string(enc),
		TipFloorLamports: plan.FloorLamports, TipLamports: plan.Lamports, TipSource: plan.Source,
	}, nil
}

func (cp *ControlPlane) SubmitBundle(ctx context.Context, req aegis.SubmitBundleRequest) (aegis.SubmitResponse, error) {
	if len(req.Transactions) == 0 {
		return aegis.SubmitResponse{}, fmt.Errorf("transactions are required")
	}
	if len(req.Transactions) >= aegis.MaxBundleTransactions {
		return aegis.SubmitResponse{}, fmt.Errorf("bundle supports at most %d transactions including tip", aegis.MaxBundleTransactions-1)
	}
	enc := tx.NormalizeEncoding(req.Encoding)
	if enc == "" {
		return aegis.SubmitResponse{}, fmt.Errorf("encoding must be base64 or base58")
	}

	txID := aegis.TransactionID("tx_" + uuid.NewString())
	policyMode := req.PolicyMode
	if policyMode == "" {
		policyMode = cp.cfg.PolicyMode
	}

	plan, err := cp.planTip(ctx, txID, policyMode, req.TipLamports)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	blockhash, err := cp.fetchProcessedBlockhash(ctx)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	tipEncoded, _, err := cp.buildSignedTipTx(ctx, plan.Lamports, blockhash, enc)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	allTxs := append(append([]string{}, req.Transactions...), tipEncoded)
	sigs, err := tx.ExtractSignaturesFromBundle(allTxs, enc)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), TipLamports: int64(plan.Lamports),
		RequestedTipLamports: int64(plan.Lamports), FloorLamports: int64(plan.FloorLamports),
		TipSource: string(plan.Source), Memo: pgtype.Text{String: req.Memo, Valid: req.Memo != ""},
		RetryAttempt: 0, SubmissionKind: string(aegis.SubmissionBundle),
		Encoding: string(enc), Signatures: sigsJSON, TxCount: int32(len(allTxs)),
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	storage.LogDBResult(cp.logger, "CreateTransaction", string(txID), nil)
	cp.logger.Info("transaction created",
		zap.String("transaction_id", string(txID)),
		zap.String("submission_kind", string(aegis.SubmissionBundle)),
		zap.Strings("signatures", sigs),
	)
	cp.commitTipDecision(ctx, txID, &plan)

	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
	})

	clientBlockhash := ""
	if len(req.Transactions) > 0 {
		clientBlockhash, _ = tx.ExtractBlockhash(req.Transactions[0], enc)
	}

	bundleID, err := cp.forwardBundle(ctx, txID, allTxs, enc, sigs, clientBlockhash, plan)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	return aegis.SubmitResponse{
		TransactionID: txID, SubmissionKind: aegis.SubmissionBundle,
		Result: bundleID, BundleID: aegis.BundleID(bundleID),
		Signatures: sigs, Signature: aegis.Signature(sigs[0]),
		Status: string(aegis.StatusSubmitted), Encoding: string(enc),
		TipFloorLamports: plan.FloorLamports, TipLamports: plan.Lamports, TipSource: plan.Source,
	}, nil
}

// landViaRPC submits each encoded transaction through the configured RPC so it
// lands on-chain even when the Jito path is best-effort (the public block engine
// frequently acknowledges but never ingests multi-tx bundles). Per-transaction
// errors are returned only when nothing could be sent; "already processed"
// duplicates are expected and ignored by the caller.
func (cp *ControlPlane) landViaRPC(ctx context.Context, encoded []string, enc aegis.Encoding) error {
	var firstErr error
	sent := 0
	for _, e := range encoded {
		if _, err := cp.rpc.SendTransaction(ctx, e, string(enc)); err != nil {
			cp.logger.Warn("rpc landing send failed", zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		sent++
	}
	if sent > 0 {
		return nil
	}
	return firstErr
}

// markSubmitted records a successful submission: it creates the bundle row (when
// Jito returned a bundle id), flips the transaction to submitted, emits the
// lifecycle event, tracks the signatures for the geyser stream, and enqueues the
// time-bounded status poll. It is shared by every submission path.
func (cp *ControlPlane) markSubmitted(ctx context.Context, txID aegis.TransactionID, bundleID string, sigs []string, blockhash string, plan tipPlan, submitPath string) (string, error) {
	now := time.Now().UTC()
	slot := cp.slotState.CurrentSlot()
	if plan.TargetSlot > 0 {
		slot = plan.TargetSlot
	}
	leader := plan.Leader
	if leader == "" {
		leader = cp.slotState.LeaderAt(slot)
	}
	primarySig := ""
	if len(sigs) > 0 {
		primarySig = sigs[0]
	}

	if bundleID != "" {
		_, err := cp.q.CreateBundle(ctx, dbgen.CreateBundleParams{
			ID: bundleID, TransactionID: pgtype.Text{String: string(txID), Valid: true},
			Status: "submitted", TipLamports: int64(plan.Lamports),
			TargetSlot:  pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
			SubmittedAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		storage.LogDBOp(cp.logger, "CreateBundle", err,
			zap.String("transaction_id", string(txID)),
			zap.String("bundle_id", bundleID),
		)
	}

	_, err := cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusSubmitted), Stage: string(aegis.StageSubmitted),
		Signature:   pgtype.Text{String: primarySig, Valid: primarySig != ""},
		BundleID:    pgtype.Text{String: bundleID, Valid: bundleID != ""},
		TipLamports: int64(plan.Lamports), TargetSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		SubmittedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		Leader:        pgtype.Text{String: leader, Valid: leader != ""},
		SubmittedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	})
	storage.LogDBOp(cp.logger, "UpdateTransactionStatus.submitted", err,
		zap.String("transaction_id", string(txID)),
		zap.String("bundle_id", bundleID),
		zap.String("signature", primarySig),
		zap.Uint64("slot", slot),
		zap.String("leader", leader),
		zap.Uint64("tip_lamports", plan.Lamports),
	)
	if err == nil {
		cp.logger.Info("transaction submitted",
			zap.String("transaction_id", string(txID)),
			zap.String("bundle_id", bundleID),
			zap.String("signature", primarySig),
			zap.Uint64("slot", slot),
			zap.String("leader", leader),
			zap.Uint64("tip_lamports", plan.Lamports),
			zap.String("submit_path", submitPath),
		)
	}

	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(primarySig),
		BundleID: aegis.BundleID(bundleID), Stage: aegis.StageSubmitted,
		Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: map[string]any{"leader": leader, "tip_lamports": plan.Lamports, "submit_path": submitPath},
	})

	for _, sig := range sigs {
		cp.sigTracker.Track(sig, stream.TrackedSignature{
			TransactionID: txID, BundleID: aegis.BundleID(bundleID), Signature: aegis.Signature(sig),
		})
	}

	cp.broadcastTransaction(txID)
	cp.enqueueStatusPoll(txID, aegis.SubmissionBundle, bundleID, sigs, blockhash)
	return bundleID, nil
}

// forwardBundle submits a client-provided bundle through Jito (for MEV/ordering)
// and mirrors the constituent transactions through the RPC so they still land if
// the public block engine drops the bundle. Submission only fails when both
// paths reject the transactions.
func (cp *ControlPlane) forwardBundle(ctx context.Context, txID aegis.TransactionID, encoded []string, enc aegis.Encoding, sigs []string, blockhash string, plan tipPlan) (string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}

	bundleID, jitoErr := cp.jito.SendBundle(ctx, encoded, enc)
	rpcErr := cp.landViaRPC(ctx, encoded, enc)
	if jitoErr != nil && rpcErr != nil {
		err := fmt.Errorf("jito sendBundle: %v; rpc fallback: %v", jitoErr, rpcErr)
		cp.handleSubmitFailure(ctx, txID, "", err)
		return "", err
	}
	if jitoErr != nil {
		cp.logger.Warn("jito sendBundle failed; transactions submitted via rpc fallback", zap.Error(jitoErr))
		bundleID = ""
	}

	return cp.markSubmitted(ctx, txID, bundleID, sigs, blockhash, plan, "jito_send_bundle+rpc")
}

// forwardClientTransaction submits a single client transaction through Jito's
// sendTransaction endpoint (auto-bundled for MEV protection) and mirrors it
// through the RPC so it lands reliably even when the client did not embed a tip.
func (cp *ControlPlane) forwardClientTransaction(ctx context.Context, txID aegis.TransactionID, encoded string, enc aegis.Encoding, sig, blockhash string, plan tipPlan) (string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}

	var bundleID string
	res, jitoErr := cp.jito.SendTransaction(ctx, encoded, enc)
	if jitoErr == nil {
		bundleID = res.BundleID
		if res.Signature != "" {
			sig = res.Signature
		}
	} else {
		cp.logger.Warn("jito sendTransaction failed; relying on rpc fallback", zap.Error(jitoErr))
	}
	rpcErr := cp.landViaRPC(ctx, []string{encoded}, enc)
	if jitoErr != nil && rpcErr != nil {
		err := fmt.Errorf("jito sendTransaction: %v; rpc fallback: %v", jitoErr, rpcErr)
		cp.handleSubmitFailure(ctx, txID, "", err)
		return "", err
	}

	return cp.markSubmitted(ctx, txID, bundleID, []string{sig}, blockhash, plan, "jito_send_transaction+rpc")
}

// forwardOpsTransaction submits a single ops transaction (with the tip embedded)
// through Jito's sendTransaction endpoint. Jito wraps it into a bundle and
// returns the bundle id via the x-bundle-id header, so the lifecycle is still
// tracked as a bundle while landing reliably on the public block engine.
func (cp *ControlPlane) forwardOpsTransaction(ctx context.Context, txID aegis.TransactionID, encoded string, enc aegis.Encoding, sig, blockhash string, plan tipPlan) (string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}

	res, err := cp.jito.SendTransaction(ctx, encoded, enc)
	if err != nil {
		cp.handleSubmitFailure(ctx, txID, "", err)
		return "", err
	}
	bundleID := res.BundleID
	if res.Signature != "" {
		sig = res.Signature
	}

	return cp.markSubmitted(ctx, txID, bundleID, []string{sig}, blockhash, plan, "jito_send_transaction")
}

func (cp *ControlPlane) handleSubmitFailure(ctx context.Context, txID aegis.TransactionID, bundleID string, err error) {
	cp.logger.Warn("jito submission failed", zap.String("transaction_id", string(txID)), zap.Error(err))
	class := failure.Classify(err, failure.Evidence{RPCMessage: err.Error(), Slot: cp.slotState.CurrentSlot()})
	cp.recordFailure(ctx, txID, bundleID, class)
	now := time.Now().UTC()
	slot := cp.slotState.CurrentSlot()
	_, dbErr := cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusFailed), Stage: string(aegis.StageFailed),
		FailureKind:   pgtype.Text{String: string(class.Kind), Valid: true},
		SubmittedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		Leader:        pgtype.Text{String: cp.slotState.LeaderAt(slot), Valid: true},
		FailedAt:      pgtype.Timestamptz{Time: now, Valid: true},
	})
	storage.LogDBOp(cp.logger, "UpdateTransactionStatus.failed", dbErr,
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(class.Kind)),
	)
	cp.logger.Info("submit failure recorded",
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(class.Kind)),
		zap.String("title", class.Title),
		zap.Uint64("slot", slot),
	)
	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageFailed, Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: map[string]any{"error": err.Error(), "failure_kind": string(class.Kind)},
	})
	cp.broadcastTransaction(txID)
}

func (cp *ControlPlane) OnStreamSignature(ctx context.Context, signature string, slot uint64) {
	entry, ok := cp.sigTracker.Lookup(signature)
	if !ok {
		return
	}
	txRow, err := cp.q.GetTransaction(ctx, string(entry.TransactionID))
	if err != nil {
		return
	}
	if txRow.ProcessedAt.Valid {
		return
	}
	now := time.Now().UTC()
	leader := cp.slotState.LeaderAt(slot)
	_, err = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(entry.TransactionID), Status: string(aegis.StatusProcessing), Stage: string(aegis.StageProcessed),
		Signature:     pgtype.Text{String: signature, Valid: true},
		ProcessedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		ProcessedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		Leader:        pgtype.Text{String: leader, Valid: leader != ""},
	})
	storage.LogDBOp(cp.logger, "UpdateTransactionStatus.stream_processed", err,
		zap.String("transaction_id", string(entry.TransactionID)),
		zap.String("signature", signature),
		zap.Uint64("slot", slot),
	)
	if err == nil {
		cp.logger.Info("geyser processed",
			zap.String("transaction_id", string(entry.TransactionID)),
			zap.String("signature", signature),
			zap.Uint64("slot", slot),
			zap.String("leader", leader),
			zap.String("source", "geyser_stream"),
		)
	}
	cp.emitLifecycle(ctx, lifecycle.StageEvent{
		TransactionID: entry.TransactionID, Signature: entry.Signature,
		BundleID: entry.BundleID, Stage: aegis.StageProcessed,
		Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: map[string]any{"source": "geyser_stream"},
	})
	cp.broadcastTransaction(entry.TransactionID)
}

func (cp *ControlPlane) enqueueStatusPoll(txID aegis.TransactionID, kind aegis.SubmissionKind, bundleID string, sigs []string, blockhash string) {
	if cp.river == nil {
		cp.logger.Warn("status poll skipped: river client not configured",
			zap.String("transaction_id", string(txID)),
		)
		return
	}
	_, err := cp.river.Insert(context.Background(), queue.StatusPollArgs{
		TransactionID:       string(txID),
		SubmissionKind:      string(kind),
		BundleID:            bundleID,
		Signatures:          sigs,
		Blockhash:           blockhash,
		Attempt:             0,
		FirstPolledAtUnixMS: time.Now().UnixMilli(),
	}, &river.InsertOpts{
		ScheduledAt: time.Now().Add(queue.PendingPollInterval),
	})
	if err != nil {
		storage.LogDBOp(cp.logger, "InsertStatusPoll", err,
			zap.String("transaction_id", string(txID)),
			zap.String("bundle_id", bundleID),
		)
		return
	}
	sig := ""
	if len(sigs) > 0 {
		sig = sigs[0]
	}
	cp.logger.Info("status poll enqueued",
		zap.String("transaction_id", string(txID)),
		zap.String("bundle_id", bundleID),
		zap.String("signature", sig),
		zap.Duration("delay", queue.PendingPollInterval),
	)
}

func (cp *ControlPlane) recordFailure(ctx context.Context, txID aegis.TransactionID, bundleID string, class failure.Classification) {
	evidence, _ := json.Marshal(class.Evidence)
	slot := cp.slotState.CurrentSlot()
	_, err := cp.q.InsertFailure(ctx, dbgen.InsertFailureParams{
		ID: "fail_" + uuid.NewString(), TransactionID: pgtype.Text{String: string(txID), Valid: true},
		BundleID: pgtype.Text{String: bundleID, Valid: bundleID != ""},
		Kind:     string(class.Kind), Title: class.Title, Severity: class.Severity,
		Slot:              pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		RecommendedAction: pgtype.Text{String: class.RecommendedAction, Valid: true}, Evidence: evidence,
	})
	storage.LogDBOp(cp.logger, "InsertFailure", err,
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(class.Kind)),
		zap.String("title", class.Title),
		zap.Uint64("slot", slot),
	)
	if cp.notify != nil {
		cp.notify.BroadcastFailure(class)
	}
}

func (cp *ControlPlane) broadcastTransaction(txID aegis.TransactionID) {
	if cp.notify == nil {
		return
	}
	txRow, err := cp.q.GetTransaction(context.Background(), string(txID))
	if err != nil {
		return
	}
	cp.notify.BroadcastTransaction(string(txID), txRow.Stage, txRow)
}

func (cp *ControlPlane) GetTransaction(ctx context.Context, id string) (dbgen.Transaction, error) {
	return cp.q.GetTransaction(ctx, id)
}

func (cp *ControlPlane) GetTimeline(ctx context.Context, id string) ([]dbgen.LifecycleEvent, error) {
	return cp.q.ListLifecycleEvents(ctx, id)
}

func (cp *ControlPlane) GetBundle(ctx context.Context, id string) (dbgen.Bundle, error) {
	return cp.q.GetBundle(ctx, id)
}

func (cp *ControlPlane) GetLatestBlockhash(ctx context.Context) (*aegisrpc.GetLatestBlockhashResponse, error) {
	return cp.rpc.GetLatestBlockhash(ctx, "processed")
}

func (cp *ControlPlane) GetTipAccounts(ctx context.Context) ([]string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		if err := cp.jito.LoadTipAccounts(ctx); err != nil {
			return nil, err
		}
	}
	return cp.jito.TipAccounts(), nil
}

func (cp *ControlPlane) PollBundleStatus(ctx context.Context, bundleID string) (any, error) {
	statuses, err := cp.jito.GetBundleStatuses(ctx, []string{bundleID})
	if err != nil {
		return nil, err
	}
	if len(statuses.Value) == 0 {
		inflight, ierr := cp.jito.GetInflightBundleStatuses(ctx, []string{bundleID})
		if ierr != nil {
			return nil, ierr
		}
		return inflight, nil
	}
	return statuses, nil
}

// RecoveryBridge exposes recovery hooks to queue workers.
type RecoveryBridge struct {
	cp *ControlPlane
}

func (b *RecoveryBridge) Trigger(ctx context.Context, txID aegis.TransactionID, kind aegis.FailureKind, title, action string, rc queue.RecoveryContext) {
	b.cp.HandleFailureRecovery(ctx, txID, kind, title, action, recoveryContext{
		TransactionID:  rc.TransactionID,
		SubmissionKind: rc.SubmissionKind,
		BundleID:       rc.BundleID,
		Signatures:     rc.Signatures,
		Blockhash:      rc.Blockhash,
		RetryAttempt:   rc.RetryAttempt,
		OpsMemo:        rc.OpsMemo,
		OpsLamports:    rc.OpsLamports,
		PolicyMode:     rc.PolicyMode,
	})
}
