// Tests für das HTML-Template der MD-Reader-Oberfläche.
//
// Prüft ob BuildInitialHTML ein gültiges, vollständiges HTML-Dokument
// mit korrekten Konfigurationswerten erzeugt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (⌘-Tastenkürzel für macOS)
package ui

import (
	"regexp"
	"strings"
	"testing"
)

// TestBuildInitialHTML_ContainsDoctype prüft ob das Dokument mit DOCTYPE beginnt.
func TestBuildInitialHTML_ContainsDoctype(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", IsPortrait: false})
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTML beginnt nicht mit DOCTYPE")
	}
}

// TestBuildInitialHTML_FontSize prüft ob die Schriftgröße korrekt eingebettet wird.
func TestBuildInitialHTML_FontSize(t *testing.T) {
	// Standardgröße
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, "--font-size: 16px") {
		t.Error("Schriftgröße 16px nicht im CSS gefunden")
	}
	if !strings.Contains(html, "var fontSize = 16") {
		t.Error("fontSize = 16 nicht im JavaScript gefunden")
	}
	if !strings.Contains(html, "var defaultFontSize = 16") {
		t.Error("defaultFontSize = 16 nicht im JavaScript gefunden")
	}

	// Andere Größe
	html32 := BuildInitialHTML(UIConfig{FontSize: 32, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html32, "--font-size: 32px") {
		t.Error("Schriftgröße 32px nicht im CSS gefunden")
	}
}

// TestBuildInitialHTML_DefaultFontSizeUnchanged prüft Bug-Fix: defaultFontSize bleibt immer 16,
// auch wenn fontSize auf 32 (200%) gespeichert wurde. Ohne diesen Fix würde der Zoom-Label
// beim Start immer "100%" anzeigen statt "200%".
func TestBuildInitialHTML_DefaultFontSizeUnchanged(t *testing.T) {
	// Simuliert gespeicherte 200%-Zoom-Einstellung (fontSize=32, Default=16)
	html := BuildInitialHTML(UIConfig{FontSize: 32, DefaultFontSize: 16, Theme: "light"})
	// fontSize muss 32 sein (gespeichert)
	if !strings.Contains(html, "var fontSize = 32") {
		t.Error("Zoom-Bug: fontSize = 32 fehlt im JavaScript")
	}
	// defaultFontSize muss 16 bleiben (konstanter Basis-Wert)
	if !strings.Contains(html, "var defaultFontSize = 16") {
		t.Error("Zoom-Bug: defaultFontSize = 16 fehlt – Zoom-Prozent würde falsch berechnet")
	}
	// defaultFontSize darf NICHT ebenfalls 32 sein
	if strings.Contains(html, "var defaultFontSize = 32") {
		t.Error("Zoom-Bug: defaultFontSize wurde auf 32 gesetzt – Zoom-Label würde 100%% statt 200%% zeigen")
	}
}

// TestBuildInitialHTML_ThemeLight prüft das Light-Theme (keine Body-Klasse).
func TestBuildInitialHTML_ThemeLight(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, `<body class="">`) {
		t.Error("Light-Theme: body class sollte leer sein")
	}
}

// TestBuildInitialHTML_ThemeDark prüft das Dark-Theme.
func TestBuildInitialHTML_ThemeDark(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "dark"})
	if !strings.Contains(html, `<body class="dark">`) {
		t.Error("Dark-Theme: body class='dark' fehlt")
	}
}

// TestBuildInitialHTML_ThemeRetro prüft das Retro-Theme.
func TestBuildInitialHTML_ThemeRetro(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "retro"})
	if !strings.Contains(html, `<body class="retro">`) {
		t.Error("Retro-Theme: body class='retro' fehlt")
	}
}

// TestBuildInitialHTML_PortraitMode prüft den Hochformat-Modus.
func TestBuildInitialHTML_PortraitMode(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", IsPortrait: true})
	if !strings.Contains(html, "var isPortrait = true") {
		t.Error("Hochformat-Modus: isPortrait = true fehlt im JavaScript")
	}
}

