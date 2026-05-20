# mdview — projektspezifische Hinweise

## Plattform-Invariante (verbindlich)

mdview muss auf **Linux, macOS und Windows** bauen und laufen. Jede Änderung,
die diese Invariante verletzt, ist abzulehnen oder vor dem Merge zu reparieren.

### Konkret heißt das

- **Keine ungetaggten POSIX-Calls in `cmd/mdview/`**. Alles aus `syscall`,
  `golang.org/x/sys/unix` o.Ä. gehört in eine Datei mit Build-Tag
  (`//go:build linux`, `//go:build darwin`, `//go:build !linux`, …). Vorbild:
  `cmd/mdview/stderr_filter.go` (Linux) + `cmd/mdview/stderr_filter_other.go`
  (non-Linux No-op).
- **Keine hartcodierten Helfer-Binaries** wie `xdg-open`, `open`, `cmd /c` ohne
  Plattform-Switch. Vorbild: `cmd/mdview/open_url_{linux,windows,darwin}.go`,
  die alle dieselbe Signatur `openCommand(url string) *exec.Cmd` liefern.
- **Neue Plattform-Abhängigkeiten** (z. B. ein OS-spezifisches CGO-Lib) brauchen
  immer eine Begründung im Commit und einen Build-Tag-getrennten Fallback.
- **`internal/...` bleibt plattformneutral.** Wenn dort doch mal ein OS-Switch
  nötig wird, gilt das gleiche Build-Tag-Pattern wie in `cmd/mdview/`.

### Verifikation

- `.github/workflows/build.yml` baut bei jedem Push und PR auf `ubuntu-latest`
  und `windows-latest`. Rote Matrix = blockt den Merge.
- Lokaler Reality-Check vor "fertig"-Meldung: `make build && go test ./...` auf
  Linux. Der Windows-Compile wird über CI bestätigt — lokales Cross-Compile
  mit mingw-w64 ist möglich, aber kein Muss.

### Build-Toolchain pro OS

| OS      | Build                                                            |
| ------- | ---------------------------------------------------------------- |
| Linux   | `make build` (braucht `libgtk-3-dev` + `libwebkit2gtk-4.{0,1}-dev`) |
| Windows | `go build ./cmd/mdview` (braucht `mingw-w64`, `CGO_ENABLED=1`, WebView2-Runtime) |
| macOS   | `go build ./cmd/mdview` (Xcode CLT)                              |

### Ausnahmen

- Tooling, das ausschließlich für die Entwicklung läuft (Scripts unter
  `scripts/`, einmalige Migrationen), darf Linux-only sein, muss aber nicht
  vom finalen Binary aus aufgerufen werden.
