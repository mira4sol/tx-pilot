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
	"github.com/mira4sol/aegis/internal/storage"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/stream"
	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"go.uber.org/zap"
)

const (
	// PendingPollInterval is how often we re-check a still-pending submission.
	// Solana slots are ~400ms and a transaction typically confirms within ~5s,
	// so a 2s cadence catches landings quickly without hammering the RPC or the
	// rate-limited Jito endpoint.
	PendingPollInterval = 2 * time.Second
	// FinalizePollInterval is used after a transaction is already confirmed and
	// we are only waiting for finalization (~13s), so we can poll less often.
	FinalizePollInterval = 3 * time.Second
	// MaxPollWindow caps the total time we will poll a single submission. A
	// transaction that has not reached a terminal state within this window is
	// treated as dropped — on mainnet a bundle either lands within a couple of
	// slots or its blockhash expires well before this.
	MaxPollWindow = 5 * time.Minute
	// InvalidGracePeriod is how long a freshly-submitted bundle may read as
	// "Invalid" on Jito inflight status before we treat it as dropped. Jito
	// needs a moment to register the bundle across regions.
	InvalidGracePeriod = 15 * time.Second
	// MaxPollAttempts is a hard safety cap that backstops MaxPollWindow.
	MaxPollAttempts = 150
)

type WorkerDeps struct {
	Queries   *dbgen.Queries
	RPC       *aegisrpc.Client
	Jito      *bundle.JitoClient
	Tracker   *lifecycle.Tracker
	Notify    **notify.Dispatcher
	Webhook   *notify.WebhookClient
	Logger    *zap.Logger
	SlotState *stream.SlotState
	Recovery  RecoveryTrigger
}

type RecoveryTrigger interface {
	Trigger(ctx context.Context, txID aegis.TransactionID, kind aegis.FailureKind, title, action string, rc RecoveryContext)
}

type RecoveryContext struct {
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

type RiverQueue struct {
	client *river.Client[pgx.Tx]
}

func NewRiverQueue(ctx context.Context, pool *pgxpool.Pool, deps WorkerDeps, logger *zap.Logger) (*RiverQueue, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &StatusPollWorker{deps: deps})
	river.AddWorker(workers, &WebhookDeliveryWorker{webhook: deps.Webhook})

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
	if logger != nil {
		logger.Info("river queue started",
			zap.Strings("workers", []string{"status_poll", "webhook_delivery"}),
		)
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
	TransactionID       string   `json:"transaction_id"`
	SubmissionKind      string   `json:"submission_kind"`
	BundleID            string   `json:"bundle_id"`
	Signatures          []string `json:"signatures"`
	Blockhash           string   `json:"blockhash"`
	Attempt             int      `json:"attempt"`
	FirstPolledAtUnixMS int64    `json:"first_polled_at_unix_ms"`
	river.JobArgs
}

// pollDeadlineExceeded reports whether the total polling window for a
// submission has elapsed.
func (a StatusPollArgs) pollDeadlineExceeded() bool {
	if a.FirstPolledAtUnixMS == 0 {
		return false
	}
	return time.Since(time.UnixMilli(a.FirstPolledAtUnixMS)) > MaxPollWindow
}

// pollElapsed returns how long we have been polling this submission.
func (a StatusPollArgs) pollElapsed() time.Duration {
	if a.FirstPolledAtUnixMS == 0 {
		return 0
	}
	return time.Since(time.UnixMilli(a.FirstPolledAtUnixMS))
}

func (StatusPollArgs) Kind() string { return "status_poll" }

type StatusPollWorker struct {
	river.WorkerDefaults[StatusPollArgs]
	deps WorkerDeps
}

func (w *StatusPollWorker) Work(ctx context.Context, job *river.Job[StatusPollArgs]) error {
	args := job.Args
	txID := aegis.TransactionID(args.TransactionID)
	sig := w.firstSig(args)
	if w.deps.Logger != nil {
		w.deps.Logger.Info("status poll tick",
			zap.String("transaction_id", string(txID)),
			zap.String("bundle_id", args.BundleID),
			zap.String("signature", sig),
			zap.Int("poll_attempt", args.Attempt),
			zap.Duration("elapsed", args.pollElapsed()),
			zap.String("submission_kind", args.SubmissionKind),
		)
	}

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
		return w.reschedule(ctx, job, args, "rpc_error")
	}

	if len(result.Value) > 0 && result.Value[0] != nil {
		st := result.Value[0]
		// The signature landed. If it carries an execution error it is a
		// terminal on-chain failure — stop immediately with the real reason.
		if isExecutionError(st.Err) {
			return w.markFailedOnChain(ctx, txID, args, args.Signatures[0], st.Err, slotFromPtr(st.Slot))
		}
		if st.ConfirmationStatus != "" {
			return w.applyConfirmation(ctx, job, txID, args.BundleID, args.Signatures[0], st.ConfirmationStatus, slotFromPtr(st.Slot))
		}
		return w.reschedule(ctx, job, args, "not_yet_processed")
	}

	if w.blockhashExpired(ctx, args.Blockhash) {
		return w.markFailed(ctx, txID, args, args.Signatures[0],
			aegis.FailureExpiredBlockhash, "Blockhash expired before the transaction landed",
			"Refresh the blockhash and resubmit; the transaction was never included", 0, nil)
	}
	return w.reschedule(ctx, job, args, "signature_not_found")
}

