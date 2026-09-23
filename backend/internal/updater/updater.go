// Package updater implements in-app self-updating: a background loop
// checks GitHub Releases for a newer version, and an explicit Apply()
// (triggered only by a manager clicking "Restart to apply update" in the
// dashboard) downloads and atomically swaps in the new binary, then
// restarts the process.
package updater

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"attendance-mgmt/backend/internal/version"
)

const checkInterval = 6 * time.Hour

// Status is the updater's current view of whether a newer release exists.
// It intentionally does not persist across restarts — re-checking is cheap.
type Status struct {
	CurrentVersion string    `json:"current_version"`
	Available      bool      `json:"available"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
	Error          string    `json:"error,omitempty"`
}

// Checker owns the background update-check loop and the ability to apply
// an update on demand.
type Checker struct {
	slug string // "owner/repo" on GitHub
	srv  *http.Server

	mu     sync.RWMutex
	status Status
}

// New creates a Checker. srv is shut down gracefully before the process
// restarts itself after a successful Apply.
func New(slug string, srv *http.Server) *Checker {
	return &Checker{
		slug: slug,
		srv:  srv,
		status: Status{
			CurrentVersion: version.Version,
		},
	}
}

// Status returns a snapshot of the current update-check state.
func (c *Checker) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// StartBackgroundLoop checks for updates shortly after startup and then on
// a fixed interval, until ctx is cancelled. It never applies anything —
// that only happens via an explicit Apply() call. No-ops entirely for
// unreleased/dev builds (version.Version == "dev"), so local development
// never makes a network call to GitHub.
func (c *Checker) StartBackgroundLoop(ctx context.Context) {
	if version.Version == "dev" {
		return
	}

	select {
	case <-time.After(1 * time.Minute):
	case <-ctx.Done():
		return
	}
	c.check()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.check()
		}
	}
}

// CheckNow runs an update check immediately, bypassing the background
// loop's interval, and returns the resulting status. Used by the manual
// "Check for update" button. Unlike StartBackgroundLoop, this runs even for
// dev builds — it'll just surface a "parse current version" error, which is
// an accurate answer for a build with no real version.
func (c *Checker) CheckNow() Status {
	c.check()
	return c.Status()
}

func (c *Checker) check() {
	now := time.Now()

	current, err := semver.ParseTolerant(version.Version)
	if err != nil {
		c.setStatus(func(s *Status) {
			s.CheckedAt = now
			s.Error = fmt.Sprintf("parse current version: %v", err)
		})
		return
	}

	latest, found, err := selfupdate.DetectLatest(c.slug)
	if err != nil {
		c.setStatus(func(s *Status) {
			s.CheckedAt = now
			s.Error = err.Error()
		})
		return
	}

	c.setStatus(func(s *Status) {
		s.CheckedAt = now
		s.Error = ""
		if !found {
			s.Available = false
			return
		}
		s.Available = latest.Version.GT(current)
		s.LatestVersion = "v" + latest.Version.String()
	})
}

func (c *Checker) setStatus(mutate func(*Status)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	mutate(&c.status)
}

// Apply downloads the latest release, atomically replaces the running
// executable, spawns a new process of the freshly-written binary, and
// then gracefully shuts down and exits — handing control to the new
// process. Only called in response to an explicit manager action.
//
// This spawns a new process rather than exec-replacing the current one
// (syscall.Exec) because Windows has no fork/exec model; the
// spawn-then-exit pattern works uniformly across Windows, macOS, and
// Linux.
func (c *Checker) Apply() error {
	current, err := semver.ParseTolerant(version.Version)
	if err != nil {
		return fmt.Errorf("parse current version: %w", err)
	}

	if _, err := selfupdate.UpdateSelf(current, c.slug); err != nil {
		return fmt.Errorf("download/replace binary: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start updated process: %w", err)
	}

	log.Printf("update applied, restarting as pid %d", cmd.Process.Pid)

	if c.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown during update: %v", err)
		}
	}

	os.Exit(0)
	return nil // unreachable
}
