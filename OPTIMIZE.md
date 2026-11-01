# Optimierungsvorschläge

## Offen aus Build 47

- **Vier ältere `viewfile`-Tests** schreiben ins echte Temp-Verzeichnis; sie sollten mit der Hilfsfunktion `useIsolatedTempDir` in ein `t.TempDir()` gelenkt werden (Befund der Gegenprüfung 2026-11-01).
- **`make all` / `deb` / `release`**: Früher scheiterte der Sammelaufruf, weil sich die Host-Pakete `:arm64`/`:armhf`/`:amd64` ausschließen. Mit dem Container-Build ist das Hindernis entfallen: Die GUI-Ziele laufen in Containern, die Images entstehen einmalig mit `make build-images`. Stand 2026-11-01: Die Makefile-Struktur (`all: increment-build test linux windows linux-arm64 linux-armhf installer deb`) erlaubt einen Aufruf; ein vollständiger `make all`-Lauf wurde in dieser Dokumentationsrunde nicht ausgeführt. `release` hängt von `all` ab und erhöht die Build-Nummer dabei erneut – vor dem Nutzen prüfen.
- **Auto-Update** (Build 48 geplant, siehe FEATURES.md).
- Kleine Reste im Starter: Meldung ohne Display nur deutsch; zwei gleichzeitig gestartete Instanzen können zwei Installationsfragen zeigen; `spawnOfferProcess` ohne echten Test.

## Performance

- **Lazy Loading**: Große Markdown-Dateien schrittweise rendern statt alles auf einmal
- **Caching**: Gerendertes HTML im Speicher cachen um Re-Rendering zu vermeiden
- **Web Worker**: Markdown-Konvertierung in einem Web Worker (JS-Thread) auslagern
- **Streaming**: Sehr große Dateien zeilenweise verarbeiten

## Architektur

- ~~**Template Engine**: HTML-Template auslagern in separate .html Datei (go:embed)~~ ✅ *Build 8: In 4 Go-Dateien aufgeteilt (styles.go, html_body.go, scripts.go, template.go)*
- **go:embed**: CSS/JS/HTML als externe Dateien einbinden (go:embed) für bessere IDE-Unterstützung (Syntax-Highlighting)
- **Config-Datei**: Benutzereinstellungen in `~/.config/md-reader/config.json` speichern
- **Plugin-System**: Erweiterbare Markdown-Renderer-Plugins
- **IPC-Protokoll**: Strukturiertes Nachrichtenprotokoll zwischen Go und JS

## UI/UX

- **Animations**: Sanfte Übergänge beim Zoom (CSS transition)
- **Responsive Toolbar**: Toolbar auf kleinen Fenstern kompakter darstellen
- **Drag-Feedback**: Animierter Drop-Indikator wenn Datei über Fenster gehalten wird
- **Scroll-Position**: Scroll-Position beim Reload beibehalten (bei Dateiänderungen)
- **Tastaturnavigation**: Vollständige Tastatursteuerung ohne Maus

## Dateigröße

- **Stripped Binary**: `go build -ldflags="-s -w"` für kleinere Binaries
- **UPX Compression**: Nachgelagerte UPX-Komprimierung der Binaries
- ~~**Statisches Linking**~~ ✅ *Build 47: Starter statisch (CGO aus), GUI mit statischem libstdc++/libgcc und glibc-Wächter*

## Windows-spezifisch

- **WebView2 Bootstrapper**: Automatische WebView2-Installation wenn nicht vorhanden
- **Installer**: NSIS- oder MSI-Installer für Windows erstellen
- **Windows-Icon**: Anwendungsicon für Windows-Binary (.ico)

## Sicherheit

- **Content Security Policy**: CSP-Header für den WebView-Inhalt setzen
- **Sandboxing**: Webview-Navigations-Einschränkungen (keine externen URLs laden)
- **Input-Sanitization**: HTML-Injection aus unsicherem Markdown prüfen (RISK-001, goldmark `html.WithUnsafe()`)
- **Binding-Validierung**: persistLastFile-Binding Pfad-Validierung gegen path traversal (RISK-004)

## Cross-Platform

- ~~**ARM-Builds Linux**~~ ✅ *Build 47: arm64/armhf in Containern*; Windows ARM64 weiterhin nicht baubar (kein mingw-Compiler)
- ~~**macOS**: Build-Config fehlt~~ ✅ *Build 46: Workflow `macos.yml`*
