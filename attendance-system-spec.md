# Attendance Management System — Phase 1
**The Backyard Project — Local HRM System**
**Purpose of this doc**: handoff spec for implementation in Claude Code.

---

## 1. Overview
A locally-hosted attendance system running on the office Windows laptop. Staff scan a QR code with their phone, which opens a web interface over the office WiFi, log in, and check in/out. Supports regular staff (fixed monthly salary) and event/temp staff (hourly wage), with payroll-ready hour calculations.

Single Go binary serves both the API and the built React frontend (embedded via `embed.FS`) — no separate runtime or install step needed on the target laptop.

---

## 2. Tech Stack

| Layer | Choice | Notes |
|---|---|---|
| Backend | **Go** | Compiles to a single native `.exe` — no Go runtime/compiler needed on the Windows laptop. Cross-compile from dev machine with `GOOS=windows GOARCH=amd64 go build`. |
| Database | **SQLite** via `modernc.org/sqlite` (pure Go driver) | No CGO — avoids needing a C compiler for cross-compilation. File-based, easy to back up. |
| Frontend | **React** (single app, role-based rendering) | One app serves both the staff (phone) view and the manager/admin (laptop) view based on logged-in role. Served as static build, embedded into the Go binary. |
| Windows interface | Browser tab at `localhost:PORT` | Electron shelved for now — not needed since it's just a browser tab. Revisit later if a tray icon / no-visible-terminal experience becomes a priority. |
| Deployment | Single `.exe`, auto-start via Windows Task Scheduler (on login/wake) | Handles laptop sleep/wake without a wrapper app. |
| QR code | Static, generated server-side | Points to laptop's local IP/hostname. Proximity enforced via office WiFi-only reachability, not QR rotation. |

**Architecture in one line**: Go API + embedded React build + SQLite, one binary, one process, runs entirely on the local network.

---

## 3. Users & Roles
- **Regular staff**: pre-registered accounts (kitchen, floor/service, management). Fixed monthly salary.
- **Event/temp staff**: registered on the spot by a manager. Hourly wage (pay = hours × rate).
- **Manager/Admin**: registers staff, views/edits attendance records, runs reports, fixes missed check-outs.
- **Staff**: can only check in/out and view their own attendance.

## 4. Authentication
- Login via **Employee ID + PIN**.
- Accounts created by admin/manager only (including on-the-spot registration for temp staff).
- **Session persistence**: staff stay logged in on their phone for the day (persistent session/cookie) — no need to re-enter PIN for every check-in/out event.
- **Device binding**: each employee account is bound to one registered device. First login registers the device (e.g. a stored device fingerprint/token); subsequent logins from a different device are rejected (or require admin/manager to re-bind/reset the device). Manager/admin should have a way to clear a lost/replaced device's binding.

## 5. Network & Proximity Control
- Server runs locally; phones must be on the **same office WiFi** to reach it.
- QR code is **static**, points to the laptop's local IP/hostname.
- Laptop needs a fixed/reserved local IP (or a stable hostname) so the QR doesn't break on reboot/DHCP renewal.

## 6. Check-in / Check-out Flow
- **Multiple check-in/out events per day** supported (split shifts).
- Each event timestamped and tied to employee ID.
- **Auto-close**: if a staff member forgets to check out, the system auto-closes the open session at a daily cutoff time (e.g., midnight), flagged as auto-closed for manager review.
- **Editing past records**: managers/admins can edit past check-in/out times (e.g. to fix auto-closed sessions). Every edit is written to an audit log (who edited, when, old value, new value) — see schema in Section 10.

## 7. Payroll & Reporting
- **Pay structure**:
  - Regular staff: fixed monthly salary (hours logged for attendance record, not pay calculation).
  - Event/temp staff: hourly wage — pay = hours worked × rate.
- No overtime rules in Phase 1.
- Reports:
  - Daily attendance log (check-in/out times, auto-closed flags).
  - Monthly summary (total hours per employee).
  - Payroll export for temp/event staff (hours × rate = pay due).
- Admin can view/export historical records.

## 8. Infrastructure Notes
- Windows laptop, **sleeps sometimes** — need power settings adjusted to prevent sleep during business hours, and/or graceful "server offline" messaging on the staff page if unreachable.
- **Backup**: periodic automatic backup of the SQLite file recommended (e.g., daily copy to a separate folder/drive) — payroll data lives here.
  - **Phase 1**: Windows Task Scheduler copies the SQLite file to a local destination folder on a schedule (e.g. daily).
  - **Later (cloud sync)**: install Google Drive for Desktop on the laptop (signed into a Google account), which creates a synced local folder (e.g. `G:\My Drive\`). Point the same scheduled copy task at that folder instead — Drive's desktop client handles the upload in the background. No app code or API integration needed; it's just a destination-path change.

---

## 9. Resolved Decisions
1. **Session persistence**: Staff stay logged in on their phone for the day — persistent session, no PIN re-entry per event.
2. **Rate/salary storage**: A single `pay_rate` field per employee is sufficient for Phase 1 (no rate-change history).
3. **Editing past records**: Managers/admins can edit past check-in/out times; all edits are audit-logged (who, when, old → new value).
4. **Multiple devices**: Device-bound — each employee's account is tied to the device they first log in from. Manager/admin can clear/reset the binding if a phone is lost or replaced.
5. **Backup destination**: Local folder for Phase 1 via Windows Task Scheduler; designed so it can point at a Google Drive Desktop–synced folder later with no code changes (see Section 8).
6. **QR display**: Printed and posted at the entrance. Relies on the laptop having a reserved/fixed local IP (Section 5) so the printed code doesn't go stale.

---

## 10. Suggested Initial Schema (draft — refine during build)
- `employees`: id, employee_code, name, role, pin_hash, pay_type (monthly/hourly), pay_rate, is_temp, device_id (nullable — set on first login, cleared by manager/admin to re-bind), created_at
- `attendance_events`: id, employee_id, event_type (check_in/check_out), timestamp, auto_closed (bool)
- `attendance_edit_log`: id, attendance_event_id, edited_by (employee_id of manager/admin), edited_at, field_changed, old_value, new_value
- `pay_periods` / payroll export: derived from attendance_events + employees, not necessarily a stored table in Phase 1

---

## 11. Phase 1 Scope Boundary
In scope: QR-based check-in/out, employee registration (regular + temp), attendance logs, monthly hour summaries, hourly payroll export for temp staff.
Out of scope (future phases): overtime rules, leave management, shift scheduling, notifications, cloud sync, Electron wrapper.
