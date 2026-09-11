// Package version holds the build-time version string, set via
// -ldflags "-X attendance-mgmt/backend/internal/version.Version=vX.Y.Z".
// Left at "dev" for local/unreleased builds (go run, dev.sh, plain
// build.sh) — the updater treats "dev" as "never check for updates".
package version

var Version = "dev"
