package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params []any, out any) error {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var envelope rpcResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("rpc %s: %s", method, envelope.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}

type BlockhashResult struct {
	Blockhash string `json:"blockhash"`
}

type GetLatestBlockhashResponse struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value struct {
		Blockhash            string `json:"blockhash"`
		LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
	} `json:"value"`
}

func (c *Client) GetLatestBlockhash(ctx context.Context, commitment string) (*GetLatestBlockhashResponse, error) {
	var out GetLatestBlockhashResponse
	if err := c.call(ctx, "getLatestBlockhash", []any{map[string]string{"commitment": commitment}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSlot(ctx context.Context) (uint64, error) {
	var slot uint64
	if err := c.call(ctx, "getSlot", []any{}, &slot); err != nil {
		return 0, err
	}
	return slot, nil
}

func (c *Client) GetTPS(ctx context.Context) (float64, error) {
	var samples []struct {
		NumTransactions uint64 `json:"numTransactions"`
		Slot            uint64 `json:"slot"`
	}
	if err := c.call(ctx, "getRecentPerformanceSamples", []any{1}, &samples); err != nil {
		return 0, err
	}
	if len(samples) == 0 {
		return 0, nil
	}
	return float64(samples[0].NumTransactions) / 0.4, nil
}

func (c *Client) GetSlotLeaders(ctx context.Context, startSlot uint64, limit uint64) ([]string, error) {
	var leaders []string
	if err := c.call(ctx, "getSlotLeaders", []any{startSlot, limit}, &leaders); err != nil {
		return nil, err
	}
	return leaders, nil
}

type SignatureStatus struct {
	Slot               *uint64         `json:"slot"`
	Confirmations      *uint64         `json:"confirmations"`
	Err                json.RawMessage `json:"err"`
	ConfirmationStatus string          `json:"confirmationStatus"`
}

type SignatureStatusesResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value []*SignatureStatus `json:"value"`
}

func (c *Client) GetSignatureStatuses(ctx context.Context, signatures []string, searchHistory bool) (*SignatureStatusesResult, error) {
	var out SignatureStatusesResult
	if err := c.call(ctx, "getSignatureStatuses", []any{signatures, map[string]bool{"searchTransactionHistory": searchHistory}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type IsBlockhashValidResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value bool `json:"value"`
}

// IsBlockhashValid reports whether a blockhash can still be used to land a
// transaction. Once it returns false, any transaction referencing that
// blockhash can never be included and should be treated as dropped.
func (c *Client) IsBlockhashValid(ctx context.Context, blockhash, commitment string) (bool, error) {
	if commitment == "" {
		commitment = "confirmed"
	}
	var out IsBlockhashValidResult
	if err := c.call(ctx, "isBlockhashValid", []any{blockhash, map[string]string{"commitment": commitment}}, &out); err != nil {
		return false, err
	}
	return out.Value, nil
}

func (c *Client) SendTransaction(ctx context.Context, encodedTx string, encoding string) (string, error) {
	var sig string
	if err := c.call(ctx, "sendTransaction", []any{encodedTx, map[string]any{
		"encoding":            encoding,
		"skipPreflight":       false,
		"preflightCommitment": "processed",
	}}, &sig); err != nil {
		return "", err
	}
	return sig, nil
}

func (c *Client) SimulateTransaction(ctx context.Context, encodedTx string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.call(ctx, "simulateTransaction", []any{encodedTx, map[string]any{
		"encoding":               "base64",
		"commitment":             "processed",
		"replaceRecentBlockhash": false,
	}}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
