package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/aegis/internal/bundle"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/internal/lifecycle"
	"github.com/mira4sol/aegis/internal/notify"
	"github.com/mira4sol/aegis/internal/queue"
	aegisrpc "github.com/mira4sol/aegis/internal/rpc"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/stream"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type ControlPlane struct {
	cfg       *config.Config
	q         *dbgen.Queries
	rpc       *aegisrpc.Client
	jito      *bundle.JitoClient
	tip       *bundle.TipResolver
	tracker   *lifecycle.Tracker
	slotState *stream.SlotState
	hub       *notify.Hub
	river     *river.Client[pgx.Tx]
	logger    *zap.Logger
}

type Dependencies struct {
	Config    *config.Config
	Queries   *dbgen.Queries
	RPC       *aegisrpc.Client
	Jito      *bundle.JitoClient
	Tip       *bundle.TipResolver
	Tracker   *lifecycle.Tracker
	SlotState *stream.SlotState
	Hub       *notify.Hub
	River     *river.Client[pgx.Tx]
	Logger    *zap.Logger
}

func NewControlPlane(deps Dependencies) *ControlPlane {
	return &ControlPlane{
		cfg: deps.Config, q: deps.Queries, rpc: deps.RPC, jito: deps.Jito, tip: deps.Tip,
		tracker: deps.Tracker, slotState: deps.SlotState, hub: deps.Hub, river: deps.River, logger: deps.Logger,
	}
}

func (cp *ControlPlane) advisoryTipFloor(ctx context.Context) uint64 {
	tipRes, err := cp.tip.ResolveTip(ctx, bundle.ResolveInput{
		PolicyMode: cp.cfg.PolicyMode,
		Congestion: 0.35,
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
	floor := cp.advisoryTipFloor(ctx)
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), FloorLamports: int64(floor), TipSource: string(aegis.TipSourceAuto),
		Memo: pgtype.Text{String: req.Memo, Valid: req.Memo != ""}, RetryAttempt: 0,
		SubmissionKind: string(aegis.SubmissionTransaction), Encoding: string(enc),
		Signatures: sigsJSON, TxCount: 1,
		Signature: pgtype.Text{String: sigs[0], Valid: true},
	})
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
	})

	blockhash, _ := tx.ExtractBlockhash(req.Transaction, enc)

	result, err := cp.forwardTransaction(ctx, txID, req.Transaction, enc, sigs, blockhash)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	return aegis.SubmitResponse{
		TransactionID: txID, SubmissionKind: aegis.SubmissionTransaction,
		Result: result, Signatures: sigs, Signature: aegis.Signature(result),
		Status: string(aegis.StatusSubmitted), Encoding: string(enc),
		TipFloorLamports: floor,
	}, nil
}

func (cp *ControlPlane) forwardTransaction(ctx context.Context, txID aegis.TransactionID, encoded string, enc aegis.Encoding, sigs []string, blockhash string) (string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}

	result, err := cp.jito.SendTransaction(ctx, encoded, enc)
	if err != nil {
		cp.handleSubmitFailure(ctx, txID, "", err)
		return "", err
	}

	now := time.Now().UTC()
	slot := cp.slotState.CurrentSlot()
	bundleID := result.BundleID

	_, _ = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusSubmitted), Stage: string(aegis.StageSubmitted),
		Signature: pgtype.Text{String: result.Signature, Valid: true},
		BundleID: pgtype.Text{String: bundleID, Valid: bundleID != ""},
		SubmittedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		SubmittedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(result.Signature),
		BundleID: aegis.BundleID(bundleID), Stage: aegis.StageSubmitted,
		Slot: aegis.Slot(slot), Timestamp: now,
	})
	cp.broadcastTransaction(txID)
	cp.enqueueStatusPoll(txID, aegis.SubmissionTransaction, bundleID, sigs, blockhash)
	return result.Signature, nil
}

