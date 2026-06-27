//go:build integration

package developer_test

import (
	"net/http"
	"testing"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestHealthz(t *testing.T) {
	status, body := helpers.GET(t, "/healthz")
	helpers.AssertStatus(t, status, http.StatusOK, body)
	helpers.AssertJSONKeys(t, body, "status")
}
