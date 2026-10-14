// Package web bundles the single-page frontend, PWA manifest and service
// worker that the Go server serves.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var assets embed.FS

// StaticFS returns the frontend file system rooted at the static directory.
func StaticFS() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
