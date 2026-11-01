# Makefile für MD Reader
# Cross-Compilation für Linux x86_64, Linux ARM64, Linux ARMhf, Windows x86_64
# Linux-Ablauf je Architektur: GUI im Docker-Container (Ubuntu 22.04, glibc 2.35)
# bauen (build/gui/) → glibc-Wächter auf dem Host → GUI und Icons als Nutzlast nach
# src/launcher/payload/ kopieren → statischen Starter bauen (build/md-reader*, Host,
# reines Go). Dazu kommt je Architektur ein .deb (reine GUI).
# Images einmalig bauen: make build-images
# macOS (Universal) nur auf einem Mac: make macos (sonst GitHub-Workflow macos.yml)
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-11-01 (GUI-Build in Ubuntu-22.04-Containern, Wächter 2.35)

PROJECT     := md-reader
BUILD_DIR   := build
SRC_DIR     := src
PKG_CONFIG  := $(SRC_DIR)/pkgconfig

# Build-Nummer aus Datei lesen und inkrementieren.
# BUILD_NUM ist bewusst rekursiv ("="): Es wird bei jeder Verwendung neu gelesen,
# damit "make all" nach increment-build die neue Nummer in Starter und .deb nutzt.
BUILD_NUM   = $(shell cat $(SRC_DIR)/build.txt 2>/dev/null || echo 0)
NEXT_BUILD  := $(shell expr $(BUILD_NUM) + 1)

# Go-Compiler
GO          := go
GOOS_LINUX  := linux
GOOS_WIN    := windows
ARCH        := amd64

# Windows Cross-Compiler
WIN_CC      := x86_64-w64-mingw32-gcc
WIN_CXX     := x86_64-w64-mingw32-g++

# Docker-Build-Images der Linux-GUI (installer/linux/docker/Dockerfile).
# Die ARM-Cross-Compiler und WebKit-Pakete stecken in den Images, nicht auf dem Host.
IMAGE_PREFIX := md-reader-build:22.04
DOCKER_DIR   := installer/linux/docker

# Caches außerhalb des Repos (Go-Modulcache und Go-Build-Cache), in den Container gemountet
CACHE_ROOT   := $(HOME)/.cache/md-reader-build
GOMOD_CACHE  := $(CACHE_ROOT)/gomod
GOBUILD_CACHE := $(CACHE_ROOT)/gobuild

# Container-Aufruf für den GUI-Build: Projekt nach /work, eigener Benutzer (Dateien
# gehören nicht root), kein Netz (Module liegen vorab im Cache, siehe gomod-download).
# Die Shims in src/pkgconfig*/ werden über den absoluten Containerpfad /work/src/... gefunden.
DOCKER_RUN = docker run --rm --network none \
	--user $(shell id -u):$(shell id -g) \
	-v "$(CURDIR)":/work -w /work/src \
	-e HOME=/tmp -e GOCACHE=/tmp/.cache/go-build \
	-e GOPROXY=off -e GOFLAGS=-mod=readonly \
	-v "$(GOMOD_CACHE)":/go/pkg/mod \
	-v "$(GOBUILD_CACHE)":/tmp/.cache/go-build

# Prüft, dass das Image der Architektur ($(1)) existiert
CHECK_IMAGE = docker image inspect $(IMAGE_PREFIX)-$(1) > /dev/null 2>&1 || { \
	echo "FEHLER: Image $(IMAGE_PREFIX)-$(1) fehlt – erst 'make build-images' ausführen." >&2; exit 1; }

# PKG_CONFIG_PATH für WebKit-Shim (nur für Host-Aufrufe wie make test; die
# Container-Builds setzen den Pfad selbst)
export PKG_CONFIG_PATH := $(abspath $(PKG_CONFIG)):$(PKG_CONFIG_PATH)

# GitHub-Einstellungen für Release
GITHUB_REPO := WaschbaerImHaus/text-reader

# Gemeinsame ldflags:
#   -s -w              → Debug-Symbole entfernen, kleinere Binary
#   -static-libstdc++  → libstdc++ statisch einlinken (verhindert CXXABI-Fehler auf älteren Systemen)
#   -static-libgcc     → libgcc statisch einlinken (verhindert libgcc_s.so-Fehler)
LDFLAGS_LINUX := -s -w -extldflags '-static-libstdc++ -static-libgcc'
LDFLAGS_WIN   := -s -w -H windowsgui

