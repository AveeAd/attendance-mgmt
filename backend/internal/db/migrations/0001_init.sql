-- Initial schema for Attendance Management System (Phase 1)

CREATE TABLE IF NOT EXISTS employees (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    employee_code TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('staff', 'manager', 'admin')),
    pin_hash      TEXT NOT NULL,
    pay_type      TEXT NOT NULL CHECK (pay_type IN ('monthly', 'hourly')),
    pay_rate      REAL NOT NULL DEFAULT 0,
    is_temp       INTEGER NOT NULL DEFAULT 0 CHECK (is_temp IN (0, 1)),
    device_id     TEXT,
    is_active     INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS attendance_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    employee_id  INTEGER NOT NULL REFERENCES employees(id),
    event_type   TEXT NOT NULL CHECK (event_type IN ('check_in', 'check_out')),
    timestamp    TEXT NOT NULL,
    auto_closed  INTEGER NOT NULL DEFAULT 0 CHECK (auto_closed IN (0, 1)),
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_attendance_events_employee_time
    ON attendance_events(employee_id, timestamp);

CREATE TABLE IF NOT EXISTS attendance_edit_log (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    attendance_event_id INTEGER NOT NULL REFERENCES attendance_events(id),
    edited_by           INTEGER NOT NULL REFERENCES employees(id),
    edited_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    field_changed       TEXT NOT NULL,
    old_value           TEXT,
    new_value           TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
    token       TEXT PRIMARY KEY,
    employee_id INTEGER NOT NULL REFERENCES employees(id),
    device_id   TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at  TEXT NOT NULL
);
