package main

import (
	"context"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"attendance-mgmt/backend/internal/db"
	"attendance-mgmt/backend/internal/handlers"
	"attendance-mgmt/backend/internal/middleware"
	"attendance-mgmt/backend/internal/models"
	"attendance-mgmt/backend/internal/service"
	"attendance-mgmt/backend/internal/updater"
	"attendance-mgmt/backend/internal/webui"
)

// updateRepoSlug is the "owner/repo" GitHub slug the in-app updater checks
// for new releases. Update this if the repo is ever renamed/transferred.
const updateRepoSlug = "AveeAd/attendance-mgmt"

func main() {
	dbPath := os.Getenv("ATTENDANCE_DB_PATH")
	if dbPath == "" {
		dbPath = "attendance.db"
	}

	// The Windows build runs with no console attached (see build tags/
	// -H=windowsgui), so log output needs somewhere to go besides stderr.
	// Written next to the database, alongside the db file itself.
	logPath := filepath.Join(filepath.Dir(dbPath), "attendance.log")
	if logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	} else {
		log.Printf("could not open log file %s: %v", logPath, err)
	}

	listener, port, err := listen(os.Getenv("ATTENDANCE_PORT"))
	if err != nil {
		log.Fatalf("failed to bind port: %v", err)
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

	srv := &http.Server{Handler: mux}

	// The updater needs a handle to the server so it can shut it down
	// gracefully before restarting the process after applying an update.
	upd := updater.New(updateRepoSlug, srv)
	mux.Handle("GET /api/update-status", managerOnly(handlers.UpdateStatus(upd)))
	mux.Handle("POST /api/update/check", managerOnly(handlers.CheckUpdate(upd)))
	mux.Handle("POST /api/update/apply", managerOnly(handlers.ApplyUpdate(upd)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go upd.StartBackgroundLoop(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("attendance-mgmt listening on :%s (db: %s)", port, dbPath)
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

// listen picks the port to bind to. An explicit ATTENDANCE_PORT is honored
// as-is (fail loudly if it's unavailable — the operator asked for it). With
// no override, 8080 is common enough to already be taken by something else
// on the machine, so we try it first and fall back to letting the OS assign
// any free port rather than refusing to start.
func listen(explicitPort string) (net.Listener, string, error) {
	if explicitPort != "" {
		ln, err := net.Listen("tcp", ":"+explicitPort)
		if err != nil {
			return nil, "", err
		}
		return ln, explicitPort, nil
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Printf("port 8080 unavailable (%v), asking OS for a free port instead", err)
		ln, err = net.Listen("tcp", ":0")
		if err != nil {
			return nil, "", err
		}
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	return ln, port, nil
}
