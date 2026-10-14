# Behobene Sicherheitsprobleme

## Behoben ✅

### RISK-005: Skript-Injection über den Seitentitel (Build 40 – 2026-10-14)
**Problem**: Der Titel (erste `# `-Zeile der Datei) wurde unescaped in `<title>` eingesetzt. Eine Datei mit `# </title><script>…</script>` konnte Skript ausführen (CSP erlaubt `'unsafe-inline'`) und damit die Go-Bindings aufrufen.
**Fix**: `html.EscapeString()` auf den Titel in `BuildInitialHTML`; Tests `TestBuildInitialHTML_TitleEscaped` / `_TitleUmlautsUnchanged`.

**Datei**: `src/ui/template.go`

### RISK-001: HTML-Injection durch Markdown (Build 11 – 2026-03-08)
**Fix**: `html.WithUnsafe()` aus goldmark-Konfiguration entfernt. Eingebettetes HTML in Markdown wird jetzt escaped statt gerendert. `<script>`, `<iframe>` usw. können nicht mehr über MD-Dateien injiziert werden.

**Datei**: `src/renderer/markdown.go`

### RISK-002: WebView ohne Content-Security-Policy (Build 11 – 2026-03-08)
**Fix**: CSP-Meta-Tag in `htmlDocHead` eingefügt: `default-src 'none'; script-src 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: blob:; connect-src 'none';`

**Datei**: `src/ui/template.go`

### RISK-004: Path-Traversal via persistLastFile-Binding (Build 11 – 2026-03-08)
**Fix**: Pfad-Validierung im `persistLastFile`-Binding: Pfad wird nur gespeichert wenn `renderer.IsSupportedFile()` true zurückgibt und `os.Stat()` eine existierende Nicht-Verzeichnis-Datei bestätigt.

**Datei**: `src/bindings.go`
