import { useState, useCallback, useEffect, type ElementType, type ReactNode } from "react";
import {
  Activity, AlertCircle, AlertTriangle, Brain, CheckCircle2,
  Clock, Cpu, Copy, Eye, ExternalLink, GitBranch, Layers, Network, Radio,
  RefreshCw, Shield, Target, TrendingUp, X, XCircle, Zap,
  ArrowRight, BarChart3, Server, ChevronRight
} from "lucide-react";
import {
  AreaChart, Area, LineChart, Line, BarChart, Bar,
  XAxis, YAxis, Tooltip, ResponsiveContainer,
} from "recharts";
import { useDashboardData, useTransactionTimeline } from "@/hooks/useDashboardData";
import type {
  Tx,
  AIDecision,
  FailureEntry,
  AIType,
  FailureType,
  TxStatus,
  LeaderWindowView,
  SlotFeedView,
  RecoveryStepView,
  TimelineEventView,
} from "@/utils/dashboard-mappers";
import { DISPLAY_LIMITS, SLOT_FEED_ROW_HEIGHT } from "@/utils/dashboard-mappers";
import { explorerAddressUrl, explorerSlotUrl, solscanTxUrl } from "@/utils/explorer";

// ─────────────────────────────────────────────────────────────
// DESIGN TOKENS
// ─────────────────────────────────────────────────────────────

const C = {
  cyan: "#00d4ff",
  green: "#00e87a",
  amber: "#f59e0b",
  red: "#ef4444",
  purple: "#8b5cf6",
  blue: "#3b82f6",
  bg: "#07090e",
  card: "#0c1018",
  cardAlt: "#0e1320",
  border: "rgba(0,212,255,0.09)",
  borderBright: "rgba(0,212,255,0.22)",
  text: "#c0d0e0",
  sub: "#8899bb",
  muted: "#4a6070",
  dim: "#1e2e3e",
  mono: "'JetBrains Mono', monospace",
  sans: "'Barlow', sans-serif",
} as const;

// ─────────────────────────────────────────────────────────────
// CSS ANIMATIONS (injected once)
// ─────────────────────────────────────────────────────────────

const AEGIS_CSS = `
  @keyframes aegis-pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.35; transform: scale(0.75); }
  }
  @keyframes aegis-glow {
    0%, 100% { box-shadow: 0 0 8px rgba(0,212,255,0.35); }
    50% { box-shadow: 0 0 22px rgba(0,212,255,0.75), 0 0 44px rgba(0,212,255,0.2); }
  }
  @keyframes aegis-flow {
    0% { left: -12%; opacity: 0; }
    8% { opacity: 1; }
    92% { opacity: 1; }
    100% { left: 108%; opacity: 0; }
  }
  @keyframes aegis-row-in {
    from { opacity: 0; transform: translateY(-5px); background: rgba(0,212,255,0.06); }
    to   { opacity: 1; transform: translateY(0);    background: transparent; }
  }
  @keyframes aegis-scan {
    0%   { transform: translateY(0); opacity: 0.6; }
    100% { transform: translateY(100%); opacity: 0; }
  }
  @keyframes aegis-spin-slow {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
  }
  @keyframes aegis-fade-in {
    from { opacity: 0; transform: translateY(4px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  @keyframes aegis-blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.2; }
  }
  .aegis-node-glow { animation: aegis-glow 2.2s ease-in-out infinite; }
  .aegis-live-dot  { animation: aegis-pulse 1.4s ease-in-out infinite; }
  .aegis-row-in    { animation: aegis-row-in 0.4s ease forwards; }
  .aegis-fade-in   { animation: aegis-fade-in 0.5s ease forwards; }

  .aegis-scroll::-webkit-scrollbar { width: 3px; height: 3px; }
  .aegis-scroll::-webkit-scrollbar-track { background: transparent; }
  .aegis-scroll::-webkit-scrollbar-thumb { background: rgba(0,212,255,0.18); border-radius: 2px; }
  .aegis-scroll:hover::-webkit-scrollbar-thumb { background: rgba(0,212,255,0.38); }

  body { font-family: 'Barlow', sans-serif; background: #07090e; }
`;

// ─────────────────────────────────────────────────────────────
// ATOMIC COMPONENTS
// ─────────────────────────────────────────────────────────────

const LiveDot = ({ color = C.green, size = 6 }: { color?: string; size?: number }) => (
  <span
    className="aegis-live-dot inline-block rounded-full flex-shrink-0"
    style={{ width: size, height: size, background: color, boxShadow: `0 0 ${size + 2}px ${color}` }}
  />
);

const SectionLabel = ({ children, icon: Icon }: { children: React.ReactNode; icon?: React.ElementType }) => (
  <div className="flex items-center gap-1.5 mb-2.5">
    {Icon && <Icon size={9} color={C.muted} />}
    <span style={{ fontFamily: C.mono, fontSize: 9, letterSpacing: "0.2em", color: C.muted, textTransform: "uppercase" as const }}>
      {children}
    </span>
  </div>
);

const Divider = () => <div style={{ height: 1, background: C.border, marginBlock: "12px" }} />;

const StatusBadge = ({ status }: { status: TxStatus }) => {
  const map: Record<TxStatus, [string, string]> = {
    finalized:  [C.green,  "FINALIZED"],
    confirmed:  [C.cyan,   "CONFIRMED"],
    processing: [C.amber,  "PROCESSING"],
    submitted:  [C.blue,   "SUBMITTED"],
    failed:     [C.red,    "FAILED"],
  };
  const [color, label] = map[status];
  return (
    <span style={{
      fontFamily: C.mono, fontSize: 9, letterSpacing: "0.1em",
      color, background: `${color}18`, border: `1px solid ${color}38`,
      padding: "1px 5px", borderRadius: 3,
    }}>
      {label}
    </span>
  );
};

const ConfBar = ({ value, color = C.cyan, label }: { value: number; color?: string; label?: string }) => (
  <div className="flex items-center gap-2">
    <div className="flex-1 rounded-full" style={{ height: 2, background: C.dim }}>
      <div className="rounded-full transition-all duration-700"
        style={{ height: 2, width: `${value * 100}%`, background: color, boxShadow: `0 0 6px ${color}60` }} />
    </div>
    <span style={{ fontFamily: C.mono, fontSize: 10, color: C.muted, minWidth: 28, textAlign: "right" as const }}>
      {label ?? `${Math.round(value * 100)}%`}
    </span>
  </div>
);