// TestBuildInitialHTML_LandscapeMode prüft den Querformat-Modus.
func TestBuildInitialHTML_LandscapeMode(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", IsPortrait: false})
	if !strings.Contains(html, "var isPortrait = false") {
		t.Error("Querformat-Modus: isPortrait = false fehlt im JavaScript")
	}
}

// TestBuildInitialHTML_ContainsToolbar prüft ob die Toolbar im HTML vorhanden ist.
func TestBuildInitialHTML_ContainsToolbar(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, `id="toolbar"`) {
		t.Error("Toolbar-Element fehlt im HTML")
	}
}

// TestBuildInitialHTML_ContainsTOCSidebar prüft ob die TOC-Seitenleiste vorhanden ist.
func TestBuildInitialHTML_ContainsTOCSidebar(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, `id="toc-sidebar"`) {
		t.Error("TOC-Seitenleiste fehlt im HTML")
	}
}

// TestBuildInitialHTML_ContainsSearchBar prüft ob die Suchleiste vorhanden ist.
func TestBuildInitialHTML_ContainsSearchBar(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, `id="search-bar"`) {
		t.Error("Suchleiste fehlt im HTML")
	}
}

// TestBuildInitialHTML_ContainsDropZone prüft ob die Drop-Zone vorhanden ist.
func TestBuildInitialHTML_ContainsDropZone(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, `id="drop-zone"`) {
		t.Error("Drop-Zone fehlt im HTML")
	}
}

// TestBuildInitialHTML_ContainsDarkCSS prüft ob Dark-Mode-CSS vorhanden ist.
func TestBuildInitialHTML_ContainsDarkCSS(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, "body.dark") {
		t.Error("Dark-Mode-CSS fehlt")
	}
}

// TestBuildInitialHTML_ContainsRetroCSS prüft ob Retro-Mode-CSS vorhanden ist.
func TestBuildInitialHTML_ContainsRetroCSS(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.Contains(html, "body.retro") {
		t.Error("Retro-Mode-CSS fehlt")
	}
}

// TestBuildInitialHTML_ClosingTags prüft ob das Dokument korrekt geschlossen wird.
func TestBuildInitialHTML_ClosingTags(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("HTML endet nicht mit </html>")
	}
	if !strings.Contains(html, "</body>") {
		t.Error("</body>-Tag fehlt")
	}
	if !strings.Contains(html, "</script>") {
		t.Error("</script>-Tag fehlt")
	}
}

// TestBuildInitialHTML_ContainsKaTeX prüft ob KaTeX-CSS und JS eingebettet werden.
func TestBuildInitialHTML_ContainsKaTeX(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	// KaTeX-CSS muss eingebettet sein (enthält typischerweise .katex-Klassen)
	if !strings.Contains(html, ".katex") {
		t.Error("KaTeX-CSS fehlt im HTML")
	}
	// KaTeX-JS muss eingebettet sein
	if !strings.Contains(html, "katex") {
		t.Error("KaTeX-JavaScript fehlt im HTML")
	}
	// Auto-Render muss eingebettet sein (renderMathInElement ist der Funktionsname)
	if !strings.Contains(html, "renderMathInElement") {
		t.Error("KaTeX Auto-Render (renderMathInElement) fehlt im HTML")
	}
	// CSP muss font-src data: erlauben (für eingebettete KaTeX-Schriften)
	if !strings.Contains(html, "font-src data:") {
		t.Error("CSP enthält font-src data: nicht – KaTeX-Schriften würden blockiert")
	}
	// Fonts müssen als base64 data URIs im CSS eingebettet sein
	if !strings.Contains(html, "data:font/woff2;base64,") {
		t.Error("KaTeX-Schriften sind nicht als base64 Data-URIs eingebettet")
	}
}

