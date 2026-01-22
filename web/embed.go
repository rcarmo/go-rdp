// Package web provides embedded static assets for the RDP HTML5 client.
package web

import (
	"embed"
	"io/fs"
)

// Embed HTML files from src/ (tracked in git)
//
//go:embed src/*.html
var srcFS embed.FS

// Embed JS/WASM from dist/ (built artifacts)
// Note: dist/js must exist with built assets for this to work
//
//go:embed dist/js
var distFS embed.FS

// DistFS returns a merged filesystem with HTML from src/ and JS from dist/.
// This ensures the server works after a fresh clone + build.
func DistFS() (fs.FS, error) {
	srcSub, err := fs.Sub(srcFS, "src")
	if err != nil {
		return nil, err
	}
	distSub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return &mergedFS{primary: srcSub, secondary: distSub}, nil
}

// mergedFS serves files from primary first, falling back to secondary.
type mergedFS struct {
	primary   fs.FS
	secondary fs.FS
}

func (m *mergedFS) Open(name string) (fs.File, error) {
	f, err := m.primary.Open(name)
	if err == nil {
		return f, nil
	}
	return m.secondary.Open(name)
}

func (m *mergedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	// Merge directory entries from both filesystems
	entries := make(map[string]fs.DirEntry)

	if rd, ok := m.primary.(fs.ReadDirFS); ok {
		if dirEntries, err := rd.ReadDir(name); err == nil {
			for _, e := range dirEntries {
				entries[e.Name()] = e
			}
		}
	}

	if rd, ok := m.secondary.(fs.ReadDirFS); ok {
		if dirEntries, err := rd.ReadDir(name); err == nil {
			for _, e := range dirEntries {
				if _, exists := entries[e.Name()]; !exists {
					entries[e.Name()] = e
				}
			}
		}
	}

	result := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, e)
	}
	return result, nil
}