const AITypeIcon = ({ type }: { type: AIType }) => {
  const map: Record<AIType, [React.ElementType, string]> = {
    tip_up:   [TrendingUp,  C.cyan],
    delay:    [Clock,       C.amber],
    warning:  [AlertTriangle, C.amber],
    strategy: [Cpu,         C.purple],
    recovery: [RefreshCw,   C.green],
  };
  const [Icon, color] = map[type];
  return <Icon size={11} style={{ color, flexShrink: 0 }} />;
};

const FailureTypeIcon = ({ type }: { type: FailureType }) => {
  const map: Record<FailureType, [React.ElementType, string]> = {
    blockhash: [Clock,         C.amber],
    rejection: [XCircle,       C.red],
    low_tip:   [TrendingUp,    C.amber],
    compute:   [Cpu,           C.red],
    skipped:   [AlertCircle,   C.amber],
  };
  const [Icon, color] = map[type];
  return <Icon size={10} style={{ color, flexShrink: 0 }} />;
};

const Card = ({ children, style }: { children: React.ReactNode; style?: React.CSSProperties }) => (
  <div style={{
    background: C.card,
    border: `1px solid ${C.border}`,
    borderRadius: 6,
    padding: "12px 14px",
    ...style,
  }}>
    {children}
  </div>
);

const ChartTooltip = ({ active, payload }: { active?: boolean; payload?: { name: string; value: number; color: string }[] }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{
      background: "#0a1018", border: `1px solid ${C.borderBright}`,
      borderRadius: 4, padding: "6px 10px",
    }}>
      {payload.map((p) => (
        <div key={p.name} style={{ fontFamily: C.mono, fontSize: 10, color: p.color }}>
          {p.name}: {p.value}
        </div>
      ))}
    </div>
  );
};

const TX_GRID = "minmax(0,1.35fr) minmax(0,0.85fr) minmax(0,0.7fr) minmax(0,0.65fr) minmax(0,0.75fr) minmax(0,0.5fr) minmax(0,0.65fr) minmax(0,0.35fr)";

function TruncCell({
  children,
  color = C.text,
  size = 10,
  mono = true,
  title,
}: {
  children: ReactNode;
  color?: string;
  size?: number;
  mono?: boolean;
  title?: string;
}) {
  return (
    <span
      title={title}
      style={{
        fontFamily: mono ? C.mono : C.sans,
        fontSize: size,
        color,
        overflow: "hidden",
        textOverflow: "ellipsis",
        whiteSpace: "nowrap",
        minWidth: 0,
        display: "block",
      }}
    >
      {children}
    </span>
  );
}

function CopyButton({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard unavailable.
    }
  }, [value]);

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={copied ? "Copied!" : label ? `Copy ${label}` : "Copy"}
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        width: 22,
        height: 22,
        borderRadius: 4,
        border: `1px solid ${copied ? C.green + "50" : C.border}`,
        background: copied ? `${C.green}14` : `${C.cardAlt}`,
        color: copied ? C.green : C.muted,
        cursor: "pointer",
        flexShrink: 0,
        padding: 0,
      }}
    >
      {copied ? <CheckCircle2 size={11} /> : <Copy size={11} />}
    </button>
  );
}

function TraceMetaRow({
  label,
  display,
  value,
  href,
}: {
  label: string;
  display: string;
  value: string;
  href?: string;
}) {
  const openExternal = useCallback((e: React.MouseEvent) => {
    if (!href) return;
    e.preventDefault();
    window.open(href, "_blank", "noopener,noreferrer");
  }, [href]);

  return (
    <div style={{
      display: "flex",
      alignItems: "center",
      gap: 8,
      padding: "6px 8px",
      borderRadius: 4,
      background: C.cardAlt,
      border: `1px solid ${C.border}`,
    }}>
      <span style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.14em", width: 52, flexShrink: 0 }}>
        {label}
      </span>
      {href ? (
        <a
          href={href}
          onClick={openExternal}
          title={value}
          style={{
            flex: 1,
            minWidth: 0,
            fontFamily: C.mono,
            fontSize: 10,
            color: C.cyan,
            textDecoration: "none",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            display: "inline-flex",
            alignItems: "center",
            gap: 4,
          }}
        >
          {display}
          <ExternalLink size={10} style={{ flexShrink: 0, opacity: 0.7 }} />
        </a>
      ) : (
        <span style={{
          flex: 1,
          minWidth: 0,
          fontFamily: C.mono,
          fontSize: 10,
          color: C.text,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}>
          {display}
        </span>
      )}
      <CopyButton value={value} label={label.toLowerCase()} />
    </div>
  );
}

function SlotLink({ slot, cluster }: { slot: number; cluster: string }) {
  if (slot <= 0) {
    return (
      <span style={{ fontFamily: C.mono, fontSize: 10, color: C.muted }}>slot —</span>
    );
  }

  const href = explorerSlotUrl(slot, cluster);

  return (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault();
        window.open(href, "_blank", "noopener,noreferrer");
      }}
      title={`Open slot ${slot.toLocaleString()} on Solana Explorer`}
      style={{
        fontFamily: C.mono,
        fontSize: 10,
        fontWeight: 600,
        color: C.cyan,
        textDecoration: "none",
        display: "inline-flex",
        alignItems: "center",
        gap: 3,
        padding: "1px 6px",
        borderRadius: 3,
        background: `${C.cyan}10`,
        border: `1px solid ${C.cyan}28`,
      }}
    >
      slot {slot.toLocaleString()}
      <ExternalLink size={9} style={{ opacity: 0.75 }} />
    </a>
  );
}

// ─────────────────────────────────────────────────────────────
// GLOBAL STATUS BAR
// ─────────────────────────────────────────────────────────────

