//go:build integration

package dashboard_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/tx-pilot/test/helpers"
)

func TestDashboardAIDecisions(t *testing.T) {
	status, body := helpers.GET(t, "/v1/dashboard/ai-decisions")
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "decisions")
}
