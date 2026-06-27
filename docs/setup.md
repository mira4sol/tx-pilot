# Aegis — Setup & Run Guide

This guide walks through installing dependencies, configuring credentials, starting infrastructure, and running the Aegis API server and dashboard.

For API usage after the server is up, see [Operations Runbook](operations.md). For architecture and data contracts, see [Architecture](architecture.md) and [Dashboard contract](dashboard-data-contract.md).

---

## What you are running

| Component | Role | Default URL |
|-----------|------|-------------|
| **Aegis API** | Transaction control plane, lifecycle tracking, dashboard REST/WS | `http://localhost:8080` |
| **Dashboard (web)** | Real-time ops UI (React + Vite) | Dev: `http://localhost:5173` · Prod: same origin as API (`/`) |
| **Postgres** | Persistence, River job queue | `localhost:5432` |
| **River UI** | Background job inspector | `http://localhost:8080/riverui` |

Aegis connects to **live mainnet** services by default (Solana RPC, Yellowstone gRPC, Jito block engine, OpenAI). You need valid credentials and a funded ops keypair before submitting real transactions.

---

## Prerequisites

### Required tools

| Tool | Version | Notes |
|------|---------|-------|
| **Go** | 1.26+ | See `go.mod` |
| **Docker** | Recent | Postgres via `deployments/docker-compose.yaml` |
| **Node.js** | 18+ | Dashboard build/dev |
| **pnpm** | 9+ | Frontend package manager (`web/`) |

Optional but useful:

- **make** — all common commands are wrapped in the root `Makefile`
- **air** — hot reload for Go (`make air`)

### External services (credentials required)

You must obtain and configure:

1. **Solana RPC + WS** — e.g. SolInfra, Helius, Triton
2. **Yellowstone gRPC** — streaming slot/signature feed (token required)
3. **OpenAI API key** — tip intelligence and recovery agent
4. **Mainnet SOL** — fund the server ops keypair (~0.05 SOL for demos/tests)

Jito endpoints are preconfigured for mainnet in `.env.example`; override if needed.

---

## 1. Clone and install

```bash
git clone <repo-url> aegis
cd aegis
```

### Go dependencies

```bash
go mod download
```

### Frontend dependencies

```bash
cd web
pnpm install
cd ..
```

If `pnpm install` fails on macOS with a missing `@rollup/rollup-darwin-arm64` error, ensure `web/pnpm-workspace.yaml` includes `darwin` under `supportedArchitectures`, then reinstall:

```bash
cd web && rm -rf node_modules && pnpm install
```

---

## 2. Environment configuration

### Backend (`.env`)

Copy the template and fill in real values:

```bash
cp .env.example .env
```

Edit `.env` at the **project root**. The server loads it automatically on startup (`config.Load(".env")`).

#### Required variables

| Variable | Description |
|----------|-------------|
| `SOLANA_RPC_URL` | HTTP JSON-RPC endpoint |
| `SOLANA_WS_URL` | WebSocket RPC (optional but recommended) |
| `YELLOWSTONE_GRPC_URL` | Geyser gRPC host (e.g. `fra.grpc.solinfra.dev:443`) |
| `YELLOWSTONE_GRPC_TOKEN` | Auth token for Yellowstone |
| `DATABASE_URL` | Postgres connection string |
| `OPENAI_API_KEY` | OpenAI API key |
| `AEGIS_KEYPAIR_PATH` | Path to server ops keypair JSON (see below) |

#### Common optional variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AEGIS_ENV` | `development` | Log/environment label |
| `AEGIS_CLUSTER` | `mainnet-beta` | Cluster name exposed to dashboard |
| `AEGIS_HTTP_ADDR` | `:8080` | API listen address |
| `AEGIS_POLICY_MODE` | `SAFE` | `SAFE`, `FAST`, `CHEAP`, or `AGGRESSIVE` |
| `AEGIS_WEB_DIR` | `web/dist` | Static dashboard files; set empty to disable |
| `JITO_MIN_TIP_LAMPORTS` | `1000` | Advisory minimum tip floor |
| `OPENAI_MODEL` | `gpt-4o-mini` | Agent model |

