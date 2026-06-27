#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go run cmd/aegis-cli/main.go -memo "aegis-expired-$(date +%s)" -inject-expired
