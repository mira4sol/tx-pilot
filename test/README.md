# TX Pilot Integration Tests

Live HTTP/WebSocket tests against a running TX Pilot server. Developer tests build and sign **real** Solana transactions client-side, then POST encoded payloads to TX Pilot for Jito forwarding.

## Who signs the transaction?

`test/helpers` loads `tx-pilot-test-keypair.json`, fetches blockhash from `GET /v1/blockhash`, builds a system transfer (1 lamport), signs locally with solana-go, and POSTs the encoded tx to:

- `POST /v1/transactions` — single tx
- `POST /v1/bundles` — transfer + tip tx bundle

## Test wallet

| Item       | Value                                                |
| ---------- | ---------------------------------------------------- |
| File       | `tx-pilot-test-keypair.json` (gitignored, project root) |
| Public key | `Hgj2HKri7Re9RWucnhoETSnNqmBMmQCLZpuigHbfMhj`        |
| Generate   | `make test-keypair`                                  |
| Fund       | ~0.05 SOL on mainnet-beta                            |

## Transfer target

Developer tests send **1 lamport** to:

`5SEZmBS8s41cJ8g3gmLS1BexujHcNZHe5qznPJMdVUsh`

Bundle tests add a second tx tipping a Jito tip account (1000 lamports minimum).

## Prerequisites

1. `make test-keypair` (once)
2. Fund the public key above on mainnet-beta
3. `make migrate-up` (includes `002_raw_submissions.sql`)
4. `make dev` (server running)

## Run

```bash
make test-integration-developer
make test-integration-dashboard
make test-integration-ws
make test-integration-lifecycle   # requires TX_PILOT_RUN_BOUNTY_LOG=1, live mainnet server
make test-integration
```

## Configuration

| Env                   | Default                 | Description  |
| --------------------- | ----------------------- | ------------ |
| `TX_PILOT_TEST_BASE_URL` | `http://localhost:8080` | API base URL |
