#!/usr/bin/env bash
# Builds the frontend into backend/internal/webui/dist, then compiles the
# Go binary with it embedded.
#
# Pass GOOS/GOARCH env vars to cross-compile, e.g.
#   GOOS=windows GOARCH=amd64 ./build.sh
# Pass VERSION to stamp a release version into the binary (defaults to
# "dev", which disables the in-app updater), e.g.
#   VERSION=v1.2.3 GOOS=linux GOARCH=arm64 ./build.sh
set -euo pipefail

cd "$(dirname "$0")"

echo "==> building frontend"
(cd frontend && npm install && npm run build)

echo "==> building backend"
goos="${GOOS:-$(go env GOOS)}"
goarch="${GOARCH:-$(go env GOARCH)}"
version="${VERSION:-dev}"

if [ -n "${GOOS:-}${GOARCH:-}" ]; then
  out="attendance-mgmt-${goos}-${goarch}"
else
  out="attendance-mgmt"
fi
if [ "$goos" = "windows" ]; then
  out="${out}.exe"
fi

(cd backend && go build -ldflags "-X attendance-mgmt/backend/internal/version.Version=${version}" -o "../$out" .)

echo "==> built ./$out"
