package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"attendance-mgmt/backend/internal/auth"
	"attendance-mgmt/backend/internal/models"
)

const sessionTTL = 16 * time.Hour

type loginRequest struct {
	EmployeeCode string `json:"employee_code"`
	Pin          string `json:"pin"`
	DeviceID     string `json:"device_id"`
}

type loginResponse struct {
	Token    string          `json:"token"`
	Employee models.Employee `json:"employee"`
}

// Login handles POST /api/login.
// Employee ID + PIN auth. Binds the account to device_id on first login;
// subsequent logins must present the same device_id, unless a
// manager/admin has cleared the binding (Employee.DeviceID == nil).
// Admin accounts are exempt from device binding altogether.
func Login(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.EmployeeCode == "" || req.Pin == "" || req.DeviceID == "" {
			http.Error(w, "employee_code, pin, and device_id are required", http.StatusBadRequest)
			return
		}

		var emp models.Employee
		var pinHash string
		var deviceID sql.NullString
		row := conn.QueryRow(`
			SELECT id, employee_code, name, role, pin_hash, pay_type, pay_rate, is_temp, device_id, is_active
			FROM employees WHERE employee_code = ?
		`, req.EmployeeCode)
		if err := row.Scan(&emp.ID, &emp.EmployeeCode, &emp.Name, &emp.Role, &pinHash, &emp.PayType,
			&emp.PayRate, &emp.IsTemp, &deviceID, &emp.IsActive); err != nil {
			http.Error(w, "invalid employee code or pin", http.StatusUnauthorized)
			return
		}

		if !emp.IsActive {
			http.Error(w, "account disabled", http.StatusForbidden)
			return
		}

		if !auth.VerifyPIN(pinHash, req.Pin) {
			http.Error(w, "invalid employee code or pin", http.StatusUnauthorized)
			return
		}

		// Admins bypass device binding entirely — they need to be able to log
		// in from any device, including to reset another employee's binding.
		if emp.Role != models.RoleAdmin {
			if deviceID.Valid && deviceID.String != "" {
				if deviceID.String != req.DeviceID {
					http.Error(w, "this account is registered to a different device; ask a manager to reset it", http.StatusForbidden)
					return
				}
			} else {
				if _, err := conn.Exec(`UPDATE employees SET device_id = ? WHERE id = ?`, req.DeviceID, emp.ID); err != nil {
					http.Error(w, "failed to register device", http.StatusInternalServerError)
					return
				}
			}
			emp.DeviceID = &req.DeviceID
		} else if deviceID.Valid {
			emp.DeviceID = &deviceID.String
		}

		token, err := auth.NewToken()
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		expiresAt := time.Now().Add(sessionTTL)
		if _, err := conn.Exec(`
			INSERT INTO sessions (token, employee_id, device_id, expires_at) VALUES (?, ?, ?, ?)
		`, token, emp.ID, req.DeviceID, expiresAt.Format(time.RFC3339)); err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, loginResponse{Token: token, Employee: emp})
	}
}

// Logout handles POST /api/logout.
func Logout(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token != "" {
			conn.Exec(`DELETE FROM sessions WHERE token = ?`, token)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}
