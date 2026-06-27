export interface NetworkTicker {
  slot: number;
  current_leader: string;
  next_jito_leader: string;
  congestion_pct: number;
  bundle_land_rate_pct: number;
  tip_floor_lamports: number;
  tip_floor_sol: number;
  tps: number;
  slot_drift_ms: number;
  network_health_pct: number;
  cluster: string;
}

export interface SlotEntry {
  slot: number;
  age_ms: number;
  leader: string;
  status: string;
  jito_leader: boolean;
  skipped: boolean;
}

export interface SlotFeed {
  slots: SlotEntry[];
}

export interface LeaderWindow {
  identity: string;
  slot_start: number;
  slot_end: number;
  jito: boolean;
  quality: string;
  current: boolean;
}

export interface LeaderSchedule {
  current_leader: string;
  next_jito_leader: string;
  leaders: LeaderWindow[];
}

export interface NetworkHealth {
  congestion_pct: number;
  confirmation_latency_ms: number;
  slot_jitter_ms: number;
  validator_stability_pct: number;
  network_health_pct: number;
}

export interface BundleMetrics {
  sent: number;
  landed: number;
  success_pct: number;
  avg_tip_lamports: number;
  avg_tip_sol: number;
  retries: number;
  pending: number;
}

export interface PipelineStage {
  stage: string;
  label: string;
  count: number;
  delta_from_previous_ms: number | null;
}

export interface PipelineSummary {
  total_in_flight: number;
  avg_end_to_end_seconds: number;
  success_rate_pct: number;
  failed_today: number;
  retried: number;
}

export interface Pipeline {
  realtime: boolean;
  stages: PipelineStage[];
  summary: PipelineSummary;
}

export interface TransactionRow {
  transaction_id: string;
  signature: string;
  slot: number;
  bundle_id: string;
  tip_lamports: number;
  tip_sol: number;
  status: string;
  latency_ms: number;
  leader: string;
  retry_attempt: number;
  agent_decision_id?: string;
}

export interface TransactionStream {
  rows: TransactionRow[];
  row_count: number;
  streaming: boolean;
}

export interface AIDecisionRow {
  decision_id: string;
  transaction_id?: string;
  type: string;
  title: string;
  summary: string;
  inputs?: Record<string, unknown>;
  action?: Record<string, unknown>;
  confidence_pct: number;
  created_at: string;
}

export interface AIDecisions {
  decisions: AIDecisionRow[];
}

export interface LandingProbability {
  probability_pct: number;
  factors: {
    bundle_tip_adequacy_pct: number;
    leader_stability_pct: number;
    slot_competition_pct: number;
    network_readiness_pct: number;
  };
}

export interface FailureRow {
  failure_id: string;
  kind: string;
  title: string;
  reason?: string;
  slot: number;
  severity: string;
  recommended_action: string;
  transaction_id: string;
  bundle_id: string;
  created_at: string;
}

export interface FailureAnalysis {
  failures: FailureRow[];
}

export interface RecoveryRow {
  action_id: string;
  label: string;
  status: string;
  transaction_id: string;
  decision_id: string;
}

export interface RecoveryPanel {
  actions: RecoveryRow[];
}

export interface DashboardSnapshot {
  generated_at: string;
  network: NetworkTicker;
  slots: SlotFeed;
  leaders: LeaderSchedule;
  health: NetworkHealth;
  bundles: BundleMetrics;
  pipeline: Pipeline;
  transactions: TransactionStream;
  ai_decisions: AIDecisions;
  landing_probability: LandingProbability;
  failures: FailureAnalysis;
  recovery: RecoveryPanel;
}

export interface ChartPointPayload {
  congestion_pct?: number;
  tip_floor_lamports?: number;
  confirmation_latency_ms?: number;
  processed_to_confirmed_ms?: number;
  confirmed_to_finalized_ms?: number;
  landed?: number;
  failed?: number;
  pending?: number;
  avg_tip_lamports?: number;
  success_rate_pct?: number;
}

export interface ChartPoint {
  time: string;
  payload: ChartPointPayload;
}

export interface ChartsResponse {
  series: string;
  window: string;
  points: ChartPoint[];
}

export interface LifecycleEvent {
  id: number;
  transaction_id: string;
  signature: string;
  bundle_id: string;
  stage: string;
  slot: number;
  latency_ms: number | null;
  created_at: string;
  metadata?: Record<string, unknown>;
}

export interface TimelineResponse {
  events: LifecycleEvent[];
}

export interface WsEnvelope<T = unknown> {
  type: string;
  sequence: number;
  server_time: string;
  payload: T;
}

export const WS_CHANNELS = [
  "network.ticker",
  "slots.feed",
  "leaders.schedule",
  "lifecycle.pipeline",
  "transactions.stream",
  "ai.decisions",
  "landing.probability",
  "failures.analysis",
  "recovery.actions",
  "charts.series",
] as const;
