//go:build integration

package developer_test

import (
	"testing"

	"github.com/mira4sol/aegis/pkg/aegis"
	"github.com/mira4sol/aegis/test/helpers"
)

func TestSubmitBundle(t *testing.T) {
	signer := helpers.LoadTestSigner(t)
	tipAccount := helpers.FetchTipAccount(t)

	transferTx := helpers.BuildSignedTransfer(t, signer, signer.PublicKey(), helpers.DevTransferLamports, "integration-bundle-transfer")
	tipTx := helpers.BuildSignedTip(t, signer, tipAccount, 1000)

	resp := helpers.SubmitSignedBundle(t, []string{transferTx, tipTx}, "integration-bundle-test")

	if resp.SubmissionKind != aegis.SubmissionBundle {
		t.Fatalf("expected submission_kind bundle, got %s", resp.SubmissionKind)
	}
	if resp.Result == "" {
		t.Fatal("expected non-empty bundle_id result")
	}
	if resp.BundleID == "" {
		t.Fatalf("expected bundle_id in response")
	}
	t.Logf("submitted bundle tx_id=%s bundle_id=%s", resp.TransactionID, resp.Result)

	status, body := helpers.GET(t, "/v1/bundles/"+string(resp.BundleID))
	helpers.AssertStatus(t, status, 200, body)
	helpers.AssertJSONKeys(t, body, "bundle", "jito_status")
}
