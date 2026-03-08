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
# Linux
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o ../build/md-reader .

# Windows
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -o ../build/md-reader.exe .
```

## Architektur
- `main.go`: WebView-Setup + Go-JS-Bindings
- `renderer/markdown.go`: goldmark MD→HTML Konvertierung
- `ui/template.go`: HTML/CSS/JS-Template (gesamte UI)
- `platform_linux.go`: GTK-Vollbild (CGo, Build-Tag: linux)
- `platform_windows.go`: WinAPI-Vollbild (Build-Tag: windows)

## UI-Interaktion (JS↔Go)
- JS ruft `window.processMarkdown(content, filename)` → Go gibt `{html, title}` zurück
- JS ruft `window.closeApp()` → Go beendet WebView
- JS ruft `window.nativeFullscreen()` → Go wechselt OS-Vollbild