# Starter (statisch, ohne CGO): Build-Nummer wird per -X eingesetzt
LDFLAGS_LAUNCHER = -s -w -X main.buildNumber=$(BUILD_NUM)

# Zwischenergebnisse und Nutzlast des Starters
GUI_DIR     := $(BUILD_DIR)/gui
PAYLOAD_DIR := $(SRC_DIR)/launcher/payload
ICON_SRC    := $(SRC_DIR)/ui/assets

# Höchste erlaubte glibc-Version der Linux-Dateien (Ubuntu 22.04 hat glibc 2.35).
# Nur auf der Befehlszeile überschreibbar (make GLIBC_MAX=2.30), um den
# Wächter zu testen. Eine Umgebungsvariable GLIBC_MAX (auch mit make -e)
# würde den Wächter still abschwächen und bricht deshalb ab (Härtung H4,
# 2026-11-01). Die Prüfung muss VOR der Zuweisung stehen: Danach meldet
# $(origin) "file" statt "environment".
ifneq ($(filter environment%,$(origin GLIBC_MAX)),)
$(error GLIBC_MAX darf nicht aus der Umgebung kommen (gefunden: $(GLIBC_MAX)). Zum Testen des Wächters auf der Befehlszeile angeben: make GLIBC_MAX=2.30 ...)
endif
GLIBC_MAX   := 2.35
CHECK_GLIBC := installer/linux/check-glibc.sh
BUILD_DEB   := installer/linux/build-deb.sh

# Kopiert die Icons in die Nutzlast des Starters (gitignoriert)
COPY_ICONS   = mkdir -p $(PAYLOAD_DIR)/icons && \
	cp $(ICON_SRC)/favicon-16.png $(ICON_SRC)/favicon-32.png \
	   $(ICON_SRC)/favicon-48.png $(ICON_SRC)/favicon-256.png \
	   $(ICON_SRC)/favicon.svg $(PAYLOAD_DIR)/icons/

# Kein paralleler Lauf (make -j): BUILD_NUM darf nicht vor increment-build gelesen
# werden, und alle Starter teilen sich das Verzeichnis payload/icons.
.NOTPARALLEL:

# Schlägt ein Rezept fehl, wird sein (halbfertiges) Ziel gelöscht
.DELETE_ON_ERROR:

# FORCE: GUI- und Starter-Ziele haben keine Quelldateien als Abhängigkeit. Mit FORCE
# werden sie bei jedem Aufruf neu gebaut (der Go-Build-Cache hält das billig). So wird
# nie eine alte build/gui/*-Datei unter einer neuen Build-Nummer weiterverwendet, auch
# nicht nach "rm -f build/md-reader*". Folge: "make linux-arm64 deb-arm64" immer in
# EINEM Aufruf starten (siehe rules/build-release.md).
FORCE:

.PHONY: FORCE all linux windows linux-arm64 linux-armhf installer macos release test clean increment-build help \
	deb deb-amd64 deb-arm64 deb-armhf build-images gomod-download

## all: Kompiliert für Linux x86_64, Windows und Linux ARM64/ARMhf
all: increment-build test linux windows linux-arm64 linux-armhf installer deb

## linux: Kompiliert das Linux x86_64 Binary (Starter mit eingebetteter GUI)
linux: $(BUILD_DIR)/$(PROJECT)

# GUI (dynamisch gelinkt, CGO) → erst nach <ziel>.tmp, dann glibc-Wächter, dann
# umbenennen. So bleibt nie eine vom Wächter abgelehnte Datei unter dem Zielnamen liegen.
$(GUI_DIR)/$(PROJECT)-amd64: FORCE
	@echo "→ Kompiliere GUI für Linux x86_64 (Container $(IMAGE_PREFIX)-amd64)..."
	@mkdir -p $(GUI_DIR)
	@$(call CHECK_IMAGE,amd64)
	@$(MAKE) --no-print-directory gomod-download
	rm -f $@ $@.tmp
	$(DOCKER_RUN) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 \
		-e PKG_CONFIG_PATH=/work/src/pkgconfig \
		$(IMAGE_PREFIX)-amd64 \
		go build -ldflags="$(LDFLAGS_LINUX)" -o ../$@.tmp . || { rm -f $@.tmp; exit 1; }
	$(CHECK_GLIBC) $(GLIBC_MAX) $@.tmp || { rm -f $@.tmp; exit 1; }
	mv $@.tmp $@

