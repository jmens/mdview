// Package renderer converts Markdown files into HTML fragments suitable
// for embedding into the mdview HTML shell.
package renderer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Result is the output of a render pass.
type Result struct {
	HTML  string
	Title string
}

// Renderer turns Markdown files into HTML fragments.
type Renderer struct {
	md goldmark.Markdown
}

// New constructs a Renderer with CommonMark + GFM + highlighting enabled.
// Math (KaTeX) and Mermaid are passed through and rendered client-side.
func New() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			emoji.Emoji,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
					chromahtml.WithLineNumbers(false),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	return &Renderer{md: md}
}

// RenderFile reads the file at path and renders it to HTML.
// Relative image paths are kept as-is; the HTML shell sets a <base>
// tag so the browser resolves them against the file's directory.
func (r *Renderer) RenderFile(path string) (Result, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", path, err)
	}
	return r.Render(src, filepath.Base(path))
}

// Render converts raw Markdown bytes into an HTML fragment.
// title is used as a default document title if no H1 is found.
func (r *Renderer) Render(src []byte, title string) (Result, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(src, &buf); err != nil {
		return Result{}, fmt.Errorf("convert markdown: %w", err)
	}
	return Result{
		HTML:  buf.String(),
		Title: extractTitle(src, title),
	}, nil
}

// extractTitle returns the first ATX H1 ("# ...") if present,
// otherwise the fallback.
func extractTitle(src []byte, fallback string) string {
	for _, line := range bytes.Split(src, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) >= 2 && trimmed[0] == '#' && trimmed[1] == ' ' {
			return string(bytes.TrimSpace(trimmed[1:]))
		}
	}
	return fallback
}
