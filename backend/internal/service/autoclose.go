package service

import (
	"database/sql"
	"log"
	"time"
)

// RunAutoClose finds employees whose most recent event is an unmatched
// check_in from a previous calendar day and inserts a check_out for them
// at the end of that check-in's day, flagged auto_closed for manager review.
func RunAutoClose(conn *sql.DB) error {
	rows, err := conn.Query(`
		SELECT e.employee_id, e.timestamp
		FROM attendance_events e
		WHERE e.event_type = 'check_in'
		AND e.timestamp = (
			SELECT MAX(timestamp) FROM attendance_events WHERE employee_id = e.employee_id
		)
	`)
	if err != nil {
		return err
	}

	type stale struct {
		employeeID int64
		checkInAt  time.Time
	}
	var toClose []stale

	now := time.Now()
	for rows.Next() {
		var employeeID int64
		var tsStr string
		if err := rows.Scan(&employeeID, &tsStr); err != nil {
			rows.Close()
			return err
		}
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue
		}
		if ts.Local().YearDay() != now.YearDay() || ts.Local().Year() != now.Year() {
			toClose = append(toClose, stale{employeeID: employeeID, checkInAt: ts})
		}
	}
	rows.Close()

	for _, s := range toClose {
		cutoff := time.Date(s.checkInAt.Year(), s.checkInAt.Month(), s.checkInAt.Day(), 23, 59, 59, 0, s.checkInAt.Location())
		if _, err := conn.Exec(`
			INSERT INTO attendance_events (employee_id, event_type, timestamp, auto_closed)
			VALUES (?, 'check_out', ?, 1)
		`, s.employeeID, cutoff.Format(time.RFC3339)); err != nil {
			log.Printf("auto-close: failed for employee %d: %v", s.employeeID, err)
		}
	}

	return nil
}

// StartAutoCloseLoop runs RunAutoClose immediately and then on a fixed
// interval for as long as the process is alive.
func StartAutoCloseLoop(conn *sql.DB, interval time.Duration) {
	if err := RunAutoClose(conn); err != nil {
		log.Printf("auto-close: initial run failed: %v", err)
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := RunAutoClose(conn); err != nil {
				log.Printf("auto-close: run failed: %v", err)
			}
		}
	}()
}
