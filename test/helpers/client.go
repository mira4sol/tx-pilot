// Package helpers provides shared utilities for live HTTP integration tests.
// Run with: go test -tags=integration ./test/...
package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/mira4sol/tx-pilot/internal/tx"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

const (
	TestKeypairFile     = "tx-pilot-test-keypair.json"
	DevRecipient        = "5SEZmBS8s41cJ8g3gmLS1BexujHcNZHe5qznPJMdVUsh"
	DevTransferLamports = uint64(1)
)

func BaseURL() string {
	if v := os.Getenv("TX_PILOT_TEST_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("TX_PILOT_PUBLIC_API_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func HTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func GET(t *testing.T, path string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, BaseURL()+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, body
}

func POSTJSON(t *testing.T, path string, payload any) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, BaseURL()+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, respBody
}

func AssertStatus(t *testing.T, got int, want int, body []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("status %d want %d body=%s", got, want, string(body))
	}
}

func AssertJSONKeys(t *testing.T, body []byte, keys ...string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing key %q in %s", k, string(body))
		}
	}
}

func LoadTestSigner(t *testing.T) *tx.Signer {
	t.Helper()
	path := TestKeypairFile
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join("..", "..", TestKeypairFile)
	}
	signer, err := tx.LoadSigner(path)
	if err != nil {
		t.Fatalf("load test signer: %v", err)
	}
	return signer
}

type blockhashResponse struct {
	Value struct {
		Blockhash string `json:"blockhash"`
	} `json:"value"`
}

func FetchBlockhash(t *testing.T) solana.Hash {
	t.Helper()
	status, body := GET(t, "/v1/blockhash")
	AssertStatus(t, status, http.StatusOK, body)
	var resp blockhashResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal blockhash: %v", err)
	}
	return solana.MustHashFromBase58(resp.Value.Blockhash)
}

func FetchTipAccount(t *testing.T) solana.PublicKey {
	t.Helper()
	status, body := GET(t, "/v1/tip-accounts")
	AssertStatus(t, status, http.StatusOK, body)
	var resp struct {
		Accounts []string `json:"accounts"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal tip accounts: %v", err)
	}
	if len(resp.Accounts) == 0 {
		t.Fatal("no tip accounts returned")
	}
	return solana.MustPublicKeyFromBase58(resp.Accounts[0])
}

func BuildSignedTransfer(t *testing.T, signer *tx.Signer, recipient solana.PublicKey, lamports uint64, memo string) string {
	t.Helper()
	factory := tx.NewFactory(signer)
	blockhash := FetchBlockhash(t)
	signed, err := factory.BuildTransfer(context.Background(), recipient, lamports, memo, blockhash)
	if err != nil {
		t.Fatalf("build transfer: %v", err)
	}
	encoded, err := tx.EncodeTransaction(signed, txpilot.EncodingBase64)
	if err != nil {
		t.Fatalf("encode tx: %v", err)
	}
	return encoded
}

func BuildSignedTip(t *testing.T, signer *tx.Signer, tipAccount solana.PublicKey, lamports uint64) string {
	t.Helper()
	factory := tx.NewFactory(signer)
	blockhash := FetchBlockhash(t)
	signed, err := factory.BuildTipTransfer(tipAccount, lamports, blockhash)
	if err != nil {
		t.Fatalf("build tip: %v", err)
	}
	encoded, err := tx.EncodeTransaction(signed, txpilot.EncodingBase64)
	if err != nil {
		t.Fatalf("encode tip: %v", err)
	}
	return encoded
}

func SubmitSignedTransaction(t *testing.T, encoded string, memo string) txpilot.SubmitResponse {
	t.Helper()
	payload := txpilot.SubmitTransactionRequest{
		Transaction: encoded,
		Encoding:    string(txpilot.EncodingBase64),
		Memo:        memo,
	}
	status, body := POSTJSON(t, "/v1/transactions", payload)
	AssertStatus(t, status, http.StatusAccepted, body)
	var resp txpilot.SubmitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal submit response: %v", err)
	}
	if resp.TransactionID == "" {
		t.Fatalf("empty transaction_id: %s", string(body))
	}
	if resp.Result == "" {
		t.Fatalf("empty result: %s", string(body))
	}
	return resp
}

func SubmitSignedBundle(t *testing.T, encoded []string, memo string) txpilot.SubmitResponse {
	t.Helper()
	payload := txpilot.SubmitBundleRequest{
		Transactions: encoded,
		Encoding:     string(txpilot.EncodingBase64),
		Memo:         memo,
	}
	status, body := POSTJSON(t, "/v1/bundles", payload)
	AssertStatus(t, status, http.StatusAccepted, body)
	var resp txpilot.SubmitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal bundle response: %v", err)
	}
	if resp.TransactionID == "" || resp.Result == "" {
		t.Fatalf("invalid bundle response: %s", string(body))
	}
	return resp
}

// SubmitDeveloperTx builds and submits a real signed 1-lamport self-transfer for
// integration tests. A self-transfer is used so the transaction actually lands:
// the signer account already exists and remains rent-exempt, whereas sending a
// sub-rent-exempt amount to a fresh account is rejected with InsufficientFundsForRent.
func SubmitDeveloperTx(t *testing.T, memo string) txpilot.SubmitResponse {
	t.Helper()
	signer := LoadTestSigner(t)
	encoded := BuildSignedTransfer(t, signer, signer.PublicKey(), DevTransferLamports, memo)
	return SubmitSignedTransaction(t, encoded, memo)
}

// PollTransaction polls GET /v1/transactions/{id} until status is terminal or timeout.
func PollTransaction(t *testing.T, txID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last map[string]any
	for time.Now().Before(deadline) {
		status, body := GET(t, fmt.Sprintf("/v1/transactions/%s", txID))
		if status == http.StatusNotFound {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		AssertStatus(t, status, http.StatusOK, body)
		if err := json.Unmarshal(body, &last); err != nil {
			t.Fatalf("unmarshal tx: %v", err)
		}
		s, _ := last["status"].(string)
		switch s {
		case "finalized", "failed", "confirmed":
			return last
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("poll timeout for tx %s last=%v", txID, last)
	return last
}
