# mdview — Design

**Datum:** 2026-05-20
**Status:** Approved (Auto-Mode)

## Ziel

Ein schlanker, schneller Markdown-Viewer für Ubuntu/Linux. CLI-Aufruf
`mdview /path/zur/datei.md` öffnet ein natives Fenster und rendert die Datei
grafisch. Lieferung als einzelnes Binary.

## Stack-Entscheidungen

| Aspekt        | Wahl                                                   |
| ------------- | ------------------------------------------------------ |
| Sprache       | Go                                                     |
| GUI           | `github.com/webview/webview_go` (WebKit2GTK)           |
| Markdown      | `github.com/yuin/goldmark` + GFM-Extensions            |
| Code-Highlight| `github.com/alecthomas/chroma` via goldmark-highlight  |
| Math          | KaTeX, eingebettet via `go:embed`                      |
| Diagramme     | Mermaid.js, eingebettet via `go:embed`                 |
| File-Watch    | `github.com/fsnotify/fsnotify`                         |
| Build         | `go build` + Makefile                                  |

## Architektur

```
mdview/
├── cmd/mdview/main.go         # CLI, Webview-Setup, Glue
├── internal/
│   ├── renderer/              # Markdown → HTML (testbar, pur Go)
│   │   ├── renderer.go
│   │   ├── renderer_test.go
│   │   └── testdata/          # *.md + *.html.golden
│   ├── watcher/               # fsnotify-Wrapper mit Debounce
│   │   └── watcher.go
│   └── assets/                # eingebettete Web-Assets
│       ├── embed.go           # //go:embed
│       ├── template.html
│       ├── theme.css
│       ├── app.js             # Shortcuts, Suche, Theme-Toggle
│       └── vendor/
│           ├── katex/...
│           └── mermaid.min.js
├── go.mod / go.sum
├── Makefile
└── README.md
```

## Datenfluss

1. `main` parst CLI-Args, validiert Dateipfad, ermittelt Basisverzeichnis.
2. `renderer.Render(path)` liest die Datei, parst Markdown und gibt
   HTML-Body + Frontmatter-loses Resultat zurück.
3. `assets.Page(body, theme)` baut die finale HTML-Seite aus dem Template,
   eingebettetem CSS und JS, KaTeX und Mermaid.
4. `webview` wird mit der HTML-Seite initialisiert; nach `Init` werden
   Go-Bindings registriert (`openExternal(url)`, `quit()`).
5. `watcher.Watch(path, callback)` startet einen fsnotify-Loop. Bei
   Datei-Änderung wird neu gerendert und via
   `webview.Eval("updateContent(<json>)")` an die Seite gepusht.
6. Tasten und Suche laufen client-seitig in `app.js` und rufen nur bei
   `quit` / `openExternal` zurück nach Go.

## Markdown-Features

- **CommonMark** (Basis)
- **GFM**: Tabellen, Strikethrough, Task-Listen, Autolinks
- **Footnote**, **Typographer**
- **Syntax-Highlighting** via Chroma, zwei Stylesheets (light/dark) in CSS
  eingebettet, Klassen-basiert
- **Math**: `$...$` inline, `$$...$$` block; Renderer lässt die Syntax
  durch, KaTeX rendert clientseitig
- **Mermaid**: ```` ```mermaid ```` → `<pre class="mermaid">`, Mermaid
  rendert clientseitig
- **Bilder**: relative Pfade gegen Datei-Verzeichnis aufgelöst, als
  `file://`-URLs ausgegeben
- **Header-IDs**: automatisch (slugifiziert), Anker-Links funktionieren

## Interaktion

| Shortcut       | Aktion                                |
| -------------- | ------------------------------------- |
| `q`, `Esc`     | Quit                                  |
| `t`            | Light/Dark-Theme togglen              |
| `r`            | Manueller Reload                      |
| `j`, `↓`       | Scroll runter                         |
| `k`, `↑`       | Scroll hoch                           |
| `g`            | An den Anfang                         |
| `G`            | Ans Ende                              |
| `Ctrl+F`, `/`  | In-Page-Suche öffnen                  |
| `n`, `Enter`   | Nächster Treffer (in Suche)           |
| `N`, `Shift+Enter` | Vorheriger Treffer                |
| `Esc`          | Suche schließen                       |

- Externe Links (`http`, `https`, `mailto`) öffnen via Go-Binding mit
  `xdg-open`.
- Interne Anker (`#heading`) scrollen im Dokument.

## Live-Reload

- fsnotify watcht das **Verzeichnis** der Datei (nicht die Datei selbst),
  weil viele Editoren atomar speichern (RENAME statt WRITE).
- Events werden auf den Dateinamen gefiltert und 50 ms debounced.
- Bei Reload wird die aktuelle Scroll-Position in JS gemerkt
  (`window.scrollY`) und nach DOM-Update wiederhergestellt.
- Render-Fehler während Reload: rote Fehler-Box oben im Dokument, alter
  Inhalt bleibt sichtbar.

## CLI

```
mdview <pfad/zu/datei.md>

Optionen:
  -h, --help       Hilfe anzeigen
  -V, --version    Version anzeigen
```

Exit-Codes:

- `0` — sauberer Exit
- `1` — Datei nicht gefunden oder nicht lesbar
- `2` — falsche Argumente

Fenstertitel: `mdview — <basename>`, Default-Größe 1024×768.

## Build & Packaging

- `make build` → `bin/mdview` (`go build -ldflags "-s -w"`)
- `make run FILE=README.md` → Build + Start
- `make install` → kopiert nach `/usr/local/bin/mdview`
- `make test` → `go test ./...`
- Build-Voraussetzungen (Ubuntu): `libwebkit2gtk-4.1-dev` (24.04) bzw.
  `4.0-dev` (22.04), `libgtk-3-dev`, `pkg-config`
- Binary-Größe Ziel: < 15 MB inkl. eingebetteter KaTeX/Mermaid

## Tests

- **Renderer**: Golden-File-Tests für die wichtigsten Konstrukte
  (CommonMark-Basis, GFM-Tabelle, Code mit Highlight, Math-Passthrough,
  Mermaid-Passthrough, relative Bildpfade).
- **Watcher / Webview**: keine automatischen Tests (Aufwand-Nutzen-Verhältnis
  schlecht; manueller Smoke-Test reicht für ein Tool).
- Alle Tests müssen vor dem Commit grün sein.

## Out of Scope

- Print / PDF-Export
- Mehrere Dateien, Tabs
- Inhaltsverzeichnis-Sidebar
- Konfigurationsdatei
- Cross-Compile, AppImage, Snap, Deb
- CI-Pipeline
- Auto-Update

## Risiken

- **WebKit2GTK-Versions-Skew (4.0 vs. 4.1)** zwischen Ubuntu-LTS-Versionen:
  Build-Voraussetzungen werden im README dokumentiert. Cross-Distro-Bundling
  ist explizit Out of Scope.
- **KaTeX/Mermaid-Größe**: ~1.5 MB eingebettet. Akzeptiert, weil offline
  und ohne CDN-Dependency.
- **fsnotify Atomic-Save**: Verzeichnis-Watching + Debounce löst das, ist
  aber nicht-trivial — daher explizit getestet (manueller Smoke-Test).
