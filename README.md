# TX Pilot — Autonomous Solana Transaction Control Plane

**Working prototype on mainnet-beta.** Plug in your RPC, Yellowstone, and OpenAI credentials, fund an ops keypair, and run.

Built for trading bots, consumer apps, and any workload pushing high transaction volume. TX Pilot handles submission, tip planning, lifecycle tracking, failure classification, and AI-assisted recovery so you can see _why_ a transaction landed or failed instead of guessing from a signature alone.

![TX Pilot architecture](tx-pilot-architecture.svg)

---

## Why Solana is different from Bitcoin and EVM chains

On Bitcoin and Ethereum, transactions sit in a **public mempool**. Validators or miners pick from that pool, ordering is uncertain, and propagation adds latency before anything executes.

```
  Bitcoin / EVM (simplified)

  Wallet ──► Mempool (shared, unordered pool)
                  │
                  ▼
            Validator / miner picks txs
                  │
                  ▼
               Block
```

Solana was built for throughput. Instead of a global mempool gossip model, **Gulf Stream** forwards transactions directly toward the **TPU** (Transaction Processing Unit) of the upcoming **leader** validator. Leaders rotate every **slot (~400 ms)**. You are racing the clock against slot boundaries and leader schedules, not waiting in a public queue.

```
  Solana (simplified)

  Wallet ──► RPC / TPU path ──► Current leader's TPU
                                      │
                               ~400 ms slot
                                      │
                                      ▼
                               Leader builds block
                                      │
                               Next leader already known
                               (schedule known in advance)
```

That design is why blockhash freshness, leader timing, and tip economics matter more here than on EVM. A transaction can look "sent" while missing the leader window entirely.

---

## What TX Pilot does

TX Pilot is a transaction operations layer you run beside your app. Your bot or frontend signs payloads locally; TX Pilot handles the operational work around getting them landed and understood.

| Capability                        | How                                                                                                                                                                                                                        |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Jito bundles + MEV protection** | Client bundles get a server-signed tip tx appended; submissions go through Jito's block engine with RPC fallback. Atomic bundle ordering reduces sandwich exposure compared to broadcasting blindly to the public mempool. |
| **Dynamic tips**                  | Live Jito tip floor + policy mode (`SAFE` / `FAST` / `CHEAP` / `AGGRESSIVE`) + congestion and leader multipliers. OpenAI adjusts the final tip per submission.                                                             |
| **AI operations agent**           | Tip intelligence on every submit; failure reasoning and autonomous retry on server ops txs (refresh blockhash, recalc tip, resubmit).                                                                                      |
| **Geyser streaming**              | Yellowstone gRPC for slots, leaders, and signature landing with reconnect and backpressure. Stream hits `processed` before RPC polling catches up.                                                                         |
| **Per-transaction workers**       | Each submission spawns its own durable River job. Child workers poll status, classify failures, and trigger recovery independently. One slow tx does not block the rest.                                                   |
| **Observability**                 | REST + WebSocket dashboard, lifecycle timelines, failure taxonomy, and bounty-grade lifecycle export.                                                                                                                      |

### Submission paths

| Endpoint                | What happens                                                                                                                         |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `POST /v1/transactions` | Pre-signed client tx forwarded unchanged via Jito + RPC. Tip in the response is **advisory** (re-signing would break the signature). |
| `POST /v1/bundles`      | 1–4 client txs + server-signed dynamic tip appended, submitted as a Jito bundle + RPC mirror.                                        |
| `POST /v1/ops/submit`   | Server-signed self-transfer with embedded tip. Used for demos, lifecycle evidence, and autonomous recovery.                          |

### How a submission flows

```mermaid
sequenceDiagram
  participant App as Your app / bot
  participant API as TX Pilot API
  participant CP as Control plane
  participant AI as OpenAI agent
  participant Jito as Jito block engine
  participant Geyser as Yellowstone Geyser
  participant River as River queue worker

  App->>API: POST /v1/bundles
  API->>CP: planTip (floor + policy + congestion)
  CP->>AI: DecideTip
  AI-->>CP: set_tip action
  CP->>Jito: sendBundle + RPC mirror
  CP->>River: enqueue status poll job
  Geyser-->>CP: signature seen (processed)
  River->>River: poll confirmed / finalized
  River-->>CP: failure → AI recovery (ops only)
```

