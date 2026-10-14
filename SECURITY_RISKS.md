# Sicherheitsrisiken

## Offen ⚠️

### RISK-008: Externe Links öffnen fremde Websites IM App-Fenster (MITTEL – gefunden 2026-10-14)
**Beschreibung**: Ein Klick auf einen `http(s)`-Link im Dokument navigiert den WebView selbst zur fremden Seite (`scripts.js` lässt externe Links bewusst durch). webview_go injiziert die Go-Bindings in JEDE geladene Seite – die fremde Website könnte also `loadNativeFile`, `persistState`, `openFilePicker` usw. aufrufen. Die CSP des Viewers gilt dort nicht.
**Auswirkung**: Erfordert einen bewussten Klick des Nutzers auf einen Link. Danach kann die Website z. B. Einstellungen verändern oder lokale Dateien in der App anzeigen lassen (aber nicht direkt auslesen: `connect-src` gilt nur auf der eigenen Seite, die Website selbst hat keinen Dateizugriff).
**Lösungsweg**: Externe Links per neuem Binding `openExternal(url)` im System-Browser öffnen (Schema-Prüfung http/https/mailto; Linux `xdg-open`, Windows `rundll32 url.dll,FileProtocolHandler`) und die Navigation im WebView verhindern.

### Hinweis zu RISK-003
Die Bewertung „nur closeApp und nativeFullscreen“ ist veraltet: Es gibt inzwischen 10 Bindings, u. a. `loadNativeFile(path)`. Seit Build 41 schützen Bereinigung (EPUB/LaTeX) und strikte CSP (Nonce) davor, dass Dokumentinhalte sie aufrufen. Offen bleibt nur der Weg über fremde Websites (RISK-008).

## Mitigiert / Behoben

### RISK-006: Skript-Ausführung durch präparierte EPUB-Dateien (BEHOBEN – Build 41)
**Beschreibung**: EPUB-Kapitel wurden nur von `<script>`/`<style>`/`<link>` befreit; Event-Attribute (`onerror=` …), `javascript:`-Links, `<iframe srcdoc>` und `<meta refresh>` gelangten in die Seite und wurden ausgeführt (CSP erlaubte `'unsafe-inline'`).

**Status**: Behoben – siehe SECURITY_FIXED.md.

### RISK-007: HTML-/Skript-Injection durch LaTeX-Dateien (BEHOBEN – Build 41)
**Beschreibung**: Der LaTeX-Parser übernahm Fließtext und Formeln ohne Escaping; `\href{javascript:…}` wurde zum ausführbaren Link.

**Status**: Behoben – siehe SECURITY_FIXED.md.


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
