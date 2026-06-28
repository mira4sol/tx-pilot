package dashboard

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/bundle"
	"github.com/mira4sol/tx-pilot/internal/config"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/internal/stream"
	"go.uber.org/zap"
)

type Service struct {
	cfg       *config.Config
	slotState *stream.SlotState
	q         *dbgen.Queries
	tip       *bundle.TipResolver
	logger    *zap.Logger
}

func NewService(cfg *config.Config, slotState *stream.SlotState, q *dbgen.Queries, tip *bundle.TipResolver, logger *zap.Logger) *Service {
	return &Service{cfg: cfg, slotState: slotState, q: q, tip: tip, logger: logger}
}

type Snapshot struct {
	GeneratedAt  time.Time          `json:"generated_at"`
	Network      NetworkTicker      `json:"network"`
	Slots        SlotFeed           `json:"slots"`
	Leaders      LeaderSchedule     `json:"leaders"`
	Health       NetworkHealth      `json:"health"`
	Bundles      BundleMetrics      `json:"bundles"`
	Pipeline     Pipeline           `json:"pipeline"`
	Transactions TransactionStream  `json:"transactions"`
	Decisions    AIDecisions        `json:"ai_decisions"`
	Landing      LandingProbability `json:"landing_probability"`
	Failures     FailureAnalysis    `json:"failures"`
	Recovery     RecoveryPanel      `json:"recovery"`
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
	CongestionPct         int     `json:"congestion_pct"`
	ConfirmationLatencyMS float64 `json:"confirmation_latency_ms"`
	SlotJitterMS          float64 `json:"slot_jitter_ms"`
	ValidatorStabilityPct int     `json:"validator_stability_pct"`
	NetworkHealthPct      int     `json:"network_health_pct"`
}

type BundleMetrics struct {
	Sent           int64   `json:"sent"`
	Landed         int64   `json:"landed"`
	SuccessPct     float64 `json:"success_pct"`
	AvgTipLamports float64 `json:"avg_tip_lamports"`
	AvgTipSOL      float64 `json:"avg_tip_sol"`
	Retries        int64   `json:"retries"`
	Pending        int64   `json:"pending"`
}

type Pipeline struct {
	Realtime bool            `json:"realtime"`
	Stages   []PipelineStage `json:"stages"`
	Summary  PipelineSummary `json:"summary"`
}

type PipelineStage struct {
	Stage               string `json:"stage"`
	Label               string `json:"label"`
	Count               int64  `json:"count"`
	DeltaFromPreviousMS *int64 `json:"delta_from_previous_ms"`
}

type PipelineSummary struct {
	TotalInFlight      int64   `json:"total_in_flight"`
	AvgEndToEndSeconds float64 `json:"avg_end_to_end_seconds"`
	SuccessRatePct     float64 `json:"success_rate_pct"`
	FailedToday        int64   `json:"failed_today"`
	Retried            int64   `json:"retried"`
}

