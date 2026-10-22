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
- Tests: `src/renderer/*_test.go` + `src/ui/template_test.go` (alle grün)
- Assets: `src/ui/assets/styles.css`, `src/ui/assets/body.html`, `src/ui/assets/scripts.js` (go:embed)

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
- 2026-03-07: Multi-Format (EPUB, FB2, TXT, HTML), Config-Persistenz, TOC, Suche, Dark/Retro-Mode.
- 2026-03-08 (Build 8): Refactoring: ui/template.go aufgeteilt in styles.go, html_body.go, scripts.go, template.go. 15 UI-Tests hinzugefügt (31 gesamt).
- 2026-03-08 (Build 9): Bug #003 (letzte Drag-&-Drop-Datei nicht gespeichert) + Bug #004 (Scroll-Position nicht persistent) behoben. persistLastFile-Binding + text/uri-list-Extraktion.
- 2026-03-08 (Build 10): Linux + Windows Builds erstellt. CLAUDE.md-Pflichten erledigt. RISK-004 identifiziert (path traversal via persistLastFile). README.md vervollständigt (76 Tests).
- 2026-03-08 (Build 11): Security (RISK-001/002/004 behoben, RISK-003 mitigiert), go:embed (Assets als externe Dateien), bindings.go (aus main.go extrahiert), ARM64-Build, Mermaid.js-Integration. Build-Größe reduziert (-s -w). CSP-Header hinzugefügt.
- 2026-07-03 (Build 38): Bug #011 behoben – EPUB-Kapitellinks führten zu leerem Fenster. Interne Links werden zu Ankern umgeschrieben, Calibre-Body-IDs erhalten, JS-Sicherheitsnetz gegen relative Links. Release v1.0.39 enthält auch diesen Build.
- 2026-07-04 (Build 39): Bug #012 behoben – große EPUBs (17.epub, 4,3 MB HTML) ließen sich nicht öffnen. Ursachen: WebView2-2-MB-Limit bei NavigateToString (Fix: file://-Temp-Datei via ui/viewfile.go ab 1,5 MB) + Calibre-SVG-Cover nicht eingebettet (Fix: SVG <image xlink:href/href> in embedEpubImages). Release v1.0.39 mit allen Binaries veröffentlicht.
- 2026-10-14 (Build 40–43): Bug #013 (Inhalt im HTML-Kommentar → unformatiert, Start mitten im Text), Bug #014 (KaTeX lud nach dem Hauptskript), RISK-005 bis RISK-010: Seitentitel escaped, Whitelist-Bereiniger für EPUB/LaTeX, strikte CSP mit Nonce, externe Links im System-Browser (extlink), img-src nur data:, Bildpfade als Erlaubnisliste. GitHub-Pages-Workflow aktiv.
- 2026-10-22 (Build 44): RISK-011 (Gerätenamen-Umgehung bei Bildpfaden). Agent-Struktur eingeführt: 20 Agents in .claude/agents/ (lead-vorarbeiter, 6 specialists, 5 analysts, 3 worker, 5 helper), gemeinsame Regeln und überschneidungsfreie Zuständigkeiten in .claude/agents-shared/, Projektwissen aufgeteilt in .claude/rules/ (7 Dateien), Stop-/SessionStart-Hook check-project-structure.py prüft Eigentümer je Datei, Modell je Ebene, CLAUDE.md-Länge und Testzahl. Start als Vorarbeiter: `claude --agent lead-vorarbeiter`.
