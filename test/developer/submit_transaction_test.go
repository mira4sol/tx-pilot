//go:build integration

package developer_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/mira4sol/aegis/test/helpers"
)

func TestSubmitTransaction(t *testing.T) {
	signer := helpers.LoadTestSigner(t)
	encoded := helpers.BuildSignedTransfer(t, signer, signer.PublicKey(), helpers.DevTransferLamports, "integration-submit-test")

	resp := helpers.SubmitSignedTransaction(t, encoded, "integration-submit-test")

	if resp.SubmissionKind != aegis.SubmissionTransaction {
		t.Fatalf("expected submission_kind transaction, got %s", resp.SubmissionKind)
	}
	if resp.Result == "" {
		t.Fatal("expected non-empty result signature")
	}
	if resp.Encoding != string(aegis.EncodingBase64) {
		t.Fatalf("expected base64 encoding, got %s", resp.Encoding)
	}
	t.Logf("submitted tx_id=%s result=%s kind=%s", resp.TransactionID, resp.Result, resp.SubmissionKind)
}

func TestSubmitTransactionBase58(t *testing.T) {
	signer := helpers.LoadTestSigner(t)
	encodedB64 := helpers.BuildSignedTransfer(t, signer, signer.PublicKey(), helpers.DevTransferLamports, "integration-base58-test")

	parsed, err := tx.DecodeTransaction(encodedB64, aegis.EncodingBase64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	encodedB58, err := tx.EncodeTransaction(parsed, aegis.EncodingBase58)
	if err != nil {
		t.Fatalf("encode base58: %v", err)
	}

	status, body := helpers.POSTJSON(t, "/v1/transactions", aegis.SubmitTransactionRequest{
		Transaction: encodedB58,
		Encoding:    string(aegis.EncodingBase58),
		Memo:        "integration-base58-test",
	})
	helpers.AssertStatus(t, status, http.StatusAccepted, body)
	helpers.AssertJSONKeys(t, body, "transaction_id", "result", "submission_kind")
}
