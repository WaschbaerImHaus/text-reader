# Makefile für MD Reader
# Cross-Compilation für Linux x86_64, Linux ARM64, Linux ARMhf, Windows x86_64
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-03-15

PROJECT     := md-reader
BUILD_DIR   := build
SRC_DIR     := src
PKG_CONFIG  := $(SRC_DIR)/pkgconfig

# Build-Nummer aus Datei lesen und inkrementieren
BUILD_NUM   := $(shell cat $(SRC_DIR)/build.txt 2>/dev/null || echo 0)
NEXT_BUILD  := $(shell expr $(BUILD_NUM) + 1)

# Go-Compiler
GO          := go
GOOS_LINUX  := linux
GOOS_WIN    := windows
ARCH        := amd64

# Windows Cross-Compiler
WIN_CC      := x86_64-w64-mingw32-gcc
WIN_CXX     := x86_64-w64-mingw32-g++

# ARM Cross-Compiler
ARM64_CC    := aarch64-linux-gnu-gcc
ARM64_CXX   := aarch64-linux-gnu-g++
ARMHF_CC    := arm-linux-gnueabihf-gcc
ARMHF_CXX   := arm-linux-gnueabihf-g++

# PKG_CONFIG_PATH für WebKit-Shim
export PKG_CONFIG_PATH := $(PKG_CONFIG):$(PKG_CONFIG_PATH)

# GitHub-Einstellungen für Release
GITHUB_REPO := WaschbaerImHaus/text-reader

# Gemeinsame ldflags:
#   -s -w              → Debug-Symbole entfernen, kleinere Binary
#   -static-libstdc++  → libstdc++ statisch einlinken (verhindert CXXABI-Fehler auf älteren Systemen)
#   -static-libgcc     → libgcc statisch einlinken (verhindert libgcc_s.so-Fehler)
LDFLAGS_LINUX := -s -w -extldflags '-static-libstdc++ -static-libgcc'
LDFLAGS_WIN   := -s -w -H windowsgui

.PHONY: all linux windows linux-arm64 linux-armhf installer release test clean increment-build docker-build docker-build-release docker-rebuild help

## all: Kompiliert für Linux x86_64, Windows und Linux ARM64/ARMhf
all: increment-build test linux windows linux-arm64 linux-armhf installer

## linux: Kompiliert das Linux x86_64 Binary
linux: $(BUILD_DIR)/$(PROJECT)

$(BUILD_DIR)/$(PROJECT):
	@echo "→ Kompiliere für Linux x86_64..."
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=$(GOOS_LINUX) GOARCH=$(ARCH) \
		$(GO) build -ldflags="$(LDFLAGS_LINUX)" -o ../$(BUILD_DIR)/$(PROJECT) .
	@echo "✓ Linux-Binary: $(BUILD_DIR)/$(PROJECT)"

## windows: Kompiliert das Windows x86_64 Binary (Cross-Compilation mit MinGW)
windows: $(BUILD_DIR)/$(PROJECT).exe

$(BUILD_DIR)/$(PROJECT).exe:
	@echo "→ Kompiliere für Windows x86_64 (Cross-Compilation)..."
	@mkdir -p $(BUILD_DIR)
	@if command -v $(WIN_CC) > /dev/null 2>&1; then \
		cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=$(GOOS_WIN) GOARCH=$(ARCH) \
			CC=$(WIN_CC) CXX=$(WIN_CXX) \
			$(GO) build -ldflags="$(LDFLAGS_WIN)" -o ../$(BUILD_DIR)/$(PROJECT).exe . && \
		echo "✓ Windows-Binary: $(BUILD_DIR)/$(PROJECT).exe"; \
	else \
		echo "⚠ $(WIN_CC) nicht gefunden. Windows-Build übersprungen."; \
		echo "  Installation: sudo apt-get install gcc-mingw-w64"; \
	fi

## linux-arm64: Kompiliert das Linux ARM64 Binary
linux-arm64: $(BUILD_DIR)/$(PROJECT)-linux-arm64

$(BUILD_DIR)/$(PROJECT)-linux-arm64:
	@echo "→ Kompiliere für Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	@if command -v $(ARM64_CC) > /dev/null 2>&1; then \
		cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
			CC=$(ARM64_CC) CXX=$(ARM64_CXX) \
			PKG_CONFIG=aarch64-linux-gnu-pkg-config \
			PKG_CONFIG_PATH=$(abspath $(SRC_DIR)/pkgconfig-arm64) \
			$(GO) build -ldflags="$(LDFLAGS_LINUX)" -o ../$(BUILD_DIR)/$(PROJECT)-linux-arm64 . && \
		echo "✓ Linux-ARM64-Binary: $(BUILD_DIR)/$(PROJECT)-linux-arm64"; \
	else \
		echo "⚠ $(ARM64_CC) nicht gefunden. ARM64-Build übersprungen."; \
		echo "  Installation: sudo apt-get install gcc-aarch64-linux-gnu"; \
	fi

