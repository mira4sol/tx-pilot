//go:build integration

package developer_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestGetTransaction(t *testing.T) {
	submit := helpers.SubmitDeveloperTx(t, "integration-get-tx")

	status, body := helpers.GET(t, fmt.Sprintf("/v1/transactions/%s", submit.TransactionID))
	helpers.AssertStatus(t, status, http.StatusOK, body)

	var tx map[string]any
	if err := json.Unmarshal(body, &tx); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tx["id"] != string(submit.TransactionID) {
		t.Fatalf("id mismatch: %v", tx["id"])
	}
}

func TestGetTransactionNotFound(t *testing.T) {
	status, body := helpers.GET(t, "/v1/transactions/tx_does_not_exist")
	helpers.AssertStatus(t, status, http.StatusNotFound, body)
	helpers.AssertJSONKeys(t, body, "error")
}
