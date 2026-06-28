//go:build integration

package developer_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/mira4sol/tx-pilot/test/helpers"
)

func TestTransactionTimeline(t *testing.T) {
	submit := helpers.SubmitDeveloperTx(t, "integration-timeline")

	status, body := helpers.GET(t, fmt.Sprintf("/v1/transactions/%s/timeline", submit.TransactionID))
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "events")

	var out struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Events) == 0 {
		t.Fatal("expected at least one lifecycle event (created)")
	}
}
