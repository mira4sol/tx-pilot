import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  DashboardWebSocket,
  dashboardApi,
  transactionApi,
  type DashboardSnapshot,
  type TransactionRow,
  type WsEnvelope,
} from "@/utils/http";
import {
  mapAIDecision,
  mapChartsToSeries,
  mapFailure,
  mapLeaderWindows,
  mapPipelineCounts,
  mapPipelineLatencies,
  mapRecoverySteps,
  mapSlotFeed,
  mapTimelineEvents,
  mapTransactionRow,
  statusBarMetrics,
  healthMetrics,
  bundleMetricTiles,
  landingFactors,
  capSlotFeed,
  type AIDecision,
  type FailureEntry,
  type LeaderWindowView,
  type RecoveryStepView,
  type SlotFeedView,
  type TimelineEventView,
  type Tx,
} from "@/utils/dashboard-mappers";

const SNAPSHOT_KEY = ["dashboard", "snapshot"] as const;
const CHARTS_KEY = ["dashboard", "charts"] as const;
const TIMELINE_KEY = (id: string) => ["transaction", id, "timeline"] as const;

function mergeTransactionRows(existing: TransactionRow[], incoming: TransactionRow): TransactionRow[] {
  const rows = [incoming, ...existing.filter((r) => r.transaction_id !== incoming.transaction_id)];
  return rows.slice(0, 50);
}

function parseSlotFeedPayload(payload: unknown): DashboardSnapshot["slots"] {
  if (Array.isArray(payload)) {
    return { slots: capSlotFeed(payload as DashboardSnapshot["slots"]["slots"]) };
  }
  if (payload && typeof payload === "object" && "slots" in payload) {
    const slots = (payload as DashboardSnapshot["slots"]).slots;
    return { slots: capSlotFeed(slots) };
  }
  return { slots: [] };
}

function normalizeSnapshotSlots(snapshot: DashboardSnapshot): DashboardSnapshot {
  return {
    ...snapshot,
    slots: { slots: capSlotFeed(snapshot.slots?.slots) },
  };
}

function applyWsUpdate(snapshot: DashboardSnapshot, message: WsEnvelope): DashboardSnapshot {
  switch (message.type) {
    case "network.ticker.updated":
      return { ...snapshot, network: message.payload as DashboardSnapshot["network"] };
    case "slots.feed.updated":
      return { ...snapshot, slots: parseSlotFeedPayload(message.payload) };
    case "leaders.schedule.updated":
      return { ...snapshot, leaders: message.payload as DashboardSnapshot["leaders"] };
    case "lifecycle.pipeline.updated":
      return { ...snapshot, pipeline: message.payload as DashboardSnapshot["pipeline"] };
    case "landing.probability.updated":
      return { ...snapshot, landing_probability: message.payload as DashboardSnapshot["landing_probability"] };
    case "recovery.actions.updated":
      return { ...snapshot, recovery: message.payload as DashboardSnapshot["recovery"] };
    case "transaction.updated": {
      const raw = message.payload as Record<string, unknown>;
      const sigObj = raw.signature as { String?: string; Valid?: boolean } | string | undefined;
      const sig = typeof sigObj === "string" ? sigObj : sigObj?.Valid ? sigObj.String ?? "" : "";
      const bundleObj = raw.bundle_id as { String?: string; Valid?: boolean } | string | undefined;
      const bundleId = typeof bundleObj === "string" ? bundleObj : bundleObj?.Valid ? bundleObj.String ?? "" : "";
      const slotObj = raw.submitted_slot as { Int64?: number; Valid?: boolean } | number | undefined;
      const slot = typeof slotObj === "number" ? slotObj : slotObj?.Valid ? Number(slotObj.Int64 ?? 0) : 0;
      const leaderObj = raw.leader as { String?: string; Valid?: boolean } | string | undefined;
      const leader = typeof leaderObj === "string" ? leaderObj : leaderObj?.Valid ? leaderObj.String ?? "" : "";
      const tipLamports = Number(raw.tip_lamports ?? 0);
      const mapped: TransactionRow = {
        transaction_id: String(raw.transaction_id ?? raw.id ?? ""),
        signature: sig,
        slot,
        bundle_id: bundleId,
        tip_lamports: tipLamports,
        tip_sol: tipLamports / 1e9,
        status: String(raw.status ?? "submitted"),
        latency_ms: Number(raw.latency_ms ?? 0),
        leader,
        retry_attempt: Number(raw.retry_attempt ?? 0),
        agent_decision_id: String(raw.agent_decision_id ?? ""),
      };
      const rows = mergeTransactionRows(snapshot.transactions.rows, mapped);
      return {
        ...snapshot,
        transactions: {
          ...snapshot.transactions,
          rows,
          row_count: rows.length,
        },
      };
    }
    case "agent.decision_made": {
      const payload = message.payload as DashboardSnapshot["ai_decisions"]["decisions"][number];
      const decisions = [payload, ...snapshot.ai_decisions.decisions.filter((d) => d.decision_id !== payload.decision_id)].slice(0, 20);
      return { ...snapshot, ai_decisions: { decisions } };
    }
    case "failure.recorded": {
      const payload = message.payload as DashboardSnapshot["failures"]["failures"][number] & {
        Kind?: string;
        Title?: string;
        RecommendedAction?: string;
        failure_id?: string;
      };
      const failure = {
        failure_id: payload.failure_id ?? "",
        kind: payload.kind ?? payload.Kind ?? "unknown",
        title: payload.title ?? payload.Title ?? "Failure",
        slot: payload.slot ?? 0,
        severity: payload.severity ?? "warning",
        recommended_action: payload.recommended_action ?? payload.RecommendedAction ?? "",
        transaction_id: payload.transaction_id ?? "",
        bundle_id: payload.bundle_id ?? "",
        created_at: payload.created_at ?? new Date().toISOString(),
      };
      const failures = [failure, ...snapshot.failures.failures.filter((f) => f.failure_id !== failure.failure_id)].slice(0, 20);
      return { ...snapshot, failures: { failures } };
    }
    default:
      return snapshot;
  }
}

