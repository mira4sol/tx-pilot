package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mira4sol/aegis/internal/api"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/dashboard"
	"github.com/mira4sol/aegis/internal/notify"
	"github.com/mira4sol/aegis/internal/stream"
	"go.uber.org/zap"
)

func TestHealthz(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{Cluster: "mainnet-beta"}
	slotState := stream.NewSlotState()
	dash := dashboard.NewService(cfg, slotState, nil, logger)
	router := api.NewRouter(api.Dependencies{
		Config: cfg, Dashboard: dash, Hub: notify.NewHub(), Logger: logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
}
