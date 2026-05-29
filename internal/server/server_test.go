package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipgate/mdview/internal/doctype"
	"github.com/sipgate/mdview/internal/renderer"
)

func TestHandleRoot_MarkdownServesMarkdownTemplate(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("# Hello\n\nWorld."), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(mdPath, doctype.Markdown, renderer.New())
	if err := s.RenderNow(); err != nil {
		t.Fatalf("RenderNow: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	got := string(body)
	if !strings.Contains(got, "<h1") || !strings.Contains(got, "Hello") {
		t.Errorf("body missing rendered Markdown: %s", got)
	}
	if !strings.Contains(got, "/assets/app.js") {
		t.Errorf("body missing Markdown app.js reference: %s", got)
	}
}

func TestHandleRoot_PDFServesPDFTemplate(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "example.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n%%EOF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(pdfPath, doctype.PDF, renderer.New())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	got := string(body)
	if !strings.Contains(got, "/assets/pdf.js") {
		t.Errorf("body missing pdf.js script: %s", got)
	}
	if !strings.Contains(got, "/files/example.pdf") {
		t.Errorf("body missing file URL: %s", got)
	}
	if strings.Contains(got, "/assets/app.js") {
		t.Errorf("PDF body must not load Markdown app.js: %s", got)
	}
}
