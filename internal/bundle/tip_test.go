package bundle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mira4sol/aegis/internal/bundle"
	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/pkg/aegis"
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
		PolicyMode:         aegis.ModeSafe,
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
	if res.TipSource != aegis.TipSourceFloorClamped {
		t.Fatalf("expected floor_clamped, got %s", res.TipSource)
	}
}

func TestResolveTipCallerAboveFloor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]bundle.TipFloor{{LandedTips25thPercentile: 0.000001}})
	}))
	defer server.Close()

	cfg := &config.Config{PolicyMode: aegis.ModeSafe, JitoTipFloorURL: server.URL, JitoMinTipLamports: 1000}
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
