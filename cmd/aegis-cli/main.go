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

	"github.com/gagliardetto/solana-go"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func main() {
	var (
		baseURL = flag.String("url", "http://localhost:8080", "Aegis API base URL")
		memo    = flag.String("memo", "aegis-cli", "memo")
		bundle  = flag.Bool("bundle", false, "submit as bundle with tip tx")
	)
	flag.Parse()

	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if cfg.PublicAPIBaseURL != "" {
		*baseURL = cfg.PublicAPIBaseURL
	}
	if cfg.KeypairPath == "" {
		fmt.Fprintln(os.Stderr, "AEGIS_KEYPAIR_PATH required for CLI signing")
		os.Exit(1)
	}

	signer, err := tx.LoadSigner(cfg.KeypairPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load signer: %v\n", err)
		os.Exit(1)
	}

	blockhash := fetchBlockhash(*baseURL)
	factory := tx.NewFactory(signer)
	recipient := signer.PublicKey()
	transfer, err := factory.BuildTransfer(context.Background(), recipient, 1, *memo, blockhash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build transfer: %v\n", err)
		os.Exit(1)
	}
	encodedTransfer, err := tx.EncodeTransaction(transfer, aegis.EncodingBase64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}

	if *bundle {
		tipAccount := fetchTipAccount(*baseURL)
		tipTx, err := factory.BuildTipTransfer(tipAccount, 1000, blockhash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "build tip: %v\n", err)
			os.Exit(1)
		}
		encodedTip, err := tx.EncodeTransaction(tipTx, aegis.EncodingBase64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode tip: %v\n", err)
			os.Exit(1)
		}
		body, _ := json.Marshal(aegis.SubmitBundleRequest{
			Transactions: []string{encodedTransfer, encodedTip},
			Encoding:     string(aegis.EncodingBase64),
			Memo:         *memo,
		})
		resp, err := postJSON(context.Background(), *baseURL+"/v1/bundles", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "submit bundle failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(resp))
		return
	}

	body, _ := json.Marshal(aegis.SubmitTransactionRequest{
		Transaction: encodedTransfer,
		Encoding:    string(aegis.EncodingBase64),
		Memo:        *memo,
	})
	resp, err := postJSON(context.Background(), *baseURL+"/v1/transactions", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "submit failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(resp))
}

func fetchBlockhash(baseURL string) solana.Hash {
	resp, err := http.Get(baseURL + "/v1/blockhash")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var out struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return solana.MustHashFromBase58(out.Value.Blockhash)
}

func fetchTipAccount(baseURL string) solana.PublicKey {
	resp, err := http.Get(baseURL + "/v1/tip-accounts")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var out struct {
		Accounts []string `json:"accounts"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return solana.MustPublicKeyFromBase58(out.Accounts[0])
}

func postJSON(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}
