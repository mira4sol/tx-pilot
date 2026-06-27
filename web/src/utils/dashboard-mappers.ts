import type {
  AIDecisionRow,
  ChartsResponse,
  DashboardSnapshot,
  FailureRow,
  LifecycleEvent,
  SlotEntry,
  TransactionRow,
} from "@/utils/http";

export type TxStatus = "finalized" | "confirmed" | "processing" | "submitted" | "failed";
export type AIType = "tip_up" | "delay" | "warning" | "strategy" | "recovery";
export type FailureType = "blockhash" | "rejection" | "low_tip" | "compute" | "skipped";

export interface Tx {
  id: string;
  sig: string;
  sigFull: string;
  slot: number;
  bundleId: string;
  tip: string;
  status: TxStatus;
  latency: number;
  leader: string;
  leaderFull: string;
  retries: number;
  ts: number;
}

export interface AIDecision {
  id: string;
  type: AIType;
  title: string;
  reasoning: string;
  confidence: number;
  context: string;
}

export interface FailureEntry {
  id: string;
  type: FailureType;
  label: string;
  action: string;
  slot: number;
  ts: number;
}

export interface LeaderWindowView {
  leader: string;
  slots: string;
  status: "active" | "next" | "upcoming";
  jito: boolean;
}

export interface SlotFeedView {
  slot: number;
  skipped: boolean;
  leader: string;
  txCount: number;
}

export interface RecoveryStepView {
  label: string;
  done: boolean;
}

/** Visible row caps for sidebar / stream panels (data still streams in full). */
export const DISPLAY_LIMITS = {
  slotFeed: 8,
  leaderSchedule: 5,
  transactionStream: 12,
} as const;

export const SLOT_FEED_ROW_HEIGHT = 22;

/** Keep newest slots only — dedupe by slot number, newest first. */
export function capSlotFeed(slots: SlotEntry[] | undefined, limit = DISPLAY_LIMITS.slotFeed): SlotEntry[] {
  if (!Array.isArray(slots) || slots.length === 0) return [];
  const seen = new Set<number>();
  const out: SlotEntry[] = [];
  for (const entry of slots) {
    if (seen.has(entry.slot)) continue;
    seen.add(entry.slot);
    out.push(entry);
  }
  out.sort((a, b) => b.slot - a.slot);
  return out.slice(0, limit);
}

export interface TimelineEventView {
  label: string;
  ts: string;
  slot: number;
  delta: string | null;
  color: string;
}

const STATUS_MAP: Record<string, TxStatus> = {
  finalized: "finalized",
  confirmed: "confirmed",
  processed: "processing",
  processing: "processing",
  submitted: "submitted",
  failed: "failed",
  created: "submitted",
};

const AI_TYPE_MAP: Record<string, AIType> = {
  tip_adjustment: "tip_up",
  submission_delay: "delay",
  leader_avoidance: "warning",
  retry_backoff: "strategy",
  blockhash_refresh: "recovery",
  abort: "warning",
};

const FAILURE_TYPE_MAP: Record<string, FailureType> = {
  expired_blockhash: "blockhash",
  bundle_rejected: "rejection",
  tip_below_floor: "low_tip",
  compute_exceeded: "compute",
  leader_skipped: "skipped",
  stream_gap: "skipped",
  rpc_error: "rejection",
  unknown: "rejection",
};

const STAGE_LABELS: Record<string, string> = {
  submitted: "Bundle Submitted",
  processed: "Processed by Leader",
  confirmed: "Confirmed",
  finalized: "Finalized",
  failed: "Failed",
};

const STAGE_COLORS: Record<string, string> = {
  submitted: "#8b5cf6",
  processed: "#00d4ff",
  confirmed: "#3b82f6",
  finalized: "#00e87a",
  failed: "#ef4444",
};

function formatSlotRange(start: number, end: number): string {
  return `${start.toLocaleString()}–${end.toLocaleString()}`;
}

function shortMiddle(value: string, head = 4, tail = 4): string {
  if (!value) return "—";
  if (value.length <= head + tail + 1) return value;
  return `${value.slice(0, head)}…${value.slice(-tail)}`;
}

function shortBundleId(id: string): string {
  if (!id) return "—";
  if (/^JTO-/i.test(id)) return id.length > 12 ? shortMiddle(id, 4, 4) : id;
  const compact = id.replace(/-/g, "");
  if (compact.length <= 8) return compact.toUpperCase();
  return `JTO-${compact.slice(0, 6).toUpperCase()}`;
}

