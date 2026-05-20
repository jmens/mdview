# Windows Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `mdview` build and run on Windows (alongside Linux) by isolating the two Linux-only spots in the codebase behind Go build tags and adding a CI job that proves the Windows build stays green.

**Architecture:** Today the project compiles only on Linux because `cmd/mdview/main.go` unconditionally calls `startStderrFilter` (defined in a file gated by `//go:build linux`) and `openExternal` shells out to `xdg-open`. Everything else (`webview_go`, `fsnotify`, `goldmark`, `chroma`, the `internal/...` packages) is already cross-platform. We split both Linux-specific helpers into per-OS files via build tags (`_linux.go`, `_windows.go`, `_other.go` for the noop fallback), give the URL opener a Windows implementation using `rundll32 url.dll,FileProtocolHandler`, and add a GitHub Actions workflow that builds `mdview.exe` on `windows-latest` so the cross-platform invariant is enforced. macOS comes along essentially for free as `darwin` falls under the `_other.go` build tag for the stderr filter and gets its own `open(1)` branch in the URL opener.

**Tech Stack:** Go 1.22, `github.com/webview/webview_go` (CGO → WebView2 on Windows, WebKitGTK on Linux, WKWebView on macOS), GitHub Actions.

**Runtime requirement on Windows:** Edge WebView2 Runtime. Preinstalled on Windows 11; on Windows 10 users may need the Microsoft Evergreen Bootstrapper. Documented in README, not enforced by code.

**Build requirement on Windows:** `mingw-w64` (`gcc.exe` in `PATH`) and `CGO_ENABLED=1`, because `webview_go` uses cgo. The existing `Makefile` and `scripts/pkg-config-shim.sh` are Linux-specific and stay as the Linux build path; Windows users invoke `go build` directly.

---

## File Structure

**New files:**
- `cmd/mdview/stderr_filter_other.go` — `//go:build !linux`, no-op `startStderrFilter` so `main.go` compiles on Windows/macOS.
- `cmd/mdview/open_url.go` — platform-neutral `openExternal(url string)` that delegates to per-platform `openCommand(url) *exec.Cmd`.
- `cmd/mdview/open_url_linux.go` — `openCommand` using `xdg-open`.
- `cmd/mdview/open_url_windows.go` — `openCommand` using `rundll32 url.dll,FileProtocolHandler`.
- `cmd/mdview/open_url_darwin.go` — `openCommand` using `open(1)`.
- `cmd/mdview/open_url_linux_test.go` — assertion for the Linux command.
- `cmd/mdview/open_url_windows_test.go` — assertion for the Windows command.
- `cmd/mdview/open_url_darwin_test.go` — assertion for the macOS command.
- `.github/workflows/build.yml` — matrix CI: `ubuntu-latest` + `windows-latest`.

**Modified files:**
- `cmd/mdview/main.go` — remove the inline `openExternal` (lines 136–144) and update the package comment (line 1).
- `README.md` — extend "Install" with a Windows section and tweak the opening sentence.
- `Makefile` — add a short note via `make deps` that Windows uses `go build` directly. (No new target — Linux flow stays untouched.)

---

## Task 1: Stub `startStderrFilter` for non-Linux

**Why first:** Currently `main.go:115` calls a function only defined in `stderr_filter.go` (`//go:build linux`). The Windows build fails at the link stage before we can do anything else. A no-op stub on non-Linux unblocks every subsequent task.

**Files:**
- Create: `cmd/mdview/stderr_filter_other.go`

- [ ] **Step 1: Add the non-Linux stub**

Create `cmd/mdview/stderr_filter_other.go`:

```go
//go:build !linux

package main

// startStderrFilter is a no-op on platforms other than Linux. The Linux
// implementation filters WebKitGTK/JSC noise; WebView2 (Windows) and WKWebView
// (macOS) do not emit equivalent stderr chatter, so there is nothing to filter.
func startStderrFilter() (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 2: Verify the Linux build still compiles and tests pass**

Run: `go build ./... && go test ./internal/renderer/...`
Expected: build succeeds, tests pass. The new file has `//go:build !linux`, so on Linux it is excluded and the existing `stderr_filter.go` is used — nothing changes.

