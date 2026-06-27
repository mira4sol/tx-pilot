package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mira4sol/aegis/internal/app"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/dashboard"
	"github.com/mira4sol/aegis/internal/notify"
	"github.com/mira4sol/aegis/pkg/aegis"
	"go.uber.org/zap"
)

type Dependencies struct {
	Config       *config.Config
	Dashboard    *dashboard.Service
	ControlPlane *app.ControlPlane
	Hub          *notify.Hub
	Logger       *zap.Logger
	RiverUI      http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	applyDefaultMiddleware(r)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if deps.RiverUI != nil {
		r.Get("/riverui", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/riverui/", http.StatusTemporaryRedirect)
		})
		// Pass full paths through; riverui strips Prefix internally.
		r.Handle("/riverui/*", deps.RiverUI)
	}

	r.Route("/v1", func(r chi.Router) {
		r.Post("/transactions", deps.submitTransaction)
		r.Post("/bundles", deps.submitBundle)
		r.Post("/ops/submit", deps.submitOps)
		r.Get("/lifecycle-log", deps.lifecycleLog)
		r.Get("/transactions/{id}", deps.getTransaction)
		r.Get("/transactions/{id}/timeline", deps.getTimeline)
		r.Get("/bundles/{id}", deps.getBundle)
		r.Get("/blockhash", deps.getBlockhash)
		r.Get("/tip-accounts", deps.getTipAccounts)
		r.Get("/dashboard/snapshot", deps.dashboardSnapshot)
		r.Get("/dashboard/network", deps.dashboardNetwork)
		r.Get("/dashboard/slots", deps.dashboardSlots)
		r.Get("/dashboard/leaders", deps.dashboardLeaders)
		r.Get("/dashboard/bundles", deps.dashboardBundles)
		r.Get("/dashboard/transactions", deps.dashboardTransactions)
		r.Get("/dashboard/ai-decisions", deps.dashboardAIDecisions)
		r.Get("/dashboard/failures", deps.dashboardFailures)
		r.Get("/dashboard/recovery", deps.dashboardRecovery)
		r.Get("/dashboard/charts", deps.dashboardCharts)
		r.Get("/ws", deps.websocket)
	})
	return r
}

func (deps Dependencies) submitTransaction(w http.ResponseWriter, r *http.Request) {
	var req aegis.SubmitTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := deps.ControlPlane.SubmitTransaction(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (deps Dependencies) submitBundle(w http.ResponseWriter, r *http.Request) {
	var req aegis.SubmitBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := deps.ControlPlane.SubmitBundle(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (deps Dependencies) submitOps(w http.ResponseWriter, r *http.Request) {
	var req aegis.SubmitOpsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := deps.ControlPlane.SubmitOps(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (deps Dependencies) lifecycleLog(w http.ResponseWriter, r *http.Request) {
	limit := int32(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			limit = int32(n)
		}
	}
	entries, err := deps.ControlPlane.ExportLifecycleLog(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "count": len(entries)})
}

func (deps Dependencies) getTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tx, err := deps.ControlPlane.GetTransaction(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (deps Dependencies) getTimeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	events, err := deps.ControlPlane.GetTimeline(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "timeline not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (deps Dependencies) getBundle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	bundleRow, err := deps.ControlPlane.GetBundle(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "bundle not found")
		return
	}
	jitoStatus, _ := deps.ControlPlane.PollBundleStatus(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{
		"bundle":      bundleRow,
		"jito_status": jitoStatus,
	})
}

func (deps Dependencies) getBlockhash(w http.ResponseWriter, r *http.Request) {
	bh, err := deps.ControlPlane.GetLatestBlockhash(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bh)
}

func (deps Dependencies) getTipAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := deps.ControlPlane.GetTipAccounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

func (deps Dependencies) dashboardSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := deps.Dashboard.Snapshot(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (deps Dependencies) dashboardNetwork(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Network)
}

func (deps Dependencies) dashboardSlots(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Slots)
}

func (deps Dependencies) dashboardLeaders(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Leaders)
}

func (deps Dependencies) dashboardBundles(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Bundles)
}

func (deps Dependencies) dashboardTransactions(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Transactions)
}

func (deps Dependencies) dashboardAIDecisions(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Decisions)
}

func (deps Dependencies) dashboardFailures(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Failures)
}

func (deps Dependencies) dashboardRecovery(w http.ResponseWriter, r *http.Request) {
	snap, _ := deps.Dashboard.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, snap.Recovery)
}

func (deps Dependencies) dashboardCharts(w http.ResponseWriter, r *http.Request) {
	series := r.URL.Query().Get("series")
	if series == "" {
		series = "network_health"
	}
	points, err := deps.Dashboard.Charts(r.Context(), series)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series, "points": points})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
