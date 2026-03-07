# Sicherheitsrisiken

## Offen ⚠️

### RISK-001: HTML-Injection durch Markdown (Mittel)
**Beschreibung**: goldmark ist mit `html.WithUnsafe()` konfiguriert, was eingebettetes HTML in Markdown erlaubt. Böswillige MD-Dateien könnten JavaScript-Code enthalten.

**Risiko**: Mittel – Betrifft nur lokal geöffnete Dateien, kein Netzwerkzugriff.

**Empfehlung**: `html.WithUnsafe()` entfernen wenn HTML-Injection verhindert werden soll. Dann werden `<script>` und ähnliche Tags aus Markdown herausgefiltert.

### RISK-002: WebView ohne Content-Security-Policy (Niedrig)
**Beschreibung**: Der WebView lädt keine externen Ressourcen, aber es gibt keine explizite Content-Security-Policy (CSP).

**Risiko**: Niedrig – WebView lädt ohnehin keine URLs außer dem initialen HTML.

**Empfehlung**: CSP-Header über WebView-JavaScript setzen.

### RISK-003: Plattform-Bindings via JS (Niedrig)
**Beschreibung**: Die Go-Funktionen `closeApp` und `nativeFullscreen` sind als globale JS-Funktionen gebunden und könnten theoretisch von eingebettetem MD-HTML aufgerufen werden.

**Risiko**: Niedrig – `closeApp` schließt nur das Fenster, `nativeFullscreen` wechselt den Vollbild-Modus.

**Empfehlung**: Akzeptabel für lokale Markdown-Anzeige.