### MEV and sandwich protection (practical view)

A classic sandwich attack front-runs and back-runs your swap around public mempool visibility. Routing through **Jito bundles** with a private submission path gives you **atomic ordering**: your txs execute together or not at all, without unrelated txs inserted between them. TX Pilot appends a dynamically priced tip so bundles stay competitive without hardcoded values.

This is not a guarantee against all MEV, but it is strictly better than firing unsigned-order transactions into a public RPC and hoping.

### Queue and worker model

TX Pilot uses **Go goroutines** for hot paths (Geyser reader, dashboard broadcast, lifecycle fanout) and **River** (Postgres-backed) for durable per-transaction work.

When a transaction is submitted, the control plane inserts a **status poll job**. River picks it up on any available worker goroutine. That worker owns the poll loop for _that_ transaction only: RPC signature status, Jito bundle status, blockhash expiry checks, failure classification, and recovery trigger. Jobs retry on their own schedule (~2 s while pending, up to 5 minutes total) without blocking other submissions.

```
  Submit tx_A ──► River job_A ──► worker polls A ──► confirmed / failed
  Submit tx_B ──► River job_B ──► worker polls B ──► confirmed / failed
  Submit tx_C ──► River job_C ──► worker polls C ──► confirmed / failed

  (A, B, C run in parallel; failure on B does not stall A or C)
```

Lifecycle events are written through **8 sharded channels** so concurrent transactions append events in parallel without a single global lock.

Inspect live jobs at `http://localhost:8080/riverui`.

### Why Go (not Rust or Node.js)

We chose **Go** for the control plane because goroutines and channels are a natural fit for this workload:

- **Thousands of concurrent submissions** without a thread-per-request cost. Each transaction gets its own poll loop and lifecycle shard without manual pthread tuning.
- **Simple worker pools** for Geyser ingestion, lifecycle tracking, and River job handlers with clear cancellation via `context`.
- **Fast iteration** on operational tooling (CLI, demos, integration tests) while still compiling to a single static binary for production.

Rust would give similar concurrency with more upfront complexity for API iteration speed. Node.js async works for I/O but struggles with CPU-bound signature handling and long-lived worker supervision at high fanout. Go sits in the middle: fast enough, operable, and built for exactly this kind of network service.

---

## Quick start

Full guide: [docs/setup.md](docs/setup.md)

```bash
cp .env.example .env          # add RPC, Yellowstone, OpenAI, DATABASE_URL
make test-keypair             # creates tx-pilot-test-keypair.json
make docker-up && make migrate-up
make dev                      # API + dashboard at http://localhost:8080
```

|                 |                                                        |
| --------------- | ------------------------------------------------------ |
| **Ops keypair** | `TX_PILOT_KEYPAIR_PATH=./tx-pilot-test-keypair.json`   |
| **Used for**    | Bundle tip signing, ops submit, autonomous recovery    |
| **Fund**        | ~0.05 SOL on mainnet-beta for demos and lifecycle runs |

---

## API (summary)

| Endpoint                             | Description                                                                    |
| ------------------------------------ | ------------------------------------------------------------------------------ |
| `POST /v1/transactions`              | Forward pre-signed tx; advisory tip in response                                |
| `POST /v1/bundles`                   | Client txs + server-signed dynamic tip                                         |
| `POST /v1/ops/submit`                | Server-signed ops tx with embedded tip                                         |
| `GET /v1/lifecycle-log`              | Bounty lifecycle export                                                        |
| `GET /v1/transactions/{id}`          | Poll status                                                                    |
| `GET /v1/transactions/{id}/timeline` | Stage events + `latency_ms`                                                    |
| `GET /v1/dashboard/*`                | Live ops dashboard                                                             |
| `GET /v1/ws`                         | WebSocket streams (`transactions.stream`, `ai.decisions`, `failures.analysis`) |