- [ ] **Step 3: Validate the new file syntactically**

Run: `gofmt -l cmd/mdview/stderr_filter_other.go`
Expected: empty output (file is already formatted).

A real Windows build verification requires `mingw-w64` + `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc` cross-compile — out of scope for a local check. The Windows compile is verified end-to-end by the CI matrix added in Task 5. If the build tag in the new file is wrong (e.g. accidentally `linux` or missing), the CI job will fail with "undefined: startStderrFilter".

- [ ] **Step 4: Commit**

```bash
git add cmd/mdview/stderr_filter_other.go
git commit -m "feat: stub startStderrFilter on non-Linux platforms"
```

---

## Task 2: Extract `openExternal` into platform-specific files

**Files:**
- Modify: `cmd/mdview/main.go:136-144` (remove the inline implementation, keep the `w.Bind("openExternal", openExternal)` call on line 124)
- Create: `cmd/mdview/open_url.go`
- Create: `cmd/mdview/open_url_linux.go`
- Create: `cmd/mdview/open_url_windows.go`
- Create: `cmd/mdview/open_url_darwin.go`
- Create: `cmd/mdview/open_url_linux_test.go`
- Create: `cmd/mdview/open_url_windows_test.go`
- Create: `cmd/mdview/open_url_darwin_test.go`

- [ ] **Step 1: Write the failing Linux test**

Create `cmd/mdview/open_url_linux_test.go`:

```go
//go:build linux

package main

import "testing"

func TestOpenCommandLinux(t *testing.T) {
	cmd := openCommand("https://example.com")
	if got := cmd.Path; got == "" || cmd.Args[0] != "xdg-open" {
		t.Fatalf("expected xdg-open, got %q (args=%v)", got, cmd.Args)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "https://example.com" {
		t.Fatalf("expected args [xdg-open https://example.com], got %v", cmd.Args)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `go test ./cmd/mdview/...`
Expected: FAIL with "undefined: openCommand".

- [ ] **Step 3: Create the platform-neutral wrapper**

Create `cmd/mdview/open_url.go`:

```go
package main

import (
	"fmt"
	"os"
)

// openExternal launches the OS default handler for url and does not wait.
// The per-platform openCommand decides which executable to invoke.
func openExternal(url string) {
	cmd := openCommand(url)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "mdview: open external failed:", err)
		return
	}
	go func() { _ = cmd.Wait() }()
}
```

- [ ] **Step 4: Create the Linux implementation**

Create `cmd/mdview/open_url_linux.go`:

```go
//go:build linux

package main

import "os/exec"

func openCommand(url string) *exec.Cmd {
	return exec.Command("xdg-open", url)
}
```

- [ ] **Step 5: Remove the inline `openExternal` from `main.go`**

In `cmd/mdview/main.go`:

1. Delete lines 136–144 (the old `func openExternal(url string) { ... }`).
2. Delete the now-unused `"os/exec"` import on line 11.
3. Leave the `w.Bind("openExternal", openExternal)` call on line 124 alone — it now resolves to the new function in `open_url.go`.

- [ ] **Step 6: Run the Linux test, verify it passes**

Run: `go test ./cmd/mdview/...`
Expected: PASS.

- [ ] **Step 7: Create the Windows implementation and test**

Create `cmd/mdview/open_url_windows.go`:

```go
//go:build windows

package main

import "os/exec"

// openCommand uses rundll32 with the URL.dll FileProtocolHandler entry point.
// This avoids spawning a visible cmd.exe window (as `cmd /c start` would)
// and handles quoting safely because rundll32 takes the URL as a single arg.
func openCommand(url string) *exec.Cmd {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
}
```

Create `cmd/mdview/open_url_windows_test.go`:

```go
//go:build windows