// TestBuildInitialHTML_PercentSigns prüft ob %% korrekt zu % wird.
//
// Alle '%%' im Template sollen durch fmt.Sprintf zu einfachem '%' werden,
// damit CSS-Werte wie '100%' korrekt ausgegeben werden.
func TestBuildInitialHTML_PercentSigns(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	// CSS-Wert 100% darf nicht als 100%% erscheinen
	if strings.Contains(html, "100%%") {
		t.Error("Doppeltes %% gefunden – fmt.Sprintf wurde nicht korrekt angewendet")
	}
	// CSS-Wert muss als einfaches % erscheinen
	if !strings.Contains(html, "100%") {
		t.Error("Einfaches % fehlt – CSS-Prozentwerte fehlen")
	}
}

// TestBuildInitialHTML_ContentInsertedOnce prüft, dass der Dokumentinhalt
// genau einmal eingefügt wird – und zwar im #content-Container.
//
// Bug #013: Ein HTML-Kommentar im Body-Template enthielt den Platzhalter
// {{CONTENT_HTML}}. Der Inhalt landete dadurch zusätzlich in einem Kommentar.
func TestBuildInitialHTML_ContentInsertedOnce(t *testing.T) {
	content := "<p>EINDEUTIGER-INHALT-4711</p>"
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", ContentHTML: content})
	if n := strings.Count(html, "EINDEUTIGER-INHALT-4711"); n != 1 {
		t.Fatalf("Inhalt %d-mal eingefügt, erwartet genau 1-mal", n)
	}
	if !strings.Contains(html, `<div id="content">`+content+`</div>`) {
		t.Error("Inhalt steht nicht im #content-Container")
	}
}

// TestBuildInitialHTML_ContentWithCommentEnd prüft den Sonderfall aus Bug #013:
// Enthält der Inhalt ein "-->" (z. B. goldmarks "<!-- raw HTML omitted -->"
// für <Platzhalter> im Fließtext), darf nichts davon vor #content erscheinen.
// Sonst rendert der Browser den Rest des Dokuments ungestylt oberhalb des
// eigentlichen Inhalts (Anzeige beginnt mitten im Text).
func TestBuildInitialHTML_ContentWithCommentEnd(t *testing.T) {
	content := "<p>Vorher <!-- raw HTML omitted --> NACH-KOMMENTAR</p>"
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "dark", ContentHTML: content})
	contentStart := strings.Index(html, `<div id="content">`)
	if contentStart < 0 {
		t.Fatal("#content-Container fehlt")
	}
	if strings.Contains(html[:contentStart], "NACH-KOMMENTAR") {
		t.Error("Inhalt erscheint vor #content – wird außerhalb des gestylten Bereichs gerendert")
	}
}

// TestBuildInitialHTML_EmptyContentPlaceholderRemoved prüft, dass ohne Inhalt
// (Drop-Zone) kein Platzhalter im Dokument übrig bleibt.
func TestBuildInitialHTML_EmptyContentPlaceholderRemoved(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if strings.Contains(html, "{{CONTENT_HTML}}") {
		t.Error("Platzhalter {{CONTENT_HTML}} nicht ersetzt")
	}
	if !strings.Contains(html, `<div id="content"></div>`) {
		t.Error("Leerer #content-Container fehlt")
	}
}

// TestBuildInitialHTML_TitleEscaped prüft, dass der Seitentitel HTML-escaped wird.
//
// Der Titel stammt aus der ersten "# "-Zeile der Datei. Unescaped könnte eine
// Datei mit "# </title><script>…" Skript ausführen (CSP erlaubt Inline-Skripte)
// und damit die Go-Bindings aufrufen.
func TestBuildInitialHTML_TitleEscaped(t *testing.T) {
	html := BuildInitialHTML(UIConfig{
		FontSize: 16, DefaultFontSize: 16, Theme: "light",
		ContentHTML: "<p>x</p>",
		PageTitle:   `</title><script>alert("x")</script> & "Mehr"`,
	})
	if strings.Contains(html, `<script>alert("x")</script>`) {
		t.Fatal("Skript aus dem Titel unescaped im Dokument")
	}
	want := `<title>&lt;/title&gt;&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt; &amp; &#34;Mehr&#34; – MD Reader</title>`
	if !strings.Contains(html, want) {
		t.Errorf("Titel nicht korrekt escaped, erwartet: %s", want)
	}
}

