# Aegis Architecture

Aegis is an autonomous Solana transaction control plane. It observes Yellowstone/Geyser streams, forwards client-signed transactions to Jito, tracks lifecycle via Solana RPC + Jito bundle status polling, classifies failures, and records AI advisory decisions.

## Components

- **API Gateway** (`internal/api`): Chi REST + WebSocket; raw tx/bundle submit, tracking, dashboard, River UI mount.
- **Stream Engine** (`internal/stream`): Yellowstone gRPC slots/transactions with reconnect + backpressure.
- **Bundle Router** (`internal/bundle`): Jito client (`sendTransaction`, `sendBundle`, `getBundleStatuses`, `getInflightBundleStatuses`, `getTipAccounts`).
- **RPC Gateway** (`internal/rpc`): Consolidated Solana RPC (`getLatestBlockhash`, `getSignatureStatuses`, etc.).
- **Lifecycle Tracker** (`internal/lifecycle`): Sharded workers, append-only events, commitment progression.
- **Failure Classifier** (`internal/failure`): Typed failure taxonomy with evidence.
- **AI Agent** (`internal/agent`): gpt-4o-mini advisory decisions (no server-side re-sign).
- **River Queue** (`internal/queue`): Status poll jobs, webhook delivery; UI at `/riverui`.
- **Notify** (`internal/notify`): WebSocket fanout + signed webhooks.
- **Dashboard** (`internal/dashboard`): Canonical aggregates per `docs/dashboard-data-contract.md`.
- **Storage** (`internal/storage`): Postgres + sqlc + goose migrations.

## Data Flow

1. Client builds + signs transaction(s) locally
2. `POST /v1/transactions` or `POST /v1/bundles` with base64/base58 payload
3. Aegis decodes for signature extraction, stores `submission_kind` (`transaction` | `bundle`)
4. Forwards to Jito block engine (returns signature or bundle_id as `result`)
5. River `status_poll` jobs poll RPC / Jito until processed → confirmed → finalized
6. Lifecycle shards record stages; failures trigger AI advisory records
7. Dashboard + WebSocket broadcast updates

## Submission kinds

| Kind | Jito method | Max txs | Tracking |
|------|-------------|---------|----------|
| `transaction` | `sendTransaction` | 1 | `getSignatureStatuses` |
| `bundle` | `sendBundle` | 5 | `getBundleStatuses` + RPC for member sigs |

Single `GET /v1/transactions/{id}` tracks both kinds via one anchor row.

## Failure Handling

- Submit errors → classified + `failed` status
- Bundle not landing → inflight + landed status polling with backoff
- AI agent records recommended actions; clients rebuild/resubmit with fresh blockhash