package main

import "testing"

func TestOpenCommandWindows(t *testing.T) {
	cmd := openCommand("https://example.com")
	if cmd.Args[0] != "rundll32.exe" {
		t.Fatalf("expected rundll32.exe, got %q", cmd.Args[0])
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "url.dll,FileProtocolHandler" || cmd.Args[2] != "https://example.com" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
```

- [ ] **Step 8: Create the macOS implementation and test**

Create `cmd/mdview/open_url_darwin.go`:

```go
//go:build darwin

package main

import "os/exec"

func openCommand(url string) *exec.Cmd {
	return exec.Command("open", url)
}
```

Create `cmd/mdview/open_url_darwin_test.go`:

```go
//go:build darwin

package main

import "testing"

func TestOpenCommandDarwin(t *testing.T) {
	cmd := openCommand("https://example.com")
	if cmd.Args[0] != "open" {
		t.Fatalf("expected open, got %q", cmd.Args[0])
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "https://example.com" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
```

- [ ] **Step 9: Confirm Linux build is green end-to-end**

Run: `make build && go test ./...`
Expected: build succeeds, all tests pass.

- [ ] **Step 10: Commit**

```bash
git add cmd/mdview/open_url.go cmd/mdview/open_url_linux.go cmd/mdview/open_url_windows.go cmd/mdview/open_url_darwin.go cmd/mdview/open_url_linux_test.go cmd/mdview/open_url_windows_test.go cmd/mdview/open_url_darwin_test.go cmd/mdview/main.go
git commit -m "refactor: split openExternal into per-OS implementations"
```

---

## Task 3: Update package comment and `make deps`

**Files:**
- Modify: `cmd/mdview/main.go:1` (package comment)
- Modify: `Makefile` (`deps` target only)

- [ ] **Step 1: Update the package comment**

In `cmd/mdview/main.go` change line 1 from:

```go
// mdview is a small, fast Markdown viewer for Ubuntu/Linux.
```

to:

```go
// mdview is a small, fast Markdown viewer for Linux, macOS, and Windows.
```

- [ ] **Step 2: Extend `make deps` with the Windows hint**

In `Makefile`, replace the `deps` target with:

```makefile
deps:
	@echo "Linux (Ubuntu) build dependencies:"
	@echo "  sudo apt install libgtk-3-dev pkg-config"
	@echo "  Ubuntu 24.04: sudo apt install libwebkit2gtk-4.1-dev"
	@echo "  Ubuntu 22.04: sudo apt install libwebkit2gtk-4.0-dev"
	@echo
	@echo "Windows build dependencies (run in PowerShell):"
	@echo "  Install mingw-w64 (e.g. via 'winget install MSYS2.MSYS2' and 'pacman -S mingw-w64-x86_64-gcc')"
	@echo "  Ensure gcc.exe is on PATH, then: go build ./cmd/mdview"
	@echo "  Runtime: Edge WebView2 (preinstalled on Win11; on Win10 install the Evergreen Bootstrapper)."
```

- [ ] **Step 3: Verify the Linux build still works**

Run: `make build`
Expected: produces `./bin/mdview`.

- [ ] **Step 4: Commit**

```bash
git add cmd/mdview/main.go Makefile
git commit -m "docs: mention Windows and macOS support in package comment and make deps"
```

---

## Task 4: Document Windows install in `README.md`

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the opening sentence**

In `README.md` change line 3 from:

```
A small, fast Markdown viewer for Ubuntu/Linux. Open a file with
```

to:

```
A small, fast Markdown viewer for Linux, macOS, and Windows. Open a file with
```

- [ ] **Step 2: Add a Windows install section**

Insert a new `### Build (Windows)` subsection in `README.md` directly after the existing `### Build` block (line 34) — so the file now lists Linux and Windows side by side:

```markdown
### Build (Windows)

Runtime: Edge WebView2 is required. It ships with Windows 11 and is delivered
by Windows Update on Windows 10; if missing, install Microsoft's "Evergreen
Bootstrapper" from the official download page.

Build dependencies: `mingw-w64` with `gcc.exe` on `PATH`. Easiest path is
[MSYS2](https://www.msys2.org/):

```powershell
winget install MSYS2.MSYS2
# in the MSYS2 shell:
pacman -S mingw-w64-x86_64-gcc
# add C:\msys64\mingw64\bin to PATH
```

Then build the executable:

```powershell
$env:CGO_ENABLED = "1"
go build -o mdview.exe .\cmd\mdview
```

Run with:

```powershell
.\mdview.exe README.md
```
```

- [ ] **Step 3: Update the external-links footnote**

In `README.md` change line 56–57 from:

```
External links open in your default browser via `xdg-open`. Relative image
paths are resolved against the directory of the Markdown file.
```

to:

```
External links open in your default browser via the OS handler
(`xdg-open` on Linux, `open` on macOS, `rundll32 url.dll,FileProtocolHandler`
on Windows). Relative image paths are resolved against the directory of the
Markdown file.
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add Windows install instructions"
```

---

## Task 5: Add a cross-platform GitHub Actions workflow

**Why:** Without CI the Windows build will silently rot. A matrix build catches a missing build tag or a stray `syscall.Dup` the first time it lands on `main`.

**Files:**
- Create: `.github/workflows/build.yml`

- [ ] **Step 1: Create the workflow**

Create `.github/workflows/build.yml`:

```yaml
name: build

on:
  push:
    branches: [main]
  pull_request:

jobs:
  build:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Install Linux dependencies
        if: matrix.os == 'ubuntu-latest'
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config

      - name: Build (Linux)
        if: matrix.os == 'ubuntu-latest'
        run: make build

      - name: Build (Windows)
        if: matrix.os == 'windows-latest'
        env:
          CGO_ENABLED: "1"
        run: go build -o mdview.exe ./cmd/mdview

      - name: Test
        run: go test ./...

      - name: Upload Linux binary
        if: matrix.os == 'ubuntu-latest'
        uses: actions/upload-artifact@v4
        with:
          name: mdview-linux-amd64
          path: bin/mdview

      - name: Upload Windows binary
        if: matrix.os == 'windows-latest'
        uses: actions/upload-artifact@v4
        with:
          name: mdview-windows-amd64
          path: mdview.exe
```

> `windows-latest` ships with a working mingw-w64 toolchain (`gcc` from the GitHub Actions image), so no extra setup step is required for cgo.

- [ ] **Step 2: Smoke-check the YAML locally**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/build.yml'))" && echo OK`
Expected: prints `OK`. (If `pyyaml` isn't installed, fall back to `yamllint` or just inspect visually — the goal is to catch typos before the first push.)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build.yml
git commit -m "ci: build and test on linux and windows"
```

- [ ] **Step 4: Push the branch and watch the Actions run**

After pushing, open the Actions tab on GitHub and confirm both matrix legs go green. If the Windows leg fails:
- "undefined: startStderrFilter" → Task 1 build tag is wrong.
- cgo / C-linker error → mingw missing on the runner; pin the image with `runs-on: windows-2022` and add `shell: msys2 {0}` if necessary.
- "undefined: openCommand" → Task 2 build tag is wrong.

---

## Out of scope

- **Static distribution of `WebView2Loader.dll`** — relying on the Evergreen runtime is the standard approach and keeps the binary small.
- **MSI / Chocolatey / winget packaging** — separate concern; the artifact uploaded by the CI workflow is enough for a first release.
- **ARM64 builds** — can be added later by extending the matrix.
- **macOS CI build** — code already supports macOS via the `darwin` build tag, but verifying it requires a `macos-latest` runner; skip until someone actually needs it.
- **Removing the `pkg-config-shim.sh`** — Linux-only, harmless on Windows (Make/Bash isn't invoked there).