// TestBuildInitialHTML_TitleUmlautsUnchanged prüft, dass normale Titel mit
// Umlauten und Satzzeichen unverändert bleiben (Escaping nur für &<>"').
func TestBuildInitialHTML_TitleUmlautsUnchanged(t *testing.T) {
	html := BuildInitialHTML(UIConfig{
		FontSize: 16, DefaultFontSize: 16, Theme: "light",
		ContentHTML: "<p>x</p>",
		PageTitle:   "Offene Fragen (aufgeräumt 14.10.2026 08:03)",
	})
	if !strings.Contains(html, "<title>Offene Fragen (aufgeräumt 14.10.2026 08:03) – MD Reader</title>") {
		t.Error("Titel mit Umlauten wurde verändert")
	}
}

// cspNonceRe findet die Nonce in der script-src-Direktive der CSP.
var cspNonceRe = regexp.MustCompile(`script-src 'nonce-([A-Za-z0-9+/=_-]+)'`)

// extractCSP liefert den Inhalt des CSP-Meta-Tags.
func extractCSP(t *testing.T, html string) string {
	t.Helper()
	m := regexp.MustCompile(`http-equiv="Content-Security-Policy" content="([^"]*)"`).FindStringSubmatch(html)
	if m == nil {
		t.Fatal("CSP-Meta-Tag fehlt")
	}
	return m[1]
}

// extractDirective liefert eine einzelne CSP-Direktive (z. B. "script-src …").
func extractDirective(csp, name string) string {
	for _, d := range strings.Split(csp, ";") {
		d = strings.TrimSpace(d)
		if strings.HasPrefix(d, name+" ") {
			return d
		}
	}
	return ""
}

// TestBuildInitialHTML_CSPScriptSrcStrict prüft RISK-006/007 (CSP-Härtung):
// script-src erlaubt KEINE Inline-Skripte/Handler mehr, sondern nur Skripte mit
// Nonce und das Mermaid-Skript von genau einer CDN-Adresse.
func TestBuildInitialHTML_CSPScriptSrcStrict(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	scriptSrc := extractDirective(extractCSP(t, html), "script-src")
	if scriptSrc == "" {
		t.Fatal("script-src-Direktive fehlt")
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src erlaubt noch 'unsafe-inline': %s", scriptSrc)
	}
	if strings.Contains(scriptSrc, "'unsafe-eval'") {
		t.Errorf("script-src erlaubt 'unsafe-eval': %s", scriptSrc)
	}
	if !cspNonceRe.MatchString(scriptSrc) {
		t.Errorf("script-src enthält keine Nonce: %s", scriptSrc)
	}
	// Nur die exakte Mermaid-Datei, nicht der ganze CDN-Host
	for _, src := range strings.Fields(scriptSrc)[1:] {
		if strings.HasPrefix(src, "https:") && src != "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js" {
			t.Errorf("Zu weit gefasste Skriptquelle: %s", src)
		}
	}
}

// TestBuildInitialHTML_CSPExtraDirectives prüft die zusätzlichen Schutz-Direktiven.
func TestBuildInitialHTML_CSPExtraDirectives(t *testing.T) {
	csp := extractCSP(t, BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"}))
	for _, want := range []string{"default-src 'none'", "base-uri 'none'", "form-action 'none'", "frame-src 'none'", "connect-src 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP enthält %q nicht: %s", want, csp)
		}
	}
}

