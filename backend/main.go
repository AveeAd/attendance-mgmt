package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"attendance-mgmt/backend/internal/db"
	"attendance-mgmt/backend/internal/handlers"
	"attendance-mgmt/backend/internal/middleware"
	"attendance-mgmt/backend/internal/models"
	"attendance-mgmt/backend/internal/service"
	"attendance-mgmt/backend/internal/webui"
)

func main() {
	dbPath := os.Getenv("ATTENDANCE_DB_PATH")
	if dbPath == "" {
		dbPath = "attendance.db"
	}
	port := os.Getenv("ATTENDANCE_PORT")
	if port == "" {
		port = "8080"
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer conn.Close()

	service.StartAutoCloseLoop(conn, 15*time.Minute)

	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/login", handlers.Login(conn))
	// Reveals only LAN IPs/port, needed to render the public QR check-in page.
	mux.HandleFunc("GET /api/server-info", handlers.ServerInfo(port))

	// Authenticated (any role)
	mux.Handle("POST /api/logout", middleware.RequireAuth(conn)(handlers.Logout(conn)))
	mux.Handle("POST /api/attendance/punch", middleware.RequireAuth(conn)(handlers.Punch(conn)))
	mux.Handle("GET /api/attendance/me", middleware.RequireAuth(conn)(handlers.ListMyAttendance(conn)))
	mux.Handle("POST /api/me/change-pin", middleware.RequireAuth(conn)(handlers.ChangeMyPin(conn)))

	// Manager/admin only
	managerOnly := func(h http.Handler) http.Handler {
		return middleware.RequireAuth(conn)(middleware.RequireRole(models.RoleManager, models.RoleAdmin)(h))
	}
	mux.Handle("POST /api/employees", managerOnly(handlers.CreateEmployee(conn)))
	mux.Handle("GET /api/employees", managerOnly(handlers.ListEmployees(conn)))
	mux.Handle("POST /api/employees/{id}/reset-device", managerOnly(handlers.ResetDevice(conn)))
	mux.Handle("POST /api/employees/{id}/reset-pin", managerOnly(handlers.ResetPin(conn)))
	mux.Handle("POST /api/employees/{id}/archive", managerOnly(handlers.ArchiveEmployee(conn)))
	mux.Handle("POST /api/employees/{id}/activate", managerOnly(handlers.ActivateEmployee(conn)))
	mux.Handle("DELETE /api/employees/{id}", managerOnly(handlers.DeleteEmployee(conn)))
	mux.Handle("GET /api/attendance", managerOnly(handlers.ListAttendance(conn)))
	mux.Handle("PATCH /api/attendance/{id}", managerOnly(handlers.EditAttendanceEvent(conn)))
	mux.Handle("GET /api/reports/monthly", managerOnly(handlers.MonthlySummary(conn)))
	mux.Handle("GET /api/reports/payroll", managerOnly(handlers.PayrollExport(conn)))

	// Static frontend (embedded React build). Falls back to index.html for
	// any path that isn't a real file, so React Router's client-side routes
	// (e.g. /login, /manager) work on direct navigation and refresh.
	frontend, err := webui.FS()
	if err != nil {
		log.Fatalf("failed to load embedded frontend: %v", err)
	}
	fileServer := http.FileServer(http.FS(frontend))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(frontend, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}))

	addr := ":" + port
	log.Printf("attendance-mgmt listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
