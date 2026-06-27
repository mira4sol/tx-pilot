package lifecycle

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/pkg/aegis"
)

type Tracker struct {
	q      *dbgen.Queries
	shards int
	chans  []chan eventJob
	wg     sync.WaitGroup
}

type eventJob struct {
	ctx context.Context
	ev  StageEvent
}

type StageEvent struct {
	TransactionID aegis.TransactionID
	Signature     aegis.Signature
	BundleID      aegis.BundleID
	Stage         aegis.LifecycleStage
	Slot          aegis.Slot
	LatencyMS     *int64
	Metadata      map[string]any
	Timestamp     time.Time
}

func NewTracker(q *dbgen.Queries, shards int) *Tracker {
	if shards <= 0 {
		shards = 8
	}
	t := &Tracker{q: q, shards: shards, chans: make([]chan eventJob, shards)}
	for i := 0; i < shards; i++ {
		t.chans[i] = make(chan eventJob, 256)
		idx := i
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			for job := range t.chans[idx] {
				_ = t.process(job.ctx, job.ev)
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

func (t *Tracker) Emit(ctx context.Context, ev StageEvent) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	// Critical stages are processed synchronously to guarantee timeline evidence.
	if ev.Stage == aegis.StageCreated || ev.Stage == aegis.StageSubmitted || ev.Stage == aegis.StageFailed {
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
	}
	_, err := t.q.InsertLifecycleEvent(ctx, dbgen.InsertLifecycleEventParams{
		ID: uuid.NewString(), TransactionID: string(ev.TransactionID),
		Signature: pgtype.Text{String: string(ev.Signature), Valid: ev.Signature != ""},
		BundleID:  pgtype.Text{String: string(ev.BundleID), Valid: ev.BundleID != ""},
		Stage: string(ev.Stage), Slot: pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0},
		LatencyMs: latency, Metadata: meta,
	})
	if err != nil {
		return err
	}

	_, err = t.q.GetTransaction(ctx, string(ev.TransactionID))
	if err != nil {
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
	case aegis.StageSubmitted:
		params.SubmittedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.SubmittedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case aegis.StageProcessed:
		params.ProcessedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.ProcessedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case aegis.StageConfirmed:
		params.ConfirmedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.ConfirmedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case aegis.StageFinalized:
		params.FinalizedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
		params.FinalizedSlot = pgtype.Int8{Int64: int64(ev.Slot), Valid: ev.Slot > 0}
	case aegis.StageFailed:
		params.FailedAt = pgtype.Timestamptz{Time: ev.Timestamp, Valid: true}
	}
	_, err = t.q.UpdateTransactionStatus(ctx, params)
	return err
}

func DeltaMS(from, to time.Time) *int64 {
	if from.IsZero() || to.IsZero() {
		return nil
	}
	ms := to.Sub(from).Milliseconds()
	return &ms
}
