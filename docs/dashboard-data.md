# TX Pilot Dashboard Data

This document defines the backend data the TX Pilot dashboard needs.

The dashboard is an operational control room for Solana transaction execution. It must feel live: current network state, slot movement, Jito leader windows, bundle execution, lifecycle transitions, AI decisions, failures, and charts should update without page refresh.

## Delivery Modes

Support all three delivery modes:

- HTTPS polling for canonical snapshots and transaction status.
- WebSocket streaming for live dashboard updates.
- Signed webhooks for external server-to-server lifecycle updates.

The frontend should prefer WebSocket for live panels and fall back to HTTPS polling when disconnected.

## Update Cadence

Recommended cadences:

- Top network ticker: every slot or 500ms, whichever is calmer.
- Live slot feed: every slot event.
- Leader schedule: refresh on epoch/leader schedule changes and when the current target window moves.
- Lifecycle pipeline: every lifecycle event or 1s aggregate tick.
- Realtime transaction stream: every transaction lifecycle event.
- AI decision feed: every agent decision.
- Landing probability: every 1-2s or after bundle metric changes.
- Failure analysis: every failure classification.
- Bottom charts: every 2-5s using rolling windows.

## WebSocket Channels

Use one WebSocket endpoint with typed messages:

```txt
GET /v1/ws
```

Client subscription message:

```json
{
  "type": "subscribe",
  "channels": [
    "network.ticker",
    "slots.feed",
    "leaders.schedule",
    "lifecycle.pipeline",
    "transactions.stream",
    "ai.decisions",
    "landing.probability",
    "failures.analysis",
    "recovery.actions",
    "charts.series"
  ]
}
```

Every server message must include:

```json
{
  "type": "network.ticker.updated",
  "sequence": 102394,
  "server_time": "2026-06-23T10:40:12.510Z",
  "payload": {}
}
```

Rules:

- `sequence` must increase per connection.
- Messages must be typed and versionable.
- Include enough IDs for the frontend to merge updates without resetting whole panels.
- Slow WebSocket consumers must be disconnected or degraded intentionally.

## HTTPS Snapshot Endpoints

Minimum endpoints:

```txt
GET /v1/dashboard/snapshot
GET /v1/dashboard/network
GET /v1/dashboard/slots
GET /v1/dashboard/leaders
GET /v1/dashboard/bundles
GET /v1/dashboard/transactions
GET /v1/dashboard/ai-decisions
GET /v1/dashboard/failures
GET /v1/dashboard/charts?window=15m
GET /v1/transactions/{id}
GET /v1/transactions/{id}/timeline
GET /v1/bundles/{bundle_id}
GET /v1/bundles/{bundle_id}/timeline
```

`/v1/dashboard/snapshot` should return enough data to render the whole dashboard before WebSocket updates arrive.

## Top Network Ticker

Panel fields:

```json
{
  "slot": 285834382,
  "current_leader": "Jito Labs",
  "next_jito_leader": "Jump Crypto",
  "congestion_pct": 41,
  "bundle_land_rate_pct": 94.2,
  "tip_floor_lamports": 12000,
  "tip_floor_sol": 0.000012,
  "tps": 3284,
  "slot_drift_ms": 2.1,
  "network_health_pct": 70,
  "cluster": "mainnet-beta"
}
```

Sources:

- Slot and leader data from Yellowstone/Geyser and leader schedule.
- Bundle land rate from recent bundle outcomes.
- Tip floor from recent Jito tip account data and observed landing outcomes.
- TPS from stream/RPC metrics.
- Slot drift from expected slot time versus observed slot timing.
- Network health from congestion, confirmation latency, slot drift, validator stability, and landing rate.

## Live Slot Feed

Panel fields:

```json
{
  "slots": [
    {
      "slot": 285834382,
      "age_ms": 265,
      "leader": "Jito Labs",
      "status": "current",
      "jito_leader": true,
      "skipped": false
    },
    {
      "slot": 285834376,
      "age_ms": 199,
      "leader": "Unknown",
      "status": "skipped",
      "jito_leader": false,
      "skipped": true
    }
  ]
}
```

Required behavior:

