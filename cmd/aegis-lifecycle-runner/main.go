package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func main() {
	count := flag.Int("count", 10, "number of bundle submissions")
	failures := flag.Int("failures", 2, "number of forced blockhash expiry failures")
	baseURL := flag.String("base", "", "API base URL (defaults to AEGIS_PUBLIC_API_BASE_URL)")
	flag.Parse()

	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if *baseURL == "" {
		*baseURL = cfg.PublicAPIBaseURL
	}

	ctx := context.Background()
	fmt.Printf("running lifecycle log on %s: %d submissions (%d forced failures)\n", *baseURL, *count, *failures)

	for i := 0; i < *count; i++ {
		injectExpired := i < *failures
		req := aegis.SubmitOpsRequest{
			Memo:                   fmt.Sprintf("lifecycle-run-%d", i+1),
			Lamports:               1,
			InjectExpiredBlockhash: injectExpired,
		}
		resp, err := postJSON(ctx, *baseURL+"/v1/ops/submit", req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "submit %d failed: %v\n", i+1, err)
			continue
		}
		fmt.Printf("[%d/%d] tx_id=%s bundle=%s inject_expired=%v\n", i+1, *count, resp.TransactionID, resp.Result, injectExpired)
		time.Sleep(3 * time.Second)
	}

	time.Sleep(30 * time.Second)

	logResp, err := http.Get(*baseURL + "/v1/lifecycle-log?limit=50")
	if err != nil {
		fmt.Fprintf(os.Stderr, "export lifecycle log: %v\n", err)
		os.Exit(1)
	}
	defer logResp.Body.Close()
	body, _ := io.ReadAll(logResp.Body)
	outPath := "lifecycle-log.json"
	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("lifecycle log written to %s\n", outPath)
}

func postJSON(ctx context.Context, url string, payload any) (aegis.SubmitResponse, error) {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return aegis.SubmitResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return aegis.SubmitResponse{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var out aegis.SubmitResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return aegis.SubmitResponse{}, err
	}
	return out, nil
}
