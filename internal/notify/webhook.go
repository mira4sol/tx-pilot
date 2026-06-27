package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mira4sol/aegis/internal/config"
)

type WebhookClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewWebhookClient(cfg *config.Config) *WebhookClient {
	return &WebhookClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type WebhookPayload struct {
	EventType      string          `json:"event_type"`
	IdempotencyKey string          `json:"idempotency_key"`
	ServerTime     time.Time       `json:"server_time"`
	Data           json.RawMessage `json:"data"`
}

func (c *WebhookClient) Enabled() bool {
	return c.cfg.WebhooksEnabled && c.cfg.WebhookURL != ""
}

func (c *WebhookClient) Deliver(ctx context.Context, eventType, idempotencyKey string, data any) error {
	if !c.Enabled() {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	payload := WebhookPayload{
		EventType:      eventType,
		IdempotencyKey: idempotencyKey,
		ServerTime:     time.Now().UTC(),
		Data:           raw,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aegis-Idempotency-Key", idempotencyKey)
	req.Header.Set("X-Aegis-Event-Type", eventType)
	if c.cfg.WebhookSigningSecret != "" {
		mac := hmac.New(sha256.New, []byte(c.cfg.WebhookSigningSecret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Aegis-Signature", "sha256="+sig)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook delivery status %d", resp.StatusCode)
	}
	return nil
}
