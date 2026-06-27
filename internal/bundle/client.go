package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/mira4sol/aegis/pkg/aegis"
)

type JitoClient struct {
	baseURL     string
	httpClient  *http.Client
	tipAccounts []solana.PublicKey
}

func NewJitoClient(baseURL string) *JitoClient {
	return &JitoClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

type jitoRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type jitoRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *JitoClient) endpointFor(method string) string {
	switch method {
	case "sendTransaction":
		return c.baseURL + "/transactions"
	case "sendBundle":
		return c.baseURL + "/bundles"
	default:
		return c.baseURL + "/" + method
	}
}

// maxRateLimitRetries bounds how many times a rate-limited request is retried
// with backoff before surfacing the error. The public Jito endpoint enforces a
// low global request rate, so transient 429/"rate limited" responses are common
// under congestion and are safe to retry for idempotent submissions.
const maxRateLimitRetries = 4

// isRateLimited reports whether a Jito response indicates the request was
// throttled, either via HTTP status or the JSON-RPC error message.
func isRateLimited(status int, msg string) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	m := strings.ToLower(msg)
	return strings.Contains(m, "rate limit") || strings.Contains(m, "rate-limited") ||
		strings.Contains(m, "congested")
}

func (c *JitoClient) call(ctx context.Context, method string, params []any, out any) (http.Header, error) {
	body, _ := json.Marshal(jitoRPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})

	var lastErr error
	for attempt := 0; attempt <= maxRateLimitRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff capped at ~4s: 500ms, 1s, 2s, 4s.
			backoff := time.Duration(250*(1<<attempt)) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointFor(method), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("jito %s: rate limited (HTTP 429)", method)
			continue
		}

		var envelope jitoRPCResponse
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return resp.Header, fmt.Errorf("jito %s: decode response: %w", method, err)
		}
		if envelope.Error != nil {
			if isRateLimited(resp.StatusCode, envelope.Error.Message) {
				lastErr = fmt.Errorf("jito %s: %s", method, envelope.Error.Message)
				continue
			}
			return resp.Header, fmt.Errorf("jito %s: %s", method, envelope.Error.Message)
		}
		if out != nil {
			if err := json.Unmarshal(envelope.Result, out); err != nil {
				return resp.Header, err
			}
		}
		return resp.Header, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("jito %s: exhausted retries", method)
	}
	return nil, lastErr
}

func (c *JitoClient) LoadTipAccounts(ctx context.Context) error {
	var accounts []string
	if _, err := c.call(ctx, "getTipAccounts", []any{}, &accounts); err != nil {
		return err
	}
	c.tipAccounts = make([]solana.PublicKey, 0, len(accounts))
	for _, a := range accounts {
		pk, err := solana.PublicKeyFromBase58(a)
		if err != nil {
			continue
		}
		c.tipAccounts = append(c.tipAccounts, pk)
	}
	return nil
}

func (c *JitoClient) TipAccounts() []string {
	out := make([]string, 0, len(c.tipAccounts))
	for _, pk := range c.tipAccounts {
		out = append(out, pk.String())
	}
	return out
}

func (c *JitoClient) PickTipAccount() (solana.PublicKey, error) {
	if len(c.tipAccounts) == 0 {
		return solana.PublicKey{}, fmt.Errorf("no tip accounts loaded")
	}
	return c.tipAccounts[time.Now().UnixNano()%int64(len(c.tipAccounts))], nil
}

func (c *JitoClient) TipAccountsCount() int {
	return len(c.tipAccounts)
}

type SendTransactionResult struct {
	Signature string
	BundleID  string
}

func (c *JitoClient) SendTransaction(ctx context.Context, encodedTx string, encoding aegis.Encoding) (*SendTransactionResult, error) {
	if encoding == "" {
		encoding = aegis.EncodingBase64
	}
	var signature string
	headers, err := c.call(ctx, "sendTransaction", []any{encodedTx, map[string]string{"encoding": string(encoding)}}, &signature)
	if err != nil {
		return nil, err
	}
	return &SendTransactionResult{
		Signature: signature,
		BundleID:  headers.Get("x-bundle-id"),
	}, nil
}

func (c *JitoClient) SendBundle(ctx context.Context, encodedTxs []string, encoding aegis.Encoding) (string, error) {
	if encoding == "" {
		encoding = aegis.EncodingBase64
	}
	var bundleID string
	_, err := c.call(ctx, "sendBundle", []any{encodedTxs, map[string]string{"encoding": string(encoding)}}, &bundleID)
	if err != nil {
		return "", err
	}
	return bundleID, nil
}

type BundleStatus struct {
	BundleID           string   `json:"bundle_id"`
	Transactions       []string `json:"transactions"`
	Slot               uint64   `json:"slot"`
	ConfirmationStatus string   `json:"confirmation_status"`
	Landed             bool     `json:"landed"`
}

type BundleStatusesResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value []BundleStatus `json:"value"`
}

func (c *JitoClient) GetBundleStatuses(ctx context.Context, bundleIDs []string) (*BundleStatusesResult, error) {
	var result BundleStatusesResult
	_, err := c.call(ctx, "getBundleStatuses", []any{bundleIDs}, &result)
	if err != nil {
		return nil, err
	}
	for i := range result.Value {
		result.Value[i].Landed = result.Value[i].ConfirmationStatus != ""
	}
	return &result, nil
}

type InflightBundleStatus struct {
	BundleID   string  `json:"bundle_id"`
	Status     string  `json:"status"`
	LandedSlot *uint64 `json:"landed_slot"`
}

type InflightBundleStatusesResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value []InflightBundleStatus `json:"value"`
}

func (c *JitoClient) GetInflightBundleStatuses(ctx context.Context, bundleIDs []string) (*InflightBundleStatusesResult, error) {
	var result InflightBundleStatusesResult
	_, err := c.call(ctx, "getInflightBundleStatuses", []any{bundleIDs}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