func (w *StatusPollWorker) pollBundle(ctx context.Context, job *river.Job[StatusPollArgs], txID aegis.TransactionID, args StatusPollArgs) error {
	// The on-chain signature status is the source of truth for landing. Jito's
	// sendTransaction path lands the tx while inflight bundle status can still
	// read "Invalid", so check the signature first: a confirmed signature means
	// the bundle landed, and a signature carrying an execution error is a
	// terminal on-chain failure.
	if len(args.Signatures) > 0 {
		if result, err := w.deps.RPC.GetSignatureStatuses(ctx, args.Signatures, false); err == nil &&
			len(result.Value) > 0 && result.Value[0] != nil {
			st := result.Value[0]
			if isExecutionError(st.Err) {
				return w.markFailedOnChain(ctx, txID, args, args.Signatures[0], st.Err, slotFromPtr(st.Slot))
			}
			if st.ConfirmationStatus != "" {
				return w.applyConfirmation(ctx, job, txID, args.BundleID, args.Signatures[0], st.ConfirmationStatus, slotFromPtr(st.Slot))
			}
		}
	}

	// A definitively expired blockhash means the bundle can never land. Check
	// this before the inflight "Invalid" fast-fail so injected/expired blockhash
	// failures are classified accurately rather than as a generic rejection.
	if w.blockhashExpired(ctx, args.Blockhash) {
		return w.markFailed(ctx, txID, args, w.firstSig(args),
			aegis.FailureExpiredBlockhash, "Blockhash expired before the bundle landed",
			"Refresh the blockhash and resubmit the bundle", 0, nil)
	}

	// getBundleStatuses is the authoritative source once a bundle lands: it
	// returns the bundle's transactions, landing slot, and confirmation status.
	statuses, err := w.deps.Jito.GetBundleStatuses(ctx, []string{args.BundleID})
	if err == nil && len(statuses.Value) > 0 && statuses.Value[0].ConfirmationStatus != "" {
		bs := statuses.Value[0]
		sig := w.firstSig(args)
		if len(bs.Transactions) > 0 {
			sig = bs.Transactions[0]
		}
		return w.applyConfirmation(ctx, job, txID, args.BundleID, sig, bs.ConfirmationStatus, bs.Slot)
	}

	// getInflightBundleStatuses reports the auction-level state for a bundle that
	// has not yet been verified as landed. Per Jito docs the status is one of:
	// Invalid, Pending, Failed, Landed.
	inflight, ierr := w.deps.Jito.GetInflightBundleStatuses(ctx, []string{args.BundleID})
	if ierr == nil && len(inflight.Value) > 0 {
		switch inflight.Value[0].Status {
		case "Failed":
			return w.markFailed(ctx, txID, args, w.firstSig(args),
				aegis.FailureBundleRejected, "Bundle rejected by Jito auction",
				"Recalculate tip above the dynamic floor and resubmit", 0, nil)
		case "Landed":
			if len(args.Signatures) > 0 {
				result, rerr := w.deps.RPC.GetSignatureStatuses(ctx, args.Signatures, false)
				if rerr == nil && len(result.Value) > 0 && result.Value[0] != nil && result.Value[0].ConfirmationStatus != "" {
					return w.applyConfirmation(ctx, job, txID, args.BundleID, args.Signatures[0], result.Value[0].ConfirmationStatus, slotFromPtr(result.Value[0].Slot))
				}
			}
		case "Invalid":
			// Invalid means Jito has no record of the bundle in its 5-minute
			// lookback. Right after submission this can be propagation lag, but
			// past the grace period it means the bundle was never accepted or was
			// dropped by the auction, so fail fast instead of waiting for expiry.
			if args.pollElapsed() > InvalidGracePeriod {
				return w.markFailed(ctx, txID, args, w.firstSig(args),
					aegis.FailureBundleRejected, "Bundle dropped by Jito (inflight status Invalid)",
					"Resubmit with a fresh blockhash and a competitive tip", 0,
					map[string]any{"inflight_status": "Invalid"})
			}
		}
	}

	return w.reschedule(ctx, job, args, "bundle_pending")
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
	leader := w.leaderAt(slot)

	if w.deps.Logger != nil {
		w.deps.Logger.Info("poll confirmation",
			zap.String("transaction_id", string(txID)),
			zap.String("confirmation_status", confirmation),
			zap.String("stage", string(stage)),
			zap.Uint64("slot", slot),
			zap.String("leader", leader),
			zap.String("source", "rpc_poll"),
		)
	}

	txRow, _ := w.deps.Queries.GetTransaction(ctx, string(txID))
	if stage != aegis.StageProcessed && stage != aegis.StageSubmitted && txRow.ID != "" && !txRow.ProcessedAt.Valid {
		if err := w.emitStage(ctx, txID, bundleID, signature, aegis.StageProcessed, slot, leader, now); err == nil {
			now = time.Now().UTC()
		}
	}

	params := dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: confirmation, Stage: string(stage),
		Signature: pgtype.Text{String: signature, Valid: signature != ""},
		BundleID:  pgtype.Text{String: bundleID, Valid: bundleID != ""},
		Leader:    pgtype.Text{String: leader, Valid: leader != ""},
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
			_, bundleErr := w.deps.Queries.UpdateBundleStatus(ctx, dbgen.UpdateBundleStatusParams{
				ID: bundleID, Status: "landed", LandedAt: pgtype.Timestamptz{Time: now, Valid: true},
			})
			storage.LogDBOp(w.deps.Logger, "UpdateBundleStatus.landed", bundleErr,
				zap.String("transaction_id", string(txID)),
				zap.String("bundle_id", bundleID),
			)
		}
	}
	_, txErr := w.deps.Queries.UpdateTransactionStatus(ctx, params)
	storage.LogDBOp(w.deps.Logger, "UpdateTransactionStatus.poll", txErr,
		zap.String("transaction_id", string(txID)),
		zap.String("stage", string(stage)),
		zap.Uint64("slot", slot),
	)
	if emitErr := w.deps.Tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(signature),
		BundleID: aegis.BundleID(bundleID), Stage: stage,
		Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: map[string]any{"leader": leader, "source": "rpc_poll"},
	}); emitErr != nil {
		storage.LogDBOp(w.deps.Logger, "EmitLifecycle.poll", emitErr,
			zap.String("transaction_id", string(txID)),
			zap.String("stage", string(stage)),
		)
	}
	w.broadcast(txID)
	if stage != aegis.StageFinalized {
		args := job.Args
		if args.FirstPolledAtUnixMS == 0 {
			args.FirstPolledAtUnixMS = time.Now().UnixMilli()
		}
		if !args.pollDeadlineExceeded() {
			args.Attempt++
			if client := river.ClientFromContext[pgx.Tx](ctx); client != nil {
				_, insertErr := client.Insert(ctx, args, &river.InsertOpts{
					ScheduledAt: time.Now().Add(FinalizePollInterval),
				})
				if insertErr != nil {
					storage.LogDBOp(w.deps.Logger, "InsertStatusPoll.finalize", insertErr,
						zap.String("transaction_id", string(txID)),
					)
				} else if w.deps.Logger != nil {
					w.deps.Logger.Info("status poll rescheduled for finalization",
						zap.String("transaction_id", string(txID)),
						zap.Duration("delay", FinalizePollInterval),
					)
				}
			}
		}
	}
	return nil
}

