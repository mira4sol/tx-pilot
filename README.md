# Aegis — Autonomous Solana Transaction Control Plane

AI-powered transaction orchestration and recovery for Solana. Aegis monitors live network state, forwards pre-signed transactions to Jito (`sendTransaction` or `sendBundle`), tracks full lifecycle commitments via Solana RPC + Jito bundle status polling, and uses an AI SRE agent for advisory failure analysis.

## Quick start

```bash
cp .env.example .env   # fill credentials
make test-keypair      # creates aegis-test-keypair.json (gitignored) for CLI/tests
make docker-up
make migrate-up
make dev
```

River Queue UI: `http://localhost:8080/riverui`

### Integration test wallet

| | |
|---|---|
| **File** | `aegis-test-keypair.json` |
| **Public key** | `Hgj2HKri7Re9RWucnhoETSnNqmBMmQCLZpuigHbfMhj` |
| **Fund** | ~0.05 SOL on mainnet-beta |

Tests build and sign transactions **client-side** with this keypair, then POST encoded txs to Aegis.

## API

| Endpoint | Description |
|----------|-------------|
| `POST /v1/transactions` | Forward single pre-signed tx (base64/base58) to Jito `sendTransaction` |
| `POST /v1/bundles` | Forward 1–5 pre-signed txs to Jito `sendBundle` |
| `GET /v1/transactions/{id}` | Poll status (single tx or bundle anchor) |
| `GET /v1/transactions/{id}/timeline` | Lifecycle events |
| `GET /v1/bundles/{id}` | Bundle DB row + Jito status |
| `GET /v1/blockhash` | Latest blockhash for client signing |
| `GET /v1/tip-accounts` | Jito tip accounts |
| `GET /v1/dashboard/snapshot` | Full dashboard payload |
| `GET /v1/ws` | Live WebSocket streams |
| `GET /riverui` | River queue job UI |

### Submit single transaction (base64)

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{"transaction":"<signed-tx-base64>","encoding":"base64"}'
```

Response includes `result` (signature), `submission_kind`, `transaction_id`.

### Submit bundle (up to 5 txs)

```bash
curl -X POST http://localhost:8080/v1/bundles \
  -H 'Content-Type: application/json' \
  -d '{"transactions":["<tx0>","<tx1>"],"encoding":"base64"}'
```

Response `result` is the Jito `bundle_id`.

## Bounty README answers

### 1. What does processed→confirmed delta tell you about network health?

A rising processed→confirmed delta means validators are seeing transactions quickly (processed) but cluster voting/fork resolution is lagging before confirmation. Under congestion or degraded leader performance, this gap widens—it's a real-time signal that landing probability is dropping even when submissions appear healthy at processed commitment.

### 2. Why not use finalized commitment for time-sensitive blockhashes?

Finalized commitment trails the chain tip by many slots (~13+ seconds). A blockhash valid at finalized may already be expired at the current tip. Time-sensitive transactions need `processed` or `confirmed` blockhashes to maximize valid signing window.

### 3. What happens if the Jito leader skips their slot?

The bundle misses that leader's block engine window. Aegis classifies this as `leader_skipped`, records the failure, and surfaces advisory AI guidance. Clients must rebuild and resubmit with a fresh blockhash.

## Stack

- Go + Chi + Postgres + sqlc + goose + River + River UI
- Yellowstone gRPC (Triton) + Jito block engine
- OpenAI gpt-4o-mini agent (advisory)

## Docs

- [Architecture](docs/architecture.md)
- [Dashboard contract](docs/dashboard-data-contract.md)
- [Operations](docs/operations.md)
- [Lifecycle log](docs/lifecycle-log.md)
