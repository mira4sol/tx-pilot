#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
curl -s "${AEGIS_PUBLIC_API_BASE_URL:-http://localhost:8080}/v1/dashboard/transactions" | jq '.rows | length'
