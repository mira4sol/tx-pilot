//go:build integration

package dashboard_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestDashboardSnapshot(t *testing.T) {
	status, body := helpers.GET(t, "/v1/dashboard/snapshot")
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "generated_at", "network", "slots", "leaders", "health", "bundles", "pipeline", "transactions")
}
