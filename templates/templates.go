// Package templates holds the text that cumin-trace init writes into a
// product repository. No embedded path holds .claude, CLAUDE.md, or
// .github/workflows; the package internal/scaffold names where each file goes.
package templates

import "embed"

// Files is the embedded text. A path is relative to this directory.
//
//go:embed settings.toml docs
var Files embed.FS
