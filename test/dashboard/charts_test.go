//go:build integration

package dashboard_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestDashboardCharts(t *testing.T) {
	status, body := helpers.GET(t, "/v1/dashboard/charts?window=15m")
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "series")
}