// markFailedOnChain handles a transaction that landed but failed execution. The
// raw on-chain error is classified into a human reason and preserved verbatim.
func (w *StatusPollWorker) markFailedOnChain(ctx context.Context, txID aegis.TransactionID, args StatusPollArgs, signature string, rawErr json.RawMessage, slot uint64) error {
	if w.deps.Logger != nil {
		w.deps.Logger.Info("on-chain execution failure",
			zap.String("transaction_id", string(txID)),
			zap.String("signature", signature),
			zap.String("on_chain_error", string(rawErr)),
			zap.Uint64("slot", slot),
		)
	}
	class := failure.ClassifyOnChain(string(rawErr), failure.Evidence{Slot: slot})
	return w.markFailed(ctx, txID, args, signature, class.Kind, class.Title, class.RecommendedAction, slot,
		map[string]any{"on_chain_error": string(rawErr)})
}

// markFailed records a terminal failure: it updates the transaction (and bundle)
// status, inserts a failure row with the reason, emits a lifecycle event, and
// broadcasts the change. The poll is not rescheduled after this.
func (w *StatusPollWorker) markFailed(ctx context.Context, txID aegis.TransactionID, args StatusPollArgs, signature string, kind aegis.FailureKind, title, action string, slot uint64, extra map[string]any) error {
	now := time.Now().UTC()
	if slot == 0 {
		slot = w.currentSlot()
	}
	leader := w.leaderAt(slot)
	tipLamports := int64(0)
	var txRow dbgen.Transaction
	if row, err := w.deps.Queries.GetTransaction(ctx, string(txID)); err == nil {
		txRow = row
		tipLamports = row.TipLamports
	}

	if w.deps.Logger != nil {
		w.deps.Logger.Info("poll failure recorded",
			zap.String("transaction_id", string(txID)),
			zap.String("failure_kind", string(kind)),
			zap.String("title", title),
			zap.String("recommended_action", action),
			zap.Uint64("slot", slot),
			zap.String("source", "rpc_poll"),
		)
	}

	_, txErr := w.deps.Queries.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(aegis.StatusFailed), Stage: string(aegis.StageFailed),
		Signature:     pgtype.Text{String: signature, Valid: signature != ""},
		BundleID:      pgtype.Text{String: args.BundleID, Valid: args.BundleID != ""},
		TipLamports:   tipLamports,
		FailureKind:   pgtype.Text{String: string(kind), Valid: true},
		SubmittedSlot: pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		Leader:        pgtype.Text{String: leader, Valid: leader != ""},
		FailedAt:      pgtype.Timestamptz{Time: now, Valid: true},
	})
	storage.LogDBOp(w.deps.Logger, "UpdateTransactionStatus.failed", txErr,
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(kind)),
	)
	if args.BundleID != "" {
		_, bundleErr := w.deps.Queries.UpdateBundleStatus(ctx, dbgen.UpdateBundleStatusParams{
			ID: args.BundleID, Status: "failed", FailedAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		storage.LogDBOp(w.deps.Logger, "UpdateBundleStatus.failed", bundleErr,
			zap.String("transaction_id", string(txID)),
			zap.String("bundle_id", args.BundleID),
		)
	}

	detail := map[string]any{"reason": title, "slot": slot}
	for k, v := range extra {
		detail[k] = v
	}
	evidence, _ := json.Marshal(detail)
	_, failErr := w.deps.Queries.InsertFailure(ctx, dbgen.InsertFailureParams{
		ID: "fail_" + uuid.NewString(), TransactionID: pgtype.Text{String: string(txID), Valid: true},
		BundleID: pgtype.Text{String: args.BundleID, Valid: args.BundleID != ""},
		Kind:     string(kind), Title: title, Severity: "error",
		Slot:              pgtype.Int8{Int64: int64(slot), Valid: slot > 0},
		RecommendedAction: pgtype.Text{String: action, Valid: action != ""},
		Evidence:          evidence,
	})
	storage.LogDBOp(w.deps.Logger, "InsertFailure", failErr,
		zap.String("transaction_id", string(txID)),
		zap.String("failure_kind", string(kind)),
		zap.String("title", title),
	)

	meta := map[string]any{"failure_kind": string(kind), "reason": title, "recommended_action": action}
	for k, v := range extra {
		meta[k] = v
	}
	if emitErr := w.deps.Tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(signature), BundleID: aegis.BundleID(args.BundleID),
		Stage: aegis.StageFailed, Slot: aegis.Slot(slot), Timestamp: now,
		Metadata: meta,
	}); emitErr != nil {
		storage.LogDBOp(w.deps.Logger, "EmitLifecycle.failed", emitErr,
			zap.String("transaction_id", string(txID)),
		)
	}
	w.broadcast(txID)

	if w.deps.Recovery != nil && txRow.ID != "" {
		memo := ""
		if txRow.Memo.Valid {
			memo = txRow.Memo.String
		}
		sigs := args.Signatures
		if len(sigs) == 0 && signature != "" {
			sigs = []string{signature}
		}
		w.deps.Recovery.Trigger(ctx, txID, kind, title, action, RecoveryContext{
			TransactionID:  txID,
			SubmissionKind: aegis.SubmissionKind(args.SubmissionKind),
			BundleID:       args.BundleID,
			Signatures:     sigs,
			Blockhash:      args.Blockhash,
			RetryAttempt:   txRow.RetryAttempt,
			OpsMemo:        memo,
			OpsLamports:    opsLamportsFromMemo(memo),
			PolicyMode:     aegis.PolicyMode(txRow.PolicyMode),
		})
	}
	return nil
}

