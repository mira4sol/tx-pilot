//go:build integration

package dashboard_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestDashboardRecovery(t *testing.T) {
	status, body := helpers.GET(t, "/v1/dashboard/recovery")
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "actions")
}
