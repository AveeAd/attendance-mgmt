# The Backyard — Attendance Management System

Locally-hosted attendance system: staff scan a QR code on the office WiFi,
log in with Employee ID + PIN, and check in/out. Single Go binary serves
the API and the embedded React frontend — see `attendance-system-spec.md`
for the full spec.

## Project layout

```
backend/     Go API + SQLite (modernc.org/sqlite, no CGO) + embedded frontend build
frontend/    React (Vite) — staff punch view + manager dashboard
installer/   Per-OS installer configs (NSIS, nfpm, pkgbuild) used by the release workflow
.github/     CI: builds and publishes installers for every vX.Y.Z tag
build.sh     Production build: frontend build -> embedded into the Go binary
dev.sh       Dev mode: Go backend + Vite dev server, both hot-reloading
seed-demo.sh Seeds a fresh DB with the demo accounts below
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

## Installing on the office laptop (Windows, macOS, or Linux)

Every push of a `vX.Y.Z` tag builds and publishes installers for all three
platforms as GitHub Release assets — download the one for your OS from the
repo's Releases page:

| Platform | File | What it does |
|---|---|---|
| Windows | `attendance-mgmt-windows-amd64-setup.exe` | Installs to `%LOCALAPPDATA%\attendance-mgmt` (no admin needed), registers a Task Scheduler entry so it starts on every login, and starts it immediately. |
| macOS | `attendance-mgmt-macos-{amd64,arm64}.pkg` | Installs to `/usr/local/attendance-mgmt`, loads a per-user LaunchAgent so it starts on every login. |
| Linux | `attendance-mgmt_linux_{amd64,arm64}.deb` / `.rpm` | Installs to `/usr/bin`, enables+starts a systemd service (`attendance-mgmt`) that runs at boot. |

These installers are **unsigned** (no code-signing certificate) — your OS
will warn you before running them:

- **Windows**: SmartScreen shows "Windows protected your PC" → click **More
  info** → **Run anyway**.
- **macOS**: Gatekeeper blocks the `.pkg` outright the first time — either
  right-click the file → **Open**, or run `xattr -d com.apple.quarantine
  attendance-mgmt-macos-*.pkg` in Terminal first, then double-click it.

A plain portable archive (`attendance-mgmt_{windows,darwin,linux}_{amd64,arm64}.{zip,tar.gz}`)
is also published on every release, if you'd rather run the binary directly
without installing anything.

### Updating

The running app checks GitHub for new releases automatically in the
background. When one is found, a manager/admin will see an **"Update
available — Restart to apply"** button in the Manager Dashboard header —
clicking it downloads and swaps in the new binary, then restarts the app
(a few seconds of downtime). Nothing is applied without that explicit
click. This requires the laptop to have outbound internet access; if it
doesn't, just download and re-run the latest installer instead — re-running
it over an existing install also updates it.

### Releasing (maintainers)

```
git tag v1.2.3 && git push origin v1.2.3
```

pushing a `vX.Y.Z` tag triggers `.github/workflows/release.yml`, which
builds all five platform/arch binaries, packages the three installers plus
portable archives, and publishes everything as a GitHub Release with a
`SHA256SUMS` file.