func (w *StatusPollWorker) reschedule(ctx context.Context, job *river.Job[StatusPollArgs], args StatusPollArgs, reason string) error {
	if args.FirstPolledAtUnixMS == 0 {
		args.FirstPolledAtUnixMS = time.Now().UnixMilli()
	}
	if args.pollDeadlineExceeded() || args.Attempt >= MaxPollAttempts {
		return w.markFailed(ctx, aegis.TransactionID(args.TransactionID), args, w.firstSig(args),
			aegis.FailureDropped, "Transaction never reached a terminal state within the polling window",
			"The transaction was neither confirmed nor rejected within 5m; resubmit with a fresh blockhash", 0,
			map[string]any{"poll_window": MaxPollWindow.String(), "attempts": args.Attempt})
	}
	args.Attempt++
	_, err := river.ClientFromContext[pgx.Tx](ctx).Insert(ctx, args, &river.InsertOpts{
		ScheduledAt: time.Now().Add(PendingPollInterval),
	})
	if err != nil {
		storage.LogDBOp(w.deps.Logger, "InsertStatusPoll.reschedule", err,
			zap.String("transaction_id", args.TransactionID),
		)
		return err
	}
	if w.deps.Logger != nil {
		w.deps.Logger.Info("status poll rescheduled",
			zap.String("transaction_id", args.TransactionID),
			zap.Int("poll_attempt", args.Attempt),
			zap.Duration("delay", PendingPollInterval),
			zap.String("reason", reason),
		)
	}
	return nil
}

