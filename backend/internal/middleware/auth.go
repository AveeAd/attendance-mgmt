package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"attendance-mgmt/backend/internal/models"
)

type ctxKey string

const employeeCtxKey ctxKey = "employee"

// RequireAuth validates the bearer session token, loads the associated
// employee, and attaches it to the request context.
func RequireAuth(conn *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				http.Error(w, "missing session token", http.StatusUnauthorized)
				return
			}

			var emp models.Employee
			var deviceID string
			var expiresAt string
			row := conn.QueryRow(`
				SELECT e.id, e.employee_code, e.name, e.role, e.pay_type, e.pay_rate,
				       e.is_temp, e.device_id, e.is_active, s.device_id, s.expires_at
				FROM sessions s
				JOIN employees e ON e.id = s.employee_id
				WHERE s.token = ?
			`, token)
			if err := row.Scan(&emp.ID, &emp.EmployeeCode, &emp.Name, &emp.Role, &emp.PayType,
				&emp.PayRate, &emp.IsTemp, &emp.DeviceID, &emp.IsActive, &deviceID, &expiresAt); err != nil {
				http.Error(w, "invalid session", http.StatusUnauthorized)
				return
			}

			expires, err := time.Parse(time.RFC3339, expiresAt)
			if err != nil || time.Now().After(expires) {
				http.Error(w, "session expired", http.StatusUnauthorized)
				return
			}

			if !emp.IsActive {
				http.Error(w, "account disabled", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), employeeCtxKey, &emp)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restricts access to the given roles. Must run after RequireAuth.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			emp := EmployeeFromContext(r.Context())
			if emp == nil || !allowed[emp.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func EmployeeFromContext(ctx context.Context) *models.Employee {
	emp, _ := ctx.Value(employeeCtxKey).(*models.Employee)
	return emp
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
