# Lifecycle Log Evidence

Export recent transactions for bounty verification:

```bash
curl -s http://localhost:8080/v1/dashboard/transactions | jq .
```

Per-transaction timeline:

```bash
curl -s http://localhost:8080/v1/transactions/{id}/timeline | jq .
```

Each entry should include slot numbers, commitment progression, timestamps, tip amounts, and failure classification when applicable.

Bounty requirement: at least 10 real bundle submissions with 2+ failure cases. Use:

```bash
./scripts/demo-normal.sh          # repeat for successful runs
./scripts/demo-expired-blockhash.sh
go run cmd/aegis-cli/main.go -tip 1   # low-tip failure scenario
```

Verify slots on Solana explorer using signatures from dashboard stream.