# Starter: GUI und Icons als Nutzlast kopieren, dann statisch bauen
$(BUILD_DIR)/$(PROJECT): $(GUI_DIR)/$(PROJECT)-amd64 FORCE
	@echo "→ Baue Starter für Linux x86_64..."
	@mkdir -p $(PAYLOAD_DIR)/amd64
	cp $(GUI_DIR)/$(PROJECT)-amd64 $(PAYLOAD_DIR)/amd64/$(PROJECT)-gui
	$(COPY_ICONS)
	cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build -tags embedgui -trimpath -ldflags "$(LDFLAGS_LAUNCHER)" \
		-o ../$(BUILD_DIR)/$(PROJECT) ./launcher
	@echo "✓ Linux-Starter: $(BUILD_DIR)/$(PROJECT)"

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

## linux-arm64: Kompiliert das Linux ARM64 Binary (Starter mit eingebetteter GUI)
linux-arm64: $(BUILD_DIR)/$(PROJECT)-linux-arm64

$(GUI_DIR)/$(PROJECT)-arm64: FORCE
	@echo "→ Kompiliere GUI für Linux ARM64 (Container $(IMAGE_PREFIX)-arm64)..."
	@mkdir -p $(GUI_DIR)
	@$(call CHECK_IMAGE,arm64)
	@$(MAKE) --no-print-directory gomod-download
	rm -f $@ $@.tmp
	$(DOCKER_RUN) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=arm64 \
		-e CC=aarch64-linux-gnu-gcc -e CXX=aarch64-linux-gnu-g++ \
		-e PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
		-e PKG_CONFIG_PATH=/work/src/pkgconfig-arm64 \
		$(IMAGE_PREFIX)-arm64 \
		go build -ldflags="$(LDFLAGS_LINUX)" -o ../$@.tmp . || { rm -f $@.tmp; exit 1; }
	$(CHECK_GLIBC) $(GLIBC_MAX) $@.tmp || { rm -f $@.tmp; exit 1; }
	mv $@.tmp $@

$(BUILD_DIR)/$(PROJECT)-linux-arm64: $(GUI_DIR)/$(PROJECT)-arm64 FORCE
	@echo "→ Baue Starter für Linux ARM64..."
	@mkdir -p $(PAYLOAD_DIR)/arm64
	cp $(GUI_DIR)/$(PROJECT)-arm64 $(PAYLOAD_DIR)/arm64/$(PROJECT)-gui
	$(COPY_ICONS)
	cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		$(GO) build -tags embedgui -trimpath -ldflags "$(LDFLAGS_LAUNCHER)" \
		-o ../$(BUILD_DIR)/$(PROJECT)-linux-arm64 ./launcher
	@echo "✓ Linux-ARM64-Starter: $(BUILD_DIR)/$(PROJECT)-linux-arm64"

## linux-armhf: Kompiliert das Linux ARMhf (ARM32) Binary (Starter mit eingebetteter GUI)
linux-armhf: $(BUILD_DIR)/$(PROJECT)-linux-armhf

$(GUI_DIR)/$(PROJECT)-arm: FORCE
	@echo "→ Kompiliere GUI für Linux ARMhf (Container $(IMAGE_PREFIX)-armhf)..."
	@mkdir -p $(GUI_DIR)
	@$(call CHECK_IMAGE,armhf)
	@$(MAKE) --no-print-directory gomod-download
	rm -f $@ $@.tmp
	$(DOCKER_RUN) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=arm -e GOARM=7 \
		-e CC=arm-linux-gnueabihf-gcc -e CXX=arm-linux-gnueabihf-g++ \
		-e PKG_CONFIG_LIBDIR=/usr/lib/arm-linux-gnueabihf/pkgconfig:/usr/share/pkgconfig \
		-e PKG_CONFIG_PATH=/work/src/pkgconfig-armhf \
		$(IMAGE_PREFIX)-armhf \
		go build -ldflags="$(LDFLAGS_LINUX)" -o ../$@.tmp . || { rm -f $@.tmp; exit 1; }
	$(CHECK_GLIBC) $(GLIBC_MAX) $@.tmp || { rm -f $@.tmp; exit 1; }
	mv $@.tmp $@