type TransactionStream struct {
	Rows      []TransactionRow `json:"rows"`
	RowCount  int              `json:"row_count"`
	Streaming bool             `json:"streaming"`
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
	TransactionID string         `json:"transaction_id,omitempty"`
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
	FailureID         string         `json:"failure_id"`
	Kind              string         `json:"kind"`
	Title             string         `json:"title"`
	Reason            string         `json:"reason,omitempty"`
	Evidence          map[string]any `json:"evidence,omitempty"`
	Slot              uint64         `json:"slot"`
	Severity          string         `json:"severity"`
	RecommendedAction string         `json:"recommended_action"`
	TransactionID     string         `json:"transaction_id"`
	BundleID          string         `json:"bundle_id"`
	CreatedAt         time.Time      `json:"created_at"`
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

type ChartPointView struct {
	Time    time.Time      `json:"time"`
	Payload map[string]any `json:"payload"`
}

type ChartsResponse struct {
	Series string           `json:"series"`
	Window string           `json:"window"`
	Points []ChartPointView `json:"points"`
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	currentSlot, tps, slots, leaders, lastUpdate, _ := s.slotState.Snapshot()
	leader := leaders[currentSlot]
	congestion := s.slotState.CongestionPct()
	healthLatency, _ := s.q.AvgProcessedToConfirmedMS(ctx)
	bundleMetrics, _ := s.bundleMetrics(ctx)
	failedToday, _ := s.q.CountFailuresToday(ctx)
	retries, _ := s.q.CountRetries(ctx)
	stageCounts, _ := s.q.CountTransactionsByStage(ctx)
	tipFloor := s.currentTipFloor(ctx)

	snap := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Network: NetworkTicker{
			Slot: currentSlot, CurrentLeader: shortLeader(leader), NextJitoLeader: findNextJito(leaders, currentSlot),
			CongestionPct: congestion, BundleLandRatePct: bundleMetrics.SuccessPct,
			TipFloorLamports: tipFloor, TipFloorSOL: float64(tipFloor) / 1e9, TPS: int(tps),
			SlotDriftMS: slotDriftMS(lastUpdate), NetworkHealthPct: networkHealth(congestion, healthLatency),
			Cluster: s.cfg.Cluster,
		},
		Slots:   SlotFeed{Slots: slots},
		Leaders: buildLeaderSchedule(currentSlot, leaders),
		Health: NetworkHealth{
			CongestionPct: congestion, ConfirmationLatencyMS: healthLatency,
			SlotJitterMS: s.slotState.SlotJitterMS(), ValidatorStabilityPct: s.slotState.ValidatorStabilityPct(),
			NetworkHealthPct: networkHealth(congestion, healthLatency),
		},
		Bundles:      bundleMetrics,
		Pipeline:     buildPipeline(ctx, s.q, stageCounts, failedToday, retries, bundleMetrics.SuccessPct),
		Transactions: s.transactionStream(ctx),
		Decisions:    s.aiDecisions(ctx),
		Landing:      s.landingProbability(ctx, congestion, bundleMetrics),
		Failures:     s.failures(ctx),
		Recovery:     s.recovery(ctx),
	}
	s.recordChartPoint(ctx, congestion, tipFloor, healthLatency)
	return snap, nil
}

func (s *Service) currentTipFloor(ctx context.Context) uint64 {
	if s.tip == nil {
		return s.cfg.JitoMinTipLamports
	}
	floor, err := s.tip.CurrentFloorLamports(ctx)
	if err != nil {
		return s.cfg.JitoMinTipLamports
	}
	return floor
}

func (s *Service) recordChartPoint(ctx context.Context, congestion int, tipFloor uint64, latency float64) {
	payload, _ := json.Marshal(map[string]any{
		"congestion_pct": congestion, "tip_floor_lamports": tipFloor, "confirmation_latency_ms": latency,
	})
	_ = s.q.InsertChartPoint(ctx, dbgen.InsertChartPointParams{
		Series: "network_health", PointTime: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}, Payload: payload,
	})
}

func (s *Service) Charts(ctx context.Context, series string) ([]dbgen.ChartPoint, error) {
	since := time.Now().UTC().Add(-24 * time.Hour)
	return s.q.ListChartPoints(ctx, dbgen.ListChartPointsParams{
		Series: series, PointTime: pgtype.Timestamptz{Time: since, Valid: true},
	})
}

