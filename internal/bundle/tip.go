package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/pkg/aegis"
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

func solToLamports(sol float64) aegis.Lamports {
	if sol <= 0 {
		return 0
	}
	return aegis.Lamports(math.Ceil(sol * 1_000_000_000))
}

func percentileForMode(mode aegis.PolicyMode) float64 {
	switch mode {
	case aegis.ModeCheap:
		return 0.25
	case aegis.ModeSafe:
		return 0.50
	case aegis.ModeFast:
		return 0.75
	case aegis.ModeAggressive:
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

type ResolveInput struct {
	RequestedTip *uint64
	TipMode      string
	PolicyMode   aegis.PolicyMode
	Congestion   float64
}

func (r *TipResolver) ResolveTip(ctx context.Context, in ResolveInput) (aegis.TipResolution, error) {
	floor, err := r.FetchTipFloor(ctx)
	if err != nil {
		return aegis.TipResolution{}, err
	}

	mode := in.PolicyMode
	if mode == "" {
		mode = r.cfg.PolicyMode
	}

	dynamicFloorSOL := pickPercentile(floor, 0.25)
	dynamicFloor := solToLamports(dynamicFloorSOL)
	if dynamicFloor < aegis.Lamports(r.cfg.JitoMinTipLamports) {
		dynamicFloor = aegis.Lamports(r.cfg.JitoMinTipLamports)
	}

	var requested aegis.Lamports
	var source aegis.TipSource = aegis.TipSourceAuto

	switch {
	case in.RequestedTip != nil:
		requested = aegis.Lamports(*in.RequestedTip)
		source = aegis.TipSourceCaller
	case in.TipMode != "" && in.TipMode != string(aegis.TipModeAuto):
		pm := aegis.PolicyMode(in.TipMode)
		autoSOL := pickPercentile(floor, percentileForMode(pm))
		if in.Congestion > 0.7 {
			autoSOL *= 1.12
		}
		requested = solToLamports(autoSOL)
		source = aegis.TipSourceAuto
	default:
		autoSOL := pickPercentile(floor, percentileForMode(mode))
		if in.Congestion > 0.7 {
			autoSOL *= 1.12
		}
		requested = solToLamports(autoSOL)
		source = aegis.TipSourceAuto
	}

	final := requested
	if final < dynamicFloor {
		final = dynamicFloor
		if source == aegis.TipSourceCaller {
			source = aegis.TipSourceFloorClamped
		}
	}

	return aegis.TipResolution{
		RequestedTipLamports: requested,
		FloorLamports:        dynamicFloor,
		FinalTipLamports:     final,
		TipSource:            source,
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
