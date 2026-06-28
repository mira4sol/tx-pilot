package bundle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mira4sol/tx-pilot/internal/bundle"
	"github.com/mira4sol/tx-pilot/internal/config"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func TestResolveTipFloorClamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]bundle.TipFloor{{
			LandedTips25thPercentile: 0.000001,
			LandedTips50thPercentile: 0.000002,
		}})
	}))
	defer server.Close()

	cfg := &config.Config{
		PolicyMode:         txpilot.ModeSafe,
		JitoTipFloorURL:    server.URL,
		JitoMinTipLamports: 1000,
	}
	resolver := bundle.NewTipResolver(cfg)
	low := uint64(100)
	res, err := resolver.ResolveTip(context.Background(), bundle.ResolveInput{RequestedTip: &low})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalTipLamports < 1000 {
		t.Fatalf("expected floor clamp >= 1000, got %d", res.FinalTipLamports)
	}
	if res.TipSource != txpilot.TipSourceFloorClamped {
		t.Fatalf("expected floor_clamped, got %s", res.TipSource)
	}
}

func TestResolveTipCallerAboveFloor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]bundle.TipFloor{{LandedTips25thPercentile: 0.000001}})
	}))
	defer server.Close()

	cfg := &config.Config{PolicyMode: txpilot.ModeSafe, JitoTipFloorURL: server.URL, JitoMinTipLamports: 1000}
	resolver := bundle.NewTipResolver(cfg)
	high := uint64(50000)
	res, err := resolver.ResolveTip(context.Background(), bundle.ResolveInput{RequestedTip: &high})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalTipLamports != 50000 {
		t.Fatalf("expected caller tip preserved, got %d", res.FinalTipLamports)
	}
}

func TestNetworkMultiplierCongestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]bundle.TipFloor{{
			LandedTips50thPercentile: 0.000002,
		}})
	}))
	defer server.Close()

	cfg := &config.Config{PolicyMode: txpilot.ModeSafe, JitoTipFloorURL: server.URL, JitoMinTipLamports: 1000}
	resolver := bundle.NewTipResolver(cfg)

	tests := []struct {
		name       string
		congestion float64
		leader     string
		wantMin    uint64
		wantMax    uint64
	}{
		{"low congestion", 0.1, "jito", 2000, 2200},
		{"medium congestion", 0.5, "good", 2200, 2400},
		{"high congestion", 0.9, "unknown", 2500, 2700},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := resolver.ResolveTip(context.Background(), bundle.ResolveInput{
				PolicyMode: txpilot.ModeSafe, Congestion: tc.congestion, LeaderQuality: tc.leader,
			})
			if err != nil {
				t.Fatal(err)
			}
			got := uint64(res.FinalTipLamports)
			if got < tc.wantMin || got > tc.wantMax {
				t.Fatalf("tip %d outside [%d,%d] for congestion=%.1f leader=%s", got, tc.wantMin, tc.wantMax, tc.congestion, tc.leader)
			}
		})
	}
}
