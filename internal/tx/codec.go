package tx

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"github.com/mr-tron/base58"
)

func NormalizeEncoding(enc string) txpilot.Encoding {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "", "base64":
		return txpilot.EncodingBase64
	case "base58":
		return txpilot.EncodingBase58
	default:
		return ""
	}
}

func DecodeTransaction(encoded string, enc txpilot.Encoding) (*solana.Transaction, error) {
	if enc == "" {
		return nil, fmt.Errorf("unsupported encoding")
	}
	var raw []byte
	var err error
	switch enc {
	case txpilot.EncodingBase64:
		raw, err = base64.StdEncoding.DecodeString(encoded)
	case txpilot.EncodingBase58:
		raw, err = base58.Decode(encoded)
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", enc)
	}
	if err != nil {
		return nil, fmt.Errorf("decode transaction: %w", err)
	}
	tx, err := solana.TransactionFromBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parse transaction: %w", err)
	}
	return tx, nil
}

func ExtractSignatures(encoded string, enc txpilot.Encoding) ([]string, error) {
	tx, err := DecodeTransaction(encoded, enc)
	if err != nil {
		return nil, err
	}
	if len(tx.Signatures) == 0 {
		return nil, fmt.Errorf("transaction has no signatures")
	}
	sigs := make([]string, 0, len(tx.Signatures))
	for _, sig := range tx.Signatures {
		sigs = append(sigs, sig.String())
	}
	return sigs, nil
}

// ExtractBlockhash returns the recent blockhash referenced by a transaction.
// It is used to detect blockhash expiry: once this blockhash is no longer
// valid on-chain, the transaction can never land.
func ExtractBlockhash(encoded string, enc txpilot.Encoding) (string, error) {
	tx, err := DecodeTransaction(encoded, enc)
	if err != nil {
		return "", err
	}
	return tx.Message.RecentBlockhash.String(), nil
}

func ExtractSignaturesFromBundle(encoded []string, enc txpilot.Encoding) ([]string, error) {
	all := make([]string, 0, len(encoded))
	for i, item := range encoded {
		sigs, err := ExtractSignatures(item, enc)
		if err != nil {
			return nil, fmt.Errorf("transaction %d: %w", i, err)
		}
		all = append(all, sigs...)
	}
	return all, nil
}

func EncodeTransaction(tx *solana.Transaction, enc txpilot.Encoding) (string, error) {
	raw, err := tx.MarshalBinary()
	if err != nil {
		return "", err
	}
	switch enc {
	case txpilot.EncodingBase64:
		return base64.StdEncoding.EncodeToString(raw), nil
	case txpilot.EncodingBase58:
		return base58.Encode(raw), nil
	default:
		return "", fmt.Errorf("unsupported encoding: %s", enc)
	}
}