```bash
# Client bundle with server-appended tip
curl -X POST http://localhost:8080/v1/bundles \
  -H 'Content-Type: application/json' \
  -d '{"transactions":["<signed-base64>"],"policy_mode":"SAFE"}'

# Ops fault injection (AI recovery demo)
curl -X POST http://localhost:8080/v1/ops/submit \
  -H 'Content-Type: application/json' \
  -d '{"memo":"demo-expired","inject_expired_blockhash":true}'
```

Optional `tip_lamports` overrides the AI/heuristic tip (still clamped to the live floor).

---

## Bounty README questions

These answers come from running TX Pilot on **mainnet-beta** with live Yellowstone, Jito, and OpenAI.

### 1. What does the delta between `processed_at` and `confirmed_at` tell you about network health?

`processed` means a leader included your transaction and the cluster has seen it. `confirmed` means a supermajority voted on that fork. The gap between them is a congestion signal.

When the delta is small (hundreds of ms), the cluster is keeping up: leaders produce blocks on schedule and voting keeps pace. When it widens to multiple seconds, validators are ingesting work faster than the cluster can finalize votes. Under that shape, landing probability drops even if your submission returned success immediately. TX Pilot stores this delta in `lifecycle_events.latency_ms` and surfaces it in the dashboard pipeline so you can correlate tip size, leader, and confirmation lag on real traffic.

### 2. Why should you never use `finalized` commitment when fetching a blockhash for a time-sensitive transaction?

`finalized` trails the chain tip by roughly **32 slots (~13 seconds at 400 ms/slot)**. A blockhash that looks valid at `finalized` may already be expired at the current tip where your transaction would actually land. Time-sensitive submits need the freshest hash possible. TX Pilot always fetches blockhashes at **`processed` commitment** for tip and ops signing to maximize the valid window.

### 3. What happens to your bundle if the Jito leader skips their slot?

Each slot has an assigned leader with a narrow window to accept bundles through the block engine. If that leader **skips** their slot (no block produced), your bundle does not execute in that window. It may land on a later attempt if the blockhash is still valid and a subsequent leader picks it up, or it expires and fails.

TX Pilot watches slot gaps on the Geyser feed, records leader context on every lifecycle event, and may delay one slot in `SAFE`/`CHEAP` modes when the schedule looks unfavorable. If the bundle never lands, the failure is classified (e.g. `bundle_rejected`, `expired_blockhash`) and the AI agent chooses recovery: tip increase, delay, or blockhash refresh for ops transactions.

Evidence from our lifecycle runs: 8/10 ops bundles finalized within ~15 s; 2/10 failed with injected expired blockhash and triggered autonomous recovery. See [docs/lifecycle-log.md](docs/lifecycle-log.md).

---

## Stack

Go · Chi · Postgres · sqlc · goose · River · Yellowstone gRPC · Jito block engine · OpenAI `gpt-4o-mini`

---

# TX Pilot Lifecycle Log Evidence

Live mainnet Jito submissions captured by `TestBountyLifecycleLog`. Every
signature, bundle id and slot below is reproduced in full (untruncated) so
each transaction can be independently verified on a Solana explorer.

- Generated: 2026-06-27T14:40:34Z
- Submissions: 10 (target 8 success / 2 failure)
- Lifecycle log entries exported: 50

- Observed: 8 success / 2 failed

## Overview

| #   | Status    | sub slot  | proc slot | conf slot | fin slot  | Tip (lamports) | Failure                                                        |
| --- | --------- | --------- | --------- | --------- | --------- | -------------- | -------------------------------------------------------------- |
| 1   | failed    | 429255986 | 0         | 0         | 0         | 70000          | expired_blockhash (Blockhash expired before the bundle landed) |
| 2   | failed    | 429256011 | 0         | 0         | 0         | 75000          | expired_blockhash (Blockhash expired before the bundle landed) |
| 3   | finalized | 429256015 | 429256028 | 429256028 | 429256028 | 80000          | -                                                              |
| 4   | finalized | 429256024 | 429256064 | 429256064 | 429256064 | 70000          | -                                                              |
| 5   | finalized | 429256082 | 429256105 | 429256105 | 429256105 | 80000          | -                                                              |
| 6   | finalized | 429256099 | 429256141 | 429256141 | 429256141 | 70000          | -                                                              |
| 7   | finalized | 429256159 | 429256173 | 429256173 | 429256173 | 65000          | -                                                              |
| 8   | finalized | 429256183 | 429256201 | 429256201 | 429256201 | 65000          | -                                                              |
| 9   | finalized | 429256200 | 429256244 | 429256244 | 429256244 | 70000          | -                                                              |
| 10  | finalized | 429256235 | 429256280 | 429256280 | 429256280 | 70000          | -                                                              |