function GlobalStatusBar({ ticker, wsConnected }: {
  ticker: ReturnType<typeof useDashboardData>["ticker"];
  wsConnected: boolean;
}) {
  const { slot, tps, health, congestion, congColor, healthColor, currentLeader, nextJitoLeader, bundleLandRate, tipFloor, slotDrift } = ticker;

  const metrics = [
    { label: "SLOT",         value: slot.toLocaleString(),              color: C.cyan },
    { label: "LEADER",       value: currentLeader,                      color: C.text },
    { label: "NEXT JITO",    value: nextJitoLeader,                     color: C.sub },
    { label: "CONGESTION",   value: `${congestion}%`,                   color: congColor },
    { label: "BUNDLE RATE",  value: `${bundleLandRate.toFixed(1)}%`,    color: C.green },
    { label: "TIP FLOOR",    value: tipFloor,                           color: C.text },
    { label: "TPS",          value: tps.toLocaleString(),               color: C.cyan },
    { label: "SLOT DRIFT",   value: `+${slotDrift.toFixed(1)}ms`,       color: C.sub },
    { label: "NET HEALTH",   value: `${health}%`,                       color: healthColor },
  ];

  return (
    <div style={{
      height: 46, flexShrink: 0,
      background: "linear-gradient(90deg,#07090e 0%,#0a0e18 50%,#07090e 100%)",
      borderBottom: `1px solid ${C.border}`,
      display: "flex", alignItems: "center", paddingInline: 16,
    }}>
      {/* Logo */}
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginRight: 20, flexShrink: 0 }}>
        <div style={{
          width: 26, height: 26, borderRadius: 5,
          background: "linear-gradient(135deg,rgba(0,212,255,0.18),rgba(0,212,255,0.05))",
          border: `1px solid rgba(0,212,255,0.35)`,
          display: "flex", alignItems: "center", justifyContent: "center",
          boxShadow: "0 0 14px rgba(0,212,255,0.2)",
        }}>
          <Shield size={13} color={C.cyan} />
        </div>
        <span style={{ fontFamily: C.sans, fontWeight: 700, fontSize: 14, color: "#ddeeff", letterSpacing: "0.08em" }}>
          AEGIS
        </span>
        <LiveDot size={5} />
      </div>

      <div style={{ width: 1, height: 22, background: C.border, marginRight: 20 }} />

      {/* Metrics */}
      <div style={{ display: "flex", alignItems: "center", flex: 1, minWidth: 0 }}>
        {metrics.map((m, i) => (
          <div key={m.label} style={{
            display: "flex", flexDirection: "column", justifyContent: "center",
            paddingInline: 14, height: 46,
            borderRight: i < metrics.length - 1 ? `1px solid ${C.border}` : "none",
            flexShrink: 0,
          }}>
            <div style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.18em", marginBottom: 2 }}>{m.label}</div>
            <div style={{ fontFamily: C.mono, fontSize: 12, fontWeight: 600, color: m.color, lineHeight: 1 }}>{m.value}</div>
          </div>
        ))}
      </div>

      {/* Status chip */}
      <div style={{
        display: "flex", alignItems: "center", gap: 6, marginLeft: 12,
        padding: "4px 10px", borderRadius: 4,
        background: `${wsConnected ? C.green : C.amber}12`,
        border: `1px solid ${wsConnected ? C.green : C.amber}28`,
        flexShrink: 0,
      }}>
        <LiveDot size={5} color={wsConnected ? C.green : C.amber} />
        <span style={{ fontFamily: C.mono, fontSize: 9, color: wsConnected ? C.green : C.amber, letterSpacing: "0.1em" }}>
          {wsConnected ? "LIVE WS" : "POLLING"}
        </span>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────
// LEFT SIDEBAR
// ─────────────────────────────────────────────────────────────

