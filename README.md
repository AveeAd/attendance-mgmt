# The Backyard — Attendance Management System

Locally-hosted attendance system: staff scan a QR code on the office WiFi,
log in with Employee ID + PIN, and check in/out. Single Go binary serves
the API and the embedded React frontend — see `attendance-system-spec.md`
for the full spec.

## Project layout

```
backend/    Go API + SQLite (modernc.org/sqlite, no CGO) + embedded frontend build
frontend/   React (Vite) — staff punch view + manager dashboard
build.sh    Production build: frontend build -> embedded into the Go binary
dev.sh      Dev mode: Go backend + Vite dev server, both hot-reloading
seed-demo.sh   Seeds a fresh DB with the demo accounts below
```

## Dev mode

```
./dev.sh
```

Starts the Go backend on `:8080` and the Vite dev server on `:5173`
(proxying `/api` to the backend, so both hot-reload independently). Open
**http://localhost:5173**.

## Production build

```
./build.sh                                # native binary for this machine
GOOS=windows GOARCH=amd64 ./build.sh       # cross-compile the office laptop .exe
```

Builds the frontend, embeds it into the Go binary, and outputs
`./attendance-mgmt` (or `attendance-mgmt.exe` for Windows). Run it directly:

```
ATTENDANCE_PORT=8080 ATTENDANCE_DB_PATH=attendance.db ./attendance-mgmt
```

## Seeding accounts

Accounts can only be created by an admin/manager (via the app), so a fresh
database needs at least one admin bootstrapped directly:

```
cd backend && go run ./cmd/seed <db_path> <employee_code> <name> <pin> [role] [pay_type] [pay_rate] [is_temp]
```

For a full demo dataset instead, run from the repo root:

```
./seed-demo.sh                       # seeds backend/attendance.dev.db
./seed-demo.sh backend/attendance.db  # seed a specific db file
```

### Test/demo credentials

Created by `seed-demo.sh`. **For local development only — do not reuse
these PINs in a real deployment.**

| Employee ID | PIN  | Role    | Pay type       |
|-------------|------|---------|-----------------|
| `ADMIN1`    | 1234 | admin   | monthly (NPR 0)    |
| `MGR1`      | 2345 | manager | monthly (NPR 3000) |
| `STAFF1`    | 3456 | staff   | monthly (NPR 2000) |
| `TEMP1`     | 4567 | staff (temp) | hourly (NPR 15/hr) |

Log in with Employee ID + PIN at `/login`. Admin/manager accounts land on
the manager dashboard; staff accounts land on the check-in/check-out page.

Note: accounts are device-bound — the first login registers the device, and
logging in from a different device is rejected until a manager clears the
binding (Employees tab → "Reset device"). If you test multiple accounts in
the same browser, each gets its own device ID automatically (stored in
localStorage), so this only matters if you reuse the same account from two
different browsers/phones.
