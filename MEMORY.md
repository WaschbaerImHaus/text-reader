# Projektspeicher MD Reader

## Überblick
Markdown-Betrachter mit GitHub-ähnlicher Darstellung. WebView-basiert für echtes HTML/CSS-Rendering.

## Technologie-Stack
- **Go 1.26** – Hauptprogrammiersprache
- **github.com/webview/webview_go** – WebView (WebKitGTK4.1 auf Linux, Edge WebView2 auf Windows)
- **github.com/yuin/goldmark** + **goldmark-highlighting/v2** – MD→HTML Konvertierung
- **PKG_CONFIG_PATH**: `src/pkgconfig/` enthält WebKit 4.0→4.1 Kompatibilitäts-Shim

## Wichtige Pfade
- Binaries: `build/md-reader` (Linux), `build/md-reader.exe` (Windows)
- Quellcode: `src/` (go.mod hier, nicht im Projektroot)
- Pkg-Config-Shim: `src/pkgconfig/webkit2gtk-4.0.pc`
- Tests: `src/renderer/markdown_test.go` (12 Tests, alle grün)

## Build-Befehle
```bash
# Immer aus src/ ausführen
cd src

# Linux:
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o ../build/md-reader .

# Windows (Cross-Compilation):
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -o ../build/md-reader.exe .

# Tests:
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go test ./renderer/... -v
```

## Bekannte Probleme
- Windows Cross-Compilation: EventToken.h Stub wurde in webview_go Module-Cache erstellt
- WebKit-Version: Nur 4.1 verfügbar auf diesem System (nicht 4.0)

## Architektur-Entscheidungen
- **WebView statt Fyne**: Für echtes GitHub-ähnliches Rendering (HTML/CSS/JS im Browser)
- **Goldmark**: GFM-Support + Syntax-Highlighting
- **Inline HTML**: Gesamtes UI (Toolbar + Markdown-Bereich) als ein HTML-Template
- **Build Tags**: platform_linux.go / platform_windows.go für native Fullscreen-APIs

## Sitzungsverlauf
- 2026-03-05: Erstversion. Go 1.26 installiert. App implementiert mit WebView.
