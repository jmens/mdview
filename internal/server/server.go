// Package server runs a local HTTP server that the webview connects to.
// It serves embedded assets, the rendered document, and live-reload events.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipgate/mdview/internal/assets"
	"github.com/sipgate/mdview/internal/doctype"
	"github.com/sipgate/mdview/internal/renderer"
)

// Server serves the mdview UI and pushes live-reload events.
type Server struct {
	docPath  string
	docDir   string
	docType  doctype.Type
	renderer *renderer.Renderer

	mu        sync.RWMutex
	cached    renderer.Result
	lastError string

	subscribersMu sync.Mutex
	subscribers   map[chan struct{}]struct{}

	httpServer *http.Server
	listener   net.Listener
}

// New constructs a Server for the given document file.
// docType selects which HTML shell to serve at `/`.
func New(docPath string, docType doctype.Type, r *renderer.Renderer) *Server {
	return &Server{
		docPath:     docPath,
		docDir:      filepath.Dir(docPath),
		docType:     docType,
		renderer:    r,
		subscribers: make(map[chan struct{}]struct{}),
	}
}

// Start renders the file once, binds a loopback listener on a free port,
// and starts serving in a goroutine. The returned URL is the address the
// webview should navigate to.
func (s *Server) Start() (string, error) {
	if s.docType == doctype.Markdown {
		if err := s.RenderNow(); err != nil {
			// We still want to come up so the user sees the error in the UI.
			s.setError(err.Error())
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.Handle("/assets/", http.StripPrefix("/assets/", s.assetsHandler()))
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(s.docDir))))
	mux.HandleFunc("/api/content", s.handleContent)
	mux.HandleFunc("/api/events", s.handleEvents)

	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "mdview: server error: %v\n", err)
		}
	}()

	return "http://" + ln.Addr().String() + "/", nil
}

// Stop shuts the HTTP server down with a short grace period.
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// RenderNow re-renders the document and stores the result. Safe to call
// concurrently.
func (s *Server) RenderNow() error {
	res, err := s.renderer.RenderFile(s.docPath)
	if err != nil {
		s.setError(err.Error())
		return err
	}
	s.mu.Lock()
	s.cached = res
	s.lastError = ""
	s.mu.Unlock()
	return nil
}

// NotifyChange re-renders and broadcasts a reload event to all subscribers.
func (s *Server) NotifyChange() {
	if s.docType == doctype.Markdown {
		if err := s.RenderNow(); err != nil {
			// Error is already cached; clients render it inline.
		}
	}
	s.broadcast()
}

func (s *Server) setError(msg string) {
	s.mu.Lock()
	s.lastError = msg
	s.mu.Unlock()
}

func (s *Server) snapshot() (renderer.Result, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cached, s.lastError
}

// ---- HTTP handlers --------------------------------------------------------

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	switch s.docType {
	case doctype.PDF:
		s.renderPDFShell(w)
	default:
		s.renderMarkdownShell(w)
	}
}

func (s *Server) renderMarkdownShell(w http.ResponseWriter) {
	tmpl, err := assets.Template()
	if err != nil {
		http.Error(w, "template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	res, renderErr := s.snapshot()
	body := res.HTML
	if renderErr != "" {
		body = `<div class="render-error">` + htmlEscape(renderErr) + `</div>` + body
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, struct {
		Title string
		Body  template.HTML
	}{
		Title: titleOrDefault(res.Title, filepath.Base(s.docPath)),
		Body:  template.HTML(body), //nolint:gosec // body comes from goldmark renderer; we control input
	})
}

func (s *Server) renderPDFShell(w http.ResponseWriter) {
	tmpl, err := assets.PDFTemplate()
	if err != nil {
		http.Error(w, "template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, struct {
		Title   string
		FileURL string
	}{
		Title:   filepath.Base(s.docPath),
		FileURL: "/files/" + filepath.Base(s.docPath),
	})
}

func (s *Server) handleContent(w http.ResponseWriter, _ *http.Request) {
	res, renderErr := s.snapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"html":  res.HTML,
		"title": titleOrDefault(res.Title, filepath.Base(s.docPath)),
		"error": renderErr,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 1)
	s.subscribersMu.Lock()
	s.subscribers[ch] = struct{}{}
	s.subscribersMu.Unlock()
	defer func() {
		s.subscribersMu.Lock()
		delete(s.subscribers, ch)
		s.subscribersMu.Unlock()
	}()

	// Initial comment to flush headers.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "event: reload\ndata: 1\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) broadcast() {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	for ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ---- Static assets --------------------------------------------------------

func (s *Server) assetsHandler() http.Handler {
	sub, err := fs.Sub(assets.FS(), ".")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "assets: "+err.Error(), http.StatusInternalServerError)
		})
	}
	return http.FileServer(http.FS(sub))
}

// ---- Helpers --------------------------------------------------------------

func titleOrDefault(t, fallback string) string {
	if strings.TrimSpace(t) == "" {
		return fallback
	}
	return t
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}
