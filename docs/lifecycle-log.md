# Lifecycle Log Evidence

Export structured lifecycle entries for bounty verification:

```bash
curl -s http://localhost:8080/v1/lifecycle-log?limit=50 | jq .
```

Each entry includes:

- Slot numbers (`submitted_slot`, `processed_slot`, `confirmed_slot`, `finalized_slot`)
- Commitment progression (`commitment_progression` array)
- Timestamps (`submitted_at`, `processed_at`, `confirmed_at`, `finalized_at`, `failed_at`)
- Tip amounts (`tip_lamports`)
- Latency deltas (`latency_processed_ms`, `latency_confirmed_ms`)
- Leader identity
- Failure classification + human-readable title when applicable

Per-transaction timeline:

```bash
curl -s http://localhost:8080/v1/transactions/{id}/timeline | jq .
```

## Bounty evidence export

The live integration test writes explorer-verifiable evidence to [lifecycle-log-evidence.md](./lifecycle-log-evidence.md):

```bash
AEGIS_RUN_BOUNTY_LOG=1 go test -tags=integration -v -timeout=20m ./test/lifecycle/...
```

This submits 10 real server-signed Jito transactions via `POST /v1/ops/submit`
(8 normal, 2 with `inject_expired_blockhash`) and exports **full, untruncated**
signatures, bundle ids, slot numbers, commitment progression, timestamps, tip
amounts, and failure classifications.

### Latest verified run (2026-06-27, mainnet-beta)

- 8/10 finalized on-chain, 2/10 failed with `expired_blockhash` (injected) — see
  [lifecycle-log-evidence.md](./lifecycle-log-evidence.md) for every full signature.
- Dynamic tips ranged 65,000–80,000 lamports, scaling with live congestion and
  leader quality, never below the Jito minimum floor.
- Each success confirmed within ~5s and finalized within ~15s of submission.
- Spot-checked on-chain via `getSignatureStatuses`, e.g. signature
  `4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP`
  is `finalized` at slot `429256028`.

### How submissions land

Each ops submission is a single transaction that carries the payload **and** the
Jito tip, sent through Jito's `sendTransaction` endpoint (base64). Jito wraps it
into a bundle and returns the bundle id via the `x-bundle-id` header, so the
lifecycle is still tracked as a bundle while landing reliably on the public
block engine. Status polling is time-bounded to 5 minutes total and reads the
on-chain signature status (the source of truth) plus Jito's
`getBundleStatuses` / `getInflightBundleStatuses`.

## Automated bounty run (mainnet)

Requires a funded ops keypair at `AEGIS_KEYPAIR_PATH` and a running Aegis server:

```bash
go run ./cmd/aegis-lifecycle-runner -count 10 -failures 2
```

This executes 10 server-signed bundle submissions via `POST /v1/ops/submit`, with the first 2 using `inject_expired_blockhash` to demonstrate autonomous AI recovery (detect → reason → refresh blockhash → recalc tip → resubmit).

Output is written to `lifecycle-log.json`. Verify slot numbers on Solana explorer using signatures from the export.

## Manual scenarios

```bash
# Normal ops bundle
curl -X POST http://localhost:8080/v1/ops/submit -H 'Content-Type: application/json' \
  -d '{"memo":"demo-normal","lamports":1}'

# Forced blockhash expiry (autonomous recovery demo)
curl -X POST http://localhost:8080/v1/ops/submit -H 'Content-Type: application/json' \
  -d '{"memo":"demo-expired","inject_expired_blockhash":true}'

# Client tx with dynamic tip (auto-wrapped bundle)
curl -X POST http://localhost:8080/v1/transactions -H 'Content-Type: application/json' \
  -d '{"transaction":"<signed-tx-base64>","tip_lamports":50000}'
```