// TestBuildInitialHTML_AllScriptsHaveNonce prüft, dass jedes eigene <script>-Tag
// die Nonce aus der CSP trägt – sonst würde die App selbst blockiert.
func TestBuildInitialHTML_AllScriptsHaveNonce(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", ContentHTML: "<p>x</p>"})
	m := cspNonceRe.FindStringSubmatch(html)
	if m == nil {
		t.Fatal("Keine Nonce in der CSP")
	}
	nonce := m[1]
	tags := regexp.MustCompile(`<script[^>]*>`).FindAllString(html, -1)
	if len(tags) < 3 {
		t.Fatalf("Erwartet mindestens 3 Skript-Tags (App, KaTeX, Auto-Render), gefunden: %d", len(tags))
	}
	for _, tag := range tags {
		if tag != `<script nonce="`+nonce+`">` {
			t.Errorf("Skript-Tag ohne passende Nonce: %s", tag)
		}
	}
}

// TestBuildInitialHTML_NonceRandomPerPage prüft, dass jede Seite eine neue,
// ausreichend lange Zufalls-Nonce bekommt.
func TestBuildInitialHTML_NonceRandomPerPage(t *testing.T) {
	cfg := UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"}
	n1 := cspNonceRe.FindStringSubmatch(BuildInitialHTML(cfg))
	n2 := cspNonceRe.FindStringSubmatch(BuildInitialHTML(cfg))
	if n1 == nil || n2 == nil {
		t.Fatal("Nonce fehlt")
	}
	if n1[1] == n2[1] {
		t.Error("Zwei Seiten haben dieselbe Nonce")
	}
	if len(n1[1]) < 22 {
		t.Errorf("Nonce zu kurz (%d Zeichen, mind. 128 Bit erwartet)", len(n1[1]))
	}
}

// TestBuildInitialHTML_NoNoncePlaceholderLeft prüft, dass kein Platzhalter übrig bleibt.
func TestBuildInitialHTML_NoNoncePlaceholderLeft(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"})
	if strings.Contains(html, "{{") && strings.Contains(html, "NONCE") {
		t.Error("Nonce-Platzhalter nicht ersetzt")
	}
}

// TestBodyTemplate_NoInlineEventHandlers prüft, dass das eigene Body-Template
// keine Inline-Handler (onclick=… usw.) mehr enthält – die strikte CSP würde
// sie blockieren. Alle Handler werden per addEventListener gesetzt.
func TestBodyTemplate_NoInlineEventHandlers(t *testing.T) {
	if m := regexp.MustCompile(`(?i)\son[a-z]+\s*=`).FindAllString(htmlBodyHTML, -1); len(m) > 0 {
		t.Errorf("Inline-Event-Handler im Body-Template: %v", m)
	}
	if strings.Contains(strings.ToLower(htmlBodyHTML), "javascript:") {
		t.Error("javascript:-URL im Body-Template")
	}
}

// TestScripts_ButtonsWiredByListener prüft, dass alle früheren Inline-Handler
// per addEventListener verdrahtet werden (Schutz vor vergessenen Buttons).
func TestScripts_ButtonsWiredByListener(t *testing.T) {
	ids := []string{"btn-toc", "btn-open", "btn-layout", "btn-zoom-out", "btn-zoom-in",
		"btn-search", "btn-theme", "btn-fullscreen", "btn-close", "search-input",
		"search-prev", "search-next", "search-close", "toc-close"}
	for _, id := range ids {
		if !strings.Contains(htmlBodyHTML, `id="`+id+`"`) {
			t.Errorf("Element mit id=%q fehlt im Body-Template", id)
		}
		if !strings.Contains(htmlJavaScript, `'`+id+`'`) {
			t.Errorf("Element %q wird in scripts.js nicht verdrahtet", id)
		}
	}
}

