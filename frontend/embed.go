// Package frontend bundles the static UI into the executable.
package frontend

import "embed"

// Files holds index.html, js/ and vendor/, so the binary needs nothing beside it.
//
//go:embed index.html js vendor
var Files embed.FS
