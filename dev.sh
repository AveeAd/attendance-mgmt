#!/usr/bin/env bash
# Runs the backend (go run, with live reload of Go code on each restart)
# and the frontend dev server (Vite, with hot reload) side by side.
#
# Visit http://localhost:5173 — Vite proxies /api to the Go server on :8080
# (see frontend/vite.config.js), so you get instant frontend reloads without
# rebuilding the embedded dist/ or restarting the backend.
#
# Ctrl+C stops both.
set -euo pipefail

cd "$(dirname "$0")"

export ATTENDANCE_DB_PATH="${ATTENDANCE_DB_PATH:-attendance.dev.db}"
export ATTENDANCE_PORT="${ATTENDANCE_PORT:-8080}"

cleanup() {
  echo
  echo "==> stopping dev servers"
  kill "${BACKEND_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> starting backend on :$ATTENDANCE_PORT (db: backend/$ATTENDANCE_DB_PATH)"
(cd backend && go run .) &
BACKEND_PID=$!

echo "==> starting frontend dev server on :5173"
(cd frontend && npm install --silent && npm run dev)