function LeftSidebar({ slotFeed, leaderWindows, networkHealth, bundleTiles }: {
  slotFeed: SlotFeedView[];
  leaderWindows: LeaderWindowView[];
  networkHealth: ReturnType<typeof useDashboardData>["networkHealth"];
  bundleTiles: ReturnType<typeof useDashboardData>["bundleTiles"];
}) {
  return (
    <div style={{
      width: 256, flexShrink: 0,
      borderRight: `1px solid ${C.border}`,
      display: "flex", flexDirection: "column",
      overflow: "hidden",
    }}>
      <div className="aegis-scroll" style={{ flex: 1, overflowY: "auto", padding: "14px 12px", display: "flex", flexDirection: "column", gap: 10 }}>

        {/* Live Slot Feed — fixed height rolling window */}
        <Card>
          <SectionLabel icon={Radio}>Live Slot Feed</SectionLabel>
          <div style={{
            display: "flex",
            flexDirection: "column",
            gap: 2,
            height: DISPLAY_LIMITS.slotFeed * SLOT_FEED_ROW_HEIGHT,
            overflow: "hidden",
          }}>
            {Array.from({ length: DISPLAY_LIMITS.slotFeed }, (_, i) => {
              const s = slotFeed[i];
              if (!s) {
                return (
                  <div key={`slot-empty-${i}`} style={{
                    height: SLOT_FEED_ROW_HEIGHT - 2,
                    flexShrink: 0,
                  }} />
                );
              }
              return (
              <div key={`slot-${s.slot}-${s.skipped ? "skip" : "live"}`} style={{
                display: "flex", alignItems: "center", gap: 6,
                padding: "4px 6px", borderRadius: 3,
                height: SLOT_FEED_ROW_HEIGHT - 2,
                flexShrink: 0,
                background: i === 0 ? `${C.cyan}0a` : "transparent",
                border: i === 0 ? `1px solid ${C.cyan}20` : "1px solid transparent",
                transition: "background 0.2s, border-color 0.2s",
              }}>
                <div style={{ width: 6, height: 6, borderRadius: "50%", flexShrink: 0, background: s.skipped ? C.amber : i === 0 ? C.cyan : C.dim, boxShadow: i === 0 ? `0 0 6px ${C.cyan}` : s.skipped ? `0 0 4px ${C.amber}` : "none" }} />
                <span style={{ fontFamily: C.mono, fontSize: 10, color: s.skipped ? C.amber : i === 0 ? C.cyan : C.sub, flex: 1 }}>
                  {s.slot.toLocaleString()}
                </span>
                {s.skipped
                  ? <span style={{ fontFamily: C.mono, fontSize: 8, color: C.amber, letterSpacing: "0.08em" }}>SKIP</span>
                  : <span style={{ fontFamily: C.mono, fontSize: 9, color: C.muted }}>{s.txCount}</span>
                }
              </div>
              );
            })}
          </div>
        </Card>

        {/* Leader Schedule Timeline */}
        <Card>
          <SectionLabel icon={GitBranch}>Leader Schedule</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {leaderWindows.slice(0, DISPLAY_LIMITS.leaderSchedule).map((w, i) => (
              <div key={`${w.leader}-${w.slots}`} style={{
                display: "flex", alignItems: "center", gap: 6,
                padding: "5px 7px", borderRadius: 4,
                background: w.status === "active" ? `${C.cyan}0d` : w.status === "next" ? `${C.green}08` : "transparent",
                border: `1px solid ${w.status === "active" ? C.cyan + "28" : w.status === "next" ? C.green + "20" : C.border}`,
              }}>
                <div style={{
                  width: 3, borderRadius: 2, alignSelf: "stretch", flexShrink: 0,
                  background: w.status === "active" ? C.cyan : w.status === "next" ? C.green : C.dim,
                  boxShadow: w.status === "active" ? `0 0 8px ${C.cyan}` : "none",
                }} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                    <span style={{ fontFamily: C.sans, fontSize: 11, fontWeight: 600, color: w.status === "active" ? C.cyan : w.status === "next" ? C.green : C.sub }}>
                      {w.leader}
                    </span>
                    {w.jito && (
                      <span style={{ fontFamily: C.mono, fontSize: 7, color: C.purple, background: `${C.purple}18`, border: `1px solid ${C.purple}30`, padding: "0 3px", borderRadius: 2, letterSpacing: "0.08em" }}>JITO</span>
                    )}
                  </div>
                  <div style={{ fontFamily: C.mono, fontSize: 9, color: C.muted, marginTop: 1 }}>{w.slots}</div>
                </div>
                {w.status === "active" && <LiveDot size={5} color={C.cyan} />}
              </div>
            ))}
          </div>
        </Card>

        {/* Network Health */}
        <Card>
          <SectionLabel icon={Network}>Network Health</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {[
              ...networkHealth,
            ].map(m => (
              <div key={m.label}>
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 4 }}>
                  <span style={{ fontFamily: C.mono, fontSize: 9, color: C.muted }}>{m.label}</span>
                  <span style={{ fontFamily: C.mono, fontSize: 10, color: m.color, fontWeight: 600 }}>{m.value}{m.unit}</span>
                </div>
                <div style={{ height: 3, borderRadius: 2, background: C.dim }}>
                  <div style={{
                    height: 3, borderRadius: 2,
                    width: `${(m.value / m.max) * 100}%`,
                    background: m.color,
                    boxShadow: `0 0 6px ${m.color}60`,
                    transition: "width 0.6s ease",
                  }} />
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Bundle Metrics */}
        <Card>
          <SectionLabel icon={Target}>Bundle Metrics</SectionLabel>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
            {bundleTiles.map(m => (
              <div key={m.label} style={{
                background: C.cardAlt, borderRadius: 4, padding: "7px 8px",
                border: `1px solid ${C.border}`,
              }}>
                <div style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.14em", marginBottom: 3 }}>{m.label}</div>
                <div style={{ fontFamily: C.mono, fontSize: 13, fontWeight: 600, color: m.color, lineHeight: 1 }}>{m.value}</div>
              </div>
            ))}
          </div>
        </Card>

      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────
// PIPELINE HERO
// ─────────────────────────────────────────────────────────────

function PipelineHero({ counts, latencies }: { counts: number[]; latencies: string[] }) {
  const stages = [
    { label: "CREATED",   color: "#8b5cf6", count: counts[0] },
    { label: "SUBMITTED", color: "#3b82f6", count: counts[1] },
    { label: "PROCESSED", color: C.cyan,    count: counts[2] },
    { label: "CONFIRMED", color: C.green,   count: counts[3] },
    { label: "FINALIZED", color: "#00ff9d", count: counts[4] },
  ];

  return (
    <Card style={{ padding: "16px 18px" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <SectionLabel icon={Layers}>Transaction Lifecycle Pipeline</SectionLabel>
        <div style={{ display: "flex", alignItems: "center", gap: 5 }}>
          <LiveDot size={5} color={C.green} />
          <span style={{ fontFamily: C.mono, fontSize: 9, color: C.green, letterSpacing: "0.1em" }}>REALTIME</span>
        </div>
      </div>

      <div style={{ display: "flex", alignItems: "center" }}>
        {stages.map((stage, i) => (
          <div key={stage.label} style={{ display: "flex", alignItems: "center", flex: 1 }}>
            {/* Stage node */}
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", flexShrink: 0 }}>
              {/* Latency badge above */}
              <div style={{ height: 18, display: "flex", alignItems: "center", marginBottom: 6 }}>
                {i > 0 && (
                  <span style={{ fontFamily: C.mono, fontSize: 9, color: C.muted, letterSpacing: "0.08em" }}>{latencies[i]}</span>
                )}
              </div>

              {/* Node circle */}
              <div
                className={i === 2 ? "aegis-node-glow" : ""}
                style={{
                  width: 48, height: 48, borderRadius: "50%",
                  background: `radial-gradient(circle, ${stage.color}22 0%, ${stage.color}08 100%)`,
                  border: `2px solid ${stage.color}`,
                  display: "flex", alignItems: "center", justifyContent: "center",
                  boxShadow: `0 0 ${i === 2 ? 20 : 10}px ${stage.color}${i === 2 ? "70" : "40"}`,
                  position: "relative",
                }}
              >
                {i === 2 && (
                  <div style={{
                    position: "absolute", inset: -6, borderRadius: "50%",
                    border: `1px solid ${stage.color}20`,
                  }} />
                )}
                <span style={{ fontFamily: C.mono, fontSize: 13, fontWeight: 600, color: stage.color, lineHeight: 1 }}>
                  {i === 0 ? "CR" : i === 1 ? "SB" : i === 2 ? "PR" : i === 3 ? "CF" : "FN"}
                </span>
              </div>

              {/* Count */}
              <div style={{ marginTop: 8, textAlign: "center" as const }}>
                <div style={{ fontFamily: C.mono, fontSize: 15, fontWeight: 600, color: stage.color, lineHeight: 1 }}>
                  {stage.count.toLocaleString()}
                </div>
                <div style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.14em", marginTop: 3 }}>
                  {stage.label}
                </div>
              </div>
            </div>

            {/* Connector */}
            {i < stages.length - 1 && (
              <div style={{ flex: 1, position: "relative", height: 2, margin: "0 4px", marginBottom: 32, overflow: "visible" }}>
                {/* Base line */}
                <div style={{ position: "absolute", inset: 0, background: `linear-gradient(90deg, ${stages[i].color}30, ${stages[i + 1].color}30)`, borderRadius: 1 }} />
                {/* Flowing particles */}
                {[0, 1, 2].map(j => (
                  <div
                    key={j}
                    style={{
                      position: "absolute", top: -1, height: 4, width: "18%",
                      background: `linear-gradient(90deg, transparent, ${stages[i + 1].color}cc, transparent)`,
                      borderRadius: 2,
                      animation: `aegis-flow ${1.8 + i * 0.2}s ${j * 0.6}s infinite linear`,
                    }}
                  />
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Summary strip */}
      <div style={{ marginTop: 14, display: "flex", gap: 8 }}>
        {[
          { label: "Total In-Flight", value: counts.slice(1, 4).reduce((a, b) => a + b, 0).toLocaleString(), color: C.cyan },
          { label: "Avg End-to-End", value: "1.19s", color: C.text },
          { label: "Success Rate", value: "94.2%", color: C.green },
          { label: "Failed Today", value: "127", color: C.red },
          { label: "Retried", value: "342", color: C.amber },
        ].map(m => (
          <div key={m.label} style={{ flex: 1, background: C.cardAlt, borderRadius: 4, padding: "6px 8px", border: `1px solid ${C.border}` }}>
            <div style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, marginBottom: 2, letterSpacing: "0.1em" }}>{m.label}</div>
            <div style={{ fontFamily: C.mono, fontSize: 12, fontWeight: 600, color: m.color }}>{m.value}</div>
          </div>
        ))}
      </div>
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────
// TRANSACTION FEED
// ─────────────────────────────────────────────────────────────

function TxFeed({ transactions, selectedId, onSelect }: {
  transactions: Tx[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const cols = ["SIGNATURE", "SLOT", "BUNDLE", "TIP", "STATUS", "LATENCY", "LEADER", "RETRY"];
  const visible = transactions.slice(0, DISPLAY_LIMITS.transactionStream);

  return (
    <Card style={{ padding: 0, overflow: "hidden" }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "10px 14px 8px", borderBottom: `1px solid ${C.border}` }}>
        <SectionLabel icon={Eye}>Realtime Transaction Stream</SectionLabel>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ fontFamily: C.mono, fontSize: 9, color: C.muted }}>
            {visible.length} rows{transactions.length > visible.length ? ` · ${transactions.length} total` : ""}
          </span>
          <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <LiveDot size={4} color={C.cyan} />
            <span style={{ fontFamily: C.mono, fontSize: 9, color: C.cyan, letterSpacing: "0.08em" }}>STREAMING</span>
          </div>
        </div>
      </div>

      <div style={{ overflowX: "auto" }}>
        {/* Column headers */}
        <div style={{
          display: "grid",
          gridTemplateColumns: TX_GRID,
          gap: 8,
          padding: "5px 14px",
          borderBottom: `1px solid ${C.border}`,
          background: C.cardAlt,
          minWidth: 620,
        }}>
          {cols.map(c => (
            <div key={c} style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.16em", minWidth: 0 }}>{c}</div>
          ))}
        </div>

        {/* Rows */}
        <div className="aegis-scroll" style={{ maxHeight: 220, overflowY: "auto", minWidth: 620 }}>
          {visible.length === 0 ? (
            <div style={{ padding: "24px 14px", textAlign: "center", fontFamily: C.mono, fontSize: 10, color: C.muted }}>
              Waiting for transactions…
            </div>
          ) : visible.map((tx, i) => (
            <div
              key={tx.id}
              className={i === 0 ? "aegis-row-in" : ""}
              onClick={() => onSelect(tx.id === selectedId ? "" : tx.id)}
              style={{
                display: "grid",
                gridTemplateColumns: TX_GRID,
                gap: 8,
                alignItems: "center",
                padding: "6px 14px",
                borderBottom: `1px solid ${C.border}`,
                cursor: "pointer",
                background: selectedId === tx.id ? `${C.cyan}08` : "transparent",
                transition: "background 0.15s",
              }}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = selectedId === tx.id ? `${C.cyan}08` : `${C.cardAlt}`; }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = selectedId === tx.id ? `${C.cyan}08` : "transparent"; }}
            >
              <TruncCell color={C.cyan} title={tx.sigFull || tx.sig}>{tx.sig}</TruncCell>
              <TruncCell color={selectedId === tx.id ? C.cyan : C.sub}>{tx.slot.toLocaleString()}</TruncCell>
              <TruncCell color={C.muted} size={9} title={tx.bundleId}>{tx.bundleId}</TruncCell>
              <TruncCell color={C.amber}>{tx.tip}</TruncCell>
              <div style={{ minWidth: 0, overflow: "hidden" }}>
                <StatusBadge status={tx.status} />
              </div>
              <TruncCell color={tx.latency > 600 ? C.amber : C.sub}>
                {tx.latency > 0 ? `${tx.latency}ms` : "—"}
              </TruncCell>
              <TruncCell color={C.sub} mono={false} title={tx.leader}>{tx.leader}</TruncCell>
              <TruncCell color={tx.retries > 0 ? C.amber : C.muted}>{tx.retries}x</TruncCell>
            </div>
          ))}
        </div>
      </div>
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────
// REPLAY TIMELINE
// ─────────────────────────────────────────────────────────────

function TraceCloseButton({ onClose }: { onClose: () => void }) {
  return (
    <button
      type="button"
      onClick={onClose}
      title="Close trace (Esc)"
      aria-label="Close lifecycle trace"
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        gap: 5,
        minWidth: 72,
        height: 30,
        borderRadius: 4,
        border: `1px solid ${C.border}`,
        background: C.cardAlt,
        color: C.sub,
        cursor: "pointer",
        flexShrink: 0,
        padding: "0 10px",
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.color = C.cyan;
        e.currentTarget.style.borderColor = `${C.cyan}40`;
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.color = C.sub;
        e.currentTarget.style.borderColor = C.border;
      }}
    >
      <X size={14} />
      <span style={{ fontFamily: C.mono, fontSize: 9, letterSpacing: "0.1em" }}>CLOSE</span>
    </button>
  );
}

function ReplayTimeline({ tx, events, cluster, onClose, docked = false }: {
  tx: Tx | null;
  events: TimelineEventView[];
  cluster: string;
  onClose: () => void;
  docked?: boolean;
}) {
  if (!tx) {
    return (
      <Card style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: 100 }}>
        <div style={{ textAlign: "center" as const }}>
          <Eye size={18} color={C.dim} style={{ margin: "0 auto 6px" }} />
          <div style={{ fontFamily: C.mono, fontSize: 10, color: C.muted }}>Select a transaction to inspect</div>
        </div>
      </Card>
    );
  }

  if (events.length === 0) {
    return (
      <Card>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10 }}>
          <SectionLabel icon={Activity}>Lifecycle Trace</SectionLabel>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <StatusBadge status={tx.status} />
            <TraceCloseButton onClose={onClose} />
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: 80 }}>
          <div style={{ textAlign: "center" as const }}>
            <Clock size={18} color={C.dim} style={{ margin: "0 auto 6px" }} />
            <div style={{ fontFamily: C.mono, fontSize: 10, color: C.muted }}>Loading lifecycle trace…</div>
          </div>
        </div>
      </Card>
    );
  }

  const stageIcons: Record<string, ElementType> = {
    "Bundle Submitted": Server,
    "Processed by Leader": Cpu,
    Confirmed: CheckCircle2,
    Finalized: CheckCircle2,
    Failed: XCircle,
  };

  const sigUrl = tx.sigFull ? solscanTxUrl(tx.sigFull, cluster) : undefined;
  const leaderUrl = tx.leaderFull && tx.leaderFull !== "Unknown"
    ? explorerAddressUrl(tx.leaderFull, cluster)
    : undefined;

  return (
    <Card style={{
      ...(docked
        ? { flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }
        : { maxHeight: "min(420px, 45vh)", display: "flex", flexDirection: "column" }),
    }}>
      <div style={{
        display: "flex", alignItems: "center", justifyContent: "space-between",
        marginBottom: 10, flexShrink: 0, gap: 8,
        ...(docked ? { paddingTop: 10 } : {}),
      }}>
        <SectionLabel icon={Activity}>Lifecycle Trace</SectionLabel>
        <div style={{ display: "flex", alignItems: "center", gap: 8, flexShrink: 0 }}>
          <StatusBadge status={tx.status} />
          <TraceCloseButton onClose={onClose} />
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, overflowY: "auto" }} className="aegis-scroll">
      <div style={{ display: "flex", flexDirection: "column", gap: 6, marginBottom: 12 }}>
        <TraceMetaRow
          label="SIG"
          display={tx.sigFull ? `${tx.sigFull.slice(0, 8)}…${tx.sigFull.slice(-8)}` : tx.sig}
          value={tx.sigFull || tx.sig}
          href={sigUrl}
        />
        <TraceMetaRow
          label="LEADER"
          display={tx.leaderFull || tx.leader}
          value={tx.leaderFull || tx.leader}
          href={leaderUrl}
        />
        <TraceMetaRow
          label="SLOT"
          display={tx.slot > 0 ? tx.slot.toLocaleString() : "—"}
          value={tx.slot > 0 ? String(tx.slot) : ""}
          href={tx.slot > 0 ? explorerSlotUrl(tx.slot, cluster) : undefined}
        />
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 0 }}>
        {events.map((e, i) => {
          const Icon = stageIcons[e.label] ?? Activity;
          return (
            <div key={`${e.label}-${e.ts}-${i}`} style={{ display: "flex", gap: 10, position: "relative" }}>
              {/* Timeline rail */}
              <div style={{ display: "flex", flexDirection: "column", alignItems: "center", width: 20, flexShrink: 0 }}>
                <div style={{
                  width: 20, height: 20, borderRadius: "50%", flexShrink: 0,
                  background: `${e.color}18`, border: `1px solid ${e.color}50`,
                  display: "flex", alignItems: "center", justifyContent: "center",
                  boxShadow: `0 0 8px ${e.color}40`,
                }}>
                  <Icon size={9} color={e.color} />
                </div>
                {i < events.length - 1 && (
                  <div style={{ width: 1, flex: 1, background: `linear-gradient(${e.color}40, ${events[i+1].color}20)`, marginBlock: 2 }} />
                )}
              </div>

              {/* Content */}
              <div style={{ flex: 1, paddingBottom: i < events.length - 1 ? 10 : 0 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                  <span style={{ fontFamily: C.sans, fontSize: 11, fontWeight: 600, color: e.color }}>{e.label}</span>
                  {e.delta && (
                    <span style={{ fontFamily: C.mono, fontSize: 9, color: C.amber, background: `${C.amber}12`, padding: "1px 5px", borderRadius: 2, border: `1px solid ${C.amber}28` }}>
                      {e.delta}
                    </span>
                  )}
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                  <span style={{ fontFamily: C.mono, fontSize: 10, color: C.sub }}>{e.ts}</span>
                  <SlotLink slot={e.slot} cluster={cluster} />
                </div>
              </div>
            </div>
          );
        })}
      </div>
      </div>
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────
// CENTER PANEL
// ─────────────────────────────────────────────────────────────

function CenterPanel({ transactions, selectedTxId, onSelectTx, pipelineCounts, pipelineLatencies, timelineEvents, cluster }: {
  transactions: Tx[];
  selectedTxId: string | null;
  onSelectTx: (id: string) => void;
  pipelineCounts: number[];
  pipelineLatencies: string[];
  timelineEvents: TimelineEventView[];
  cluster: string;
}) {
  const selectedTx = transactions.find(t => t.id === selectedTxId) ?? null;
  const traceOpen = !!selectedTx;

  useEffect(() => {
    if (!traceOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onSelectTx("");
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [traceOpen, onSelectTx]);

  return (
    <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden", minHeight: 0 }}>
      <div
        className="aegis-scroll"
        style={{
          flex: 1,
          minHeight: 0,
          overflowY: "auto",
          padding: 12,
          display: "flex",
          flexDirection: "column",
          gap: 10,
        }}
      >
        <PipelineHero counts={pipelineCounts} latencies={pipelineLatencies} />
        <TxFeed transactions={transactions} selectedId={selectedTxId} onSelect={onSelectTx} />
        <ReplayTimeline
          tx={selectedTx}
          events={timelineEvents}
          cluster={cluster}
          onClose={() => onSelectTx("")}
        />
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────
// RIGHT PANEL — AI OPERATIONS BRAIN
// ─────────────────────────────────────────────────────────────

function RightPanel({ decisions, failures, landingProb, landingFactorBars, recoverySteps }: {
  decisions: AIDecision[];
  failures: FailureEntry[];
  landingProb: number;
  landingFactorBars: { label: string; value: number }[];
  recoverySteps: RecoveryStepView[];
}) {
  const activeStep = recoverySteps.findIndex(s => !s.done);

  return (
    <div style={{
      width: 260, flexShrink: 0,
      borderLeft: `1px solid ${C.border}`,
      display: "flex", flexDirection: "column",
      overflow: "hidden",
    }}>
      <div className="aegis-scroll" style={{ flex: 1, overflowY: "auto", padding: "14px 12px", display: "flex", flexDirection: "column", gap: 10 }}>

        {/* AI Header */}
        <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 2 }}>
          <div style={{
            width: 22, height: 22, borderRadius: 4,
            background: `${C.purple}18`, border: `1px solid ${C.purple}35`,
            display: "flex", alignItems: "center", justifyContent: "center",
          }}>
            <Brain size={11} color={C.purple} />
          </div>
          <span style={{ fontFamily: C.sans, fontWeight: 700, fontSize: 12, color: "#dde8f5", letterSpacing: "0.04em" }}>AI OPERATIONS</span>
          <LiveDot size={5} color={C.purple} />
        </div>

        {/* AI Decision Feed */}
        <Card>
          <SectionLabel icon={Zap}>AI Decision Feed</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {decisions.map((d, i) => (
              <div key={d.id} className={i === 0 ? "aegis-fade-in" : ""} style={{
                background: C.cardAlt, borderRadius: 4, padding: "8px 9px",
                border: `1px solid ${C.border}`,
              }}>
                <div style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 5 }}>
                  <AITypeIcon type={d.type} />
                  <span style={{ fontFamily: C.sans, fontSize: 11, fontWeight: 600, color: C.text, flex: 1 }}>{d.title}</span>
                </div>
                <div style={{ fontFamily: C.sans, fontSize: 10, color: C.sub, lineHeight: 1.45, marginBottom: 5 }}>
                  {d.reasoning}
                </div>
                <div style={{ fontFamily: C.mono, fontSize: 9, color: C.muted, marginBottom: 5 }}>{d.context}</div>
                <ConfBar value={d.confidence} color={C.purple} label={`${Math.round(d.confidence * 100)}% conf`} />
              </div>
            ))}
          </div>
        </Card>

        {/* Landing Probability */}
        <Card>
          <SectionLabel icon={Target}>Landing Probability</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
            {/* Big number */}
            <div style={{ position: "relative", display: "flex", alignItems: "center", justifyContent: "center" }}>
              <svg width={100} height={58} viewBox="0 0 100 58">
                <path d="M 8 52 A 44 44 0 0 1 92 52" fill="none" stroke={C.dim} strokeWidth={6} strokeLinecap="round" />
                <path
                  d="M 8 52 A 44 44 0 0 1 92 52"
                  fill="none"
                  stroke={landingProb > 80 ? C.green : landingProb > 50 ? C.amber : C.red}
                  strokeWidth={6}
                  strokeLinecap="round"
                  strokeDasharray={`${(landingProb / 100) * 138} 999`}
                  style={{ filter: `drop-shadow(0 0 6px ${landingProb > 80 ? C.green : C.amber})`, transition: "stroke-dasharray 0.8s ease" }}
                />
              </svg>
              <div style={{ position: "absolute", bottom: 2, textAlign: "center" as const }}>
                <div style={{ fontFamily: C.mono, fontSize: 20, fontWeight: 600, color: landingProb > 80 ? C.green : C.amber, lineHeight: 1 }}>
                  {landingProb}%
                </div>
              </div>
            </div>
            <div style={{ width: "100%", display: "flex", flexDirection: "column", gap: 5 }}>
              {landingFactorBars.map(m => (
                <div key={m.label}>
                  <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 3 }}>
                    <span style={{ fontFamily: C.mono, fontSize: 8, color: C.muted }}>{m.label}</span>
                  </div>
                  <ConfBar value={m.value} color={C.cyan} />
                </div>
              ))}
            </div>
          </div>
        </Card>

        {/* Failure Analysis */}
        <Card>
          <SectionLabel icon={AlertCircle}>Failure Analysis</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {failures.map(f => (
              <div key={f.id} style={{
                background: `${C.red}08`, borderRadius: 4, padding: "7px 8px",
                border: `1px solid ${C.red}1a`,
              }}>
                <div style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 4 }}>
                  <FailureTypeIcon type={f.type} />
                  <span style={{ fontFamily: C.sans, fontSize: 10, fontWeight: 600, color: "#e8a0a0" }}>{f.label}</span>
                  <span style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, marginLeft: "auto" }}>
                    {f.slot.toLocaleString()}
                  </span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                  <ArrowRight size={9} color={C.green} style={{ flexShrink: 0 }} />
                  <span style={{ fontFamily: C.sans, fontSize: 9, color: C.green }}>{f.action}</span>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Autonomous Recovery */}
        <Card>
          <SectionLabel icon={RefreshCw}>Autonomous Recovery</SectionLabel>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {recoverySteps.map((step, i) => (
              <div key={step.label} style={{ display: "flex", alignItems: "center", gap: 8, position: "relative" }}>
                {/* Connector line */}
                {i < recoverySteps.length - 1 && (
                  <div style={{ position: "absolute", left: 9, top: 18, width: 1, height: "calc(100% - 4px)", background: `${C.border}`, zIndex: 0 }} />
                )}
                {/* Step icon */}
                <div style={{
                  width: 18, height: 18, borderRadius: "50%", flexShrink: 0, zIndex: 1,
                  background: step.done ? `${C.green}20` : i === activeStep ? `${C.cyan}18` : C.dim,
                  border: `1px solid ${step.done ? C.green + "50" : i === activeStep ? C.cyan + "60" : C.border}`,
                  display: "flex", alignItems: "center", justifyContent: "center",
                  boxShadow: i === activeStep ? `0 0 8px ${C.cyan}50` : "none",
                }}>
                  {step.done
                    ? <CheckCircle2 size={9} color={C.green} />
                    : i === activeStep
                      ? <div style={{ width: 5, height: 5, borderRadius: "50%", background: C.cyan, animation: "aegis-pulse 1s ease-in-out infinite" }} />
                      : <div style={{ width: 4, height: 4, borderRadius: "50%", background: C.dim }} />
                  }
                </div>
                <span style={{
                  fontFamily: C.sans, fontSize: 10,
                  color: step.done ? C.green : i === activeStep ? C.cyan : C.muted,
                  fontWeight: i === activeStep ? 600 : 400,
                }}>
                  {step.label}
                </span>
                {i === activeStep && <LiveDot size={4} color={C.cyan} />}
              </div>
            ))}
          </div>
        </Card>

      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────
