# Sicherheitsrisiken

## Offen ⚠️

(Keine offenen Sicherheitsrisiken – Stand Build 44, 2026-10-22)

### Hinweis zu RISK-003
Die Bewertung „nur closeApp und nativeFullscreen“ ist veraltet: Es gibt inzwischen 10 Bindings, u. a. `loadNativeFile(path)`. Seit Build 41 schützen Bereinigung (EPUB/LaTeX) und strikte CSP (Nonce) davor, dass Dokumentinhalte sie aufrufen; seit Build 42 verlässt das App-Fenster die eigene Seite nicht mehr (RISK-008), fremde Websites erhalten die Bindings also nicht.

## Mitigiert / Behoben

### RISK-011: Umgehung der Gerätenamen-Sperre bei Bildpfaden (BEHOBEN – Build 44)
**Beschreibung**: `COM1 .png`, `CONIN$`, `COM¹` u. ä. umgingen die Gerätenamen-Prüfung (DoS durch blockierendes Lesen).

**Status**: Behoben – siehe SECURITY_FIXED.md.


### RISK-010: Umgehung der Netzwerkpfad-Erkennung bei Bildpfaden (BEHOBEN – Build 43)
**Beschreibung**: Sperrlisten-Prüfung für UNC-Pfade war unter Windows umgehbar (`\??\UNC\…`).

**Status**: Behoben – siehe SECURITY_FIXED.md.


### RISK-008: Externe Links öffneten fremde Websites IM App-Fenster (BEHOBEN – Build 42)
**Beschreibung**: Ein Klick auf einen http(s)-Link navigierte den WebView zur fremden Seite, die dort die Go-Bindings erhielt.

**Status**: Behoben – siehe SECURITY_FIXED.md.

### RISK-009: NTLM-Hash-Leck über file:-Ressourcen (BEHOBEN – Build 42)
**Beschreibung**: Gefunden vom automatischen Security-Review zu Build 41. `style`-Attribute in EPUBs durften `url(file://server/…)` enthalten, die CSP erlaubte `img-src file:`; Markdown-Bilder mit UNC-Pfad blieben als lokale Pfade stehen. Unter Windows löst das einen SMB-Zugriff aus, der den NTLM-Hash des Nutzers preisgeben kann.

**Status**: Behoben – siehe SECURITY_FIXED.md.


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
