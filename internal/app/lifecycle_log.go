package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func (cp *ControlPlane) ExportLifecycleLog(ctx context.Context, limit int32) ([]txpilot.LifecycleLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	txs, err := cp.q.ListRecentTransactions(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]txpilot.LifecycleLogEntry, 0, len(txs))
	for _, txRow := range txs {
		entry := txpilot.LifecycleLogEntry{
			TransactionID:         txpilot.TransactionID(txRow.ID),
			SubmissionKind:        txpilot.SubmissionKind(txRow.SubmissionKind),
			Status:                txRow.Status,
			Stage:                 txpilot.LifecycleStage(txRow.Stage),
			TipLamports:           txRow.TipLamports,
			RetryAttempt:          txRow.RetryAttempt,
			CommitmentProgression: []string{},
		}
		if txRow.Signature.Valid {
			entry.Signature = txpilot.Signature(txRow.Signature.String)
		}
		if txRow.BundleID.Valid {
			entry.BundleID = txpilot.BundleID(txRow.BundleID.String)
		}
		if txRow.SubmittedSlot.Valid {
			entry.SubmittedSlot = uint64(txRow.SubmittedSlot.Int64)
		}
		if txRow.ProcessedSlot.Valid {
			entry.ProcessedSlot = uint64(txRow.ProcessedSlot.Int64)
		}
		if txRow.ConfirmedSlot.Valid {
			entry.ConfirmedSlot = uint64(txRow.ConfirmedSlot.Int64)
		}
		if txRow.FinalizedSlot.Valid {
			entry.FinalizedSlot = uint64(txRow.FinalizedSlot.Int64)
		}
		if txRow.Leader.Valid {
			entry.Leader = txRow.Leader.String
		}
		if txRow.FailureKind.Valid {
			entry.FailureKind = txpilot.FailureKind(txRow.FailureKind.String)
		}
		entry.SubmittedAt = tsPtr(txRow.SubmittedAt)
		entry.ProcessedAt = tsPtr(txRow.ProcessedAt)
		entry.ConfirmedAt = tsPtr(txRow.ConfirmedAt)
		entry.FinalizedAt = tsPtr(txRow.FinalizedAt)
		entry.FailedAt = tsPtr(txRow.FailedAt)

		if txRow.SubmittedAt.Valid && txRow.ConfirmedAt.Valid {
			ms := txRow.ConfirmedAt.Time.Sub(txRow.SubmittedAt.Time).Milliseconds()
			entry.LatencyConfirmedMS = &ms
		}
		if txRow.SubmittedAt.Valid && txRow.ProcessedAt.Valid {
			ms := txRow.ProcessedAt.Time.Sub(txRow.SubmittedAt.Time).Milliseconds()
			entry.LatencyProcessedMS = &ms
		}
		if txRow.ProcessedAt.Valid && txRow.ConfirmedAt.Valid {
			ms := txRow.ConfirmedAt.Time.Sub(txRow.ProcessedAt.Time).Milliseconds()
			entry.LatencySubmittedMS = &ms
		}

		events, _ := cp.q.ListLifecycleEvents(ctx, txRow.ID)
		entry.CommitmentProgression = progressionFromEvents(events)

		failures, _ := cp.q.ListRecentFailures(ctx, 20)
		for _, f := range failures {
			if f.TransactionID.Valid && f.TransactionID.String == txRow.ID {
				entry.FailureTitle = f.Title
				break
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func progressionFromEvents(events []dbgen.LifecycleEvent) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, ev := range events {
		if seen[ev.Stage] {
			continue
		}
		seen[ev.Stage] = true
		out = append(out, ev.Stage)
	}
	return out
}

func jsonEncode(v any) ([]byte, error) {
	return json.Marshal(v)
}
