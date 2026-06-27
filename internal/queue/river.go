package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mira4sol/aegis/internal/bundle"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/internal/lifecycle"
	"github.com/mira4sol/aegis/internal/notify"
	aegisrpc "github.com/mira4sol/aegis/internal/rpc"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"go.uber.org/zap"
)

const (
	// PendingPollInterval is how often we re-check a still-pending submission.
	// Solana slots are ~400ms, so a 1s cadence catches landings quickly without
	// hammering the RPC.
	PendingPollInterval = 1 * time.Second
	// FinalizePollInterval is used after a transaction is already confirmed and
	// we are only waiting for finalization (~13s), so we can poll less often.
	FinalizePollInterval = 2 * time.Second
	// MaxPollAttempts is a hard safety cap. In practice a transaction reaches a
	// terminal state long before this via on-chain error or blockhash expiry.
	MaxPollAttempts = 150
)

type WorkerDeps struct {
	Queries *dbgen.Queries
	RPC     *aegisrpc.Client
	Jito    *bundle.JitoClient
	Tracker *lifecycle.Tracker
	Hub     *notify.Hub
	Logger  *zap.Logger
}

type RiverQueue struct {
	client *river.Client[pgx.Tx]
}

func NewRiverQueue(ctx context.Context, pool *pgxpool.Pool, deps WorkerDeps, logger *zap.Logger) (*RiverQueue, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &StatusPollWorker{deps: deps})
	river.AddWorker(workers, &WebhookDeliveryWorker{})

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 20},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("create river client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("start river client: %w", err)
	}
	return &RiverQueue{client: client}, nil
}

func (q *RiverQueue) Client() *river.Client[pgx.Tx] {
	return q.client
}

func (q *RiverQueue) Stop(ctx context.Context) error {
	return q.client.Stop(ctx)
}

type StatusPollArgs struct {
	TransactionID  string   `json:"transaction_id"`
	SubmissionKind string   `json:"submission_kind"`
	BundleID       string   `json:"bundle_id"`
	Signatures     []string `json:"signatures"`
	Blockhash      string   `json:"blockhash"`
	Attempt        int      `json:"attempt"`
	river.JobArgs
}

func (StatusPollArgs) Kind() string { return "status_poll" }

type StatusPollWorker struct {
	river.WorkerDefaults[StatusPollArgs]
	deps WorkerDeps
}

func (w *StatusPollWorker) Work(ctx context.Context, job *river.Job[StatusPollArgs]) error {
	args := job.Args
	txID := aegis.TransactionID(args.TransactionID)

	if args.SubmissionKind == string(aegis.SubmissionBundle) && args.BundleID != "" {
		return w.pollBundle(ctx, job, txID, args)
	}
	return w.pollTransaction(ctx, job, txID, args)
}

func (w *StatusPollWorker) pollTransaction(ctx context.Context, job *river.Job[StatusPollArgs], txID aegis.TransactionID, args StatusPollArgs) error {
	if len(args.Signatures) == 0 {
		return nil
	}
	result, err := w.deps.RPC.GetSignatureStatuses(ctx, args.Signatures, false)
	if err != nil {
		// Transient RPC error: try again, do not penalize the transaction.
		return w.reschedule(ctx, job, args)
	}

	if len(result.Value) > 0 && result.Value[0] != nil {
		st := result.Value[0]
		// The signature landed. If it carries an execution error it is a
		// terminal on-chain failure — stop immediately with the real reason.
		if isExecutionError(st.Err) {
			return w.markFailedOnChain(ctx, txID, args.BundleID, args.Signatures[0], st.Err, slotFromPtr(st.Slot))
		}
		if st.ConfirmationStatus != "" {
			return w.applyConfirmation(ctx, job, txID, args.BundleID, args.Signatures[0], st.ConfirmationStatus, slotFromPtr(st.Slot))
		}
		// Seen by the cluster but not yet processed — keep polling.
		return w.reschedule(ctx, job, args)
	}

	// Signature not found. Either it is still propagating, or it will never land
	// (expired blockhash / dropped / fee-payer could not pay). Disambiguate via
	// blockhash validity so we don't loop pointlessly.
	if w.blockhashExpired(ctx, args.Blockhash) {
		return w.markFailed(ctx, txID, args.BundleID, args.Signatures[0],
			aegis.FailureExpiredBlockhash, "Blockhash expired before the transaction landed",
			"Refresh the blockhash and resubmit; the transaction was never included", 0, nil)
	}
	return w.reschedule(ctx, job, args)
}