$(BUILD_DIR)/$(PROJECT)-linux-armhf: $(GUI_DIR)/$(PROJECT)-arm FORCE
	@echo "→ Baue Starter für Linux ARMhf..."
	@mkdir -p $(PAYLOAD_DIR)/arm
	cp $(GUI_DIR)/$(PROJECT)-arm $(PAYLOAD_DIR)/arm/$(PROJECT)-gui
	$(COPY_ICONS)
	cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
		$(GO) build -tags embedgui -trimpath -ldflags "$(LDFLAGS_LAUNCHER)" \
		-o ../$(BUILD_DIR)/$(PROJECT)-linux-armhf ./launcher
	@echo "✓ Linux-ARMhf-Starter: $(BUILD_DIR)/$(PROJECT)-linux-armhf"

## build-images: Baut die drei Docker-Build-Images (idempotent, Schichten-Cache)
build-images:
	@for a in amd64 arm64 armhf; do \
		echo "→ Image $(IMAGE_PREFIX)-$$a..."; \
		docker build --build-arg TARGET_ARCH=$$a -t $(IMAGE_PREFIX)-$$a $(DOCKER_DIR) || exit 1; \
	done

## gomod-download: Lädt die Go-Module auf dem Host in den Cache (der Container läuft ohne Netz)
gomod-download:
	@mkdir -p "$(GOMOD_CACHE)" "$(GOBUILD_CACHE)"
	cd $(SRC_DIR) && GOMODCACHE="$(GOMOD_CACHE)" GOFLAGS=-mod=mod GOPROXY=https://proxy.golang.org,direct $(GO) mod download

## deb: Baut die drei .deb-Pakete (amd64, arm64, armhf; reine GUI ohne Starter)
deb: deb-amd64 deb-arm64 deb-armhf

# Die Pakete sind absichtlich PHONY: Der Dateiname enthält die Build-Nummer, die
# erst zur Laufzeit feststeht (siehe BUILD_NUM). Die GUI-Dateien werden wegen FORCE
# in jedem make-Aufruf frisch gebaut; build-deb.sh prüft Architektur und Bauart.
deb-amd64: $(GUI_DIR)/$(PROJECT)-amd64
	$(BUILD_DEB) amd64 $(GUI_DIR)/$(PROJECT)-amd64 $(BUILD_NUM)

deb-arm64: $(GUI_DIR)/$(PROJECT)-arm64
	$(BUILD_DEB) arm64 $(GUI_DIR)/$(PROJECT)-arm64 $(BUILD_NUM)

deb-armhf: $(GUI_DIR)/$(PROJECT)-arm
	$(BUILD_DEB) armhf $(GUI_DIR)/$(PROJECT)-arm $(BUILD_NUM)

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

## macos: Baut "MD Reader.app" + build/md-reader-macos.zip (nur unter macOS)
macos:
	installer/macos/build-app.sh

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
		$(BUILD_DIR)/$(PROJECT)-linux-armhf \
		$(BUILD_DIR)/$(PROJECT)_1.0.$(NEXT_BUILD)_amd64.deb \
		$(BUILD_DIR)/$(PROJECT)_1.0.$(NEXT_BUILD)_arm64.deb \
		$(BUILD_DIR)/$(PROJECT)_1.0.$(NEXT_BUILD)_armhf.deb
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
	@echo "| \`md-reader\` | Linux x86_64 – single file, installs itself on request |"
	@echo "| \`md-reader-linux-arm64\` | Linux ARM64 – single file, installs itself on request |"
	@echo "| \`md-reader-linux-armhf\` | Linux ARMhf (32-bit) – single file, installs itself on request |"
	@echo "| \`md-reader_1.0.$(BUILD_NUM)_amd64.deb\` | Debian/Ubuntu/Mint x86_64 package |"
	@echo "| \`md-reader_1.0.$(BUILD_NUM)_arm64.deb\` | Debian/Ubuntu/Mint ARM64 package |"
	@echo "| \`md-reader_1.0.$(BUILD_NUM)_armhf.deb\` | Debian/Ubuntu/Mint ARMhf (32-bit) package |"
	@echo "| \`md-reader-macos.zip\` | macOS 13+ – Apple Silicon and Intel (attached by CI a few minutes after release) |"
	@echo ""
	@echo "### Requirements"
	@echo ""
	@echo "**Windows:** Windows 10 or newer (WebView2 pre-installed)"
	@echo "**Linux:** Ubuntu 22.04+, Debian 12+, Mint 21+, KDE neon, Raspberry Pi OS Bookworm; WebKitGTK is offered for installation on first start"
	@echo "**macOS:** macOS 13 or newer; unsigned build – on first start allow it in System Settings → Privacy & Security (\"Open Anyway\")"

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
