// Package frontend bundles the built UI into the executable.
//
// The UI source lives in src/ and is built with Vite into dist/
// (npm ci && npm run build, in this folder) before go build.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Files holds the built UI with index.html at its root, so the binary needs nothing beside it.
var Files = mustSub(dist, "dist")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
