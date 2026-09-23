package handlers

import (
	"log"
	"net/http"
	"time"

	"attendance-mgmt/backend/internal/updater"
)

// UpdateStatus handles GET /api/update-status (manager/admin only).
func UpdateStatus(upd *updater.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, upd.Status())
	}
}

// CheckUpdate handles POST /api/update/check (manager/admin only). Runs an
// immediate check instead of waiting for the background loop's next tick.
func CheckUpdate(upd *updater.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, upd.CheckNow())
	}
}

// ApplyUpdate handles POST /api/update/apply (manager/admin only).
// Responds first, then applies the update shortly after so the response
// has a chance to flush before the server restarts.
func ApplyUpdate(upd *updater.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !upd.Status().Available {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no update ready to apply"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "restarting"})

		time.AfterFunc(500*time.Millisecond, func() {
			if err := upd.Apply(); err != nil {
				log.Printf("apply update: %v", err)
			}
		})
	}
}
