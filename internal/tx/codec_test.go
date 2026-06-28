package tx_test

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/mira4sol/tx-pilot/internal/tx"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	priv := solana.NewWallet()
	blockhash := solana.Hash{1, 2, 3}
	instr := system.NewTransferInstruction(1, priv.PublicKey(), priv.PublicKey()).Build()
	stx, err := solana.NewTransaction([]solana.Instruction{instr}, blockhash, solana.TransactionPayer(priv.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = stx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(priv.PublicKey()) {
			return &priv.PrivateKey
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, enc := range []txpilot.Encoding{txpilot.EncodingBase64, txpilot.EncodingBase58} {
		encoded, err := tx.EncodeTransaction(stx, enc)
		if err != nil {
			t.Fatalf("encode %s: %v", enc, err)
		}
		decoded, err := tx.DecodeTransaction(encoded, enc)
		if err != nil {
			t.Fatalf("decode %s: %v", enc, err)
		}
		if len(decoded.Signatures) != 1 {
			t.Fatalf("expected 1 signature, got %d", len(decoded.Signatures))
		}
	}
}

func TestNormalizeEncoding(t *testing.T) {
	if tx.NormalizeEncoding("base64") != txpilot.EncodingBase64 {
		t.Fatal("expected base64")
	}
	if tx.NormalizeEncoding("base58") != txpilot.EncodingBase58 {
		t.Fatal("expected base58")
	}
	if tx.NormalizeEncoding("hex") != "" {
		t.Fatal("expected empty for invalid")
	}
}
