# Behobene Sicherheitsprobleme

## Behoben ✅

### RISK-006/007: Skript-Ausführung über EPUB- und LaTeX-Dateien (Build 41 – 2026-10-14)
**Problem**: Präparierte EPUB-Dateien (Event-Attribute, `javascript:`-Links, `<iframe srcdoc>`, `<meta refresh>`) und LaTeX-Dateien (ungefilterter Fließtext, `\href{javascript:…}`) konnten JavaScript im Viewer ausführen und die Go-Bindings aufrufen. Nachgewiesen in Chrome-Headless (Build 40: `onerror` lief).
**Fix (zweistufig)**:
1. **Bereinigung**: Neuer Whitelist-Bereiniger `renderer.SanitizeHTML` (Tokenizer `golang.org/x/net/html`): nur bekannte Elemente/Attribute, Links nur http/https/mailto/#/relativ (Prüfung nach Entfernen von Steuerzeichen, Entities werden vorher entschlüsselt), Bilder nur `data:image/…`, gefährliche Container (script, style, iframe, object, template, foreignObject …) samt Inhalt entfernt, Kommentare verworfen, Text und Attribute neu escaped. Angewendet auf das gesamte EPUB-HTML und die LaTeX-Ausgabe; LaTeX-Formeln werden zusätzlich HTML-escaped, Makros als nicht ausführbarer JSON-Datenblock übergeben.
2. **Strikte CSP**: `script-src` ohne `'unsafe-inline'`, stattdessen Zufalls-Nonce (128 Bit) je Seite für die eigenen Skripte und exakt `https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js`; zusätzlich `base-uri 'none'`, `form-action 'none'`, `frame-src 'none'`. Alle Inline-Handler (`onclick=` …) im Template durch `addEventListener` ersetzt. Eingeschleuste Handler werden damit auch dann blockiert, wenn die Bereinigung etwas übersehen sollte (in Chrome nachgewiesen).

**Dateien**: `src/renderer/sanitize.go`, `src/renderer/epub.go`, `src/renderer/latex.go`, `src/ui/template.go`, `src/ui/assets/body.html`, `src/ui/assets/scripts.js`
**Tests**: `sanitize_test.go` (17), `TestParseEpubMalicious*`, `TestParseLaTeX{RawHTMLSanitized,JavascriptLinksRemoved,MathEscaped,MacrosJSONEscaped}`, `TestBuildInitialHTML_{CSPScriptSrcStrict,CSPExtraDirectives,AllScriptsHaveNonce,NonceRandomPerPage}`, `TestBodyTemplate_NoInlineEventHandlers`, `TestScripts_ButtonsWiredByListener`

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