func (cp *ControlPlane) SubmitBundle(ctx context.Context, req aegis.SubmitBundleRequest) (aegis.SubmitResponse, error) {
	if len(req.Transactions) == 0 {
		return aegis.SubmitResponse{}, fmt.Errorf("transactions are required")
	}
	if len(req.Transactions) > aegis.MaxBundleTransactions {
		return aegis.SubmitResponse{}, fmt.Errorf("bundle supports at most %d transactions", aegis.MaxBundleTransactions)
	}
	enc := tx.NormalizeEncoding(req.Encoding)
	if enc == "" {
		return aegis.SubmitResponse{}, fmt.Errorf("encoding must be base64 or base58")
	}

	sigs, err := tx.ExtractSignaturesFromBundle(req.Transactions, enc)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	txID := aegis.TransactionID("tx_" + uuid.NewString())
	policyMode := req.PolicyMode
	if policyMode == "" {
		policyMode = cp.cfg.PolicyMode
	}
	floor := cp.advisoryTipFloor(ctx)
	sigsJSON, _ := json.Marshal(sigs)

	_, err = cp.q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID: string(txID), Status: string(aegis.StatusPending), Stage: string(aegis.StageCreated),
		PolicyMode: string(policyMode), FloorLamports: int64(floor), TipSource: string(aegis.TipSourceAuto),
		Memo: pgtype.Text{String: req.Memo, Valid: req.Memo != ""}, RetryAttempt: 0,
		SubmissionKind: string(aegis.SubmissionBundle), Encoding: string(enc),
		Signatures: sigsJSON, TxCount: int32(len(req.Transactions)),
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
	})
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageCreated, Timestamp: time.Now().UTC(),
	})

	blockhash := ""
	if len(req.Transactions) > 0 {
		blockhash, _ = tx.ExtractBlockhash(req.Transactions[0], enc)
	}

	bundleID, err := cp.forwardBundle(ctx, txID, req.Transactions, enc, sigs, blockhash)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}

	return aegis.SubmitResponse{
		TransactionID: txID, SubmissionKind: aegis.SubmissionBundle,
		Result: bundleID, BundleID: aegis.BundleID(bundleID),
		Signatures: sigs, Signature: aegis.Signature(sigs[0]),
		Status: string(aegis.StatusSubmitted), Encoding: string(enc),
		TipFloorLamports: floor,
	}, nil
}

func (cp *ControlPlane) forwardBundle(ctx context.Context, txID aegis.TransactionID, encoded []string, enc aegis.Encoding, sigs []string, blockhash string) (string, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}

	bundleID, err := cp.jito.SendBundle(ctx, encoded, enc)
	if err != nil {
		cp.handleSubmitFailure(ctx, txID, "", err)
		return "", err
	}

	now := time.Now().UTC()
	slot := cp.slotState.CurrentSlot()

	_, _ = cp.q.CreateBundle(ctx, dbgen.CreateBundleParams{
		ID: bundleID, TransactionID: pgtype.Text{String: string(txID), Valid: true},
		Status: "submitted", TargetSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		SubmittedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	_, _ = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusSubmitted), Stage: string(aegis.StageSubmitted),
		Signature: pgtype.Text{String: sigs[0], Valid: len(sigs) > 0},
		BundleID: pgtype.Text{String: bundleID, Valid: true},
		SubmittedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		SubmittedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})

	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(sigs[0]),
		BundleID: aegis.BundleID(bundleID), Stage: aegis.StageSubmitted,
		Slot: aegis.Slot(slot), Timestamp: now,
	})
	cp.broadcastTransaction(txID)
	cp.enqueueStatusPoll(txID, aegis.SubmissionBundle, bundleID, sigs, blockhash)
	return bundleID, nil
}

func (cp *ControlPlane) handleSubmitFailure(ctx context.Context, txID aegis.TransactionID, bundleID string, err error) {
	cp.logger.Warn("jito submission failed", zap.String("transaction_id", string(txID)), zap.Error(err))
	class := failure.Classify(err, failure.Evidence{RPCMessage: err.Error()})
	cp.recordFailure(ctx, txID, bundleID, class)
	now := time.Now().UTC()
	_, _ = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusFailed), Stage: string(aegis.StageFailed),
		FailedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	_ = cp.tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Stage: aegis.StageFailed, Timestamp: now,
		Metadata: map[string]any{"error": err.Error()},
	})
	cp.broadcastTransaction(txID)
}

func (cp *ControlPlane) enqueueStatusPoll(txID aegis.TransactionID, kind aegis.SubmissionKind, bundleID string, sigs []string, blockhash string) {
	if cp.river == nil {
		return
	}
	_, _ = cp.river.Insert(context.Background(), queue.StatusPollArgs{
		TransactionID:  string(txID),
		SubmissionKind: string(kind),
		BundleID:       bundleID,
		Signatures:     sigs,
		Blockhash:      blockhash,
		Attempt:        0,
	}, &river.InsertOpts{
		ScheduledAt: time.Now().Add(queue.PendingPollInterval),
	})
}

func (cp *ControlPlane) recordFailure(ctx context.Context, txID aegis.TransactionID, bundleID string, class failure.Classification) {
	evidence, _ := json.Marshal(class.Evidence)
	_, _ = cp.q.InsertFailure(ctx, dbgen.InsertFailureParams{
		ID: "fail_" + uuid.NewString(), TransactionID: pgtype.Text{String: string(txID), Valid: true},
		BundleID: pgtype.Text{String: bundleID, Valid: bundleID != ""},
		Kind: string(class.Kind), Title: class.Title, Severity: class.Severity,
		RecommendedAction: pgtype.Text{String: class.RecommendedAction, Valid: true}, Evidence: evidence,
	})
	if cp.hub != nil {
		cp.hub.Broadcast("failures.analysis", "failure.recorded", class)
	}
}

func (cp *ControlPlane) broadcastTransaction(txID aegis.TransactionID) {
	if cp.hub == nil {
		return
	}
	txRow, err := cp.q.GetTransaction(context.Background(), string(txID))
	if err != nil {
		return
	}
	cp.hub.Broadcast("transactions.stream", "transaction.updated", txRow)
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