// BOTTOM ANALYTICS
// ─────────────────────────────────────────────────────────────

function BottomAnalytics({ latencyData, congestionData, bundleData, tipData, failureData }: {
  latencyData: { t: number; v: number; v2: number }[];
  congestionData: { t: number; v: number; v2: number }[];
  bundleData: { t: number; v: number; v2: number }[];
  tipData: { t: number; v: number }[];
  failureData: { t: number; v: number }[];
}) {
  const chartBase = { margin: { top: 4, right: 4, bottom: 0, left: -20 } };
  const axisStyle = { tick: { fill: C.muted, fontSize: 8, fontFamily: C.mono }, axisLine: false, tickLine: false };

  const charts = [
    {
      label: "Confirmation Latency",
      icon: Clock,
      el: (
        <AreaChart data={latencyData} {...chartBase}>
          <defs>
            <linearGradient id="lgLat" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={C.cyan} stopOpacity={0.3} />
              <stop offset="100%" stopColor={C.cyan} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis dataKey="t" {...axisStyle} hide />
          <YAxis {...axisStyle} />
          <Tooltip content={<ChartTooltip />} />
          <Area type="monotone" dataKey="v" name="p50 ms" stroke={C.cyan} strokeWidth={1.5} fill="url(#lgLat)" dot={false} />
          <Area type="monotone" dataKey="v2" name="p99 ms" stroke={C.purple} strokeWidth={1} fill="none" dot={false} strokeDasharray="3 3" />
        </AreaChart>
      ),
    },
    {
      label: "Network Congestion",
      icon: Activity,
      el: (
        <AreaChart data={congestionData} {...chartBase}>
          <defs>
            <linearGradient id="lgCong" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={C.amber} stopOpacity={0.35} />
              <stop offset="100%" stopColor={C.amber} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis dataKey="t" {...axisStyle} hide />
          <YAxis {...axisStyle} />
          <Tooltip content={<ChartTooltip />} />
          <Area type="monotone" dataKey="v" name="congestion %" stroke={C.amber} strokeWidth={1.5} fill="url(#lgCong)" dot={false} />
        </AreaChart>
      ),
    },
    {
      label: "Bundle Success",
      icon: Target,
      el: (
        <BarChart data={bundleData} {...chartBase}>
          <XAxis dataKey="t" {...axisStyle} hide />
          <YAxis {...axisStyle} />
          <Tooltip content={<ChartTooltip />} />
          <Bar dataKey="v" name="landed" fill={C.green} fillOpacity={0.8} radius={[2, 2, 0, 0]} />
          <Bar dataKey="v2" name="failed" fill={C.red} fillOpacity={0.7} radius={[2, 2, 0, 0]} />
        </BarChart>
      ),
    },
    {
      label: "Tip vs Success Rate",
      icon: TrendingUp,
      el: (
        <LineChart data={tipData} {...chartBase}>
          <XAxis dataKey="t" {...axisStyle} hide />
          <YAxis {...axisStyle} />
          <Tooltip content={<ChartTooltip />} />
          <Line type="monotone" dataKey="v" name="success %" stroke={C.green} strokeWidth={1.5} dot={false} />
        </LineChart>
      ),
    },
    {
      label: "Failure Frequency",
      icon: AlertTriangle,
      el: (
        <BarChart data={failureData} {...chartBase}>
          <XAxis dataKey="t" {...axisStyle} hide />
          <YAxis {...axisStyle} />
          <Tooltip content={<ChartTooltip />} />
          <Bar dataKey="v" name="failures" fill={C.red} fillOpacity={0.75} radius={[2, 2, 0, 0]} />
        </BarChart>
      ),
    },
  ];

  return (
    <div style={{
      height: 168, flexShrink: 0,
      borderTop: `1px solid ${C.border}`,
      display: "flex", gap: 0,
      background: C.bg,
    }}>
      {charts.map((chart, i) => {
        const Icon = chart.icon;
        return (
          <div key={chart.label} style={{
            flex: 1, borderRight: i < charts.length - 1 ? `1px solid ${C.border}` : "none",
            padding: "10px 12px", display: "flex", flexDirection: "column",
          }}>
            <div style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 6 }}>
              <Icon size={9} color={C.muted} />
              <span style={{ fontFamily: C.mono, fontSize: 8, color: C.muted, letterSpacing: "0.14em", textTransform: "uppercase" as const }}>
                {chart.label}
              </span>
            </div>
            <div style={{ flex: 1, minHeight: 0 }}>
              <ResponsiveContainer width="100%" height="100%">
                {chart.el}
              </ResponsiveContainer>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────
// MAIN APP
// ─────────────────────────────────────────────────────────────

export default function App() {
  const [selectedTxId, setSelectedTxId] = useState<string | null>(null);
  const dashboard = useDashboardData();
  const timelineQuery = useTransactionTimeline(selectedTxId);
  const handleSelectTx = useCallback((id: string) => setSelectedTxId(id || null), []);

  const {
    isLoading,
    isError,
    wsConnected,
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
    cluster,
  } = dashboard;

  if (isLoading && transactions.length === 0) {
    return (
      <>
        <style>{AEGIS_CSS}</style>
        <div style={{ height: "100vh", display: "flex", alignItems: "center", justifyContent: "center", background: C.bg, color: C.sub, fontFamily: C.mono }}>
          Loading dashboard…
        </div>
      </>
    );
  }

  return (
    <>
      <style>{AEGIS_CSS}</style>
      <div
        style={{
          height: "100vh", display: "flex", flexDirection: "column",
          background: C.bg, fontFamily: C.sans, overflow: "hidden",
        }}
      >
        {isError && (
          <div style={{ background: `${C.red}18`, color: C.red, fontFamily: C.mono, fontSize: 10, padding: "4px 16px", textAlign: "center" }}>
            API unreachable — showing last snapshot. Polling every 2s…
          </div>
        )}
        <GlobalStatusBar ticker={ticker} wsConnected={wsConnected} />

        <div style={{ flex: 1, display: "flex", minHeight: 0, overflow: "hidden" }}>
          <LeftSidebar
            slotFeed={slotFeed}
            leaderWindows={leaderWindows}
            networkHealth={networkHealth}
            bundleTiles={bundleTiles}
          />
          <CenterPanel
            transactions={transactions}
            selectedTxId={selectedTxId}
            onSelectTx={handleSelectTx}
            pipelineCounts={pipelineCounts}
            pipelineLatencies={pipelineLatencies}
            timelineEvents={timelineQuery.data ?? []}
            cluster={cluster}
          />
          <RightPanel
            decisions={decisions}
            failures={failures}
            landingProb={landingProb}
            landingFactorBars={landingFactorBars}
            recoverySteps={recoverySteps}
          />
        </div>

        <BottomAnalytics
          latencyData={chartSeries.latencyData}
          congestionData={chartSeries.congestionData}
          bundleData={chartSeries.bundleData}
          tipData={chartSeries.tipData}
          failureData={chartSeries.failureData}
        />
      </div>
    </>
  );
}
