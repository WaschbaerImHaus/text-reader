# Research: MD Reader Optimierungen

## Was wir über das Projekt wissen

### Kernfunktion
Markdown-Betrachter für Desktop (Linux/Windows). Zielgruppe: Entwickler und technische Nutzer die Markdown-Dokumentation lesen möchten ohne GitHub-Browser-Zugang.

### Technische Basis
- WebView: Echter Browser (WebKit/Chromium-basiert) für pixelgenaues Rendering
- goldmark: Schnellster Go-Markdown-Parser, GFM-kompatibel
- Syntax-Highlighting: chroma library (selber Autor wie Pygments)

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

## Code-Qualität (Stand 2026-03-08)

### Architektur-Verbesserungen (Build 8)
- `ui/template.go` aufgeteilt: CSS (styles.go), HTML (html_body.go), JS (scripts.go)
- 15 neue UI-Tests → 31 Tests gesamt
- Jede Datei hat klaren Verantwortungsbereich

### Nächste Refactoring-Optionen
1. `main.go` + Bindings in eigene Datei auslagern (bindings.go)
2. CSS in externe .css Datei (go:embed) → Syntax-Highlighting im Editor
3. JS in externe .js Datei (go:embed) → Lint-Unterstützung

## Verbesserungsideen basierend auf Recherche
1. GitHub's exakten Primer-CSS direkt einbinden (MIT-Lizenz)
2. cmark-gfm via CGo für 100% GitHub-Kompatibilität
3. Datei-Watcher mit `fsnotify` für Auto-Reload
4. Mermaid.js für Diagramme in Markdown-Dateien
