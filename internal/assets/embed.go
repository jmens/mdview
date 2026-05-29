// Package assets embeds the HTML shell, theme CSS, app JS, and vendored
// KaTeX/Mermaid files served to the webview.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed template.html pdf.html theme.css app.js pdf.js vendor
var files embed.FS

// FS returns the embedded filesystem (template.html, theme.css, app.js, vendor/...).
func FS() fs.FS { return files }

// Static returns a sub-filesystem suitable for serving as static files
// (everything except the template).
func Static() (fs.FS, error) {
	return fs.Sub(files, ".")
}

// Template parses the HTML shell template.
func Template() (*template.Template, error) {
	b, err := files.ReadFile("template.html")
	if err != nil {
		return nil, err
	}
	return template.New("shell").Parse(string(b))
}

// PDFTemplate parses the HTML shell template used for PDF documents.
func PDFTemplate() (*template.Template, error) {
	b, err := files.ReadFile("pdf.html")
	if err != nil {
		return nil, err
	}
	return template.New("pdf-shell").Parse(string(b))
}
