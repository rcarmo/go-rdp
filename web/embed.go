// Package web provides embedded static assets for the RDP HTML5 client.
//
// The dist/ directory is populated by `make build-frontend`, which copies
// HTML/manifest/PWA assets from src/ and builds JS bundles and WASM modules.
// All assets must be built before compiling the Go binary.
package web

import (
	"embed"
	"io/fs"
)

// Embed all built web assets from dist/.
//
//go:embed dist
var distFS embed.FS

// DistFS returns the filesystem containing all web assets for serving.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