func (w *StatusPollWorker) pollBundle(ctx context.Context, job *river.Job[StatusPollArgs], txID aegis.TransactionID, args StatusPollArgs) error {
	// Inspect the underlying transaction first: even a landed bundle can contain
	// a transaction that failed execution on-chain.
	if len(args.Signatures) > 0 {
		if result, err := w.deps.RPC.GetSignatureStatuses(ctx, args.Signatures, false); err == nil &&
			len(result.Value) > 0 && result.Value[0] != nil && isExecutionError(result.Value[0].Err) {
			return w.markFailedOnChain(ctx, txID, args.BundleID, args.Signatures[0], result.Value[0].Err, slotFromPtr(result.Value[0].Slot))
		}
	}

	statuses, err := w.deps.Jito.GetBundleStatuses(ctx, []string{args.BundleID})
	if err == nil && len(statuses.Value) > 0 && statuses.Value[0].ConfirmationStatus != "" {
		bs := statuses.Value[0]
		sig := w.firstSig(args)
		if len(bs.Transactions) > 0 {
			sig = bs.Transactions[0]
		}
		return w.applyConfirmation(ctx, job, txID, args.BundleID, sig, bs.ConfirmationStatus, bs.Slot)
	}

	inflight, ierr := w.deps.Jito.GetInflightBundleStatuses(ctx, []string{args.BundleID})
	if ierr == nil && len(inflight.Value) > 0 {
		switch inflight.Value[0].Status {
		case "Failed":
			return w.markFailed(ctx, txID, args.BundleID, w.firstSig(args),
				aegis.FailureBundleRejected, "Bundle rejected by Jito",
				"Recalculate tip above the dynamic floor and resubmit", 0, nil)
		case "Landed":
			if len(args.Signatures) > 0 {
				result, rerr := w.deps.RPC.GetSignatureStatuses(ctx, args.Signatures, false)
				if rerr == nil && len(result.Value) > 0 && result.Value[0] != nil && result.Value[0].ConfirmationStatus != "" {
					return w.applyConfirmation(ctx, job, txID, args.BundleID, args.Signatures[0], result.Value[0].ConfirmationStatus, slotFromPtr(result.Value[0].Slot))
				}
			}
		}
	}

	// Not landed yet. If the blockhash has expired the bundle can never land.
	if w.blockhashExpired(ctx, args.Blockhash) {
		return w.markFailed(ctx, txID, args.BundleID, w.firstSig(args),
			aegis.FailureExpiredBlockhash, "Blockhash expired before the bundle landed",
			"Refresh the blockhash and resubmit the bundle", 0, nil)
	}
	return w.reschedule(ctx, job, args)
}

func (w *StatusPollWorker) firstSig(args StatusPollArgs) string {
	if len(args.Signatures) > 0 {
		return args.Signatures[0]
	}
	return ""
}

// blockhashExpired reports whether the referenced blockhash can no longer land a
// transaction. A nil/empty blockhash or an RPC error returns false so we never
// fail a transaction we cannot definitively prove is dead.
func (w *StatusPollWorker) blockhashExpired(ctx context.Context, blockhash string) bool {
	if blockhash == "" {
		return false
	}
	valid, err := w.deps.RPC.IsBlockhashValid(ctx, blockhash, "confirmed")
	if err != nil {
		return false
	}
	return !valid
}

func isExecutionError(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	s := string(raw)
	return s != "null" && s != "{}"
}

func (w *StatusPollWorker) applyConfirmation(ctx context.Context, job *river.Job[StatusPollArgs], txID aegis.TransactionID, bundleID, signature, confirmation string, slot uint64) error {
	now := time.Now().UTC()
	stage := mapConfirmation(confirmation)
	params := dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: confirmation, Stage: string(stage),
		Signature: pgtype.Text{String: signature, Valid: signature != ""},
		BundleID: pgtype.Text{String: bundleID, Valid: bundleID != ""},
	}
	switch stage {
	case aegis.StageProcessed:
		params.ProcessedAt = pgtype.Timestamptz{Time: now, Valid: true}
		params.ProcessedSlot = pgtype.Int8{Int64: int64(slot), Valid: slot > 0}
	case aegis.StageConfirmed:
		params.ConfirmedAt = pgtype.Timestamptz{Time: now, Valid: true}
		params.ConfirmedSlot = pgtype.Int8{Int64: int64(slot), Valid: slot > 0}
	case aegis.StageFinalized:
		params.FinalizedAt = pgtype.Timestamptz{Time: now, Valid: true}
		params.FinalizedSlot = pgtype.Int8{Int64: int64(slot), Valid: slot > 0}
		if bundleID != "" {
			_, _ = w.deps.Queries.UpdateBundleStatus(ctx, dbgen.UpdateBundleStatusParams{
				ID: bundleID, Status: "landed", LandedAt: pgtype.Timestamptz{Time: now, Valid: true},
			})
		}
	}
	_, _ = w.deps.Queries.UpdateTransactionStatus(ctx, params)
	_ = w.deps.Tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(signature),
		BundleID: aegis.BundleID(bundleID), Stage: stage,
		Slot: aegis.Slot(slot), Timestamp: now,
	})
	w.broadcast(txID)
	if stage != aegis.StageFinalized {
		args := job.Args
		args.Attempt++
		if client := river.ClientFromContext[pgx.Tx](ctx); client != nil {
			_, _ = client.Insert(ctx, args, &river.InsertOpts{
				ScheduledAt: time.Now().Add(FinalizePollInterval),
			})
		}
	}
	return nil
}

