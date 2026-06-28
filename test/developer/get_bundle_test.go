//go:build integration

package developer_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mira4sol/tx-pilot/test/helpers"
)

func TestGetBundle(t *testing.T) {
	signer := helpers.LoadTestSigner(t)
	tipAccount := helpers.FetchTipAccount(t)
	transferTx := helpers.BuildSignedTransfer(t, signer, signer.PublicKey(), helpers.DevTransferLamports, "integration-get-bundle")
	tipTx := helpers.BuildSignedTip(t, signer, tipAccount, 1000)
	submit := helpers.SubmitSignedBundle(t, []string{transferTx, tipTx}, "integration-get-bundle")

	time.Sleep(2 * time.Second)

	status, body := helpers.GET(t, fmt.Sprintf("/v1/bundles/%s", submit.Result))
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "bundle", "jito_status")

	var resp struct {
		Bundle struct {
			ID string `json:"id"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	if resp.Bundle.ID != submit.Result {
		t.Fatalf("bundle id mismatch: %s vs %s", resp.Bundle.ID, submit.Result)
	}
}