## linux-armhf: Kompiliert das Linux ARMhf (ARM32) Binary
linux-armhf: $(BUILD_DIR)/$(PROJECT)-linux-armhf

$(BUILD_DIR)/$(PROJECT)-linux-armhf:
	@echo "→ Kompiliere für Linux ARMhf..."
	@mkdir -p $(BUILD_DIR)
	@if command -v $(ARMHF_CC) > /dev/null 2>&1; then \
		cd $(SRC_DIR) && CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 \
			CC=$(ARMHF_CC) CXX=$(ARMHF_CXX) \
			PKG_CONFIG=arm-linux-gnueabihf-pkg-config \
			PKG_CONFIG_PATH=$(abspath $(SRC_DIR)/pkgconfig-armhf) \
			$(GO) build -ldflags="$(LDFLAGS_LINUX)" -o ../$(BUILD_DIR)/$(PROJECT)-linux-armhf . && \
		echo "✓ Linux-ARMhf-Binary: $(BUILD_DIR)/$(PROJECT)-linux-armhf"; \
	else \
		echo "⚠ $(ARMHF_CC) nicht gefunden. ARMhf-Build übersprungen."; \
		echo "  Installation: sudo apt-get install gcc-arm-linux-gnueabihf"; \
	fi

## installer: Erstellt den Windows NSIS Installer (benötigt makensis)
installer: $(BUILD_DIR)/$(PROJECT).exe
	@echo "→ Erstelle Windows Installer..."
	@if command -v makensis > /dev/null 2>&1; then \
		makensis installer/$(PROJECT).nsi && \
		echo "✓ Installer: $(BUILD_DIR)/$(PROJECT)-setup.exe"; \
	else \
		echo "⚠ makensis nicht gefunden. Installer-Build übersprungen."; \
		echo "  Installation: sudo apt-get install nsis"; \
	fi

## release: Kompiliert alles und erstellt ein GitHub Release
release: all
	@echo "→ Erstelle GitHub Release v1.0.$(NEXT_BUILD)..."
	@if [ -z "$$GH_TOKEN" ] && ! gh auth status > /dev/null 2>&1; then \
		echo "⚠ Kein GitHub-Token. GH_TOKEN setzen oder 'gh auth login' ausführen."; \
		exit 1; \
	fi
	GH_TOKEN=$${GH_TOKEN} gh release create "v1.0.$(NEXT_BUILD)" \
		--repo $(GITHUB_REPO) \
		--title "MD Reader v1.0.$(NEXT_BUILD) (Build $(NEXT_BUILD))" \
		--target main \
		--notes "$$($(MAKE) --no-print-directory release-notes BUILD_NUM=$(NEXT_BUILD))" \
		$(BUILD_DIR)/$(PROJECT)-setup.exe \
		$(BUILD_DIR)/$(PROJECT).exe \
		$(BUILD_DIR)/$(PROJECT) \
		$(BUILD_DIR)/$(PROJECT)-linux-arm64 \
		$(BUILD_DIR)/$(PROJECT)-linux-armhf
	@echo "✓ GitHub Release v1.0.$(NEXT_BUILD) veröffentlicht"

## release-notes: Gibt die Release Notes für den aktuellen Build aus
release-notes:
	@echo "## MD Reader v1.0.$(BUILD_NUM) (Build $(BUILD_NUM))"
	@echo ""
	@echo "### Downloads"
	@echo ""
	@echo "| File | Platform |"
	@echo "|------|----------|"
	@echo "| \`md-reader-setup.exe\` | Windows x86_64 – Installer (recommended) |"
	@echo "| \`md-reader.exe\` | Windows x86_64 – Standalone binary |"
	@echo "| \`md-reader\` | Linux x86_64 |"
	@echo "| \`md-reader-linux-arm64\` | Linux ARM64 |"
	@echo "| \`md-reader-linux-armhf\` | Linux ARMhf (32-bit) |"
	@echo ""
	@echo "### Requirements"
	@echo ""
	@echo "**Windows:** Windows 10 or newer (WebView2 pre-installed)"
	@echo "**Linux:** \`libwebkit2gtk-4.1-0\`, \`libgtk-3-0\`"

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

## docker-build: Vollständiger Build via Docker (Ubuntu 24.04, läuft auf Mint 22+)
docker-build:
	@echo "→ Starte Docker-Build (Ubuntu 24.04 / glibc 2.39)..."
	@bash docker/build.sh all

## docker-build-release: Docker-Build + GitHub Release
docker-build-release:
	@echo "→ Starte Docker-Build mit GitHub Release..."
	@bash docker/build.sh release

## docker-rebuild: Docker-Image neu bauen (nach Dockerfile-Änderungen)
docker-rebuild:
	@echo "→ Erzwinge Neubau des Docker-Images..."
	@FORCE_REBUILD=1 bash docker/build.sh all

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