// markFailedOnChain handles a transaction that landed but failed execution. The
// raw on-chain error is classified into a human reason and preserved verbatim.
func (w *StatusPollWorker) markFailedOnChain(ctx context.Context, txID aegis.TransactionID, bundleID, signature string, rawErr json.RawMessage, slot uint64) error {
	class := failure.ClassifyOnChain(string(rawErr), failure.Evidence{Slot: slot})
	return w.markFailed(ctx, txID, bundleID, signature, class.Kind, class.Title, class.RecommendedAction, slot,
		map[string]any{"on_chain_error": string(rawErr)})
}

// markFailed records a terminal failure: it updates the transaction (and bundle)
// status, inserts a failure row with the reason, emits a lifecycle event, and
// broadcasts the change. The poll is not rescheduled after this.
func (w *StatusPollWorker) markFailed(ctx context.Context, txID aegis.TransactionID, bundleID, signature string, kind aegis.FailureKind, title, action string, slot uint64, extra map[string]any) error {
	now := time.Now().UTC()
	_, _ = w.deps.Queries.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusFailed), Stage: string(aegis.StageFailed),
		Signature:   pgtype.Text{String: signature, Valid: signature != ""},
		BundleID:    pgtype.Text{String: bundleID, Valid: bundleID != ""},
		FailureKind: pgtype.Text{String: string(kind), Valid: true},
		FailedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if bundleID != "" {
		_, _ = w.deps.Queries.UpdateBundleStatus(ctx, dbgen.UpdateBundleStatusParams{
			ID: bundleID, Status: "failed", FailedAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
	}

	detail := map[string]any{"reason": title, "slot": slot}
	for k, v := range extra {
		detail[k] = v
	}
	evidence, _ := json.Marshal(detail)
	_, _ = w.deps.Queries.InsertFailure(ctx, dbgen.InsertFailureParams{
		ID: "fail_" + uuid.NewString(), TransactionID: pgtype.Text{String: string(txID), Valid: true},
		BundleID: pgtype.Text{String: bundleID, Valid: bundleID != ""},
		Kind: string(kind), Title: title, Severity: "error",
		Slot:              pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		RecommendedAction: pgtype.Text{String: action, Valid: action != ""},
		Evidence:          evidence,
	})

	meta := map[string]any{"failure_kind": string(kind), "reason": title, "recommended_action": action}
	for k, v := range extra {
		meta[k] = v
	}
	_ = w.deps.Tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(signature), BundleID: aegis.BundleID(bundleID),
		Stage: aegis.StageFailed, Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: meta,
	})
	w.broadcast(txID)
	return nil
}

func (w *StatusPollWorker) reschedule(ctx context.Context, job *river.Job[StatusPollArgs], args StatusPollArgs) error {
	if args.Attempt >= MaxPollAttempts {
		return w.markFailed(ctx, aegis.TransactionID(args.TransactionID), args.BundleID, w.firstSig(args),
			aegis.FailureDropped, "Transaction never reached a terminal state",
			"The transaction was neither confirmed nor rejected within the polling window; resubmit with a fresh blockhash", 0, nil)
	}
	args.Attempt++
	_, err := river.ClientFromContext[pgx.Tx](ctx).Insert(ctx, args, &river.InsertOpts{
		ScheduledAt: time.Now().Add(PendingPollInterval),
	})
	return err
}

func (w *StatusPollWorker) broadcast(txID aegis.TransactionID) {
	if w.deps.Hub == nil {
		return
	}
	txRow, err := w.deps.Queries.GetTransaction(context.Background(), string(txID))
	if err != nil {
		return
	}
	w.deps.Hub.Broadcast("transactions.stream", "transaction.updated", txRow)
}

func mapConfirmation(status string) aegis.LifecycleStage {
	switch status {
	case "processed":
		return aegis.StageProcessed
	case "confirmed":
		return aegis.StageConfirmed
	case "finalized":
		return aegis.StageFinalized
	default:
		return aegis.StageSubmitted
	}
}

func slotFromPtr(slot *uint64) uint64 {
	if slot == nil {
		return 0
	}
	return *slot
}

type WebhookDeliveryArgs struct {
	DeliveryID string `json:"delivery_id"`
	river.JobArgs
}

func (WebhookDeliveryArgs) Kind() string { return "webhook_delivery" }

type WebhookDeliveryWorker struct {
	river.WorkerDefaults[WebhookDeliveryArgs]
}

func (w *WebhookDeliveryWorker) Work(ctx context.Context, job *river.Job[WebhookDeliveryArgs]) error {
	return nil
}
