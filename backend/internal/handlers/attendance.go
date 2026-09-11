package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"attendance-mgmt/backend/internal/middleware"
	"attendance-mgmt/backend/internal/models"
)

// minShiftDuration is the minimum time that must elapse after a check-in
// before a check-out is accepted. Guards against accidental double-taps
// (staff meaning to check in, immediately checking out) and duplicate entries.
const minShiftDuration = 1 * time.Hour

// Punch handles POST /api/attendance/punch.
// Toggles the caller's own open/closed state: if their last event today is
// an unmatched check_in, this records a check_out, otherwise a check_in.
// Supports multiple check-in/out events per day (split shifts).
func Punch(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emp := middleware.EmployeeFromContext(r.Context())

		var lastType, lastTimestamp string
		err := conn.QueryRow(`
			SELECT event_type, timestamp FROM attendance_events
			WHERE employee_id = ? ORDER BY timestamp DESC LIMIT 1
		`, emp.ID).Scan(&lastType, &lastTimestamp)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "failed to read attendance history", http.StatusInternalServerError)
			return
		}

		nextType := models.EventCheckIn
		if err == nil && lastType == string(models.EventCheckIn) {
			nextType = models.EventCheckOut

			checkedInAt, parseErr := time.Parse(time.RFC3339, lastTimestamp)
			if parseErr == nil {
				if remaining := minShiftDuration - time.Since(checkedInAt); remaining > 0 {
					writeJSON(w, http.StatusConflict, map[string]any{
						"error":             "too early to check out",
						"eligible_at":       checkedInAt.Add(minShiftDuration).Format(time.RFC3339),
						"remaining_seconds": int(remaining.Seconds()),
					})
					return
				}
			}
		}

		now := time.Now().Format(time.RFC3339)
		res, err := conn.Exec(`
			INSERT INTO attendance_events (employee_id, event_type, timestamp) VALUES (?, ?, ?)
		`, emp.ID, nextType, now)
		if err != nil {
			http.Error(w, "failed to record attendance event", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()

		writeJSON(w, http.StatusCreated, models.AttendanceEvent{
			ID:         id,
			EmployeeID: emp.ID,
			EventType:  nextType,
			Timestamp:  now,
		})
	}
}

// ListMyAttendance handles GET /api/attendance/me.
func ListMyAttendance(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emp := middleware.EmployeeFromContext(r.Context())
		events, err := queryEvents(conn, `WHERE employee_id = ? ORDER BY timestamp DESC`, emp.ID)
		if err != nil {
			http.Error(w, "failed to list attendance", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, events)
	}
}

// ListAttendance handles GET /api/attendance (manager/admin only).
// Optional query params: employee_id, from, to (RFC3339).
func ListAttendance(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `WHERE 1=1`
		args := []any{}

		if v := q.Get("employee_id"); v != "" {
			query += ` AND employee_id = ?`
			args = append(args, v)
		}
		if v := q.Get("from"); v != "" {
			query += ` AND timestamp >= ?`
			args = append(args, v)
		}
		if v := q.Get("to"); v != "" {
			query += ` AND timestamp <= ?`
			args = append(args, v)
		}
		query += ` ORDER BY timestamp DESC`

		events, err := queryEvents(conn, query, args...)
		if err != nil {
			http.Error(w, "failed to list attendance", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, events)
	}
}

func queryEvents(conn *sql.DB, whereOrderClause string, args ...any) ([]models.AttendanceEvent, error) {
	rows, err := conn.Query(`
		SELECT id, employee_id, event_type, timestamp, auto_closed, created_at
		FROM attendance_events `+whereOrderClause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.AttendanceEvent{}
	for rows.Next() {
		var e models.AttendanceEvent
		if err := rows.Scan(&e.ID, &e.EmployeeID, &e.EventType, &e.Timestamp, &e.AutoClosed, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

type editEventRequest struct {
	Timestamp string `json:"timestamp"`
}

// EditAttendanceEvent handles PATCH /api/attendance/{id} (manager/admin only).
// Corrects a past check-in/out time and writes an audit log entry.
func EditAttendanceEvent(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid attendance event id", http.StatusBadRequest)
			return
		}

		var req editEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Timestamp == "" {
			http.Error(w, "timestamp is required", http.StatusBadRequest)
			return
		}
		if _, err := time.Parse(time.RFC3339, req.Timestamp); err != nil {
			http.Error(w, "timestamp must be RFC3339", http.StatusBadRequest)
			return
		}

		actor := middleware.EmployeeFromContext(r.Context())

		var oldTimestamp string
		if err := conn.QueryRow(`SELECT timestamp FROM attendance_events WHERE id = ?`, id).Scan(&oldTimestamp); err != nil {
			http.Error(w, "attendance event not found", http.StatusNotFound)
			return
		}

		tx, err := conn.Begin()
		if err != nil {
			http.Error(w, "failed to start transaction", http.StatusInternalServerError)
			return
		}

		if _, err := tx.Exec(`UPDATE attendance_events SET timestamp = ?, auto_closed = 0 WHERE id = ?`, req.Timestamp, id); err != nil {
			tx.Rollback()
			http.Error(w, "failed to update attendance event", http.StatusInternalServerError)
			return
		}

		if _, err := tx.Exec(`
			INSERT INTO attendance_edit_log (attendance_event_id, edited_by, field_changed, old_value, new_value)
			VALUES (?, ?, 'timestamp', ?, ?)
		`, id, actor.ID, oldTimestamp, req.Timestamp); err != nil {
			tx.Rollback()
			http.Error(w, "failed to write audit log", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "failed to commit changes", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
