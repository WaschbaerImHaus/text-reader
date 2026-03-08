# Sicherheitsrisiken

## Offen ⚠️

(Keine offenen Sicherheitsrisiken)

## Mitigiert / Behoben

### RISK-001: HTML-Injection durch Markdown (BEHOBEN – Build 11)
**Beschreibung**: goldmark war mit `html.WithUnsafe()` konfiguriert, was eingebettetes HTML in Markdown erlaubt. Böswillige MD-Dateien könnten JavaScript-Code enthalten.

**Status**: Behoben. `html.WithUnsafe()` wurde entfernt. `<script>`, `<iframe>` etc. aus Markdown werden jetzt escaped statt gerendert.

### RISK-002: WebView ohne Content-Security-Policy (BEHOBEN – Build 11)
**Beschreibung**: Der WebView lud keine externen Ressourcen, aber es gab keine explizite Content-Security-Policy (CSP).

**Status**: Behoben. CSP-Meta-Tag wurde in `htmlDocHead` eingefügt. Erlaubt nur: Inline-Scripts/Styles, CDN-Scripts von cdn.jsdelivr.net (für Mermaid.js), Data-URIs und Blob-URIs für Bilder.

### RISK-003: Plattform-Bindings via JS (MITIGIERT – Build 11)
**Beschreibung**: Die Go-Funktionen `closeApp` und `nativeFullscreen` sind als globale JS-Funktionen gebunden und könnten theoretisch von eingebettetem MD-HTML aufgerufen werden.

**Status**: Mitigiert. Durch RISK-001-Fix (html.WithUnsafe() entfernt) kann eingebettetes JavaScript in Markdown diese Bindings nicht mehr aufrufen. Das Risiko ist weiterhin akzeptabel für lokale Markdown-Anzeige, da closeApp nur das Fenster schließt und nativeFullscreen nur den Vollbild-Modus wechselt.

### RISK-004: Path-Traversal via persistLastFile-Binding (BEHOBEN – Build 11)
**Beschreibung**: Das Go-Binding `persistLastFile(path, scrollPos)` übernahm einen Dateipfad aus JavaScript ohne Validierung.

**Status**: Behoben. Pfad-Validierung im `persistLastFile`-Binding: Nur Pfade die `renderer.IsSupportedFile()` passieren und tatsächlich als existierende Datei (nicht Verzeichnis) bestätigt werden, werden als LastFile akzeptiert.