// TestBuildInitialHTML_KaTeXLoadedBeforeMainScript prüft Bug #014: KaTeX muss
// VOR dem Hauptskript geladen werden. initEmbeddedContent() ruft initKaTeX()
// sofort auf – stand KaTeX dahinter, war renderMathInElement noch undefiniert
// und keine Formel wurde gerendert.
func TestBuildInitialHTML_KaTeXLoadedBeforeMainScript(t *testing.T) {
	html := BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light", ContentHTML: "<p>$x$</p>"})
	autoRender := strings.Index(html, "renderMathInElement=")
	if autoRender < 0 {
		autoRender = strings.Index(html, "renderMathInElement")
	}
	mainInit := strings.Index(html, "(function initEmbeddedContent()")
	if autoRender < 0 || mainInit < 0 {
		t.Fatalf("Marker nicht gefunden (autoRender=%d, mainInit=%d)", autoRender, mainInit)
	}
	if autoRender > mainInit {
		t.Error("KaTeX-Auto-Render wird erst nach dem Hauptskript geladen – Formeln bleiben ungerendert")
	}
}

// TestScripts_ExternalLinksOpenedNatively prüft RISK-008: Externe Links dürfen
// das App-Fenster nicht mehr selbst navigieren, sondern werden über das
// Go-Binding openExternal im System-Browser geöffnet. Mittelklicks auf Links
// werden ebenfalls abgefangen.
func TestScripts_ExternalLinksOpenedNatively(t *testing.T) {
	if !strings.Contains(htmlJavaScript, "window.openExternal(") {
		t.Error("scripts.js ruft window.openExternal nicht auf")
	}
	if strings.Contains(htmlJavaScript, "/^(https?:|mailto:|data:)/i.test(href)) return;") {
		t.Error("scripts.js lässt externe Links noch im App-Fenster navigieren")
	}
	if !strings.Contains(htmlJavaScript, "'auxclick'") {
		t.Error("Mittelklicks auf Links werden nicht abgefangen")
	}
}

// TestBuildInitialHTML_CSPImagesOnlyData prüft (Security-Review Build 41), dass
// Bilder nur noch als data:-URIs geladen werden dürfen. file: erlaubte z. B.
// url(file://server/…) → SMB-Zugriff unter Windows (NTLM-Hash-Leck).
func TestBuildInitialHTML_CSPImagesOnlyData(t *testing.T) {
	csp := extractCSP(t, BuildInitialHTML(UIConfig{FontSize: 16, DefaultFontSize: 16, Theme: "light"}))
	if got := extractDirective(csp, "img-src"); got != "img-src data:" {
		t.Errorf("img-src = %q, erwartet \"img-src data:\"", got)
	}
}

// TestScripts_CommandKeyWorksOnMac prüft, dass alle Tastenkürzel auch mit ⌘
// (metaKey, macOS) funktionieren: Der Handler fragt die Modifikatortaste nur
// über hasCommandModifier ab, nie mehr direkt über e.ctrlKey.
func TestScripts_CommandKeyWorksOnMac(t *testing.T) {
	if !regexp.MustCompile(`function hasCommandModifier\(e\)\s*\{\s*return e\.ctrlKey \|\| e\.metaKey;`).MatchString(htmlJavaScript) {
		t.Error("hasCommandModifier(e) mit ctrlKey || metaKey fehlt in scripts.js")
	}
	if strings.Contains(htmlJavaScript, "if (e.ctrlKey") {
		t.Error("scripts.js prüft e.ctrlKey noch direkt (⌘ unter macOS wirkungslos)")
	}
	for _, shortcut := range []string{
		"hasCommandModifier(e) && e.key === 'o'",
		"hasCommandModifier(e) && e.key === 'f'",
		"hasCommandModifier(e) && !inInput",
	} {
		if !strings.Contains(htmlJavaScript, shortcut) {
			t.Errorf("Tastenkürzel %q fehlt", shortcut)
		}
	}
}
