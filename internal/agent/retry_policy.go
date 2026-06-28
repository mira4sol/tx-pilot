package agent

import (
	"fmt"

	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func fallbackTipDecision(facts TipFacts) Decision {
	tip := facts.BaseTip
	deltaPct := 0
	if facts.CongestionPct > 70 {
		deltaPct = 12
		tip = uint64(float64(tip) * 1.12)
	}
	if tip < facts.FloorLamports {
		tip = facts.FloorLamports
	}
	title := "Tip set from floor"
	summary := fmt.Sprintf("Fallback: using %d lamports tip at %.0f%% congestion", tip, facts.CongestionPct)
	if deltaPct > 0 {
		title = fmt.Sprintf("Tip raised +%d%%", deltaPct)
		summary = fmt.Sprintf("Fallback: congestion %.0f%% — raised tip to %d lamports", facts.CongestionPct, tip)
	}
	return Decision{
		Type: "tip_intelligence", Title: title, Summary: summary,
		Inputs:        map[string]any{"congestion_pct": facts.CongestionPct, "floor_tip_lamports": facts.FloorLamports},
		Action:        map[string]any{"kind": "set_tip", "tip_lamports": tip, "tip_delta_pct": deltaPct},
		ConfidencePct: 75,
	}
}

func fallbackDecision(facts DecisionFacts) Decision {
	switch facts.Failure.Kind {
	case txpilot.FailureExpiredBlockhash:
		return Decision{
			Type: "blockhash_refresh", Title: "Refresh blockhash",
			Summary:       "Fallback: expired blockhash requires refresh before resubmit",
			Inputs:        map[string]any{"failure_kind": string(facts.Failure.Kind)},
			Action:        map[string]any{"kind": "refresh_blockhash"},
			ConfidencePct: 70,
		}
	case txpilot.FailureTipBelowFloor, txpilot.FailureBundleRejected:
		return Decision{
			Type: "tip_adjustment", Title: "Increase tip",
			Summary:       "Fallback: raise tip above observed floor",
			Inputs:        map[string]any{"floor_tip_lamports": facts.FloorTip},
			Action:        map[string]any{"kind": "increase_tip", "tip_delta_pct": 15},
			ConfidencePct: 75,
		}
	default:
		return Decision{
			Type: "retry_backoff", Title: "Delay and retry",
			Summary:       "Fallback: wait one leader window before retry",
			Inputs:        map[string]any{"failure_kind": string(facts.Failure.Kind)},
			Action:        map[string]any{"kind": "delay_submission", "delay_slots": 1},
			ConfidencePct: 60,
		}
	}
}
