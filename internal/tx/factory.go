package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
)

type Signer struct {
	privateKey solana.PrivateKey
	publicKey  solana.PublicKey
}

func LoadSigner(path string) (*Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keypair: %w", err)
	}
	var raw []byte
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse keypair json: %w", err)
	}
	priv := solana.PrivateKey(raw)
	return &Signer{privateKey: priv, publicKey: priv.PublicKey()}, nil
}

func (s *Signer) PublicKey() solana.PublicKey { return s.publicKey }

type Factory struct {
	signer *Signer
}

func NewFactory(signer *Signer) *Factory {
	return &Factory{signer: signer}
}

func (f *Factory) BuildTransfer(_ context.Context, recipient solana.PublicKey, lamports uint64, memo string, blockhash solana.Hash) (*solana.Transaction, error) {
	instructions := []solana.Instruction{
		system.NewTransferInstruction(lamports, f.signer.publicKey, recipient).Build(),
	}
	if memo != "" {
		instructions = append(instructions, buildMemoInstruction(memo))
	}
	tx, err := solana.NewTransaction(instructions, blockhash, solana.TransactionPayer(f.signer.publicKey))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(f.signer.publicKey) {
			return &f.signer.privateKey
		}
		return nil
	})
	return tx, err
}

func (f *Factory) BuildSelfTransfer(_ context.Context, lamports uint64, memo string, blockhash solana.Hash) (*solana.Transaction, error) {
	return f.BuildTransfer(context.Background(), f.signer.publicKey, lamports, memo, blockhash)
}

func (f *Factory) BuildTipTransfer(tipAccount solana.PublicKey, lamports uint64, blockhash solana.Hash) (*solana.Transaction, error) {
	instructions := []solana.Instruction{
		system.NewTransferInstruction(lamports, f.signer.publicKey, tipAccount).Build(),
	}
	tx, err := solana.NewTransaction(instructions, blockhash, solana.TransactionPayer(f.signer.publicKey))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(f.signer.publicKey) {
			return &f.signer.privateKey
		}
		return nil
	})
	return tx, err
}

func buildMemoInstruction(memo string) solana.Instruction {
	programID := solana.MustPublicKeyFromBase58("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
	return solana.NewInstruction(programID, solana.AccountMetaSlice{}, []byte(memo))
}

// Ensure imports used for future token support.
var (
	_ = associatedtokenaccount.Instruction_Create
	_ = token.Instruction_Transfer
)