function formatTipSol(tipSol: number): string {
  if (tipSol === 0) return "—";
  const trimmed = tipSol.toFixed(5).replace(/(\.\d*?)0+$/, "$1").replace(/\.$/, "");
  return `${trimmed}◎`;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
}

export function mapTransactionRow(row: TransactionRow): Tx {
  const sigFull = row.signature || "";
  const leaderFull = row.leader || "Unknown";
  return {
    id: row.transaction_id,
    sig: shortMiddle(sigFull, 4, 4),
    sigFull,
    slot: row.slot,
    bundleId: shortBundleId(row.bundle_id),
    tip: formatTipSol(row.tip_sol),
    status: STATUS_MAP[row.status] ?? "submitted",
    latency: row.latency_ms,
    leader: shortMiddle(leaderFull, 4, 4),
    leaderFull,
    retries: row.retry_attempt,
    ts: Date.now(),
  };
}

export function mapAIDecision(row: AIDecisionRow): AIDecision {
  const congestion = row.inputs?.congestion_pct;
  const slot = row.inputs?.slot;
  const contextParts: string[] = [];
  if (typeof congestion === "number") contextParts.push(`Congestion ${congestion}%`);
  if (typeof slot === "number") contextParts.push(`Slot ${Number(slot).toLocaleString()}`);

  return {
    id: row.decision_id,
    type: AI_TYPE_MAP[row.type] ?? "strategy",
    title: row.title,
    reasoning: row.summary,
    confidence: row.confidence_pct / 100,
    context: contextParts.join(" · ") || row.transaction_id || "",
  };
}

export function mapFailure(row: FailureRow): FailureEntry {
  return {
    id: row.failure_id,
    type: FAILURE_TYPE_MAP[row.kind] ?? "rejection",
    label: row.title,
    action: row.recommended_action,
    slot: row.slot,
    ts: new Date(row.created_at).getTime() || Date.now(),
  };
}

export function mapLeaderWindows(snapshot: DashboardSnapshot): LeaderWindowView[] {
  const leaders = snapshot.leaders.leaders.slice(0, DISPLAY_LIMITS.leaderSchedule);
  const currentIdx = leaders.findIndex((l) => l.current);
  return leaders.map((l, i) => ({
    leader: l.identity,
    slots: formatSlotRange(l.slot_start, l.slot_end),
    status: l.current ? "active" : i === currentIdx + 1 ? "next" : "upcoming",
    jito: l.jito,
  }));
}

export function mapSlotFeed(slots: SlotEntry[] | undefined): SlotFeedView[] {
  return capSlotFeed(slots).map((s) => ({
    slot: s.slot,
    skipped: s.skipped,
    leader: s.leader || "Unknown",
    txCount: s.skipped ? 0 : Math.max(0, 150 - Math.floor(s.age_ms / 10)),
  }));
}

export function mapPipelineCounts(snapshot: DashboardSnapshot): number[] {
  const order = ["created", "submitted", "processed", "confirmed", "finalized"];
  return order.map((stage) => {
    const found = snapshot.pipeline.stages.find((s) => s.stage === stage);
    return found?.count ?? 0;
  });
}

export function mapPipelineLatencies(snapshot: DashboardSnapshot): string[] {
  return snapshot.pipeline.stages.map((stage, i) => {
    if (i === 0 || stage.delta_from_previous_ms == null) return "—";
    return `${stage.delta_from_previous_ms}ms`;
  });
}

export function mapRecoverySteps(snapshot: DashboardSnapshot): RecoveryStepView[] {
  return snapshot.recovery.actions.slice(0, 5).map((action) => ({
    label: action.label,
    done: action.status === "completed",
  }));
}

export function mapTimelineEvents(events: LifecycleEvent[]): TimelineEventView[] {
  return events.map((event, i) => {
    const prev = i > 0 ? events[i - 1] : null;
    let delta: string | null = null;
    const latency = parseOptionalNumber(event.latency_ms);
    const slot = parseOptionalNumber(event.slot) ?? 0;
    if (prev && latency != null) {
      delta = `+${latency}ms`;
    }
    return {
      label: STAGE_LABELS[event.stage] ?? event.stage,
      ts: formatTime(parseTimestamp(event.created_at)),
      slot,
      delta,
      color: STAGE_COLORS[event.stage] ?? "#8899bb",
    };
  });
}

function parseOptionalNumber(value: unknown): number | null {
  if (typeof value === "number") return value;
  if (value && typeof value === "object" && "Int64" in value) {
    const obj = value as { Int64?: number; Valid?: boolean };
    return obj.Valid ? Number(obj.Int64 ?? 0) : null;
  }
  return null;
}

