#!/usr/bin/env bash

set -euo pipefail

PORT=8082

pid="$(lsof -ti tcp:"$PORT" 2>/dev/null || true)"
if [ -n "$pid" ]; then
  kill "$pid"
  printf 'inventory server on :%s stopped (PID %s)\n' "$PORT" "$pid"
else
  printf 'no inventory server running on :%s\n' "$PORT"
fi
