package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"attendance-mgmt/backend/internal/auth"
	"attendance-mgmt/backend/internal/middleware"
)

type changePinRequest struct {
	CurrentPin string `json:"current_pin"`
	NewPin     string `json:"new_pin"`
}

// ChangeMyPin handles POST /api/me/change-pin.
// Any authenticated employee can change their own PIN by proving they know
// the current one.
func ChangeMyPin(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emp := middleware.EmployeeFromContext(r.Context())

		var req changePinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPin == "" || req.NewPin == "" {
			http.Error(w, "current_pin and new_pin are required", http.StatusBadRequest)
			return
		}

		var pinHash string
		if err := conn.QueryRow(`SELECT pin_hash FROM employees WHERE id = ?`, emp.ID).Scan(&pinHash); err != nil {
			http.Error(w, "employee not found", http.StatusNotFound)
			return
		}

		if !auth.VerifyPIN(pinHash, req.CurrentPin) {
			http.Error(w, "current PIN is incorrect", http.StatusUnauthorized)
			return
		}

		newHash, err := auth.HashPIN(req.NewPin)
		if err != nil {
			http.Error(w, "failed to hash pin", http.StatusInternalServerError)
			return
		}

		if _, err := conn.Exec(`UPDATE employees SET pin_hash = ? WHERE id = ?`, newHash, emp.ID); err != nil {
			http.Error(w, "failed to update pin", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type resetPinRequest struct {
	NewPin string `json:"new_pin"`
}

// ResetPin handles POST /api/employees/{id}/reset-pin (manager/admin only).
// Sets a new PIN directly without needing to know the old one — e.g. when
// an employee forgets theirs.
func ResetPin(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid employee id", http.StatusBadRequest)
			return
		}

		var req resetPinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPin == "" {
			http.Error(w, "new_pin is required", http.StatusBadRequest)
			return
		}

		newHash, err := auth.HashPIN(req.NewPin)
		if err != nil {
			http.Error(w, "failed to hash pin", http.StatusInternalServerError)
			return
		}

		res, err := conn.Exec(`UPDATE employees SET pin_hash = ? WHERE id = ?`, newHash, id)
		if err != nil {
			http.Error(w, "failed to update pin", http.StatusInternalServerError)
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			http.Error(w, "employee not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
