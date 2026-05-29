// Package doctype determines how to render a file based on its extension.
package doctype

import (
	"path/filepath"
	"strings"
)

// Type enumerates the document kinds mdview can render.
type Type int

const (
	Markdown Type = iota
	PDF
)

// String returns a lowercase identifier suitable for logging.
func (t Type) String() string {
	switch t {
	case Markdown:
		return "markdown"
	case PDF:
		return "pdf"
	default:
		return "unknown"
	}
}

// Detect returns the type implied by the file extension of path.
// Unknown extensions default to Markdown so existing behavior is preserved.
func Detect(path string) Type {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".pdf" {
		return PDF
	}
	return Markdown
}
