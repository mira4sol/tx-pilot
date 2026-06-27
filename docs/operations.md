# Aegis Operations Runbook

## Startup

See [Setup & run guide](setup.md) for full installation. Minimal:

```bash
make docker-up          # Postgres
export $(grep -v '^#' .env | xargs)
make migrate-up
make dev
```

River UI: `http://localhost:8080/riverui`

## Client signing flow

1. `GET /v1/blockhash` — fetch recent blockhash
2. `GET /v1/tip-accounts` — pick a Jito tip account (bundles)
3. Build + sign transaction(s) locally (solana-go, CLI, etc.)
4. Base64 or base58 encode signed txs
5. `POST /v1/transactions` (single) or `POST /v1/bundles` (1–5 txs)

## Submit single transaction

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{"transaction":"<signed-base64>","encoding":"base64","memo":"hello"}'
```

## Submit bundle

```bash
curl -X POST http://localhost:8080/v1/bundles \
  -H 'Content-Type: application/json' \
  -d '{"transactions":["<main-tx>","<tip-tx>"],"encoding":"base64"}'
```

## Poll status

```bash
curl http://localhost:8080/v1/transactions/tx_<id>
curl http://localhost:8080/v1/bundles/<bundle_id>
```

## CLI helper

```bash
go run cmd/aegis-cli/main.go -memo demo
go run cmd/aegis-cli/main.go -bundle -memo bundle-demo
```

Requires `AEGIS_KEYPAIR_PATH` for local signing.

## Dashboard

- Snapshot: `GET /v1/dashboard/snapshot`
- WebSocket: `GET /v1/ws` then subscribe to channels from `docs/dashboard-data-contract.md`

## Required env

- `SOLANA_RPC_URL`, `YELLOWSTONE_GRPC_URL`, `YELLOWSTONE_GRPC_TOKEN`
- `DATABASE_URL`, `OPENAI_API_KEY`
- `JITO_MIN_TIP_LAMPORTS` (default 1000, advisory floor in responses)
- `AEGIS_KEYPAIR_PATH` (optional for server; required for CLI/integration test signing)

## Health

`GET /healthz`
