//go:build integration

package lifecycle_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"github.com/mira4sol/tx-pilot/test/helpers"
)

const (
	totalSubmissions = 10
	successTarget    = 8
	failureTarget    = 2
)

type submissionResult struct {
	Index              int
	TransactionID      string
	BundleID           string
	InjectExpired      bool
	TerminalStatus     string
	Stage              string
	FailureKind        string
	FailureTitle       string
	SubmittedSlot      uint64
	ProcessedSlot      uint64
	ConfirmedSlot      uint64
	FinalizedSlot      uint64
	TipLamports        int64
	Signature          string
	CommitmentProgress []string
	SubmittedAt        string
	ProcessedAt        string
	ConfirmedAt        string
	FinalizedAt        string
	FailedAt           string
}

func TestBountyLifecycleLog(t *testing.T) {
	if os.Getenv("TX_PILOT_RUN_BOUNTY_LOG") != "1" {
		t.Skip("set TX_PILOT_RUN_BOUNTY_LOG=1 to run live 10-bundle bounty evidence test")
	}

	results := make([]submissionResult, 0, totalSubmissions)
	for i := 0; i < totalSubmissions; i++ {
		injectExpired := i < failureTarget
		memo := fmt.Sprintf("bounty-log-%d", i+1)
		resp := submitOps(t, memo, injectExpired)
		t.Logf("[%d/%d] submitted tx_id=%s bundle=%s inject_expired=%v", i+1, totalSubmissions, resp.TransactionID, resp.Result, injectExpired)

		row := pollOpsTransaction(t, string(resp.TransactionID), 2*time.Minute)
		entry := submissionResult{
			Index:          i + 1,
			TransactionID:  string(resp.TransactionID),
			BundleID:       string(resp.BundleID),
			InjectExpired:  injectExpired,
			TerminalStatus: stringField(row, "status"),
			Stage:          stringField(row, "stage"),
			FailureKind:    stringField(row, "failure_kind"),
			TipLamports:    int64Field(row, "tip_lamports"),
			Signature:      stringField(row, "signature"),
		}
		if entry.TipLamports == 0 {
			entry.TipLamports = int64Field(row, "requested_tip_lamports")
		}
		results = append(results, entry)
		time.Sleep(2 * time.Second)
	}

	// Allow late finalizations to settle on mainnet before scoring. Finalization
	// is ~13s after confirmation, so a short settle window is sufficient.
	time.Sleep(45 * time.Second)
	for i := range results {
		status, body := helpers.GET(t, fmt.Sprintf("/v1/transactions/%s", results[i].TransactionID))
		if status != http.StatusOK {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(body, &row); err != nil {
			continue
		}
		results[i].TerminalStatus = stringField(row, "status")
		results[i].FailureKind = stringField(row, "failure_kind")
		if results[i].TipLamports == 0 {
			results[i].TipLamports = int64Field(row, "tip_lamports")
		}
		if results[i].TipLamports == 0 {
			results[i].TipLamports = int64Field(row, "requested_tip_lamports")
		}
	}

	status, body := helpers.GET(t, "/v1/lifecycle-log?limit=50")
	helpers.AssertStatus(t, status, http.StatusOK, body)

	var logResp struct {
		Entries []txpilot.LifecycleLogEntry `json:"entries"`
		Count   int                       `json:"count"`
	}
	if err := json.Unmarshal(body, &logResp); err != nil {
		t.Fatalf("unmarshal lifecycle log: %v", err)
	}
	enrichFromLifecycleLog(results, logResp.Entries)

	successes := 0
	failures := 0
	for _, r := range results {
		switch r.TerminalStatus {
		case "confirmed", "finalized":
			successes++
		case "failed":
			failures++
		}
	}

	evidencePath := filepath.Join("..", "..", "docs", "lifecycle-log-evidence.md")
	if err := writeEvidenceMarkdown(evidencePath, results, logResp.Count); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	t.Logf("wrote %s (successes=%d failures=%d)", evidencePath, successes, failures)

	if successes < successTarget {
		t.Fatalf("expected >=%d successes, got %d (see %s)", successTarget, successes, evidencePath)
	}
	if failures < failureTarget {
		t.Fatalf("expected >=%d failures, got %d (see %s)", failureTarget, failures, evidencePath)
	}
}

func submitOps(t *testing.T, memo string, injectExpired bool) txpilot.SubmitResponse {
	t.Helper()
	payload := txpilot.SubmitOpsRequest{
		Memo:                   memo,
		Lamports:               1,
		PolicyMode:             txpilot.ModeAggressive,
		InjectExpiredBlockhash: injectExpired,
	}
	status, body := helpers.POSTJSON(t, "/v1/ops/submit", payload)
	helpers.AssertStatus(t, status, http.StatusAccepted, body)
	var resp txpilot.SubmitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal ops response: %v", err)
	}
	return resp
}

