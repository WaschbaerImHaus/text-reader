# Optimierungsvorschläge

## Performance

- **Lazy Loading**: Große Markdown-Dateien schrittweise rendern statt alles auf einmal
- **Caching**: Gerendertes HTML im Speicher cachen um Re-Rendering zu vermeiden
- **Web Worker**: Markdown-Konvertierung in einem Web Worker (JS-Thread) auslagern
- **Streaming**: Sehr große Dateien zeilenweise verarbeiten

## Architektur

- **Template Engine**: HTML-Template auslagern in separate .html Datei (go:embed)
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
- **Statisches Linking**: Statisch gelinkte Binaries für bessere Portabilität

## Windows-spezifisch

- **WebView2 Bootstrapper**: Automatische WebView2-Installation wenn nicht vorhanden
- **Installer**: NSIS- oder MSI-Installer für Windows erstellen
- **Windows-Icon**: Anwendungsicon für Windows-Binary (.ico)

## Sicherheit

- **Content Security Policy**: CSP-Header für den WebView-Inhalt setzen
- **Sandboxing**: Webview-Navigations-Einschränkungen (keine externen URLs laden)
- **Input-Sanitization**: HTML-Injection aus unsicherem Markdown prüfen
