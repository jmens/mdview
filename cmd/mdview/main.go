// mdview is a small, fast Markdown viewer for Ubuntu/Linux.
//
// Usage: mdview <path/to/file.md>
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	webview "github.com/webview/webview_go"

	"github.com/sipgate/mdview/internal/doctype"
	"github.com/sipgate/mdview/internal/renderer"
	"github.com/sipgate/mdview/internal/server"
	"github.com/sipgate/mdview/internal/watcher"
)

const usage = `mdview — small, fast Markdown and PDF viewer

Usage:
  mdview <path/to/file.md|.pdf>

Options:
  -h, --help      Show this help
  -V, --version   Show version

Shortcuts (in the viewer window):
  q, Esc          Quit
  t               Toggle light/dark theme
  r               Reload
  Ctrl+F, /       Find in page (Markdown only)
  n, N            Next / previous match (Markdown only)
  j, k, ↑, ↓      Scroll
  g, G            Top / bottom
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mdview:", err)
		var ec exitCodeError
		if errors.As(err, &ec) {
			os.Exit(ec.code)
		}
		os.Exit(1)
	}
}

type exitCodeError struct {
	code int
	msg  string
}

func (e exitCodeError) Error() string { return e.msg }

func run(args []string) error {
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	showHelp := fs.Bool("help", false, "")
	fs.BoolVar(showHelp, "h", false, "")
	showVersion := fs.Bool("version", false, "")
	fs.BoolVar(showVersion, "V", false, "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	if err := fs.Parse(args); err != nil {
		return exitCodeError{code: 2, msg: err.Error()}
	}
	if *showHelp {
		fmt.Print(usage)
		return nil
	}
	if *showVersion {
		fmt.Println("mdview", version())
		return nil
	}
	if fs.NArg() != 1 {
		fmt.Fprint(os.Stderr, usage)
		return exitCodeError{code: 2, msg: "expected exactly one file path"}
	}

	docPath, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(docPath)
	if err != nil {
		return exitCodeError{code: 1, msg: err.Error()}
	}
	if info.IsDir() {
		return exitCodeError{code: 1, msg: docPath + " is a directory"}
	}

	docType := doctype.Detect(docPath)
	r := renderer.New()
	srv := server.New(docPath, docType, r)
	url, err := srv.Start()
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	defer func() { _ = srv.Stop() }()

	stopWatch, err := watcher.Watch(docPath, srv.NotifyChange, func(werr error) {
		fmt.Fprintln(os.Stderr, "mdview: watcher:", werr)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdview: live-reload disabled:", err)
	} else {
		defer func() { _ = stopWatch() }()
	}

	// WebKit/JSC emit harmless signal-handler and option-parser warnings
	// on startup and shutdown. Filter them out before initializing webview.
	if restoreStderr, ferr := startStderrFilter(); ferr == nil {
		defer restoreStderr()
	}

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("mdview — " + filepath.Base(docPath))
	w.SetSize(1024, 768, webview.HintNone)

	if err := w.Bind("openExternal", openExternal); err != nil {
		return fmt.Errorf("bind openExternal: %w", err)
	}
	if err := w.Bind("quit", func() { w.Terminate() }); err != nil {
		return fmt.Errorf("bind quit: %w", err)
	}

	w.Navigate(url)
	w.Run()
	return nil
}

func openExternal(url string) {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "mdview: xdg-open failed:", err)
		return
	}
	// Release child immediately; we don't care about its exit.
	go func() { _ = cmd.Wait() }()
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return s.Value[:7]
		}
	}
	return "(dev)"
}