func pollOpsTransaction(t *testing.T, txID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last map[string]any
	for time.Now().Before(deadline) {
		status, body := helpers.GET(t, fmt.Sprintf("/v1/transactions/%s", txID))
		if status == http.StatusNotFound {
			time.Sleep(1 * time.Second)
			continue
		}
		helpers.AssertStatus(t, status, http.StatusOK, body)
		if err := json.Unmarshal(body, &last); err != nil {
			t.Fatalf("unmarshal tx: %v", err)
		}
		s := stringField(last, "status")
		switch s {
		case "finalized", "failed", "confirmed":
			return last
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("poll timeout for tx %s last=%v", txID, last)
	return last
}

func enrichFromLifecycleLog(results []submissionResult, entries []txpilot.LifecycleLogEntry) {
	byID := map[string]txpilot.LifecycleLogEntry{}
	for _, e := range entries {
		byID[string(e.TransactionID)] = e
	}
	for i := range results {
		e, ok := byID[results[i].TransactionID]
		if !ok {
			continue
		}
		results[i].SubmittedSlot = e.SubmittedSlot
		results[i].ProcessedSlot = e.ProcessedSlot
		results[i].ConfirmedSlot = e.ConfirmedSlot
		results[i].FinalizedSlot = e.FinalizedSlot
		results[i].FailureTitle = e.FailureTitle
		if results[i].FailureKind == "" && e.FailureKind != "" {
			results[i].FailureKind = string(e.FailureKind)
		}
		if results[i].Signature == "" {
			results[i].Signature = string(e.Signature)
		}
		if results[i].TipLamports == 0 {
			results[i].TipLamports = e.TipLamports
		}
		results[i].CommitmentProgress = append([]string{}, e.CommitmentProgression...)
		results[i].SubmittedAt = formatTime(e.SubmittedAt)
		results[i].ProcessedAt = formatTime(e.ProcessedAt)
		results[i].ConfirmedAt = formatTime(e.ConfirmedAt)
		results[i].FinalizedAt = formatTime(e.FinalizedAt)
		results[i].FailedAt = formatTime(e.FailedAt)
	}
}

func writeEvidenceMarkdown(path string, results []submissionResult, logCount int) error {
	var b strings.Builder
	b.WriteString("# TX Pilot Lifecycle Log Evidence\n\n")
	b.WriteString("Live mainnet Jito submissions captured by `TestBountyLifecycleLog`. Every\n")
	b.WriteString("signature, bundle id and slot below is reproduced in full (untruncated) so\n")
	b.WriteString("each transaction can be independently verified on a Solana explorer.\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Submissions: %d (target %d success / %d failure)\n", len(results), successTarget, failureTarget))
	b.WriteString(fmt.Sprintf("- Lifecycle log entries exported: %d\n\n", logCount))

	successes, failures := 0, 0
	for _, r := range results {
		switch r.TerminalStatus {
		case "confirmed", "finalized":
			successes++
		case "failed":
			failures++
		}
	}
	b.WriteString(fmt.Sprintf("- Observed: %d success / %d failed\n\n", successes, failures))

	// Compact status overview (no identifiers truncated — full values follow in
	// the per-transaction section below).
	b.WriteString("## Overview\n\n")
	b.WriteString("| # | Status | sub slot | proc slot | conf slot | fin slot | Tip (lamports) | Failure |\n")
	b.WriteString("|---|--------|----------|-----------|-----------|----------|----------------|---------|\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("| %d | %s | %d | %d | %d | %d | %d | %s |\n",
			r.Index, r.TerminalStatus, r.SubmittedSlot, r.ProcessedSlot, r.ConfirmedSlot, r.FinalizedSlot,
			r.TipLamports, failureCell(r)))
	}

	b.WriteString("\n## Full transaction records\n\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("### %d. %s\n\n", r.Index, r.TerminalStatus))
		b.WriteString(fmt.Sprintf("- transaction_id: `%s`\n", r.TransactionID))
		b.WriteString(fmt.Sprintf("- bundle_id: `%s`\n", r.BundleID))
		b.WriteString(fmt.Sprintf("- signature: `%s`\n", r.Signature))
		if r.Signature != "" {
			b.WriteString(fmt.Sprintf("- explorer: https://explorer.solana.com/tx/%s\n", r.Signature))
			b.WriteString(fmt.Sprintf("- solscan: https://solscan.io/tx/%s\n", r.Signature))
		}
		b.WriteString(fmt.Sprintf("- inject_expired_blockhash: %v\n", r.InjectExpired))
		b.WriteString(fmt.Sprintf("- tip_lamports: %d\n", r.TipLamports))
		b.WriteString(fmt.Sprintf("- slots: submitted=%d processed=%d confirmed=%d finalized=%d\n",
			r.SubmittedSlot, r.ProcessedSlot, r.ConfirmedSlot, r.FinalizedSlot))
		b.WriteString(fmt.Sprintf("- commitment_progression: %s\n", strings.Join(r.CommitmentProgress, " -> ")))
		b.WriteString(fmt.Sprintf("- submitted_at: %s\n", r.SubmittedAt))
		b.WriteString(fmt.Sprintf("- processed_at: %s\n", r.ProcessedAt))
		b.WriteString(fmt.Sprintf("- confirmed_at: %s\n", r.ConfirmedAt))
		b.WriteString(fmt.Sprintf("- finalized_at: %s\n", r.FinalizedAt))
		b.WriteString(fmt.Sprintf("- failed_at: %s\n", r.FailedAt))
		if r.FailureKind != "" {
			b.WriteString(fmt.Sprintf("- failure_kind: %s\n", r.FailureKind))
			b.WriteString(fmt.Sprintf("- failure_title: %s\n", r.FailureTitle))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Explorer verification\n\n")
	b.WriteString("Open any `explorer` link above, or paste the full signature into\n")
	b.WriteString("[Solana Explorer](https://explorer.solana.com/) / [Solscan](https://solscan.io/),\n")
	b.WriteString("and cross-reference the confirmed/finalized slot recorded here.\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func int64Field(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func formatTime(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func failureCell(r submissionResult) string {
	if r.FailureKind == "" {
		return "-"
	}
	if r.FailureTitle != "" {
		return r.FailureKind + " (" + r.FailureTitle + ")"
	}
	return r.FailureKind
}
