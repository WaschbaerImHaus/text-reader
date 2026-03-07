# Makefile für MD Reader
# Cross-Compilation für Linux und Windows
#
# Autor: Reisen macht Spass... mit Pia und Dirk e.Kfm.
# Letzte Änderung: 2026-03-05

PROJECT     := md-reader
BUILD_DIR   := build
SRC_DIR     := src
PKG_CONFIG  := $(SRC_DIR)/pkgconfig

# Build-Nummer aus Datei lesen und inkrementieren
BUILD_NUM   := $(shell cat $(SRC_DIR)/build.txt 2>/dev/null || echo 0)
NEXT_BUILD  := $(shell expr $(BUILD_NUM) + 1)

# Kompilier-Flags
GO          := go
GOOS_LINUX  := linux
GOOS_WIN    := windows
ARCH        := amd64

# Windows Cross-Compiler
WIN_CC      := x86_64-w64-mingw32-gcc

# PKG_CONFIG_PATH für WebKit-Shim
export PKG_CONFIG_PATH := $(PKG_CONFIG):$(PKG_CONFIG_PATH)

.PHONY: all linux windows test clean increment-build

## all: Kompiliert für Linux und Windows
all: increment-build test linux windows

## linux: Kompiliert das Linux-Binary
linux: $(BUILD_DIR)/$(PROJECT)

$(BUILD_DIR)/$(PROJECT):
	@echo "→ Kompiliere für Linux..."
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=$(GOOS_LINUX) GOARCH=$(ARCH) \
		$(GO) build -o ../$(BUILD_DIR)/$(PROJECT) .
	@echo "✓ Linux-Binary: $(BUILD_DIR)/$(PROJECT)"

## windows: Kompiliert das Windows-Binary (Cross-Compilation)
windows: $(BUILD_DIR)/$(PROJECT).exe

$(BUILD_DIR)/$(PROJECT).exe:
	@echo "→ Kompiliere für Windows (Cross-Compilation)..."
	@mkdir -p $(BUILD_DIR)
	@if command -v $(WIN_CC) > /dev/null 2>&1; then \
		cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=$(GOOS_WIN) GOARCH=$(ARCH) \
			CC=$(WIN_CC) CXX=$(WIN_CXX) \
			$(GO) build -ldflags="-H windowsgui" -o ../$(BUILD_DIR)/$(PROJECT).exe . && \
		echo "✓ Windows-Binary: $(BUILD_DIR)/$(PROJECT).exe"; \
	else \
		echo "⚠ $(WIN_CC) nicht gefunden. Windows-Build übersprungen."; \
		echo "  Installation: sudo apt-get install gcc-mingw-w64"; \
	fi

## test: Führt alle Tests aus
test:
	@echo "→ Führe Tests aus..."
	cd $(SRC_DIR) && PKG_CONFIG_PATH=$(abspath $(PKG_CONFIG)):$$PKG_CONFIG_PATH \
		CGO_ENABLED=1 $(GO) test ./... -v 2>&1 | tail -30
	@echo "✓ Tests abgeschlossen"

## increment-build: Erhöht die Build-Nummer
increment-build:
	@echo $(NEXT_BUILD) > $(SRC_DIR)/build.txt
	@echo "→ Build-Nummer: $(NEXT_BUILD)"

## clean: Löscht kompilierte Binaries
clean:
	@echo "→ Lösche Build-Verzeichnis..."
	rm -rf $(BUILD_DIR)/*
	@echo "✓ Bereinigt"

## help: Zeigt diese Hilfe
help:
	@echo "MD Reader - Build-System"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