- Keep a rolling window of recent slots.
- Mark skipped slots distinctly.
- Include leader identity when known.
- Preserve current slot highlight.

## Leader Schedule

Panel fields:

```json
{
  "current_leader": "Jito Labs",
  "next_jito_leader": "Jump Crypto",
  "leaders": [
    {
      "identity": "Jito Labs",
      "slot_start": 285834221,
      "slot_end": 285834225,
      "jito": true,
      "quality": "good",
      "current": true
    },
    {
      "identity": "Jump Crypto",
      "slot_start": 285834226,
      "slot_end": 285834230,
      "jito": true,
      "quality": "good",
      "current": false
    }
  ]
}
```

## Network Health Panel

Panel fields:

```json
{
  "congestion_pct": 72,
  "confirmation_latency_ms": 342,
  "slot_jitter_ms": 2.4,
  "validator_stability_pct": 97,
  "network_health_pct": 70
}
```

Definitions:

- `congestion_pct`: composite pressure score from TPS, confirmation latency, slot jitter, pending bundles, and recent failures.
- `confirmation_latency_ms`: rolling delta between processed and confirmed.
- `slot_jitter_ms`: observed slot timing variance.
- `validator_stability_pct`: recent leader/slot reliability score.
- `network_health_pct`: composite score for top ticker and dashboard summary.

## Bundle Metrics Panel

Panel fields:

```json
{
  "sent": 1931,
  "landed": 1820,
  "success_pct": 94.3,
  "avg_tip_lamports": 18000,
  "avg_tip_sol": 0.000018,
  "retries": 47,
  "pending": 12
}
```

## Transaction Lifecycle Pipeline

Panel fields:

```json
{
  "realtime": true,
  "stages": [
    {
      "stage": "created",
      "label": "CR",
      "count": 2754,
      "delta_from_previous_ms": null
    },
    {
      "stage": "submitted",
      "label": "SB",
      "count": 1951,
      "delta_from_previous_ms": 14
    },
    {
      "stage": "processed",
      "label": "PR",
      "count": 1455,
      "delta_from_previous_ms": 42
    },
    {
      "stage": "confirmed",
      "label": "CF",
      "count": 1214,
      "delta_from_previous_ms": 310
    },
    {
      "stage": "finalized",
      "label": "FN",
      "count": 1000,
      "delta_from_previous_ms": 820
    }
  ],
  "summary": {
    "total_in_flight": 4620,
    "avg_end_to_end_seconds": 1.19,
    "success_rate_pct": 94.2,
    "failed_today": 127,
    "retried": 342
  }
}
```

## Realtime Transaction Stream

Panel fields:

```json
{
  "rows": [
    {
      "transaction_id": "tx_01J...",
      "signature": "3KpR...K3x8",
      "slot": 285834380,
      "bundle_id": "JTO-D3DE82",
      "tip_lamports": 30000,
      "tip_sol": 0.00003,
      "status": "confirmed",
      "latency_ms": 848,
      "leader": "Anatoly.sol",
      "retry_attempt": 2,
      "agent_decision_id": "dec_01J..."
    }
  ],
  "row_count": 24,
  "streaming": true
}
```

Rows should be updated incrementally by `transaction.updated` events. Selecting a row should load:

```txt
GET /v1/transactions/{id}/timeline
```

## AI Operations Feed

Panel fields:

```json
{
  "decisions": [
    {
      "decision_id": "dec_01J...",
      "type": "tip_adjustment",
      "title": "Tip raised +12%",
      "summary": "Bundle competition: 3 concurrent submitters on this slot window",
      "inputs": {
        "congestion_pct": 78,
        "slot": 285834221
      },
      "action": {
        "kind": "increase_tip",
        "tip_delta_pct": 12
      },
      "confidence_pct": 91,
      "created_at": "2026-06-23T10:40:12.510Z"
    }
  ]
}
```

Decision types:

- `tip_adjustment`
- `blockhash_refresh`
- `retry_backoff`
- `submission_delay`
- `leader_avoidance`
- `abort`

## Landing Probability

Panel fields:

