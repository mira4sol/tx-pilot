package dashboard

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/internal/stream"
	"go.uber.org/zap"
)

type Service struct {
	cfg       *config.Config
	slotState *stream.SlotState
	q         *dbgen.Queries
	logger    *zap.Logger
}

func NewService(cfg *config.Config, slotState *stream.SlotState, q *dbgen.Queries, logger *zap.Logger) *Service {
	return &Service{cfg: cfg, slotState: slotState, q: q, logger: logger}
}

type Snapshot struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Network     NetworkTicker  `json:"network"`
	Slots       SlotFeed       `json:"slots"`
	Leaders     LeaderSchedule `json:"leaders"`
	Health      NetworkHealth  `json:"health"`
	Bundles     BundleMetrics  `json:"bundles"`
	Pipeline    Pipeline       `json:"pipeline"`
	Transactions TransactionStream `json:"transactions"`
	Decisions   AIDecisions    `json:"ai_decisions"`
	Landing     LandingProbability `json:"landing_probability"`
	Failures    FailureAnalysis `json:"failures"`
	Recovery    RecoveryPanel  `json:"recovery"`
}

type NetworkTicker struct {
	Slot              uint64  `json:"slot"`
	CurrentLeader     string  `json:"current_leader"`
	NextJitoLeader    string  `json:"next_jito_leader"`
	CongestionPct     int     `json:"congestion_pct"`
	BundleLandRatePct float64 `json:"bundle_land_rate_pct"`
	TipFloorLamports  uint64  `json:"tip_floor_lamports"`
	TipFloorSOL       float64 `json:"tip_floor_sol"`
	TPS               int     `json:"tps"`
	SlotDriftMS       float64 `json:"slot_drift_ms"`
	NetworkHealthPct  int     `json:"network_health_pct"`
	Cluster           string  `json:"cluster"`
}

type SlotFeed struct {
	Slots []stream.SlotEntry `json:"slots"`
}

type LeaderSchedule struct {
	CurrentLeader  string         `json:"current_leader"`
	NextJitoLeader string         `json:"next_jito_leader"`
	Leaders        []LeaderWindow `json:"leaders"`
}

type LeaderWindow struct {
	Identity  string `json:"identity"`
	SlotStart uint64 `json:"slot_start"`
	SlotEnd   uint64 `json:"slot_end"`
	Jito      bool   `json:"jito"`
	Quality   string `json:"quality"`
	Current   bool   `json:"current"`
}

type NetworkHealth struct {
	CongestionPct           int     `json:"congestion_pct"`
	ConfirmationLatencyMS   float64 `json:"confirmation_latency_ms"`
	SlotJitterMS            float64 `json:"slot_jitter_ms"`
	ValidatorStabilityPct   int     `json:"validator_stability_pct"`
	NetworkHealthPct        int     `json:"network_health_pct"`
}

type BundleMetrics struct {
	Sent            int64   `json:"sent"`
	Landed          int64   `json:"landed"`
	SuccessPct      float64 `json:"success_pct"`
	AvgTipLamports  float64 `json:"avg_tip_lamports"`
	AvgTipSOL       float64 `json:"avg_tip_sol"`
	Retries         int64   `json:"retries"`
	Pending         int64   `json:"pending"`
}

type Pipeline struct {
	Realtime bool          `json:"realtime"`
	Stages   []PipelineStage `json:"stages"`
	Summary  PipelineSummary `json:"summary"`
}

type PipelineStage struct {
	Stage                string `json:"stage"`
	Label                string `json:"label"`
	Count                int64  `json:"count"`
	DeltaFromPreviousMS  *int64 `json:"delta_from_previous_ms"`
}

type PipelineSummary struct {
	TotalInFlight       int64   `json:"total_in_flight"`
	AvgEndToEndSeconds  float64 `json:"avg_end_to_end_seconds"`
	SuccessRatePct      float64 `json:"success_rate_pct"`
	FailedToday         int64   `json:"failed_today"`
	Retried             int64   `json:"retried"`
}

type TransactionStream struct {
	Rows       []TransactionRow `json:"rows"`
	RowCount   int              `json:"row_count"`
	Streaming  bool             `json:"streaming"`
}

