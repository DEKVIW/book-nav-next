#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

export BOOKNAV_ENV=development
export BOOKNAV_DATA_DIR="$ROOT/data"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "==> Starting Go server on :8080"
(cd apps/server && go run ./cmd/server) &
SERVER_PID=$!

echo "==> Starting Vite on :5173"
cd apps/web
if [[ ! -d node_modules ]]; then
  npm install
fi
npm run dev
