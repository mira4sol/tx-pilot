package notify

import "github.com/riverqueue/river"

// WebhookDeliveryArgs is a durable River job for signed webhook delivery.
type WebhookDeliveryArgs struct {
	DeliveryID     string `json:"delivery_id"`
	EventType      string `json:"event_type"`
	IdempotencyKey string `json:"idempotency_key"`
	Payload        []byte `json:"payload"`
	river.JobArgs
}

func (WebhookDeliveryArgs) Kind() string { return "webhook_delivery" }
