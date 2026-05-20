BINARY := mdview
PREFIX ?= /usr/local
LDFLAGS := -s -w
PKG_CONFIG := $(abspath scripts/pkg-config-shim.sh)

export GOTOOLCHAIN ?= local
export PKG_CONFIG

.PHONY: build run install uninstall test clean tidy deps

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mdview

run: build
ifndef FILE
	$(error usage: make run FILE=path/to/file.md)
endif
	./bin/$(BINARY) $(FILE)

install: build
	install -Dm755 bin/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

test:
	go test ./internal/renderer/...

tidy:
	go mod tidy

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

clean:
	rm -rf bin
