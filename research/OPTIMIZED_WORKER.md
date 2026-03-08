# Research: MD Reader Optimierungen

**Stand: 2026-03-08 (Build 10)**

## Was wir über das Projekt wissen

### Kernfunktion
Markdown-Betrachter für Desktop (Linux/Windows). Zielgruppe: Entwickler und technische Nutzer die Markdown-Dokumentation lesen möchten ohne GitHub-Browser-Zugang.

### Technische Basis
- WebView: Echter Browser (WebKit/Chromium-basiert) für pixelgenaues Rendering
- goldmark: Schnellster Go-Markdown-Parser, GFM-kompatibel
- Syntax-Highlighting: chroma library (selber Autor wie Pygments)

### Aktueller Funktionsstand (Build 10)
- Multi-Format: MD, EPUB, FB2, TXT, HTML
- Config-Persistenz: Zoom, Theme (Hell/Dark/Retro), Layout, letzte Datei, Scroll-Position
- TOC-Seitenleiste, Volltext-Suche
- Drag & Drop mit Pfad-Extraktion via text/uri-list
- Plattformbindings: Linux (GTK), Windows (WinAPI)
- 76 Tests (renderer + ui)

## Recherche-Ergebnisse

### Markdown-Rendering-Alternativen
1. **goldmark** (aktuell): Schnell, GFM-Support, erweiterbar
2. **blackfriday**: Älterer Standard, weniger GFM-Features
3. **cmark-gfm**: C-Bibliothek, offiziell von GitHub, via CGo
4. **Pandoc** (via subprocess): Universalkonverter, aber externe Abhängigkeit

→ goldmark ist die beste Wahl für Go-native Lösungen

### WebView-Alternativen
1. **webview_go** (aktuell): Minimalistisch, gut für einfache Apps
2. **Wails**: Vollständiges Framework, komplexer aber mehr Features
3. **Lorca**: Chrome-basiert, Chrome muss installiert sein
4. **go-astilectron**: Electron-basiert, große Binary

→ webview_go optimal für einfachen Markdown-Betrachter

### GitHub Markdown CSS
GitHub nutzt "Primer" Design-System. Die wichtigsten CSS-Eigenschaften:
- Font: `-apple-system, BlinkMacSystemFont, "Segoe UI"`
- Code-Font: `ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas`
- Base font-size: 16px
- Line-height: 1.5 (Fließtext), 1.25 (Überschriften)
- Max content width: ~800px bei GitHub

### Brainstorming: Verbindungen zum Projekt
- **Accessibility (a11y)**: Markdown-Reader sollte Screen-Reader-kompatibel sein
- **Offline-First**: Lokale Anzeige ohne Internet ist Kernvorteil
- **Portable Apps**: Statisch gelinkte Version für USB-Sticks?
- **Bildschirmschoner**: Markdown als Präsentation/Slideshow?
- **Git-Integration**: Automatisches Öffnen von README.md aus Git-Repos?
- **Datei-Watcher**: fsnotify für Auto-Reload bei Dateiänderungen (Autor-Workflow)
- **Mermaid.js**: Diagramme direkt in Markdown-Dateien rendern
- **KaTeX**: Mathematische Formeln in Markdown (LaTeX-Syntax)

## Architektur (Stand Build 10)

### Datei-Aufteilung
```
src/
├── main.go              # WebView-Setup, Go-JS-Bindings, Startup-Logik
├── config.go            # AppConfig, loadConfig, saveConfig
├── platform_linux.go    # GTK-Vollbild (CGo, Build-Tag: linux)
├── platform_windows.go  # WinAPI-Vollbild (Build-Tag: windows)
├── renderer/
│   ├── markdown.go      # goldmark MD→HTML, TXT→HTML, HTML-Parsing
│   ├── epub.go          # EPUB ZIP+XHTML-Parser
│   ├── fb2.go           # FictionBook XML-Parser
│   └── images.go        # Lokale Bildpfad → Base64 Data-URI
└── ui/
    ├── template.go      # BuildInitialHTML (fügt Teile zusammen)
    ├── styles.go        # CSS (htmlCSS)
    ├── html_body.go     # HTML-Grundstruktur (htmlBodyHTML)
    └── scripts.go       # JavaScript-Logik (htmlJavaScript)
```

### Nächste Refactoring-Optionen
1. `main.go` Bindings in eigene Datei auslagern (`bindings.go`)
2. CSS in externe .css Datei (go:embed) → Syntax-Highlighting im Editor
3. JS in externe .js Datei (go:embed) → Lint-Unterstützung
4. Integration Tests für vollständige Go→JS-Bindings

## Sicherheits-Analyse
- RISK-001: html.WithUnsafe() in goldmark → eingebettetes HTML erlaubt
- RISK-002: Kein CSP-Header im WebView
- RISK-003: Go-Bindings (closeApp, nativeFullscreen) als globale JS-Funktionen
- **RISK-004 (NEU)**: persistLastFile-Binding akzeptiert beliebige Pfade aus JS → path traversal möglich bei manipulierten MD-Dateien

## Verbesserungsideen (priorisiert)

### Hoch (Sicherheit)
1. RISK-001 beheben: html.WithUnsafe() deaktivieren oder mit allowlist arbeiten
2. RISK-004 beheben: Pfad-Validierung in persistLastFile-Binding

### Mittel (Features)
3. Datei-Watcher mit `fsnotify` für Auto-Reload (produktiv beim Schreiben)
4. Tab-Leiste für mehrere gleichzeitige Dateien
5. Verlauf mehrerer zuletzt geöffneter Dateien

### Niedrig (Qualität)
6. Stripped Binary (`-ldflags="-s -w"`) für kleinere Builds
7. go:embed für CSS/JS für bessere IDE-Unterstützung
8. Mermaid.js für Diagramme