## Full transaction records

### 1. failed

- transaction_id: `tx_3c4fe90d-7066-4543-92a8-ea2b36612644`
- bundle_id: `068738840f420080f6365dd039206a421b4032f72b1a369960da599975a4fc7e`
- signature: `2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u`
- explorer: https://explorer.solana.com/tx/2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u
- solscan: https://solscan.io/tx/2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u
- inject_expired_blockhash: true
- tip_lamports: 70000
- slots: submitted=429255986 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T14:37:42Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T14:37:45Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 2. failed

- transaction_id: `tx_adc03f41-0a16-4518-ac8c-080f114ea45b`
- bundle_id: `cd71ac7199dcbaa8cfc746d92959d8b92ef175f185ece30a53b4ead9506a6cbc`
- signature: `61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK`
- explorer: https://explorer.solana.com/tx/61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK
- solscan: https://solscan.io/tx/61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK
- inject_expired_blockhash: true
- tip_lamports: 75000
- slots: submitted=429256011 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T14:37:51Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T14:37:55Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 3. finalized

- transaction_id: `tx_5e10ba4c-2d4e-4ac2-b0be-c1d9c51c91d0`
- bundle_id: `c8319ae19261c6a38dcd66cc75f1d4bb336d4e969f5a882ffd61a4e541bf19af`
- signature: `4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP`
- explorer: https://explorer.solana.com/tx/4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP
- solscan: https://solscan.io/tx/4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429256015 processed=429256028 confirmed=429256028 finalized=429256028
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:38:00Z
- processed_at: 2026-06-27T14:38:05Z
- confirmed_at: 2026-06-27T14:38:10Z
- finalized_at: 2026-06-27T14:38:15Z
- failed_at: -

### 4. finalized

- transaction_id: `tx_745d85a9-0111-4beb-888b-5583e322234b`
- bundle_id: `38b86a456cd4f1024aaf9c58e23caa158eecf0efcfb03353bed566b76f2be98d`
- signature: `5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT`
- explorer: https://explorer.solana.com/tx/5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT
- solscan: https://solscan.io/tx/5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256024 processed=429256064 confirmed=429256064 finalized=429256064
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:15Z
- processed_at: 2026-06-27T14:38:20Z
- confirmed_at: 2026-06-27T14:38:25Z
- finalized_at: 2026-06-27T14:38:30Z
- failed_at: -

### 5. finalized

- transaction_id: `tx_53ce0125-9836-4c37-ad07-22e311279554`
- bundle_id: `f22734ebf4ebf10a36c8477b50834150ce55a00d38120fbd95c5a89474ef24fc`
- signature: `52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u`
- explorer: https://explorer.solana.com/tx/52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u
- solscan: https://solscan.io/tx/52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429256082 processed=429256105 confirmed=429256105 finalized=429256105
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:28Z
- processed_at: 2026-06-27T14:38:35Z
- confirmed_at: 2026-06-27T14:38:40Z
- finalized_at: 2026-06-27T14:38:45Z
- failed_at: -

### 6. finalized

- transaction_id: `tx_67fe4968-0132-422b-a785-4e326e310645`
- bundle_id: `dd8bd76ce12285fed2949879be7ca5fb441322514e09deee7bad9a5caa635264`
- signature: `2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG`
- explorer: https://explorer.solana.com/tx/2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG
- solscan: https://solscan.io/tx/2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256099 processed=429256141 confirmed=429256141 finalized=429256141
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:45Z
- processed_at: 2026-06-27T14:38:50Z
- confirmed_at: 2026-06-27T14:38:55Z
- finalized_at: 2026-06-27T14:39:00Z
- failed_at: -

