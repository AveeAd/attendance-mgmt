package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// FS returns the embedded frontend build, rooted at the dist directory.
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
