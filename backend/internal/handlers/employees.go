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
