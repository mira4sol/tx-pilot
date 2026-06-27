import { get } from "./client";
import type {
  AIDecisions,
  BundleMetrics,
  ChartsResponse,
  DashboardSnapshot,
  FailureAnalysis,
  LandingProbability,
  LeaderSchedule,
  NetworkHealth,
  NetworkTicker,
  Pipeline,
  RecoveryPanel,
  SlotFeed,
  TimelineResponse,
  TransactionStream,
} from "./types";

export const dashboardApi = {
  snapshot: () => get<DashboardSnapshot>("/v1/dashboard/snapshot"),
  network: () => get<NetworkTicker>("/v1/dashboard/network"),
  slots: () => get<SlotFeed>("/v1/dashboard/slots"),
  leaders: () => get<LeaderSchedule>("/v1/dashboard/leaders"),
  health: () => get<NetworkHealth>("/v1/dashboard/health"),
  bundles: () => get<BundleMetrics>("/v1/dashboard/bundles"),
  pipeline: () => get<Pipeline>("/v1/dashboard/pipeline"),
  transactions: () => get<TransactionStream>("/v1/dashboard/transactions"),
  aiDecisions: () => get<AIDecisions>("/v1/dashboard/ai-decisions"),
  landing: () => get<LandingProbability>("/v1/dashboard/landing"),
  failures: () => get<FailureAnalysis>("/v1/dashboard/failures"),
  recovery: () => get<RecoveryPanel>("/v1/dashboard/recovery"),
  charts: (series = "network_health", window = "15m") =>
    get<ChartsResponse>("/v1/dashboard/charts", { series, window }),
};

export const transactionApi = {
  timeline: (id: string) => get<TimelineResponse>(`/v1/transactions/${id}/timeline`),
};

export const healthApi = {
  check: () => get<{ status: string }>("/healthz"),
};
