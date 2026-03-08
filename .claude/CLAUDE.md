# MD Reader - Projektspezifische Claude-Anweisungen

## Projektübersicht
Markdown-Betrachter mit GitHub-ähnlichem HTML/CSS-Rendering via WebView.

## Technologie
- Go 1.26, webview_go, goldmark (+highlighting)
- WebKitGTK 4.1 (Linux), Edge WebView2 (Windows)

## Wichtig beim Entwickeln
- **Tests zuerst** ausführen: `go test ./renderer/... -v` aus `src/`
- **PKG_CONFIG_PATH** immer setzen: `PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH`
- **Build-Nummer** in `src/build.txt` bei jeder Änderung inkrementieren
- **EventToken.h Stub** im GOPATH/pkg/mod für Windows Cross-Compilation benötigt

## Build-Befehle (aus src/ ausführen!)
```bash
# Linux x86_64
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o ../build/md-reader .

# Linux ARM64
PKG_CONFIG=aarch64-linux-gnu-pkg-config PKG_CONFIG_PATH=$(pwd)/pkgconfig-arm64 \
  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
  CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ \
  go build -o ../build/md-reader-linux-arm64 .

# Linux ARMhf (ARM32)
PKG_CONFIG=arm-linux-gnueabihf-pkg-config PKG_CONFIG_PATH=$(pwd)/pkgconfig-armhf \
  CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 \
  CC=arm-linux-gnueabihf-gcc CXX=arm-linux-gnueabihf-g++ \
  go build -o ../build/md-reader-linux-armhf .

# Windows x86_64
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -o ../build/md-reader.exe .

# Windows ARM64: nicht möglich (kein aarch64-w64-mingw32-gcc in Ubuntu-Repos)
```

## ARM Cross-Compilation Voraussetzungen
```bash
# Compiler:
sudo apt-get install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
                     gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf

# Bibliotheken (Multiarch):
# ARM64-Quelle: /etc/apt/sources.list.d/arm64-ports.sources (ports.ubuntu.com)
# ARMhf-Quelle: /etc/apt/sources.list.d/armhf-ports.sources (ports.ubuntu.com)
sudo apt-get install libwebkit2gtk-4.1-dev:arm64 libwebkit2gtk-4.1-dev:armhf

# PKG_CONFIG-Shims: src/pkgconfig-arm64/ und src/pkgconfig-armhf/
```

## Architektur
- `main.go`: WebView-Setup + Go-JS-Bindings
- `bindings.go`: Alle Go→JS-Bindings (processMarkdown, saveScrollPos, openFilePicker …)
- `icon.go`: go:embed für App-Icon PNG (appIconPNG)
- `renderer/markdown.go`: goldmark MD→HTML Konvertierung
- `ui/template.go`: HTML/CSS/JS-Template zusammenbauen
- `ui/favicon.go`: go:embed SVG-Favicon als base64 Data-URI
- `platform_linux.go`: GTK-Vollbild + Dateidialog + Icon-Setzung (CGo, Build-Tag: linux)
- `platform_windows.go`: WinAPI-Vollbild + Dateidialog + WM_SETICON (Build-Tag: windows)
- `rsrc_windows_amd64.syso`: Windows-Ressource mit eingebettetem Icon (IDI_ICON1)

## UI-Interaktion (JS↔Go)
- `window.processMarkdown(content, filename)` → `{html, title, fileHash, scrollPos}`
- `window.processEpub(base64, filename)` → `{html, title, fileHash, scrollPos}`
- `window.saveScrollPos(hash, scrollPos)` → speichert Scroll-Position in ScrollHistory
- `window.persistLastFile(path)` → speichert zuletzt geöffnete Datei
- `window.persistState(fontSize, theme, layout)` → speichert Einstellungen
- `window.openFilePicker()` → öffnet nativen OS-Dateidialog
- `window.nativeFullscreen()` → wechselt OS-Vollbild
- `window._closeAppNative()` → beendet App
