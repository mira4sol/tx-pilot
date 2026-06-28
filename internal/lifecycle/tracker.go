package lifecycle

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/storage"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"go.uber.org/zap"
)

type Tracker struct {
	q      *dbgen.Queries
	logger *zap.Logger
	shards int
	chans  []chan eventJob
	wg     sync.WaitGroup
}

type eventJob struct {
	ctx context.Context
	ev  StageEvent
}

type StageEvent struct {
	TransactionID txpilot.TransactionID
	Signature     txpilot.Signature
	BundleID      txpilot.BundleID
	Stage         txpilot.LifecycleStage
	Slot          txpilot.Slot
	LatencyMS     *int64
	Metadata      map[string]any
	Timestamp     time.Time
}

func NewTracker(q *dbgen.Queries, logger *zap.Logger, shards int) *Tracker {
	if shards <= 0 {
		shards = 8
	}
	t := &Tracker{q: q, logger: logger, shards: shards, chans: make([]chan eventJob, shards)}
	for i := 0; i < shards; i++ {
		t.chans[i] = make(chan eventJob, 256)
		idx := i
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			for job := range t.chans[idx] {
				if err := t.process(job.ctx, job.ev); err != nil {
					storage.LogDBOp(t.logger, "InsertLifecycleEvent.async", err,
						zap.String("transaction_id", string(job.ev.TransactionID)),
						zap.String("stage", string(job.ev.Stage)),
					)
				}
			}
		}()
	}
	return t
}

func (t *Tracker) shardKey(id string) int {
	var h uint32
	for i := 0; i < len(id); i++ {
		h = h*16777619 ^ uint32(id[i])
	}
	return int(h % uint32(t.shards))
}

func (t *Tracker) isSyncStage(stage txpilot.LifecycleStage) bool {
	switch stage {
	case txpilot.StageCreated, txpilot.StageSubmitted, txpilot.StageFailed,
		txpilot.StageProcessed, txpilot.StageConfirmed, txpilot.StageFinalized:
		return true
	default:
		return false
	}
}

func (t *Tracker) Emit(ctx context.Context, ev StageEvent) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if t.isSyncStage(ev.Stage) {
		return t.process(ctx, ev)
	}
	select {
	case t.chans[t.shardKey(string(ev.TransactionID))] <- eventJob{ctx: ctx, ev: ev}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *Tracker) Close() {
	for _, ch := range t.chans {
		close(ch)
	}
	t.wg.Wait()
}

func (t *Tracker) process(ctx context.Context, ev StageEvent) error {
	meta, _ := json.Marshal(ev.Metadata)
	latency := pgtype.Int8{}
	if ev.LatencyMS != nil {
		latency = pgtype.Int8{Int64: *ev.LatencyMS, Valid: true}
	} else if prior := t.priorTimestamp(ctx, string(ev.TransactionID), ev.Stage); !prior.IsZero() {
		if delta := DeltaMS(prior, ev.Timestamp); delta != nil {
			latency = pgtype.Int8{Int64: *delta, Valid: true}
			ev.LatencyMS = delta
		}
	}
	_, err := t.q.InsertLifecycleEvent(ctx, dbgen.InsertLifecycleEventParams{
		ID: uuid.NewString(), TransactionID: string(ev.TransactionID),
		Signature: pgtype.Text{String: string(ev.Signature), Valid: ev.Signature != ""},
		BundleID:  pgtype.Text{String: string(ev.BundleID), Valid: ev.BundleID != ""},
		Stage:     string(ev.Stage), Slot: pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0},
		LatencyMs: latency, Metadata: meta,
	})
	if err != nil {
		storage.LogDBOp(t.logger, "InsertLifecycleEvent", err,
			zap.String("transaction_id", string(ev.TransactionID)),
			zap.String("stage", string(ev.Stage)),
			zap.Uint64("slot", uint64(ev.Slot)),
		)
		return err
	}

	_, err = t.q.GetTransaction(ctx, string(ev.TransactionID))
	if err != nil {
		storage.LogDBOp(t.logger, "GetTransaction.lifecycle", err,
			zap.String("transaction_id", string(ev.TransactionID)),
		)
		return err
	}

	params := dbgen.UpdateTransactionStatusParams{ID: string(ev.TransactionID), Status: string(ev.Stage), Stage: string(ev.Stage)}
	if ev.Signature != "" {
		params.Signature = pgtype.Text{String: string(ev.Signature), Valid: true}
	}
	if ev.BundleID != "" {
		params.BundleID = pgtype.Text{String: string(ev.BundleID), Valid: true}
	}
	switch ev.Stage {
	case txpilot.StageSubmitted:
		params.SubmittedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.SubmittedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case txpilot.StageProcessed:
		params.ProcessedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.ProcessedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case txpilot.StageConfirmed:
		params.ConfirmedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.ConfirmedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case txpilot.StageFinalized:
		params.FinalizedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.FinalizedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case txpilot.StageFailed:
		params.FailedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		if ev.Slot > 0 {
			params.SubmittedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: true}
		}
	}
	_, err = t.q.UpdateTransactionStatus(ctx, params)
	if err != nil {
		storage.LogDBOp(t.logger, "UpdateTransactionStatus.lifecycle", err,
			zap.String("transaction_id", string(ev.TransactionID)),
			zap.String("stage", string(ev.Stage)),
			zap.Uint64("slot", uint64(ev.Slot)),
		)
		return err
	}
	storage.LogDBOp(t.logger, "lifecycle stage committed", nil,
		zap.String("transaction_id", string(ev.TransactionID)),
		zap.String("stage", string(ev.Stage)),
		zap.Uint64("slot", uint64(ev.Slot)),
	)
	return nil
}

func DeltaMS(from, to time.Time) *int64 {
	if from.IsZero() || to.IsZero() {
		return nil
	}
	ms := to.Sub(from).Milliseconds()
	return &ms
}

func (t *Tracker) priorTimestamp(ctx context.Context, txID string, stage txpilot.LifecycleStage) time.Time {
	row, err := t.q.GetTransaction(ctx, txID)
	if err != nil {
		return time.Time{}
	}
	switch stage {
	case txpilot.StageSubmitted:
		return row.CreatedAt.Time
	case txpilot.StageProcessed:
		if row.SubmittedAt.Valid {
			return row.SubmittedAt.Time
		}
	case txpilot.StageConfirmed:
		if row.ProcessedAt.Valid {
			return row.ProcessedAt.Time
		}
		if row.SubmittedAt.Valid {
			return row.SubmittedAt.Time
		}
	case txpilot.StageFinalized:
		if row.ConfirmedAt.Valid {
			return row.ConfirmedAt.Time
		}
	case txpilot.StageFailed:
		if row.SubmittedAt.Valid {
			return row.SubmittedAt.Time
		}
		return row.CreatedAt.Time
	}
	return time.Time{}
}
