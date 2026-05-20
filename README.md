# mdview

A small, fast Markdown viewer for Ubuntu/Linux. Open a file with
`mdview path/to/file.md` and a native window pops up rendering it.

## Features

- CommonMark + GitHub Flavored Markdown (tables, task lists, strikethrough,
  autolinks)
- Syntax-highlighted code blocks (Chroma, ~200 languages)
- Math via KaTeX (`$inline$`, `$$block$$`)
- Mermaid diagrams (` ```mermaid ` blocks)
- Light / dark theme (follows OS by default, toggle with `t`)
- In-page search (`Ctrl+F` or `/`)
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
