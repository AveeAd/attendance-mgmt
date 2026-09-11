#!/usr/bin/env bash
# Builds the frontend into backend/internal/webui/dist, then compiles the
# Go binary with it embedded. Pass GOOS/GOARCH env vars to cross-compile,
# e.g. GOOS=windows GOARCH=amd64 ./build.sh
set -euo pipefail

cd "$(dirname "$0")"

echo "==> building frontend"
(cd frontend && npm install && npm run build)

echo "==> building backend"
out="attendance-mgmt"
if [ "${GOOS:-}" = "windows" ]; then
  out="attendance-mgmt.exe"
fi
(cd backend && go build -o "../$out" .)

echo "==> built ./$out"
