#!/usr/bin/env bash
set -euo pipefail
go run cmd/tx-pilot-cli/main.go -memo "tx-pilot-expired-$(date +%s)" -inject-expired