export function useDashboardData() {
  const queryClient = useQueryClient();
  const wsRef = useRef<DashboardWebSocket | null>(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [slotFeed, setSlotFeed] = useState<SlotFeedView[]>([]);

  const snapshotQuery = useQuery({
    queryKey: SNAPSHOT_KEY,
    queryFn: async () => normalizeSnapshotSlots(await dashboardApi.snapshot()),
    refetchInterval: wsConnected ? false : 2000,
    staleTime: 1000,
  });

  const chartsQuery = useQuery({
    queryKey: CHARTS_KEY,
    queryFn: () => dashboardApi.charts("network_health", "15m"),
    refetchInterval: wsConnected ? 5000 : 3000,
    staleTime: 2000,
  });

  useEffect(() => {
    const ws = new DashboardWebSocket();
    wsRef.current = ws;

    const offStatus = ws.onStatus(setWsConnected);
    const offMessage = ws.onMessage((message) => {
      queryClient.setQueryData<DashboardSnapshot>(SNAPSHOT_KEY, (prev) => {
        if (!prev) return prev;
        return applyWsUpdate(prev, message);
      });
      if (message.type === "charts.series.updated") {
        void queryClient.invalidateQueries({ queryKey: CHARTS_KEY });
      }
    });

    ws.connect();

    return () => {
      offStatus();
      offMessage();
      ws.disconnect();
      wsRef.current = null;
    };
  }, [queryClient]);

  const snapshot = snapshotQuery.data;
  const charts = chartsQuery.data;

  // Fixed-size rolling window — always exactly DISPLAY_LIMITS.slotFeed rows max.
  useEffect(() => {
    if (!snapshot?.slots?.slots?.length) return;
    setSlotFeed(mapSlotFeed(snapshot.slots.slots));
  }, [snapshot?.network?.slot, snapshot?.generated_at, snapshot?.slots?.slots]);

  const transactions = useMemo<Tx[]>(
    () => (snapshot?.transactions.rows ?? []).map(mapTransactionRow),
    [snapshot],
  );

  const decisions = useMemo<AIDecision[]>(
    () => (snapshot?.ai_decisions.decisions ?? []).slice(0, 3).map(mapAIDecision),
    [snapshot],
  );

  const failures = useMemo<FailureEntry[]>(
    () => (snapshot?.failures.failures ?? []).slice(0, 4).map(mapFailure),
    [snapshot],
  );

  const leaderWindows = useMemo<LeaderWindowView[]>(
    () => (snapshot ? mapLeaderWindows(snapshot) : []),
    [snapshot],
  );

  const pipelineCounts = useMemo(
    () => (snapshot ? mapPipelineCounts(snapshot) : [0, 0, 0, 0, 0]),
    [snapshot],
  );

  const pipelineLatencies = useMemo(
    () => (snapshot ? mapPipelineLatencies(snapshot) : ["—", "—", "—", "—", "—"]),
    [snapshot],
  );

  const recoverySteps = useMemo<RecoveryStepView[]>(
    () => (snapshot ? mapRecoverySteps(snapshot) : []),
    [snapshot],
  );

  const chartSeries = useMemo(() => mapChartsToSeries(charts), [charts]);
  const ticker = useMemo(() => statusBarMetrics(snapshot), [snapshot]);
  const networkHealth = useMemo(() => healthMetrics(snapshot), [snapshot]);
  const bundleTiles = useMemo(() => bundleMetricTiles(snapshot), [snapshot]);
  const landingFactorBars = useMemo(() => landingFactors(snapshot), [snapshot]);
  const landingProb = snapshot?.landing_probability.probability_pct ?? 0;
  const cluster = snapshot?.network.cluster ?? "mainnet-beta";

  return {
    isLoading: snapshotQuery.isLoading,
    isError: snapshotQuery.isError,
    error: snapshotQuery.error,
    wsConnected,
    refetch: snapshotQuery.refetch,
    cluster,
    ticker,
    slotFeed,
    leaderWindows,
    networkHealth,
    bundleTiles,
    transactions,
    pipelineCounts,
    pipelineLatencies,
    decisions,
    failures,
    landingProb,
    landingFactorBars,
    recoverySteps,
    chartSeries,
  };
}

export function useTransactionTimeline(transactionId: string | null) {
  return useQuery({
    queryKey: transactionId ? TIMELINE_KEY(transactionId) : ["transaction", "none", "timeline"],
    queryFn: () => transactionApi.timeline(transactionId!),
    enabled: Boolean(transactionId),
    select: (data) => mapTimelineEvents(data.events),
  });
}

export type { TimelineEventView };