function parseTimestamp(value: unknown): string {
  if (typeof value === "string") return value;
  if (value && typeof value === "object" && "Time" in value) {
    const obj = value as { Time?: string; Valid?: boolean };
    if (obj.Valid && obj.Time) return obj.Time;
  }
  return new Date().toISOString();
}

export function mapChartsToSeries(charts: ChartsResponse | undefined) {
  const points = charts?.points ?? [];
  const latencyData = points.map((p, i) => ({
    t: i,
    v: p.payload.confirmation_latency_ms ?? 0,
    v2: (p.payload.confirmation_latency_ms ?? 0) + 400,
  }));
  const congestionData = points.map((p, i) => ({
    t: i,
    v: p.payload.congestion_pct ?? 0,
    v2: (p.payload.congestion_pct ?? 0) + 10,
  }));
  const bundleData = points.map((p, i) => ({
    t: i,
    v: p.payload.landed ?? Math.max(0, 40 - i),
    v2: p.payload.failed ?? 2,
  }));
  const tipData = points.map((p, i) => ({
    t: i,
    v: p.payload.success_rate_pct ?? Math.min(99, 80 + i),
  }));
  const failureData = points.map((p, i) => ({
    t: i,
    v: Math.max(0, (p.payload.failed ?? 0) + (i % 3)),
  }));

  return { latencyData, congestionData, bundleData, tipData, failureData };
}

export function landingFactors(snapshot: DashboardSnapshot | undefined) {
  const factors = snapshot?.landing_probability.factors;
  return [
    { label: "Tip adequacy", value: (factors?.bundle_tip_adequacy_pct ?? 0) / 100 },
    { label: "Leader stability", value: (factors?.leader_stability_pct ?? 0) / 100 },
    { label: "Slot competition", value: (factors?.slot_competition_pct ?? 0) / 100 },
    { label: "Network readiness", value: (factors?.network_readiness_pct ?? 0) / 100 },
  ];
}

export function healthMetrics(snapshot: DashboardSnapshot | undefined) {
  const health = snapshot?.health;
  return [
    { label: "Congestion", value: health?.congestion_pct ?? 0, max: 100, unit: "%", color: "#f59e0b" },
    { label: "Conf. Latency", value: health?.confirmation_latency_ms ?? 0, max: 1000, unit: "ms", color: "#00d4ff" },
    { label: "Slot Jitter", value: health?.slot_jitter_ms ?? 0, max: 20, unit: "ms", color: "#00e87a" },
    { label: "Validator Stab.", value: health?.validator_stability_pct ?? 0, max: 100, unit: "%", color: "#00e87a" },
  ];
}

export function bundleMetricTiles(snapshot: DashboardSnapshot | undefined) {
  const bundles = snapshot?.bundles;
  return [
    { label: "SENT", value: (bundles?.sent ?? 0).toLocaleString(), color: "#c0d0e0" },
    { label: "LANDED", value: (bundles?.landed ?? 0).toLocaleString(), color: "#00e87a" },
    { label: "SUCCESS", value: `${(bundles?.success_pct ?? 0).toFixed(1)}%`, color: "#00d4ff" },
    { label: "AVG TIP", value: formatTipSol(bundles?.avg_tip_sol ?? 0), color: "#f59e0b" },
    { label: "RETRIES", value: String(bundles?.retries ?? 0), color: "#8899bb" },
    { label: "PENDING", value: String(bundles?.pending ?? 0), color: "#8b5cf6" },
  ];
}

export function statusBarMetrics(snapshot: DashboardSnapshot | undefined) {
  const network = snapshot?.network;
  const congestion = network?.congestion_pct ?? 0;
  const health = network?.network_health_pct ?? 0;
  return {
    slot: network?.slot ?? 0,
    tps: network?.tps ?? 0,
    health,
    congestion,
    currentLeader: network?.current_leader ?? "Unknown",
    nextJitoLeader: network?.next_jito_leader ?? "Unknown",
    bundleLandRate: network?.bundle_land_rate_pct ?? 0,
    tipFloor: formatTipSol(network?.tip_floor_sol ?? 0),
    slotDrift: network?.slot_drift_ms ?? 0,
    congColor: congestion > 80 ? "#ef4444" : congestion > 60 ? "#f59e0b" : "#00e87a",
    healthColor: health > 80 ? "#00e87a" : health > 60 ? "#f59e0b" : "#ef4444",
  };
}
