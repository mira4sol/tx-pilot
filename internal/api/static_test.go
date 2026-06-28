package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mira4sol/tx-pilot/internal/api"
	"github.com/mira4sol/tx-pilot/internal/config"
	"github.com/mira4sol/tx-pilot/internal/dashboard"
	"github.com/mira4sol/tx-pilot/internal/notify"
	"github.com/mira4sol/tx-pilot/internal/stream"
	"go.uber.org/zap"
)

func TestWebDashboardStatic(t *testing.T) {
	webDir := t.TempDir()
	writeTestFile(t, filepath.Join(webDir, "index.html"), "<html><body>dashboard</body></html>")
	writeTestFile(t, filepath.Join(webDir, "assets", "app.css"), "body{color:black}")

	router := testRouter(t, webDir)

	t.Run("healthz still works", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
	})

	t.Run("root serves index", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "dashboard") {
			t.Fatalf("expected index.html body, got %q", rr.Body.String())
		}
	})

	t.Run("asset file served", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app.css", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		if rr.Body.String() != "body{color:black}" {
			t.Fatalf("unexpected body %q", rr.Body.String())
		}
	})

	t.Run("spa fallback", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/settings/pipeline", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "dashboard") {
			t.Fatalf("expected index.html fallback, got %q", rr.Body.String())
		}
	})
}

func TestWebDashboardMissingDir(t *testing.T) {
	router := testRouter(t, filepath.Join(t.TempDir(), "missing"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestWebDashboardRealDist(t *testing.T) {
	webDir := filepath.Join("..", "..", "web", "dist")
	if _, err := os.Stat(filepath.Join(webDir, "index.html")); err != nil {
		t.Skip("web/dist not built; run make build-web first")
	}

	router := testRouter(t, webDir)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<!") {
		t.Fatalf("expected html document")
	}
}

func testRouter(t *testing.T, webDir string) http.Handler {
	t.Helper()
	logger := zap.NewNop()
	cfg := &config.Config{Cluster: "mainnet-beta"}
	slotState := stream.NewSlotState()
	dash := dashboard.NewService(cfg, slotState, nil, nil, logger)
	return api.NewRouter(api.Dependencies{
		Config: cfg, Dashboard: dash, Hub: notify.NewHub(), Logger: logger, WebDir: webDir,
	})
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
