#!/usr/bin/env bash
set -euo pipefail
go run cmd/tx-pilot-cli/main.go -memo "tx-pilot-normal-$(date +%s)"