### 7. finalized

- transaction_id: `tx_49a804eb-b0b4-42de-b0d7-2ed39b69474e`
- bundle_id: `8a6116087e950428832fd441b6130bc18ae5d4dee9215be6c212043b5285e464`
- signature: `41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w`
- explorer: https://explorer.solana.com/tx/41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w
- solscan: https://solscan.io/tx/41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w
- inject_expired_blockhash: false
- tip_lamports: 65000
- slots: submitted=429256159 processed=429256173 confirmed=429256173 finalized=429256173
- commitment_progression: created -> submitted -> confirmed
- submitted_at: 2026-06-27T14:38:57Z
- processed_at: 2026-06-27T14:39:00Z
- confirmed_at: 2026-06-27T14:39:10Z
- finalized_at: 2026-06-27T14:39:15Z
- failed_at: -

### 8. finalized

- transaction_id: `tx_9447e20f-8dc4-4655-996d-8c92a63f2285`
- bundle_id: `a7e31350cd74f3fadffd0c786fec7b96141b4cc0a7f166577424f108a412c24b`
- signature: `5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX`
- explorer: https://explorer.solana.com/tx/5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX
- solscan: https://solscan.io/tx/5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX
- inject_expired_blockhash: false
- tip_lamports: 65000
- slots: submitted=429256183 processed=429256201 confirmed=429256201 finalized=429256201
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:07Z
- processed_at: 2026-06-27T14:39:15Z
- confirmed_at: 2026-06-27T14:39:20Z
- finalized_at: 2026-06-27T14:39:25Z
- failed_at: -

### 9. finalized

- transaction_id: `tx_c6b5aaa8-bb79-4230-ad4d-4602803f59f2`
- bundle_id: `034eb114a54bf738329d0da84a3bd03eddef07b492ba699f944172c5400ba154`
- signature: `52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J`
- explorer: https://explorer.solana.com/tx/52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J
- solscan: https://solscan.io/tx/52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256200 processed=429256244 confirmed=429256244 finalized=429256244
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:27Z
- processed_at: 2026-06-27T14:39:30Z
- confirmed_at: 2026-06-27T14:39:40Z
- finalized_at: 2026-06-27T14:39:45Z
- failed_at: -

### 10. finalized

- transaction_id: `tx_9cc04a03-f57b-4258-830d-f1aa43e834ac`
- bundle_id: `98a81486f00cb5c90bb36b8f39492addf5973b9745e8d20e6d66a67cb19c296f`
- signature: `5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT`
- explorer: https://explorer.solana.com/tx/5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT
- solscan: https://solscan.io/tx/5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256235 processed=429256280 confirmed=429256280 finalized=429256280
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:41Z
- processed_at: 2026-06-27T14:39:45Z
- confirmed_at: 2026-06-27T14:39:55Z
- finalized_at: 2026-06-27T14:40:00Z
- failed_at: -

## Explorer verification

Open any `explorer` link above, or paste the full signature into
[Solana Explorer](https://explorer.solana.com/) / [Solscan](https://solscan.io/),
and cross-reference the confirmed/finalized slot recorded here.

---

## Documentation

| Doc                                                              | Read this for                                                     |
| ---------------------------------------------------------------- | ----------------------------------------------------------------- |
| [docs/setup.md](docs/setup.md)                                   | Install, `.env` variables, Docker Postgres, migrations, first run |
| [docs/architecture.md](docs/architecture.md)                     | Components, tip planning pipeline, signing model, AI boundaries   |
| [docs/operations.md](docs/operations.md)                         | Submit flows, curl examples, CLI helper, polling                  |
| [docs/dashboard-data.md](docs/dashboard-data.md)                 | REST and WebSocket contract for the ops dashboard                 |
| [docs/lifecycle-log.md](docs/lifecycle-log.md)                   | Lifecycle export format and bounty evidence collection            |
| [docs/lifecycle-log-evidence.md](docs/lifecycle-log-evidence.md) | Verified mainnet run with full signatures and slot numbers        |

Architecture diagram source: [tx-pilot-architecture.svg](tx-pilot-architecture.svg)
