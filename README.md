# Aegis — Autonomous Solana Transaction Control Plane

AI-powered transaction orchestration and recovery for Solana. Aegis monitors live network state via Yellowstone/Geyser, applies dynamic Jito tips using server-signed tip transactions, tracks full lifecycle commitments (stream + RPC + Jito polling), and uses an OpenAI SRE agent for tip intelligence and autonomous recovery.

## Quick start

See the full [Setup & run guide](docs/setup.md). Minimal steps:

```bash
cp .env.example .env   # fill credentials (AEGIS_KEYPAIR_PATH required)
make test-keypair      # creates aegis-test-keypair.json (gitignored)
make docker-up
make migrate-up
make dev
```

River Queue UI: `http://localhost:8080/riverui`

### Ops keypair

| | |
|---|---|
| **Env** | `AEGIS_KEYPAIR_PATH=./aegis-test-keypair.json` |
| **Used for** | Dynamic tip txs, ops submissions, autonomous recovery resubmits |
| **Fund** | ~0.05 SOL on mainnet-beta for demo/lifecycle runs |

Integration tests still sign client txs locally; the server signs tip txs appended to every bundle.

## API

| Endpoint | Description |
|----------|-------------|
| `POST /v1/transactions` | Client tx auto-wrapped into bundle with dynamic server tip |
| `POST /v1/bundles` | 1–4 client txs + dynamic server tip tx |
| `POST /v1/ops/submit` | Server-signed ops bundle (fault injection supported) |
| `GET /v1/lifecycle-log` | Structured lifecycle export for bounty evidence |
| `GET /v1/transactions/{id}` | Poll status |
| `GET /v1/transactions/{id}/timeline` | Lifecycle events with latency_ms |
| `GET /v1/bundles/{id}` | Bundle DB row + Jito status |
| `GET /v1/blockhash` | Latest `processed` blockhash |
| `GET /v1/tip-accounts` | Jito tip accounts |
| `GET /v1/dashboard/*` | Live dashboard aggregates |
| `GET /v1/ws` | WebSocket (`transactions.stream`, `failures.analysis`, `ai.decisions`) |

Optional `tip_lamports` on submit endpoints overrides the AI/heuristic tip.

### Submit with dynamic tip (auto bundle)

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{"transaction":"<signed-tx-base64>","encoding":"base64","tip_lamports":50000}'
```

### Ops submission with fault injection

```bash
curl -X POST http://localhost:8080/v1/ops/submit \
  -H 'Content-Type: application/json' \
  -d '{"memo":"demo-expired","inject_expired_blockhash":true}'
```

### Lifecycle log run (bounty evidence)

```bash
go run ./cmd/aegis-lifecycle-runner -count 10 -failures 2
```

Writes `lifecycle-log.json` with slot numbers, commitment progression, timestamps, tips, latency, leader, and failure classification.

## Bounty README answers

### 1. What does processed→confirmed delta tell you about network health?

A rising processed→confirmed delta means validators ingest transactions quickly but cluster voting/fork resolution lags before confirmation. Under congestion or degraded leader performance this gap widens—landing probability drops even when submissions look healthy at processed commitment. Aegis records this delta per transaction in `lifecycle_events.latency_ms` and aggregates it in the dashboard pipeline.

### 2. Why not use finalized commitment for time-sensitive blockhashes?

Finalized commitment trails the chain tip by many slots (~13+ seconds). A blockhash valid at finalized may already be expired at the current tip. Aegis always fetches blockhashes at `processed` commitment for tip/ops signing to maximize the valid window.

### 3. What happens if the Jito leader skips their slot?

The bundle misses that leader's block engine window. Aegis detects skipped slots via Geyser slot gaps, may delay submission one slot (SAFE/CHEAP modes), and records leader context on each lifecycle event. If the bundle never lands, failure is classified and the AI agent decides recovery actions (tip increase, delay, or blockhash refresh for ops txs).

## Stack

- Go + Chi + Postgres + sqlc + goose + River + River UI
- Yellowstone gRPC (Triton) + Jito block engine (mainnet)
- OpenAI gpt-4o-mini agent (tip intelligence + recovery)

## Docs

- [Setup & run guide](docs/setup.md)
- [Architecture](docs/architecture.md)
- [Dashboard contract](docs/dashboard-data-contract.md)
- [Lifecycle log](docs/lifecycle-log.md)
