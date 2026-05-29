BINARY := mdview
PREFIX ?= /usr/local
LDFLAGS := -s -w
PKG_CONFIG := $(abspath scripts/pkg-config-shim.sh)

export GOTOOLCHAIN ?= local
export PKG_CONFIG

.PHONY: build run install uninstall test clean tidy deps vendor-pdfjs

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
	go test ./internal/...

tidy:
	go mod tidy

deps:
	@echo "Required Ubuntu packages:"
	@echo "  sudo apt install libgtk-3-dev pkg-config"
	@echo "  Ubuntu 24.04: sudo apt install libwebkit2gtk-4.1-dev"
	@echo "  Ubuntu 22.04: sudo apt install libwebkit2gtk-4.0-dev"

clean:
	rm -rf bin

vendor-pdfjs:
	scripts/vendor-pdfjs.sh