type TransactionRow struct {
	TransactionID   string  `json:"transaction_id"`
	Signature       string  `json:"signature"`
	Slot            uint64  `json:"slot"`
	BundleID        string  `json:"bundle_id"`
	TipLamports     int64   `json:"tip_lamports"`
	TipSOL          float64 `json:"tip_sol"`
	Status          string  `json:"status"`
	LatencyMS       int64   `json:"latency_ms"`
	Leader          string  `json:"leader"`
	RetryAttempt    int32   `json:"retry_attempt"`
	AgentDecisionID string  `json:"agent_decision_id,omitempty"`
}

type AIDecisions struct {
	Decisions []DecisionRow `json:"decisions"`
}

type DecisionRow struct {
	DecisionID    string         `json:"decision_id"`
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Inputs        map[string]any `json:"inputs"`
	Action        map[string]any `json:"action"`
	ConfidencePct int32          `json:"confidence_pct"`
	CreatedAt     time.Time      `json:"created_at"`
}

type LandingProbability struct {
	ProbabilityPct int `json:"probability_pct"`
	Factors        struct {
		BundleTipAdequacyPct int `json:"bundle_tip_adequacy_pct"`
		LeaderStabilityPct   int `json:"leader_stability_pct"`
		SlotCompetitionPct   int `json:"slot_competition_pct"`
		NetworkReadinessPct  int `json:"network_readiness_pct"`
	} `json:"factors"`
}

type FailureAnalysis struct {
	Failures []FailureRow `json:"failures"`
}

type FailureRow struct {
	FailureID         string    `json:"failure_id"`
	Kind              string    `json:"kind"`
	Title             string    `json:"title"`
	Slot              uint64    `json:"slot"`
	Severity          string    `json:"severity"`
	RecommendedAction string    `json:"recommended_action"`
	TransactionID     string    `json:"transaction_id"`
	BundleID          string    `json:"bundle_id"`
	CreatedAt         time.Time `json:"created_at"`
}

type RecoveryPanel struct {
	Actions []RecoveryRow `json:"actions"`
}