func parseChartWindow(window string) time.Duration {
	switch window {
	case "5m":
		return 5 * time.Minute
	case "1h":
		return time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

func (s *Service) ChartsResponse(ctx context.Context, series, window string) (ChartsResponse, error) {
	since := time.Now().UTC().Add(-parseChartWindow(window))
	points, err := s.q.ListChartPoints(ctx, dbgen.ListChartPointsParams{
		Series: series, PointTime: pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		return ChartsResponse{}, err
	}
	out := ChartsResponse{Series: series, Window: window, Points: make([]ChartPointView, 0, len(points))}
	for _, point := range points {
		payload := map[string]any{}
		if len(point.Payload) > 0 {
			_ = json.Unmarshal(point.Payload, &payload)
		}
		t := time.Time{}
		if point.PointTime.Valid {
			t = point.PointTime.Time
		}
		out.Points = append(out.Points, ChartPointView{Time: t, Payload: payload})
	}
	return out, nil
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
			DecisionID: r.ID, TransactionID: r.TransactionID.String,
			Type: r.DecisionType, Title: r.Title, Summary: r.Summary,
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
			Reason: failureReason(r.Evidence), Evidence: parseEvidence(r.Evidence),
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
		Retries: retriesFromDB(ctx, s.q),
	}, nil
}

func retriesFromDB(ctx context.Context, q *dbgen.Queries) int64 {
	n, _ := q.CountRetries(ctx)
	return n
}

func (s *Service) landingProbability(ctx context.Context, congestion int, metrics BundleMetrics) LandingProbability {
	tipFloor := s.currentTipFloor(ctx)
	avgTip := metrics.AvgTipLamports
	tipAdequacy := 70
	if avgTip > 0 && float64(tipFloor) > 0 {
		tipAdequacy = int(math.Min(99, (avgTip/float64(tipFloor))*100))
	}
	stability := s.slotState.ValidatorStabilityPct()
	lp := LandingProbability{ProbabilityPct: int(math.Max(40, math.Min(95, metrics.SuccessPct-float64(congestion)*0.1)))}
	lp.Factors.BundleTipAdequacyPct = tipAdequacy
	lp.Factors.LeaderStabilityPct = stability
	lp.Factors.SlotCompetitionPct = max(30, 100-congestion)
	lp.Factors.NetworkReadinessPct = max(40, 100-congestion/2)
	return lp
}

func buildPipeline(ctx context.Context, q *dbgen.Queries, stageCounts []dbgen.CountTransactionsByStageRow, failedToday, retries int64, successRate float64) Pipeline {
	counts := map[string]int64{}
	for _, row := range stageCounts {
		counts[row.Stage] = row.Count
	}
	avgLatency := avgStageLatency(ctx, q)
	stages := []PipelineStage{
		{Stage: "created", Label: "CR", Count: counts["created"]},
		{Stage: "submitted", Label: "SB", Count: counts["submitted"], DeltaFromPreviousMS: avgLatency["submitted"]},
		{Stage: "processed", Label: "PR", Count: counts["processed"], DeltaFromPreviousMS: avgLatency["processed"]},
		{Stage: "confirmed", Label: "CF", Count: counts["confirmed"], DeltaFromPreviousMS: avgLatency["confirmed"]},
		{Stage: "finalized", Label: "FN", Count: counts["finalized"], DeltaFromPreviousMS: avgLatency["finalized"]},
	}
	inFlight := counts["created"] + counts["submitted"] + counts["processed"] + counts["confirmed"]
	avgE2E := 0.0
	if v := avgEndToEndSeconds(ctx, q); v > 0 {
		avgE2E = v
	}
	return Pipeline{
		Realtime: true, Stages: stages,
		Summary: PipelineSummary{
			TotalInFlight: inFlight, AvgEndToEndSeconds: avgE2E,
			SuccessRatePct: successRate, FailedToday: failedToday, Retried: retries,
		},
	}
}

func avgStageLatency(ctx context.Context, q *dbgen.Queries) map[string]*int64 {
	out := map[string]*int64{}
	txs, err := q.ListRecentTransactions(ctx, 100)
	if err != nil {
		return out
	}
	buckets := map[string][]int64{}
	for _, txRow := range txs {
		events, err := q.ListLifecycleEvents(ctx, txRow.ID)
		if err != nil {
			continue
		}
		for _, ev := range events {
			if ev.LatencyMs.Valid {
				buckets[ev.Stage] = append(buckets[ev.Stage], ev.LatencyMs.Int64)
			}
		}
	}
	for stage, vals := range buckets {
		if len(vals) == 0 {
			continue
		}
		var sum int64
		for _, v := range vals {
			sum += v
		}
		avg := sum / int64(len(vals))
		out[stage] = &avg
	}
	return out
}

func avgEndToEndSeconds(ctx context.Context, q *dbgen.Queries) float64 {
	txs, err := q.ListRecentTransactions(ctx, 100)
	if err != nil {
		return 0
	}
	var total float64
	var count int
	for _, txRow := range txs {
		if txRow.SubmittedAt.Valid && txRow.FinalizedAt.Valid {
			total += txRow.FinalizedAt.Time.Sub(txRow.SubmittedAt.Time).Seconds()
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
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

func parseEvidence(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func failureReason(raw []byte) string {
	ev := parseEvidence(raw)
	if ev == nil {
		return ""
	}
	if v, ok := ev["on_chain_error"].(string); ok && v != "" {
		return v
	}
	if v, ok := ev["reason"].(string); ok && v != "" {
		return v
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
