package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"attendance-mgmt/backend/internal/models"
)

type monthlySummaryRow struct {
	EmployeeID   int64          `json:"employee_id"`
	EmployeeCode string         `json:"employee_code"`
	Name         string         `json:"name"`
	PayType      models.PayType `json:"pay_type"`
	PayRate      float64        `json:"pay_rate"`
	TotalHours   float64        `json:"total_hours"`
	PayDue       *float64       `json:"pay_due,omitempty"`
}

// MonthlySummary handles GET /api/reports/monthly?year=YYYY&month=MM (manager/admin only).
// Returns total hours per employee, plus computed pay for hourly (temp) staff.
func MonthlySummary(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, end, err := monthRange(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		rows, err := conn.Query(`
			SELECT id, employee_code, name, pay_type, pay_rate FROM employees ORDER BY name
		`)
		if err != nil {
			http.Error(w, "failed to list employees", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := []monthlySummaryRow{}
		for rows.Next() {
			var e monthlySummaryRow
			if err := rows.Scan(&e.EmployeeID, &e.EmployeeCode, &e.Name, &e.PayType, &e.PayRate); err != nil {
				http.Error(w, "failed to read employees", http.StatusInternalServerError)
				return
			}

			hours, err := totalHoursWorked(conn, e.EmployeeID, start, end)
			if err != nil {
				http.Error(w, "failed to compute hours", http.StatusInternalServerError)
				return
			}
			e.TotalHours = hours

			if e.PayType == models.PayTypeHourly {
				pay := hours * e.PayRate
				e.PayDue = &pay
			}

			out = append(out, e)
		}

		writeJSON(w, http.StatusOK, out)
	}
}

// PayrollExport handles GET /api/reports/payroll?year=YYYY&month=MM (manager/admin only).
// Hours x rate = pay due, for hourly/temp staff only.
func PayrollExport(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, end, err := monthRange(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		rows, err := conn.Query(`
			SELECT id, employee_code, name, pay_rate FROM employees WHERE pay_type = 'hourly' ORDER BY name
		`)
		if err != nil {
			http.Error(w, "failed to list employees", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := []monthlySummaryRow{}
		for rows.Next() {
			var e monthlySummaryRow
			if err := rows.Scan(&e.EmployeeID, &e.EmployeeCode, &e.Name, &e.PayRate); err != nil {
				http.Error(w, "failed to read employees", http.StatusInternalServerError)
				return
			}
			e.PayType = models.PayTypeHourly

			hours, err := totalHoursWorked(conn, e.EmployeeID, start, end)
			if err != nil {
				http.Error(w, "failed to compute hours", http.StatusInternalServerError)
				return
			}
			e.TotalHours = hours
			pay := hours * e.PayRate
			e.PayDue = &pay

			out = append(out, e)
		}

		writeJSON(w, http.StatusOK, out)
	}
}

func monthRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if v := q.Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("invalid year query parameter")
		}
		year = parsed
	}
	if v := q.Get("month"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 1 || parsed > 12 {
			return time.Time{}, time.Time{}, errors.New("invalid month query parameter")
		}
		month = parsed
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}

func totalHoursWorked(conn *sql.DB, employeeID int64, start, end time.Time) (float64, error) {
	rows, err := conn.Query(`
		SELECT event_type, timestamp FROM attendance_events
		WHERE employee_id = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp ASC
	`, employeeID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total float64
	var openCheckIn *time.Time
	for rows.Next() {
		var eventType, tsStr string
		if err := rows.Scan(&eventType, &tsStr); err != nil {
			return 0, err
		}
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue
		}
		if eventType == string(models.EventCheckIn) {
			openCheckIn = &ts
		} else if eventType == string(models.EventCheckOut) && openCheckIn != nil {
			total += ts.Sub(*openCheckIn).Hours()
			openCheckIn = nil
		}
	}
	return total, nil
}