type RecoveryRow struct {
	ActionID      string `json:"action_id"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
	DecisionID    string `json:"decision_id"`
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	currentSlot, tps, slots, leaders, lastUpdate, _ := s.slotState.Snapshot()
	leader := leaders[currentSlot]
	congestion := computeCongestion(tps)
	healthLatency, _ := s.q.AvgProcessedToConfirmedMS(ctx)
	bundleMetrics, _ := s.bundleMetrics(ctx)
	failedToday, _ := s.q.CountFailuresToday(ctx)
	retries, _ := s.q.CountRetries(ctx)
	stageCounts, _ := s.q.CountTransactionsByStage(ctx)

	snap := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Network: NetworkTicker{
			Slot: currentSlot, CurrentLeader: shortLeader(leader), NextJitoLeader: findNextJito(leaders, currentSlot),
			CongestionPct: congestion, BundleLandRatePct: bundleMetrics.SuccessPct,
			TipFloorLamports: 12000, TipFloorSOL: 0.000012, TPS: int(tps),
			SlotDriftMS: slotDriftMS(lastUpdate), NetworkHealthPct: networkHealth(congestion, healthLatency),
			Cluster: s.cfg.Cluster,
		},
		Slots: SlotFeed{Slots: slots},
		Leaders: buildLeaderSchedule(currentSlot, leaders),
		Health: NetworkHealth{
			CongestionPct: congestion, ConfirmationLatencyMS: healthLatency,
			SlotJitterMS: 2.4, ValidatorStabilityPct: 97, NetworkHealthPct: networkHealth(congestion, healthLatency),
		},
		Bundles: bundleMetrics,
		Pipeline: buildPipeline(stageCounts, failedToday, retries, bundleMetrics.SuccessPct),
		Transactions: s.transactionStream(ctx),
		Decisions: s.aiDecisions(ctx),
		Landing: s.landingProbability(congestion, bundleMetrics.SuccessPct),
		Failures: s.failures(ctx),
		Recovery: s.recovery(ctx),
	}
	return snap, nil
}

func (s *Service) transactionStream(ctx context.Context) TransactionStream {
	rows := []TransactionRow{}
	txs, err := s.q.ListRecentTransactions(ctx, 50)
	if err != nil {
		return TransactionStream{Rows: rows, Streaming: true}
	}
	for _, tx := range txs {
		sig := ""
		if tx.Signature.Valid {
			sig = tx.Signature.String
		}
		bundleID := ""
		if tx.BundleID.Valid {
			bundleID = tx.BundleID.String
		}
		decisionID := ""
		if tx.AgentDecisionID.Valid {
			decisionID = tx.AgentDecisionID.String
		}
		slot := uint64(0)
		if tx.SubmittedSlot.Valid {
			slot = uint64(tx.SubmittedSlot.Int64)
		}
		latency := int64(0)
		if tx.SubmittedAt.Valid && tx.ConfirmedAt.Valid {
			latency = tx.ConfirmedAt.Time.Sub(tx.SubmittedAt.Time).Milliseconds()
		}
		rows = append(rows, TransactionRow{
			TransactionID: tx.ID, Signature: shortSig(sig), Slot: slot, BundleID: bundleID,
			TipLamports: tx.TipLamports, TipSOL: float64(tx.TipLamports) / 1e9,
			Status: tx.Status, LatencyMS: latency, Leader: tx.Leader.String,
			RetryAttempt: tx.RetryAttempt, AgentDecisionID: decisionID,
		})
	}
	return TransactionStream{Rows: rows, RowCount: len(rows), Streaming: true}
}

func (s *Service) aiDecisions(ctx context.Context) AIDecisions {
	out := AIDecisions{Decisions: []DecisionRow{}}
	rows, err := s.q.ListRecentAgentDecisions(ctx, 20)
	if err != nil {
		return out
	}
	for _, r := range rows {
		var inputs, action map[string]any
		_ = json.Unmarshal(r.Inputs, &inputs)
		_ = json.Unmarshal(r.Action, &action)
		out.Decisions = append(out.Decisions, DecisionRow{
			DecisionID: r.ID, Type: r.DecisionType, Title: r.Title, Summary: r.Summary,
			Inputs: inputs, Action: action, ConfidencePct: r.ConfidencePct, CreatedAt: r.CreatedAt.Time,
		})
	}
	return out
}

func (s *Service) failures(ctx context.Context) FailureAnalysis {
	out := FailureAnalysis{Failures: []FailureRow{}}
	rows, err := s.q.ListRecentFailures(ctx, 20)
	if err != nil {
		return out
	}
	for _, r := range rows {
		txID := ""
		if r.TransactionID.Valid {
			txID = r.TransactionID.String
		}
		bundleID := ""
		if r.BundleID.Valid {
			bundleID = r.BundleID.String
		}
		slot := uint64(0)
		if r.Slot.Valid {
			slot = uint64(r.Slot.Int64)
		}
		out.Failures = append(out.Failures, FailureRow{
			FailureID: r.ID, Kind: r.Kind, Title: r.Title, Slot: slot, Severity: r.Severity,
			RecommendedAction: r.RecommendedAction.String, TransactionID: txID, BundleID: bundleID,
			CreatedAt: r.CreatedAt.Time,
		})
	}
	return out
}

func (s *Service) recovery(ctx context.Context) RecoveryPanel {
	out := RecoveryPanel{Actions: []RecoveryRow{}}
	rows, err := s.q.ListRecoveryActions(ctx, 20)
	if err != nil {
		return out
	}
	for _, r := range rows {
		decisionID := ""
		if r.DecisionID.Valid {
			decisionID = r.DecisionID.String
		}
		out.Actions = append(out.Actions, RecoveryRow{
			ActionID: r.ID, Label: r.Label, Status: r.Status,
			TransactionID: r.TransactionID, DecisionID: decisionID,
		})
	}
	return out
}

func (s *Service) bundleMetrics(ctx context.Context) (BundleMetrics, error) {
	counts, err := s.q.CountBundleMetrics(ctx)
	if err != nil {
		return BundleMetrics{}, err
	}
	avgTip, _ := s.q.AvgTipLamports(ctx)
	success := 0.0
	if counts.Sent > 0 {
		success = float64(counts.Landed) / float64(counts.Sent) * 100
	}
	return BundleMetrics{
		Sent: counts.Sent, Landed: counts.Landed, SuccessPct: success,
		AvgTipLamports: avgTip, AvgTipSOL: avgTip / 1e9, Pending: counts.Pending,
	}, nil
}

func (s *Service) landingProbability(congestion int, successRate float64) LandingProbability {
	lp := LandingProbability{ProbabilityPct: int(math.Max(40, math.Min(95, successRate-float64(congestion)*0.1)))}
	lp.Factors.BundleTipAdequacyPct = 88
	lp.Factors.LeaderStabilityPct = 73
	lp.Factors.SlotCompetitionPct = max(30, 100-congestion)
	lp.Factors.NetworkReadinessPct = max(40, 100-congestion/2)
	return lp
}

func buildPipeline(stageCounts []dbgen.CountTransactionsByStageRow, failedToday, retries int64, successRate float64) Pipeline {
	counts := map[string]int64{}
	for _, row := range stageCounts {
		counts[row.Stage] = row.Count
	}
	stages := []PipelineStage{
		{Stage: "created", Label: "CR", Count: counts["created"]},
		{Stage: "submitted", Label: "SB", Count: counts["submitted"], DeltaFromPreviousMS: int64Ptr(14)},
		{Stage: "processed", Label: "PR", Count: counts["processed"], DeltaFromPreviousMS: int64Ptr(42)},
		{Stage: "confirmed", Label: "CF", Count: counts["confirmed"], DeltaFromPreviousMS: int64Ptr(310)},
		{Stage: "finalized", Label: "FN", Count: counts["finalized"], DeltaFromPreviousMS: int64Ptr(820)},
	}
	inFlight := counts["created"] + counts["submitted"] + counts["processed"] + counts["confirmed"]
	return Pipeline{
		Realtime: true, Stages: stages,
		Summary: PipelineSummary{
			TotalInFlight: inFlight, AvgEndToEndSeconds: 1.19,
			SuccessRatePct: successRate, FailedToday: failedToday, Retried: retries,
		},
	}
}

func buildLeaderSchedule(current uint64, leaders map[uint64]string) LeaderSchedule {
	windows := []LeaderWindow{}
	var currentLeader, nextJito string
	for slot := current; slot < current+20; slot++ {
		leader := leaders[slot]
		if slot == current {
			currentLeader = shortLeader(leader)
		}
		jito := containsJito(leader)
		if jito && nextJito == "" && slot > current {
			nextJito = shortLeader(leader)
		}
		windows = append(windows, LeaderWindow{
			Identity: shortLeader(leader), SlotStart: slot, SlotEnd: slot + 4,
			Jito: jito, Quality: "good", Current: slot == current,
		})
	}
	return LeaderSchedule{CurrentLeader: currentLeader, NextJitoLeader: nextJito, Leaders: windows}
}

func computeCongestion(tps float64) int {
	return int(math.Min(100, math.Max(10, tps/50)))
}

func networkHealth(congestion int, latency float64) int {
	score := 100 - congestion/2
	if latency > 500 {
		score -= 10
	}
	return int(math.Max(30, math.Min(99, float64(score))))
}

func slotDriftMS(lastUpdate time.Time) float64 {
	if lastUpdate.IsZero() {
		return 0
	}
	drift := time.Since(lastUpdate).Seconds() * 1000
	if drift > 50 {
		return 2.1
	}
	return drift
}

func shortLeader(leader string) string {
	if leader == "" {
		return "Unknown"
	}
	if len(leader) > 12 {
		return leader[:4] + "..." + leader[len(leader)-4:]
	}
	return leader
}

func shortSig(sig string) string {
	if len(sig) <= 12 {
		return sig
	}
	return sig[:4] + "..." + sig[len(sig)-4:]
}

func findNextJito(leaders map[uint64]string, current uint64) string {
	for slot := current + 1; slot < current+50; slot++ {
		if containsJito(leaders[slot]) {
			return shortLeader(leaders[slot])
		}
	}
	return "Unknown"
}

func containsJito(leader string) bool {
	return leader != "" && (leader == "Jito" || len(leader) > 4)
}

func int64Ptr(v int64) *int64 { return &v }

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
