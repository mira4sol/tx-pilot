# TX Pilot — Setup & Run Guide

This guide walks through installing dependencies, configuring credentials, starting infrastructure, and running the TX Pilot API server and dashboard.

For API usage after the server is up, see [Operations Runbook](operations.md). For architecture and data contracts, see [Architecture design (GitBook)](https://mira4sol.gitbook.io/tx-pilot), [architecture-design.md](architecture-design.md), and [Dashboard data contract](dashboard-data.md).

---

## Build everything with one command

After dependencies are installed and `.env` is configured, **`make build` is all you need to prepare a runnable release**:

```bash
make build
```

This single target:

1. **Builds the dashboard** — `pnpm build` in `web/`, output to `web/dist/` (same-origin API URLs baked in)
2. **Builds the Go binary** — compiles `cmd/tx-pilot/main.go` to **`./tx-pilot`** in the project root

You get one binary that serves the API, WebSocket streams, embedded dashboard, and River UI on `:8080`. No separate frontend server in production.

```bash
./tx-pilot
# or: make run
```

Open **http://localhost:8080/** for the dashboard.

`make build-all` is an alias for `make build`. Use `make build-go` or `make build-web` only when you need to rebuild one half.

---

## What you are running

| Component | Role | Default URL |
|-----------|------|-------------|
| **TX Pilot API** | Transaction control plane, lifecycle tracking, dashboard REST/WS | `http://localhost:8080` |
| **Dashboard (web)** | Real-time ops UI (React + Vite) | After `make build`: **http://localhost:8080/** (served by `./tx-pilot`) · Dev hot reload: `http://localhost:5173` |
| **Postgres** | Persistence, River job queue | `localhost:5432` |
| **River UI** | Background job inspector | `http://localhost:8080/riverui` |

TX Pilot connects to **live mainnet** services by default (Solana RPC, Yellowstone gRPC, Jito block engine, OpenAI). You need valid credentials and a funded ops keypair before submitting real transactions.

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
git clone <repo-url> tx-pilot
cd tx-pilot
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
| `TX_PILOT_KEYPAIR_PATH` | Path to server ops keypair JSON (see below) |

#### Common optional variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TX_PILOT_ENV` | `development` | Log/environment label |
| `TX_PILOT_CLUSTER` | `mainnet-beta` | Cluster name exposed to dashboard |
| `TX_PILOT_HTTP_ADDR` | `:8080` | API listen address |
| `TX_PILOT_POLICY_MODE` | `SAFE` | `SAFE`, `FAST`, `CHEAP`, or `AGGRESSIVE` |
| `TX_PILOT_WEB_DIR` | `web/dist` | Static dashboard files; set empty to disable |
| `JITO_MIN_TIP_LAMPORTS` | `1000` | Advisory minimum tip floor |
| `OPENAI_MODEL` | `gpt-4o-mini` | Agent model |

See `.env.example` for SolInfra REST, webhooks, and Jito URL overrides.

#### Database URL and Docker

`deployments/docker-compose.yaml` creates database **`txpilot`**:

```
postgresql://postgres:admin@localhost:5432/txpilot?sslmode=disable&search_path=public
```

The sample `.env.example` uses database name `txpilot`. Either:

- Point `DATABASE_URL` at the Docker database **`txpilot`** (recommended for local dev), or
- Create `txpilot` manually and keep the example URL.

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

`TX_PILOT_KEYPAIR_PATH` is the server ops keypair. TX Pilot uses it to:

- Sign **separate tip transactions** appended to client bundles (`POST /v1/bundles`)
- Sign **ops transactions** with embedded tips (`POST /v1/ops/submit`)
- Sign **autonomous recovery resubmits** after AI-driven failure handling

Client transactions submitted via `POST /v1/transactions` are forwarded unchanged — the server never re-signs a client payload.

```bash
make test-keypair
```

This creates `tx-pilot-test-keypair.json` in the project root (gitignored). Set in `.env`:

```env
TX_PILOT_KEYPAIR_PATH=./tx-pilot-test-keypair.json
```

Fund the printed public key with **~0.05 SOL on mainnet-beta** before running integration tests or live submissions.

Integration tests use the same file for **client-side** signing of test transfers; the server signs tip txs for bundle submissions and all ops paths.

---

## 4. Database

### Start Postgres

```bash
make docker-up
```

Verify the container is healthy:

```bash
docker ps --filter name=tx-pilot-postgres
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

### Production — `make build` then run

This is the normal path once setup is done. **`make build` prepares everything**: frontend assets in `web/dist` plus the `./tx-pilot` binary.

```bash
make build
```

What runs under the hood:

| Step | Makefile target | Output |
|------|-----------------|--------|
| 1 | `build-web` | `web/dist/` (dashboard static files, same-origin `/v1` and `/v1/ws`) |
| 2 | `build-go` | `./tx-pilot` (Go binary at project root) |

Run the server:

```bash
make run
# or: ./tx-pilot
```

Open **http://localhost:8080/** for the dashboard. API routes are under `/v1/*`. River UI at **http://localhost:8080/riverui**.

Static files are read from `TX_PILOT_WEB_DIR` (default `web/dist`). If `index.html` is missing, the API still runs but the UI is not mounted — run `make build` (or at least `make build-web`) first.

To rebuild only one part:

```bash
make build-go    # Go binary only (dashboard unchanged)
make build-web   # Dashboard only (binary unchanged)
```

### Development — API only

```bash
make dev
```

Equivalent to `go run cmd/tx-pilot/main.go`. Expect logs:

- `db connected`
- `serving web dashboard` (if `web/dist/index.html` exists)
- `starting tx-pilot api` on `:8080`

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

Use this only while editing frontend code. For a deployable artifact or to run without Node at runtime, use **`make build`** instead.

---

## 6. Verify the stack

### API

```bash
curl -s http://localhost:8080/v1/dashboard/snapshot | head -c 200
curl -s http://localhost:8080/v1/blockhash
```

### Dashboard (production mode)

After `make build && ./tx-pilot`:

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
| `make build` | **Prepare everything**: build dashboard (`web/dist`) + Go binary (`./tx-pilot`) |
| `make build-all` | Alias for `make build` |
| `make build-go` | Go binary only |
| `make build-web` | Frontend production build (same-origin API) |
| `make dev` | Run API with `go run` |
| `make dev-web` | Vite dev server on `:5173` |
| `make run` | Run `./tx-pilot` binary |
| `make docker-up` / `make docker-down` | Postgres container |
| `make migrate-up` / `make migrate-down` | Database migrations |
| `make test-keypair` | Generate `tx-pilot-test-keypair.json` |
| `make test` | Go unit tests |
| `make test-integration*` | Live HTTP/WS tests (server must be running) |
| `make sqlc` | Regenerate type-safe queries |
| `make demo-normal` / `make demo-expired` | Demo scripts |
| `make verify-lifecycle` | Lifecycle log verification |
| `make clean` | Remove `./tx-pilot` binary |
| `make air` | Go hot reload |

---

## 8. CLI helpers

Submit a signed ops transaction from the command line:

```bash
go run cmd/tx-pilot-cli/main.go -memo demo
go run cmd/tx-pilot-cli/main.go -bundle -memo bundle-demo
```

Lifecycle evidence export (writes `lifecycle-log.json`):

```bash
go run ./cmd/tx-pilot-lifecycle-runner -count 10 -failures 2
```

Both require a running server and configured `TX_PILOT_KEYPAIR_PATH`.

---

## 9. Troubleshooting

### Server exits on startup: missing required config

Fill every variable listed in the error (RPC, Yellowstone, `DATABASE_URL`, `OPENAI_API_KEY`, `TX_PILOT_KEYPAIR_PATH`).

### Database connection refused

- Run `make docker-up`
- Confirm `DATABASE_URL` host/port/user/password match Docker
- Confirm database **name** matches (`txpilot`)

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

Change `TX_PILOT_HTTP_ADDR=:8081` in `.env`, or stop the process on `:8080` / `:5173`.

---

## 10. Related documentation

| Doc | Contents |
|-----|----------|
| [README.md](../README.md) | Project overview and API summary |
| [architecture.md](architecture.md) | System design |
| [operations.md](operations.md) | Submit, poll, CLI, runbook |
| [dashboard-data.md](dashboard-data.md) | REST + WebSocket shapes |
| [lifecycle-log.md](lifecycle-log.md) | Bounty lifecycle export |
| [test/README.md](../test/README.md) | Integration test setup |

---

## Quick start (copy-paste)

Minimal path from zero to a built binary with dashboard embedded:

```bash
cp .env.example .env          # edit credentials; DATABASE_URL → .../txpilot
make test-keypair             # fund the printed pubkey on mainnet
make docker-up
export $(grep -v '^#' .env | xargs)
make migrate-up
make build                    # dashboard (web/dist) + binary (./tx-pilot) — one command
./tx-pilot                    # http://localhost:8080/ — API + dashboard + River UI
```

For frontend hot reload during development, use `make dev` + `make dev-web` instead of `make build`.
