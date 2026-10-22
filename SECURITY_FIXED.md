# Behobene Sicherheitsprobleme

## Behoben ✅

### RISK-011: Umgehung der Gerätenamen-Sperre bei Bildpfaden (Build 44 – 2026-10-22)
**Problem** (Security-Review zu Build 43): Windows entfernt Leerzeichen/Punkte am Segmentende – `COM1 .png` zeigt trotzdem auf die serielle Schnittstelle; `CONIN$`, `CONOUT$`, `CLOCK$` und hochgestellte Ziffern (`COM¹`) fehlten; nur der Dateiname, nicht die Verzeichnisse wurden geprüft. Folge: Lesen eines Geräts kann die App blockieren (DoS).
**Fix**: `isSafePathSegment` prüft jedes Pfadsegment: Segmente mit Leerzeichen/Punkt am Ende werden abgelehnt (außer `.`/`..`), Gerätename = Teil vor dem ersten Punkt ohne Leerzeichen am Ende; Sperrliste um `conin$`, `conout$`, `clock$`, `com¹²³`, `lpt¹²³` ergänzt.

**Datei**: `src/renderer/images.go`
**Tests**: `TestIsReadableImagePath_DeviceNameEvasionsRejected`

### RISK-010: Umgehung der Netzwerkpfad-Erkennung bei Bildpfaden (Build 43 – 2026-10-14)
**Problem** (Security-Review zu Build 42): `isNetworkPath` arbeitete als Sperrliste (`//`, `\\` …). Unter Windows führen weitere Schreibweisen ins Netzwerk, z. B. `\??\UNC\server\share\x.png` (Go hält ihn für absolut, Windows reicht ihn an SMB weiter) → NTLM-Hash-Leck. Für Markdown praktisch nicht ausnutzbar (goldmark kodiert Backslashes als `%5C`), aber nicht robust.
**Fix**: `isReadableImagePath` als plattformunabhängige Erlaubnisliste (fail closed): nur relative Pfade, Laufwerksbuchstabe mit Trenner (`C:\…`) oder einfacher Wurzel-Trenner; abgelehnt werden `\\…`, `//…`, `\??\…` (inkl. `\\?\`, `\\.\`), laufwerksrelative Pfade (`C:bild.png`), NTFS-Datenströme (`bild.png:x`) und reservierte Gerätenamen (`COM1.png`, `NUL.png` …).

**Datei**: `src/renderer/images.go`
**Tests**: `images_internal_test.go` (3 Tests, inkl. Windows-Fälle unter Linux)

### RISK-008: Externe Links öffnen fremde Websites im App-Fenster (Build 42 – 2026-10-14)
**Problem**: Ein Klick auf einen http(s)-Link navigierte das App-Fenster zur fremden Website; webview_go injiziert die Go-Bindings in jede Seite.
**Fix**: Neues Paket `extlink` (Typ `Opener`): strenge URL-Prüfung (nur http/https mit Host und mailto, keine Steuer-/Leer-/Anführungszeichen, max. 2048 Zeichen) und Start des Standardprogramms ohne Shell (`xdg-open` bzw. `rundll32 url.dll,FileProtocolHandler`). Neues Binding `openExternal`; der Link-Handler in `scripts.js` verhindert jede Navigation des App-Fensters (auch `data:`, `file:`, Mittelklick) und übergibt nur http/https/mailto an Go. Verifiziert in WebKitGTK mit protokollierendem Fake-`xdg-open`.

**Dateien**: `src/extlink/extlink.go`, `src/bindings.go`, `src/ui/assets/scripts.js`
**Tests**: `extlink_test.go` (8), `TestScripts_ExternalLinksOpenedNatively`

### RISK-009: NTLM-Hash-Leck über file:-Ressourcen (Build 42 – 2026-10-14)
**Problem** (Security-Review zu Build 41): `style="background:url(file://server/…)"` aus EPUBs und `img-src file:` in der CSP; Markdown-Bilder mit UNC-Pfad. Unter Windows → SMB-Zugriff auf fremden Server → NTLM-Hash kann abfließen.
**Fix**: (1) Bereiniger verwirft `style`-Werte mit `url`, `image`, `import`, `expression`, `behavior`, `binding`, `position`, Backslash (CSS-Escapes), `@` oder `/*`; einfache Stile bleiben. (2) CSP `img-src data:` (ohne `file:`/`blob:`). (3) `ResolveImagePaths` bettet jetzt auch absolute lokale Bildpfade ein (funktionieren dadurch überall), liest UNC-/Netzwerkpfade nie und bettet nur echte Bilddateien ein (feste Endungstabelle; vorher wurde jede Datei als „Bild“ eingebettet).

**Dateien**: `src/renderer/sanitize.go`, `src/renderer/images.go`, `src/ui/template.go`
**Tests**: `TestSanitizeHTML_StyleWithResourceLoadsRemoved`, `TestSanitizeHTML_SimpleStylesKept`, `TestResolveImagePaths{AbsoluteLocalFileEmbedded,UNCNeverRead,OnlyImageTypes}`, `TestBuildInitialHTML_CSPImagesOnlyData`

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
