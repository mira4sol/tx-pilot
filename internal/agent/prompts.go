package agent

import "fmt"

const decisionSchema = `Return ONLY valid JSON with keys:
type, title, summary, confidence_pct, action.
action must include kind one of: refresh_blockhash, increase_tip, delay_submission, change_mode, abort.
For increase_tip include tip_delta_pct number. For delay_submission include delay_slots number.`

const tipDecisionSchema = `Return ONLY valid JSON with keys:
type, title, summary, confidence_pct, action.
type must be "tip_intelligence".
action must include: kind ("set_tip"), tip_lamports (number), tip_delta_pct (optional number).`

func buildDecisionPrompt(facts DecisionFacts) string {
	return fmt.Sprintf(`You are Aegis, an autonomous Solana transaction SRE.
Analyze the failure and decide the next operational action.
Facts:
- transaction_id: %s
- failure_kind: %s
- failure_title: %s
- retry_attempt: %d
- congestion_pct: %.1f
- current_tip_lamports: %d
- floor_tip_lamports: %d
- current_slot: %d
- leader: %s
%s`, facts.TransactionID, facts.Failure.Kind, facts.Failure.Title, facts.RetryAttempt,
		facts.CongestionPct, facts.CurrentTip, facts.FloorTip, facts.CurrentSlot, facts.Leader, decisionSchema)
}

func buildTipPrompt(facts TipFacts) string {
	return fmt.Sprintf(`You are Aegis, an autonomous Solana transaction SRE optimizing Jito bundle tips.
Balance cost vs landing probability using live network facts.
Facts:
- transaction_id: %s
- policy_mode: %s
- congestion_pct: %.1f
- floor_tip_lamports: %d
- base_tip_lamports: %d
- leader: %s
- leader_quality: %s
- recent_landing_rate_pct: %.1f
%s`, facts.TransactionID, facts.PolicyMode, facts.CongestionPct, facts.FloorLamports,
		facts.BaseTip, facts.Leader, facts.LeaderQuality, facts.LandingRate, tipDecisionSchema)
}