```json
{
  "probability_pct": 85,
  "factors": {
    "bundle_tip_adequacy_pct": 88,
    "leader_stability_pct": 73,
    "slot_competition_pct": 62,
    "network_readiness_pct": 91
  },
  "basis": {
    "recent_window_slots": 150,
    "recent_success_rate_pct": 94.2,
    "similar_tip_success_pct": 88.5,
    "leader_recent_land_rate_pct": 82.1
  }
}
```

Landing probability should be calculated from recent success rates, tip adequacy, leader stability, slot competition, and network readiness. It must not be a static value.

## Failure Analysis

Panel fields:

```json
{
  "failures": [
    {
      "failure_id": "fail_01J...",
      "kind": "expired_blockhash",
      "title": "Blockhash expired",
      "slot": 285834198,
      "severity": "warning",
      "recommended_action": "Refresh + resubmit scheduled",
      "transaction_id": "tx_01J...",
      "bundle_id": "JTO-...",
      "created_at": "2026-06-23T10:40:12.510Z"
    }
  ]
}
```

Failure kinds:

- `expired_blockhash`
- `tip_below_floor`
- `compute_exceeded`
- `bundle_rejected`
- `leader_skipped`
- `stream_gap`
- `rpc_error`
- `unknown`

## Autonomous Recovery

Panel fields:

```json
{
  "actions": [
    {
      "action_id": "act_01J...",
      "label": "Refresh blockhash",
      "status": "completed",
      "transaction_id": "tx_01J...",
      "decision_id": "dec_01J..."
    },
    {
      "action_id": "act_01K...",
      "label": "Recalculate tip",
      "status": "running",
      "transaction_id": "tx_01J...",
      "decision_id": "dec_01J..."
    }
  ]
}
```

Action statuses:

- `queued`
- `running`
- `completed`
- `failed`
- `skipped`

## Chart Series

The bottom charts should receive rolling time-series points.

### Confirmation Latency

```json
{
  "series": "confirmation_latency",
  "points": [
    {
      "time": "2026-06-23T10:40:12Z",
      "processed_to_confirmed_ms": 342,
      "confirmed_to_finalized_ms": 820
    }
  ]
}
```

### Network Congestion

```json
{
  "series": "network_congestion",
  "points": [
    {
      "time": "2026-06-23T10:40:12Z",
      "congestion_pct": 72
    }
  ]
}
```

### Bundle Success

```json
{
  "series": "bundle_success",
  "points": [
    {
      "time": "2026-06-23T10:40:12Z",
      "landed": 44,
      "failed": 3,
      "pending": 2
    }
  ]
}
```

### Tip vs Success Rate

```json
{
  "series": "tip_vs_success_rate",
  "points": [
    {
      "time": "2026-06-23T10:40:12Z",
      "avg_tip_lamports": 18000,
      "success_rate_pct": 94.2
    }
  ]
}
```

### Failure Frequency

```json
{
  "series": "failure_frequency",
  "points": [
    {
      "time": "2026-06-23T10:40:12Z",
      "expired_blockhash": 2,
      "tip_below_floor": 1,
      "compute_exceeded": 1,
      "bundle_rejected": 0,
      "leader_skipped": 1
    }
  ]
}
```

## Backend Aggregation Rules

- The backend owns canonical calculations for congestion, landing probability, bundle land rate, tip floor, slot drift, TPS, and network health.
- The frontend should render values and visual states, not recalculate operational truth.
- All dashboard aggregates must include source timestamps so stale data can be shown honestly.
- Each panel should tolerate missing data and show degraded/stale states instead of inventing values.
- WebSocket deltas must be mergeable into the latest HTTPS snapshot.
- Chart series should be bounded rolling windows to avoid unbounded browser memory growth.

## Minimum Demo Data

For the bounty demo, the dashboard should be able to show:

- Live slot feed with at least one skipped slot.
- Leader schedule with current leader and next Jito leader.
- Transaction lifecycle pipeline with live stage counts.
- Realtime transaction stream with signatures, slots, bundle IDs, tips, status, latency, leader, and retries.
- AI decision feed showing at least one real autonomous decision.
- Landing probability based on recent observed success rates.
- Failure analysis with at least two classified failures.
- Autonomous recovery actions for an expired blockhash scenario.
- Charts for confirmation latency, congestion, bundle success, tip versus success rate, and failure frequency.
