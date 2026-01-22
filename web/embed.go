// Package web provides embedded static assets for the RDP HTML5 client.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// DistFS returns a filesystem rooted at the dist/ directory.
// This strips the "dist" prefix so files are served from root.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
