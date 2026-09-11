package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"attendance-mgmt/backend/internal/auth"
	"attendance-mgmt/backend/internal/models"
)

type createEmployeeRequest struct {
	EmployeeCode string         `json:"employee_code"`
	Name         string         `json:"name"`
	Role         models.Role    `json:"role"`
	Pin          string         `json:"pin"`
	PayType      models.PayType `json:"pay_type"`
	PayRate      float64        `json:"pay_rate"`
	IsTemp       bool           `json:"is_temp"`
}

// CreateEmployee handles POST /api/employees (manager/admin only).
// Covers both regular-staff pre-registration and on-the-spot temp-staff registration.
func CreateEmployee(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createEmployeeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.EmployeeCode == "" || req.Name == "" || req.Pin == "" {
			http.Error(w, "employee_code, name, and pin are required", http.StatusBadRequest)
			return
		}
		req.EmployeeCode = strings.ToUpper(strings.TrimSpace(req.EmployeeCode))
		if req.Role == "" {
			req.Role = models.RoleStaff
		}
		if req.PayType == "" {
			req.PayType = models.PayTypeHourly
		}

		hash, err := auth.HashPIN(req.Pin)
		if err != nil {
			http.Error(w, "failed to hash pin", http.StatusInternalServerError)
			return
		}

		res, err := conn.Exec(`
			INSERT INTO employees (employee_code, name, role, pin_hash, pay_type, pay_rate, is_temp)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, req.EmployeeCode, req.Name, req.Role, hash, req.PayType, req.PayRate, req.IsTemp)
		if err != nil {
			http.Error(w, "failed to create employee (code may already exist)", http.StatusConflict)
			return
		}
		id, _ := res.LastInsertId()

		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
	}
}

// ListEmployees handles GET /api/employees (manager/admin only).
func ListEmployees(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := conn.Query(`
			SELECT id, employee_code, name, role, pay_type, pay_rate, is_temp, device_id, is_active, created_at
			FROM employees ORDER BY name
		`)
		if err != nil {
			http.Error(w, "failed to list employees", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := []models.Employee{}
		for rows.Next() {
			var e models.Employee
			var deviceID sql.NullString
			if err := rows.Scan(&e.ID, &e.EmployeeCode, &e.Name, &e.Role, &e.PayType, &e.PayRate,
				&e.IsTemp, &deviceID, &e.IsActive, &e.CreatedAt); err != nil {
				http.Error(w, "failed to read employees", http.StatusInternalServerError)
				return
			}
			if deviceID.Valid {
				e.DeviceID = &deviceID.String
			}
			out = append(out, e)
		}

		writeJSON(w, http.StatusOK, out)
	}
}

// ResetDevice handles POST /api/employees/{id}/reset-device (manager/admin only).
// Clears the device binding so the employee can log in from a new phone.
func ResetDevice(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid employee id", http.StatusBadRequest)
			return
		}

		if _, err := conn.Exec(`UPDATE employees SET device_id = NULL WHERE id = ?`, id); err != nil {
			http.Error(w, "failed to reset device", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ArchiveEmployee handles POST /api/employees/{id}/archive (manager/admin only).
// Soft-deletes the employee: they can no longer log in, but their
// attendance history and payroll figures are preserved.
func ArchiveEmployee(conn *sql.DB) http.HandlerFunc {
	return setActive(conn, false)
}

// ActivateEmployee handles POST /api/employees/{id}/activate (manager/admin only).
// Reverses an archive, restoring the employee's ability to log in.
func ActivateEmployee(conn *sql.DB) http.HandlerFunc {
	return setActive(conn, true)
}

func setActive(conn *sql.DB, active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid employee id", http.StatusBadRequest)
			return
		}

		res, err := conn.Exec(`UPDATE employees SET is_active = ? WHERE id = ?`, active, id)
		if err != nil {
			http.Error(w, "failed to update employee", http.StatusInternalServerError)
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			http.Error(w, "employee not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteEmployee handles DELETE /api/employees/{id} (manager/admin only).
// Permanently removes an employee, but only if they have no attendance
// history — deleting one with history would corrupt past payroll figures
// and audit trails. Use ArchiveEmployee instead for employees who have
// actually worked; this is for cleaning up mis-registered/never-used ones.
func DeleteEmployee(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid employee id", http.StatusBadRequest)
			return
		}

		var eventCount, editCount int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM attendance_events WHERE employee_id = ?`, id).Scan(&eventCount); err != nil {
			http.Error(w, "failed to check attendance history", http.StatusInternalServerError)
			return
		}
		if err := conn.QueryRow(`SELECT COUNT(*) FROM attendance_edit_log WHERE edited_by = ?`, id).Scan(&editCount); err != nil {
			http.Error(w, "failed to check edit history", http.StatusInternalServerError)
			return
		}
		if eventCount > 0 || editCount > 0 {
			http.Error(w, "cannot delete an employee with attendance history; archive them instead", http.StatusConflict)
			return
		}

		tx, err := conn.Begin()
		if err != nil {
			http.Error(w, "failed to start transaction", http.StatusInternalServerError)
			return
		}

		if _, err := tx.Exec(`DELETE FROM sessions WHERE employee_id = ?`, id); err != nil {
			tx.Rollback()
			http.Error(w, "failed to delete employee", http.StatusInternalServerError)
			return
		}

		res, err := tx.Exec(`DELETE FROM employees WHERE id = ?`, id)
		if err != nil {
			tx.Rollback()
			http.Error(w, "failed to delete employee", http.StatusInternalServerError)
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			tx.Rollback()
			http.Error(w, "employee not found", http.StatusNotFound)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "failed to commit changes", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
