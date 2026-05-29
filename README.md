# mdview

A small, fast Markdown and PDF viewer for Ubuntu/Linux. Open a file with
`mdview path/to/file.md` (or `.pdf`) and a native window pops up rendering it.

## Features

- **Markdown:** CommonMark + GitHub Flavored Markdown (tables, task lists,
  strikethrough, autolinks)
- **Markdown:** Syntax-highlighted code blocks (Chroma, ~200 languages)
- **Markdown:** Math via KaTeX (`$inline$`, `$$block$$`)
- **Markdown:** Mermaid diagrams (` ```mermaid ` blocks)
- **Markdown:** In-page search (`Ctrl+F` or `/`)
- **PDF:** Page-by-page rendering via PDF.js (vendored, fully offline)
- Light / dark theme (follows OS by default, toggle with `t`)
- Live reload on file changes
- Single binary, embedded assets, fully offline

## Install

### Build dependencies (Ubuntu)

```sh
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev pkg-config
```

On Ubuntu 22.04 the packages are `libwebkit2gtk-4.0-dev` (the Go bindings
detect either flavor at build time).

### Build

```sh
make build               # produces ./bin/mdview
sudo make install        # installs to /usr/local/bin/mdview
```

## Usage

```sh
mdview README.md
```

### Shortcuts

| Key             | Action                       |
| --------------- | ---------------------------- |
| `q`, `Esc`      | Quit                         |
| `t`             | Toggle light / dark          |
| `r`             | Reload                       |
| `Ctrl+F`, `/`   | Find in page                 |
| `n`, `N`        | Next / previous match        |
| `Enter`, `Shift+Enter` | Next / previous match (in find bar) |
| `j`, `↓`        | Scroll down                  |
| `k`, `↑`        | Scroll up                    |
| `g`, `G`        | Top / bottom                 |

External links open in your default browser via `xdg-open`. Relative image
paths are resolved against the directory of the Markdown file.

## Development

```sh
make test                # runs the renderer golden tests
go test ./internal/renderer -update   # regenerate goldens
```

To refresh the vendored PDF.js (override `PDFJS_VERSION` to pin a different
release):

```sh
make vendor-pdfjs
PDFJS_VERSION=4.10.38 make vendor-pdfjs
```
