//go:build ignore

// Generate the integration-test keypair: go run scripts/generate-test-keypair.go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
)

const outFile = "tx-pilot-test-keypair.json"

func main() {
	if _, err := os.Stat(outFile); err == nil {
		wallet, err := solana.WalletFromPrivateKeyBase58(readExisting())
		if err == nil {
			fmt.Printf("keypair already exists: %s\n", outFile)
			fmt.Printf("public key (fund this): %s\n", wallet.PublicKey().String())
			return
		}
	}

	wallet := solana.NewWallet()
	raw := []byte(wallet.PrivateKey)
	data, err := json.Marshal(raw)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(outFile, data, 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", outFile)
	fmt.Printf("public key (fund this): %s\n", wallet.PublicKey().String())
}

func readExisting() string {
	data, err := os.ReadFile(outFile)
	if err != nil {
		panic(err)
	}
	var raw []byte
	if err := json.Unmarshal(data, &raw); err != nil {
		panic(err)
	}
	return solana.PrivateKey(raw).String()
}
