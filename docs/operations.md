# TX Pilot Operations Runbook

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

1. `GET /v1/blockhash` — fetch recent blockhash at `processed` commitment
2. `GET /v1/tip-accounts` — list Jito tip accounts (needed if you embed your own tip in a client bundle)
3. Build + sign transaction(s) locally (solana-go, CLI, etc.)
4. Base64 or base58 encode signed txs
5. Choose a submit path:
   - **`POST /v1/transactions`** — single pre-signed tx (unchanged on wire; server reports advisory tip)
   - **`POST /v1/bundles`** — 1–4 client txs; server appends a dynamically planned + signed tip tx

## Submit single transaction

The server forwards your signed payload via Jito `sendTransaction` + RPC mirror. It does **not** append a server tip (that would invalidate your signature). The response includes the dynamically planned advisory tip for observability.

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{"transaction":"<signed-base64>","encoding":"base64","memo":"hello","policy_mode":"SAFE"}'
```

Optional `tip_lamports` overrides the AI/heuristic recommendation in the response metadata.

## Submit bundle (server-signed tip appended)

Send 1–4 pre-signed client transactions. TX Pilot plans a dynamic tip, signs a separate tip transfer to a Jito tip account, appends it as the last bundle tx, and submits via Jito `sendBundle` + RPC mirror.

```bash
curl -X POST http://localhost:8080/v1/bundles \
  -H 'Content-Type: application/json' \
  -d '{"transactions":["<client-tx-base64>"],"encoding":"base64","policy_mode":"SAFE"}'
```

You do **not** need to include a tip tx in the client payload — the server adds one. Max 4 client txs (5 total including tip).

## Submit ops (server-signed, embedded tip)

Server builds and signs a self-transfer with an embedded Jito tip. Used for demos, lifecycle evidence, and autonomous recovery.

```bash
curl -X POST http://localhost:8080/v1/ops/submit \
  -H 'Content-Type: application/json' \
  -d '{"memo":"demo-normal","lamports":1,"policy_mode":"SAFE"}'
```

Fault injection (expired blockhash → AI recovery):

```bash
curl -X POST http://localhost:8080/v1/ops/submit \
  -H 'Content-Type: application/json' \
  -d '{"memo":"demo-expired","inject_expired_blockhash":true}'
```

## Poll status

```bash
curl http://localhost:8080/v1/transactions/tx_<id>
curl http://localhost:8080/v1/transactions/tx_<id>/timeline
curl http://localhost:8080/v1/bundles/<bundle_id>
```

## CLI helper

```bash
go run cmd/tx-pilot-cli/main.go -memo demo
go run cmd/tx-pilot-cli/main.go -bundle -memo bundle-demo
```

Requires `TX_PILOT_KEYPAIR_PATH` for local signing.

## Dashboard

- Snapshot: `GET /v1/dashboard/snapshot`
- WebSocket: `GET /v1/ws` then subscribe to channels from [dashboard-data.md](dashboard-data.md)

## Required env

- `SOLANA_RPC_URL`, `YELLOWSTONE_GRPC_URL`, `YELLOWSTONE_GRPC_TOKEN`
- `DATABASE_URL`, `OPENAI_API_KEY`
- `JITO_MIN_TIP_LAMPORTS` (default 1000; floor clamp for dynamic tips)
- `TX_PILOT_KEYPAIR_PATH` (required for bundle tip signing, ops submit, and recovery; optional if only forwarding client txs via `/v1/transactions`)

## Health

`GET /healthz`
