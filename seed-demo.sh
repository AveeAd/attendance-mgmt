#!/usr/bin/env bash
# Seeds a fresh database with demo/test accounts covering every role and
# pay type — see README.md for the resulting credentials table.
#
# Usage: ./seed-demo.sh [db_path]   (default: backend/attendance.dev.db)
set -euo pipefail

cd "$(dirname "$0")/backend"

db_path="${1:-attendance.dev.db}"

go run ./cmd/seed "$db_path" ADMIN1 "Alex Admin"    1234 admin   monthly 0     false
go run ./cmd/seed "$db_path" MGR1   "Morgan Manager" 2345 manager monthly 3000  false
go run ./cmd/seed "$db_path" STAFF1 "Sam Staff"      3456 staff   monthly 2000  false
go run ./cmd/seed "$db_path" TEMP1  "Toni Temp"      4567 staff   hourly  15    true

echo "==> seeded demo accounts into $db_path"
