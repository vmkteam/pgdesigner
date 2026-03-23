// Package frontend embeds the built Vue frontend (dist/).
//
// Build frontend first: cd frontend && pnpm build
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded dist/ filesystem.
// Returns nil if dist/ is empty (frontend not built).
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	return sub
}