See `.env.example` for SolInfra REST, webhooks, and Jito URL overrides.

#### Database URL and Docker

`deployments/docker-compose.yaml` creates database **`aegis`**:

```
postgresql://postgres:admin@localhost:5432/aegis?sslmode=disable&search_path=public
```

The sample `.env.example` uses database name `aegis_v2`. Either:

- Point `DATABASE_URL` at the Docker database **`aegis`** (recommended for local dev), or
- Create `aegis_v2` manually and keep the example URL.

Mismatch here is the most common local startup failure.

### Frontend (`web/.env`)

For **split dev** (Vite on `:5173`, API on `:8080`):

```bash
cp web/.env.example web/.env
```

```env
VITE_API_URL=http://localhost:8080
```

Vite proxies `/v1` and `/healthz` when using the dev server, but setting `VITE_API_URL` makes the client call the API directly (CORS is enabled on the Go server).

For **production builds** served by the Go binary (`make build`), leave `VITE_API_URL` unset so the dashboard uses same-origin `/v1` and `/v1/ws`. The Makefile clears it automatically:

```bash
make build-web   # VITE_API_URL= VITE_API_BASE= pnpm build
```

---

## 3. Ops keypair

The server signs dynamic tip transactions and recovery ops using a dedicated keypair.

```bash
make test-keypair
```

This creates `aegis-test-keypair.json` in the project root (gitignored). Set in `.env`:

```env
AEGIS_KEYPAIR_PATH=./aegis-test-keypair.json
```

Fund the printed public key with **~0.05 SOL on mainnet-beta** before running integration tests or live submissions.

Integration tests use the same file for **client-side** signing of test transfers; the server still signs tip txs appended to bundles.

---

## 4. Database

### Start Postgres

```bash
make docker-up
```

Verify the container is healthy:

```bash
docker ps --filter name=aegis-postgres
```

### Run migrations

Ensure `DATABASE_URL` is exported (from `.env`):

```bash
export $(grep -v '^#' .env | xargs)
make migrate-up
```

Migrations live in `internal/storage/migrations/` (goose). To roll back one step:

```bash
make migrate-down
```

### Regenerate sqlc (after schema changes)

```bash
make sqlc
```

---

## 5. Running the server

### Development — API only

```bash
make dev
```

Equivalent to `go run cmd/aegis/main.go`. Expect logs:

- `db connected`
- `serving web dashboard` (if `web/dist/index.html` exists)
- `starting aegis api` on `:8080`

Health check:

```bash
curl -s http://localhost:8080/healthz
# {"status":"ok"}
```

### Development — API + dashboard (hot reload)

Terminal 1 — backend:

```bash
make dev
```

Terminal 2 — frontend:

```bash
make dev-web
```

Open **http://localhost:5173**. The dashboard polls `GET /v1/dashboard/snapshot` and connects to `ws://localhost:8080/v1/ws` (via `VITE_API_URL`).

### Production — single binary + embedded static UI

Build frontend and Go binary together:

```bash
make build
# or: make build-all
```

This runs `build-web` (same-origin API config) then `build-go`, producing `./aegis`.

Run:

```bash
make run
# or: ./aegis
```

Open **http://localhost:8080/** for the dashboard. API routes remain under `/v1/*`.

To build Go without rebuilding the web app:

```bash
make build-go
```

To build only the web app:

```bash
make build-web
```

Static files are read from `AEGIS_WEB_DIR` (default `web/dist`). If `index.html` is missing, the API still runs; the UI is simply not mounted.

---

## 6. Verify the stack

### API

```bash
curl -s http://localhost:8080/v1/dashboard/snapshot | head -c 200
curl -s http://localhost:8080/v1/blockhash
```

### Dashboard (production mode)

After `make build && ./aegis`:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/
# 200

curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/assets/
# 200 or 404 for directory; asset paths return 200
```

### River queue UI

Visit **http://localhost:8080/riverui** while the server is running.

### Unit tests (no live server)

```bash
make test
```

Static dashboard serving is covered in `internal/api/static_test.go`.

### Integration tests (live server required)

With `make dev` running and the ops keypair funded:

```bash
make test-integration-developer
make test-integration-dashboard
make test-integration-ws
```

See [test/README.md](../test/README.md) for wallet funding, transfer targets, and lifecycle bounty tests.

---

## 7. Makefile reference

| Command | Description |
|---------|-------------|
| `make build` | Build web (`web/dist`) + Go binary (`./aegis`) |
| `make build-all` | Alias for `make build` |
| `make build-go` | Go binary only |
| `make build-web` | Frontend production build (same-origin API) |
| `make dev` | Run API with `go run` |
| `make dev-web` | Vite dev server on `:5173` |
| `make run` | Run `./aegis` binary |
| `make docker-up` / `make docker-down` | Postgres container |
| `make migrate-up` / `make migrate-down` | Database migrations |
| `make test-keypair` | Generate `aegis-test-keypair.json` |
| `make test` | Go unit tests |
| `make test-integration*` | Live HTTP/WS tests (server must be running) |
| `make sqlc` | Regenerate type-safe queries |
| `make demo-normal` / `make demo-expired` | Demo scripts |
| `make verify-lifecycle` | Lifecycle log verification |
| `make clean` | Remove `./aegis` binary |
| `make air` | Go hot reload |

---

## 8. CLI helpers

Submit a signed ops transaction from the command line:

```bash
go run cmd/aegis-cli/main.go -memo demo
go run cmd/aegis-cli/main.go -bundle -memo bundle-demo
```

Lifecycle evidence export (writes `lifecycle-log.json`):

```bash
go run ./cmd/aegis-lifecycle-runner -count 10 -failures 2
```

Both require a running server and configured `AEGIS_KEYPAIR_PATH`.

---

## 9. Troubleshooting

### Server exits on startup: missing required config

Fill every variable listed in the error (RPC, Yellowstone, `DATABASE_URL`, `OPENAI_API_KEY`, `AEGIS_KEYPAIR_PATH`).

### Database connection refused

- Run `make docker-up`
- Confirm `DATABASE_URL` host/port/user/password match Docker
- Confirm database **name** matches (`aegis` vs `aegis_v2`)

### `serving web dashboard` not in logs

Run `make build-web` or `make build`. The server only mounts the UI when `web/dist/index.html` exists.

### Dashboard shows "API unreachable"

- **Dev split mode:** ensure `make dev` is running and `web/.env` has `VITE_API_URL=http://localhost:8080`
- **Production mode:** use `make build` (not a dev build with hardcoded localhost in JS)
- Check CORS only matters for cross-origin dev; same-origin production does not need it

### Yellowstone / Geyser connection failed

Verify `YELLOWSTONE_GRPC_URL` and `YELLOWSTONE_GRPC_TOKEN`. The server will not start streaming without a valid gRPC connection.

### Integration tests fail with insufficient funds

Fund the public key from `make test-keypair` on mainnet-beta (~0.05 SOL).

### `pnpm dev` / Rollup native module errors

Reinstall with platform support in `web/pnpm-workspace.yaml` (`darwin` + `linux`, `arm64` + `x64`).

### Port already in use

Change `AEGIS_HTTP_ADDR=:8081` in `.env`, or stop the process on `:8080` / `:5173`.

---

## 10. Related documentation

| Doc | Contents |
|-----|----------|
| [README.md](../README.md) | Project overview and API summary |
| [architecture.md](architecture.md) | System design |
| [operations.md](operations.md) | Submit, poll, CLI, runbook |
| [dashboard-data-contract.md](dashboard-data-contract.md) | REST + WebSocket shapes |
| [lifecycle-log.md](lifecycle-log.md) | Bounty lifecycle export |
| [test/README.md](../test/README.md) | Integration test setup |

---

## Quick start (copy-paste)

Minimal path from zero to running API + dashboard UI:

```bash
cp .env.example .env          # edit credentials; fix DATABASE_URL → .../aegis
make test-keypair             # fund the printed pubkey on mainnet
make docker-up
export $(grep -v '^#' .env | xargs)
make migrate-up
make build                    # web + binary
./aegis                       # http://localhost:8080/
```

For frontend hot reload during development, use `make dev` + `make dev-web` instead of `make build`.
