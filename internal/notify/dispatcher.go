package notify

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

const (
	ChannelTransactions = "transactions.stream"
	ChannelAIDecisions  = "ai.decisions"
	ChannelFailures     = "failures.analysis"
)

// Event types for webhooks and WebSocket fanout.
const (
	EventTransactionSubmitted = "transaction.submitted"
	EventTransactionProcessed = "transaction.processed"
	EventTransactionConfirmed = "transaction.confirmed"
	EventTransactionFinalized = "transaction.finalized"
	EventTransactionFailed    = "transaction.failed"
	EventTransactionRetried   = "transaction.retried"
	EventBundleSubmitted      = "bundle.submitted"
	EventBundleLanded         = "bundle.landed"
	EventBundleFailed         = "bundle.failed"
	EventAgentDecisionMade    = "agent.decision_made"
)

type WebhookEnqueuer interface {
	EnqueueWebhook(ctx context.Context, eventType, idempotencyKey string, payload any) error
}

type Dispatcher struct {
	hub         *Hub
	webhook     *WebhookClient
	idempotency *IdempotencyStore
	enqueuer    WebhookEnqueuer
}

func NewDispatcher(hub *Hub, webhook *WebhookClient, enqueuer WebhookEnqueuer) *Dispatcher {
	return &Dispatcher{
		hub:         hub,
		webhook:     webhook,
		idempotency: NewIdempotencyStore(),
		enqueuer:    enqueuer,
	}
}

func (d *Dispatcher) BroadcastTransaction(txID, stage string, payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelTransactions, "transaction.updated", payload)
	d.emitWebhook(context.Background(), lifecycleEventType(stage), txID, stage, payload)
}

func (d *Dispatcher) BroadcastDecision(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelAIDecisions, EventAgentDecisionMade, payload)
	txID := extractTransactionID(payload)
	d.emitWebhook(context.Background(), EventAgentDecisionMade, txID, "decision", payload)
}

func (d *Dispatcher) BroadcastFailure(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelFailures, "failure.recorded", payload)
	txID := extractTransactionID(payload)
	d.emitWebhook(context.Background(), EventTransactionFailed, txID, "failed", payload)
}

func (d *Dispatcher) emitWebhook(ctx context.Context, eventType, transactionID, stage string, payload any) {
	if d == nil || d.webhook == nil || !d.webhook.Enabled() {
		return
	}
	key := IdempotencyKey(eventType, transactionID, stage)
	if d.idempotency.Seen(key) {
		return
	}
	d.idempotency.Mark(key)
	if d.enqueuer != nil {
		_ = d.enqueuer.EnqueueWebhook(ctx, eventType, key, payload)
	}
}

func lifecycleEventType(stage string) string {
	switch stage {
	case "submitted":
		return EventTransactionSubmitted
	case "processed", "processing":
		return EventTransactionProcessed
	case "confirmed":
		return EventTransactionConfirmed
	case "finalized":
		return EventTransactionFinalized
	case "failed":
		return EventTransactionFailed
	default:
		return "transaction.updated"
	}
}

func extractTransactionID(payload any) string {
	switch v := payload.(type) {
	case map[string]any:
		if id, ok := v["transaction_id"]; ok {
			return fmtAnyString(id)
		}
	default:
		raw, err := json.Marshal(payload)
		if err != nil {
			return ""
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			if id, ok := m["transaction_id"]; ok {
				return fmtAnyString(id)
			}
		}
	}
	return ""
}

func fmtAnyString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	default:
		return ""
	}
}

// RiverEnqueuer delivers webhooks through River durable jobs.
type RiverEnqueuer struct {
	client *river.Client[pgx.Tx]
}

func NewRiverEnqueuer(client *river.Client[pgx.Tx]) *RiverEnqueuer {
	return &RiverEnqueuer{client: client}
}

func (r *RiverEnqueuer) EnqueueWebhook(ctx context.Context, eventType, idempotencyKey string, payload any) error {
	if r == nil || r.client == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.client.Insert(ctx, WebhookDeliveryArgs{
		DeliveryID:     "wh_" + uuid.NewString(),
		EventType:      eventType,
		IdempotencyKey: idempotencyKey,
		Payload:        raw,
	}, &river.InsertOpts{ScheduledAt: time.Now()})
	return err
}
