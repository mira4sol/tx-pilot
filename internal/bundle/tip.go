package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/mira4sol/tx-pilot/internal/config"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

type TipFloor struct {
	Time                        time.Time `json:"time"`
	LandedTips25thPercentile    float64   `json:"landed_tips_25th_percentile"`
	LandedTips50thPercentile    float64   `json:"landed_tips_50th_percentile"`
	LandedTips75thPercentile    float64   `json:"landed_tips_75th_percentile"`
	LandedTips95thPercentile    float64   `json:"landed_tips_95th_percentile"`
	LandedTips99thPercentile    float64   `json:"landed_tips_99th_percentile"`
	EMALandedTips50thPercentile float64   `json:"ema_landed_tips_50th_percentile"`
}

type TipResolver struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewTipResolver(cfg *config.Config) *TipResolver {
	return &TipResolver{cfg: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (r *TipResolver) FetchTipFloor(ctx context.Context) (*TipFloor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.cfg.JitoTipFloorURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var floors []TipFloor
	if err := json.Unmarshal(body, &floors); err != nil {
		return nil, err
	}
	if len(floors) == 0 {
		return nil, fmt.Errorf("empty tip floor response")
	}
	return &floors[0], nil
}

func solToLamports(sol float64) txpilot.Lamports {
	if sol <= 0 {
		return 0
	}
	return txpilot.Lamports(math.Ceil(sol * 1_000_000_000))
}

func percentileForMode(mode txpilot.PolicyMode) float64 {
	switch mode {
	case txpilot.ModeCheap:
		return 0.25
	case txpilot.ModeSafe:
		return 0.50
	case txpilot.ModeFast:
		return 0.75
	case txpilot.ModeAggressive:
		return 0.95
	default:
		return 0.50
	}
}

func pickPercentile(floor *TipFloor, p float64) float64 {
	switch {
	case p <= 0.25:
		return floor.LandedTips25thPercentile
	case p <= 0.50:
		return floor.LandedTips50thPercentile
	case p <= 0.75:
		return floor.LandedTips75thPercentile
	case p <= 0.95:
		return floor.LandedTips95thPercentile
	default:
		return floor.LandedTips99thPercentile
	}
}

// networkMultiplier scales tips continuously with congestion (0.0-1.0).
func networkMultiplier(congestion float64) float64 {
	if congestion < 0 {
		congestion = 0
	}
	if congestion > 1 {
		congestion = 1
	}
	return 1.0 + congestion*0.25
}

// leaderMultiplier applies a small premium when leader quality is uncertain.
func leaderMultiplier(quality string) float64 {
	switch quality {
	case "jito":
		return 1.0
	case "good":
		return 1.02
	case "unknown", "":
		return 1.05
	default:
		return 1.03
	}
}

func applyNetworkAdjustments(sol float64, congestion float64, leaderQuality string) float64 {
	return sol * networkMultiplier(congestion) * leaderMultiplier(leaderQuality)
}

type ResolveInput struct {
	RequestedTip  *uint64
	TipMode       string
	PolicyMode    txpilot.PolicyMode
	Congestion    float64
	LeaderQuality string
}

type RecommendInput struct {
	ResolveInput
	LeaderQuality string
	LandingRate   float64
}

type RecommendResult struct {
	Resolution  txpilot.TipResolution
	Percentile  float64
	Congestion  float64
	FloorSOL    float64
	SelectedSOL float64
	PolicyMode  txpilot.PolicyMode
}

func (r *TipResolver) ResolveTip(ctx context.Context, in ResolveInput) (txpilot.TipResolution, error) {
	floor, err := r.FetchTipFloor(ctx)
	if err != nil {
		return txpilot.TipResolution{}, err
	}

	mode := in.PolicyMode
	if mode == "" {
		mode = r.cfg.PolicyMode
	}

	dynamicFloorSOL := pickPercentile(floor, 0.25)
	dynamicFloor := solToLamports(dynamicFloorSOL)
	if dynamicFloor < txpilot.Lamports(r.cfg.JitoMinTipLamports) {
		dynamicFloor = txpilot.Lamports(r.cfg.JitoMinTipLamports)
	}

	var requested txpilot.Lamports
	var source txpilot.TipSource = txpilot.TipSourceAuto

	switch {
	case in.RequestedTip != nil:
		requested = txpilot.Lamports(*in.RequestedTip)
		source = txpilot.TipSourceCaller
	case in.TipMode != "" && in.TipMode != string(txpilot.TipModeAuto):
		pm := txpilot.PolicyMode(in.TipMode)
		autoSOL := pickPercentile(floor, percentileForMode(pm))
		autoSOL = applyNetworkAdjustments(autoSOL, in.Congestion, in.LeaderQuality)
		requested = solToLamports(autoSOL)
		source = txpilot.TipSourceAuto
	default:
		autoSOL := pickPercentile(floor, percentileForMode(mode))
		autoSOL = applyNetworkAdjustments(autoSOL, in.Congestion, in.LeaderQuality)
		requested = solToLamports(autoSOL)
		source = txpilot.TipSourceAuto
	}

	final := requested
	if final < dynamicFloor {
		final = dynamicFloor
		if source == txpilot.TipSourceCaller {
			source = txpilot.TipSourceFloorClamped
		}
	}

	return txpilot.TipResolution{
		RequestedTipLamports: requested,
		FloorLamports:        dynamicFloor,
		FinalTipLamports:     final,
		TipSource:            source,
	}, nil
}

// Recommend resolves a dynamic tip from live Jito floor data and current network conditions.
func (r *TipResolver) Recommend(ctx context.Context, in RecommendInput) (RecommendResult, error) {
	res, err := r.ResolveTip(ctx, in.ResolveInput)
	if err != nil {
		return RecommendResult{}, err
	}
	floor, _ := r.FetchTipFloor(ctx)
	mode := in.PolicyMode
	if mode == "" {
		mode = r.cfg.PolicyMode
	}
	pct := percentileForMode(mode)
	selectedSOL := pickPercentile(floor, pct)
	selectedSOL = applyNetworkAdjustments(selectedSOL, in.Congestion, in.LeaderQuality)
	floorSOL := pickPercentile(floor, 0.25)
	return RecommendResult{
		Resolution:  res,
		Percentile:  pct,
		Congestion:  in.Congestion,
		FloorSOL:    floorSOL,
		SelectedSOL: selectedSOL,
		PolicyMode:  mode,
	}, nil
}

func (r *TipResolver) CurrentFloorLamports(ctx context.Context) (uint64, error) {
	floor, err := r.FetchTipFloor(ctx)
	if err != nil {
		return r.cfg.JitoMinTipLamports, nil
	}
	lamports := uint64(solToLamports(pickPercentile(floor, 0.25)))
	if lamports < r.cfg.JitoMinTipLamports {
		return r.cfg.JitoMinTipLamports, nil
	}
	return lamports, nil
}