func opsLamportsFromMemo(memo string) uint64 {
	// Ops txs default to 1 lamport when amount is not stored separately.
	if memo == "" {
		return 0
	}
	return 1
}

func (w *StatusPollWorker) broadcast(txID aegis.TransactionID) {
	if w.deps.Notify == nil || *w.deps.Notify == nil {
		return
	}
	txRow, err := w.deps.Queries.GetTransaction(context.Background(), string(txID))
	if err != nil {
		return
	}
	(*w.deps.Notify).BroadcastTransaction(string(txID), txRow.Stage, txRow)
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

func (w *StatusPollWorker) leaderAt(slot uint64) string {
	if w.deps.SlotState == nil {
		return ""
	}
	return w.deps.SlotState.LeaderAt(slot)
}

func (w *StatusPollWorker) currentSlot() uint64 {
	if w.deps.SlotState == nil {
		return 0
	}
	return w.deps.SlotState.CurrentSlot()
}

func (w *StatusPollWorker) emitStage(ctx context.Context, txID aegis.TransactionID, bundleID, signature string, stage aegis.LifecycleStage, slot uint64, leader string, ts time.Time) error {
	params := dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: string(stage), Stage: string(stage),
		Signature: pgtype.Text{String: signature, Valid: signature != ""},
		BundleID:  pgtype.Text{String: bundleID, Valid: bundleID != ""},
		Leader:    pgtype.Text{String: leader, Valid: leader != ""},
	}
	switch stage {
	case aegis.StageProcessed:
		params.ProcessedAt = pgtype.Timestamptz{Time: ts, Valid: true}
		params.ProcessedSlot = pgtype.Int8{Int64: int64(slot), Valid: slot > 0}
	}
	_, err := w.deps.Queries.UpdateTransactionStatus(ctx, params)
	if err != nil {
		storage.LogDBOp(w.deps.Logger, "UpdateTransactionStatus.backfill", err,
			zap.String("transaction_id", string(txID)),
			zap.String("stage", string(stage)),
		)
		return err
	}
	return w.deps.Tracker.Emit(ctx, lifecycle.StageEvent{
		TransactionID: txID, Signature: aegis.Signature(signature), BundleID: aegis.BundleID(bundleID),
		Stage: stage, Slot: aegis.Slot(slot), Timestamp: ts,
		Metadata: map[string]any{"leader": leader, "backfill": true},
	})
}

type WebhookDeliveryWorker struct {
	river.WorkerDefaults[notify.WebhookDeliveryArgs]
	webhook *notify.WebhookClient
}

func (w *WebhookDeliveryWorker) Work(ctx context.Context, job *river.Job[notify.WebhookDeliveryArgs]) error {
	if w.webhook == nil || !w.webhook.Enabled() {
		return nil
	}
	var data any
	if len(job.Args.Payload) > 0 {
		_ = json.Unmarshal(job.Args.Payload, &data)
	}
	return w.webhook.Deliver(ctx, job.Args.EventType, job.Args.IdempotencyKey, data)
}
