// Tests für den Text-Modus (C64-ähnliche Monospace-Ansicht).
//
// Prüft die Farbprüfung, die Normalisierung der Einstellungen und wie
// BuildInitialHTML den Modus in die Seite einbaut (Body-Klasse, Farben,
// eingebettete Schrift, Bedienelemente, Skript mit Nonce).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24
package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestIsHexColor prüft, dass nur exakt "#rrggbb" als Farbe gilt.
// Alles andere könnte als CSS- oder JS-Code in die Seite gelangen.
func TestIsHexColor(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"#40318d", true},
		{"#ABCDEF", true},
		{"#000000", true},
		{"#ffffff", true},
		{"", false},
		{"#fff", false},       // Kurzform nicht erlaubt
		{"#1234567", false},   // zu lang
		{"40318d", false},     // ohne Raute
		{"#gggggg", false},    // keine Hex-Ziffern
		{"red", false},        // Farbname
		{" #40318d", false},   // Leerzeichen vorne
		{"#40318d ", false},   // Leerzeichen hinten
		{"#40318d\n", false},  // Zeilenumbruch hinten
		{"#123456;}", false},  // CSS-Ausbruch
		{"#123456'", false},   // JS-String-Ausbruch
		{"rgb(1,2,3)", false}, // andere Schreibweise
		{"#12345٠", false},    // arabisch-indische Ziffer
		{"＃123456", false},    // Vollbreiten-Raute
	}
	for _, tt := range tests {
		if got := IsHexColor(tt.value); got != tt.want {
			t.Errorf("IsHexColor(%q) = %v, erwartet %v", tt.value, got, tt.want)
		}
	}
}

// TestNewTextMode_ValidColorsLowercased prüft, dass gültige Farben übernommen
// und einheitlich klein geschrieben werden.
func TestNewTextMode_ValidColorsLowercased(t *testing.T) {
	m := NewTextMode(true, "#FFAA00", "#0000Cc")
	if !m.Enabled {
		t.Error("Enabled ging verloren")
	}
	if m.Foreground != "#ffaa00" {
		t.Errorf("Foreground = %q, erwartet #ffaa00", m.Foreground)
	}
	if m.Background != "#0000cc" {
		t.Errorf("Background = %q, erwartet #0000cc", m.Background)
	}
}

// TestNewTextMode_InvalidColorsFallBack prüft, dass ungültige Farben durch die
// C64-Standardfarben ersetzt werden – je Farbe einzeln.
func TestNewTextMode_InvalidColorsFallBack(t *testing.T) {
	m := NewTextMode(false, "red;}", "")
	if m.Foreground != DefaultTextModeForeground {
		t.Errorf("Foreground = %q, erwartet Standard %q", m.Foreground, DefaultTextModeForeground)
	}
	if m.Background != DefaultTextModeBackground {
		t.Errorf("Background = %q, erwartet Standard %q", m.Background, DefaultTextModeBackground)
	}
	// Nur die ungültige Farbe wird ersetzt, die gültige bleibt
	m = NewTextMode(false, "#123456", "kaputt")
	if m.Foreground != "#123456" || m.Background != DefaultTextModeBackground {
		t.Errorf("Teil-Fallback falsch: %+v", m)
	}
}

// TestNewTextMode_DefaultsAreC64Colors prüft die Startfarben: Hellblau auf Blau.
func TestNewTextMode_DefaultsAreC64Colors(t *testing.T) {
	if DefaultTextModeForeground != "#7869c4" {
		t.Errorf("Standard-Vordergrund = %q, erwartet C64-Hellblau #7869c4", DefaultTextModeForeground)
	}
	if DefaultTextModeBackground != "#40318d" {
		t.Errorf("Standard-Hintergrund = %q, erwartet C64-Blau #40318d", DefaultTextModeBackground)
	}
	if !IsHexColor(DefaultTextModeForeground) || !IsHexColor(DefaultTextModeBackground) {
		t.Error("Standardfarben sind selbst keine gültigen Farben")
	}
}

// bodyClasses liest die Klassen des <body>-Tags aus der Seite.
func bodyClasses(t *testing.T, html string) []string {
	t.Helper()
	m := regexp.MustCompile(`</head>\s*<body class="([^"]*)">`).FindStringSubmatch(html)
	if m == nil {
		t.Fatal("<body class=…> nicht gefunden")
	}
	return strings.Fields(m[1])
}

// hasTextModeClass prüft, ob der Body die Klasse "textmode" trägt.
func hasTextModeClass(t *testing.T, html string) bool {
	t.Helper()
	for _, c := range bodyClasses(t, html) {
		if c == "textmode" {
			return true
		}
	}
	return false
}

// textModeConfig liefert eine UIConfig mit Inhalt und eingeschaltetem Text-Modus.
func textModeConfig() UIConfig {
	return UIConfig{
		FontSize: 16, DefaultFontSize: 16, Theme: "light",
		ContentHTML:       "<h1>Titel</h1><p>Text</p>",
		TextMode:          NewTextMode(true, "#112233", "#445566"),
		TextModeAvailable: true,
	}
}

// TestBuildInitialHTML_TextModeActiveBodyClass prüft, dass die Seite bei
// eingeschaltetem und verfügbarem Modus die Body-Klasse "textmode" trägt.
func TestBuildInitialHTML_TextModeActiveBodyClass(t *testing.T) {
	html := BuildInitialHTML(textModeConfig())
	if !strings.Contains(html, `<body class="textmode">`) {
		t.Error(`Body-Klasse "textmode" fehlt bei aktivem Text-Modus`)
	}
}

// TestBuildInitialHTML_TextModeKeepsTheme prüft, dass Theme und Text-Modus
// zusammen als Klassen am Body stehen.
func TestBuildInitialHTML_TextModeKeepsTheme(t *testing.T) {
	cfg := textModeConfig()
	cfg.Theme = "dark"
	html := BuildInitialHTML(cfg)
	if !strings.Contains(html, `<body class="dark textmode">`) {
		t.Error(`Body-Klasse "dark textmode" fehlt`)
	}
}

// TestBuildInitialHTML_TextModeDisabled prüft, dass ein ausgeschalteter Modus
// keine Klasse setzt (bisheriges Verhalten unverändert).
func TestBuildInitialHTML_TextModeDisabled(t *testing.T) {
	cfg := textModeConfig()
	cfg.TextMode.Enabled = false
	html := BuildInitialHTML(cfg)
	if hasTextModeClass(t, html) {
		t.Error("Text-Modus aktiv, obwohl ausgeschaltet")
	}
	if !strings.Contains(html, "var textModeEnabled = false;") {
		t.Error("JS-Variable textModeEnabled = false fehlt")
	}
}

// TestBuildInitialHTML_TextModeUnavailable prüft PDF/PS: Der Modus bleibt
// gespeichert eingeschaltet, wirkt aber nicht, und die Schrift wird nicht
// eingebettet (spart rund 160 KB je PDF-Seite).
func TestBuildInitialHTML_TextModeUnavailable(t *testing.T) {
	cfg := textModeConfig()
	cfg.TextModeAvailable = false
	html := BuildInitialHTML(cfg)
	if hasTextModeClass(t, html) {
		t.Error("Text-Modus aktiv, obwohl für dieses Format nicht verfügbar")
	}
	if !strings.Contains(html, "var textModeAvailable = false;") {
		t.Error("JS-Variable textModeAvailable = false fehlt")
	}
	if !strings.Contains(html, "var textModeEnabled = true;") {
		t.Error("gespeicherte Einstellung textModeEnabled = true ging verloren")
	}
	if strings.Contains(html, textModeFontFamily) {
		t.Error("Text-Modus-Schrift eingebettet, obwohl nicht verfügbar")
	}
}

// TestBuildInitialHTML_TextModeUnavailableWithoutContent prüft die Drop-Zone:
// ohne Dokument ist der Modus nicht verfügbar.
func TestBuildInitialHTML_TextModeUnavailableWithoutContent(t *testing.T) {
	cfg := textModeConfig()
	cfg.ContentHTML = ""
	html := BuildInitialHTML(cfg)
	if !strings.Contains(html, "var textModeAvailable = false;") {
		t.Error("Text-Modus ohne Inhalt als verfügbar markiert")
	}
	if hasTextModeClass(t, html) {
		t.Error("Text-Modus ohne Inhalt aktiv")
	}
}

// TestBuildInitialHTML_TextModeColors prüft, dass die gewählten Farben als
// CSS-Variablen, JS-Werte und Startwerte der Farbfelder in der Seite stehen.
func TestBuildInitialHTML_TextModeColors(t *testing.T) {
	html := BuildInitialHTML(textModeConfig())
	for _, want := range []string{
		"--tm-fg: #112233;",
		"--tm-bg: #445566;",
		"var textModeForeground = '#112233';",
		"var textModeBackground = '#445566';",
		`id="tm-fg" value="#112233"`,
		`id="tm-bg" value="#445566"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("%q fehlt in der Seite", want)
		}
	}
}

// TestBuildInitialHTML_TextModeColorInjection prüft die Gegenprobe: Auch wenn
// eine UIConfig ungeprüfte Farben enthält (z. B. aus einer manipulierten
// state.json), landen sie nicht in CSS oder JS – BuildInitialHTML normalisiert selbst.
func TestBuildInitialHTML_TextModeColorInjection(t *testing.T) {
	cfg := textModeConfig()
	cfg.TextMode = TextMode{
		Enabled:    true,
		Foreground: "red;}</style><script>alert(1)</script>",
		Background: "#000000'; alert(1); '",
	}
	html := BuildInitialHTML(cfg)
	if strings.Contains(html, "alert(1)") {
		t.Fatal("ungeprüfte Farbe in die Seite übernommen")
	}
	if !strings.Contains(html, "--tm-fg: "+DefaultTextModeForeground+";") {
		t.Error("ungültige Vordergrundfarbe nicht durch Standard ersetzt")
	}
	if !strings.Contains(html, "--tm-bg: "+DefaultTextModeBackground+";") {
		t.Error("ungültige Hintergrundfarbe nicht durch Standard ersetzt")
	}
}

// TestBuildInitialHTML_TextModeFontEmbedded prüft, dass die Pixelschrift als
// data:-URI eingebettet wird (CSP font-src data:, offline nutzbar).
func TestBuildInitialHTML_TextModeFontEmbedded(t *testing.T) {
	html := BuildInitialHTML(textModeConfig())
	if !strings.Contains(html, `font-family: "`+textModeFontFamily+`"`) {
		t.Error("@font-face der Text-Modus-Schrift fehlt")
	}
	if !strings.Contains(html, "url(data:font/ttf;base64,") {
		t.Error("Schrift nicht als data:-URI eingebettet")
	}
	if strings.Contains(html, "PressStart2P-Regular.ttf") {
		t.Error("Schrift wird über einen Dateinamen statt eingebettet geladen")
	}
}

// TestTextModeFontFaceCSS_ValidBase64 prüft, dass die eingebettete Schrift
// vollständig ist (TrueType-Signatur am Anfang der dekodierten Daten).
func TestTextModeFontFaceCSS_ValidBase64(t *testing.T) {
	css := TextModeFontFaceCSS()
	m := regexp.MustCompile(`base64,([A-Za-z0-9+/=]+)\)`).FindStringSubmatch(css)
	if m == nil {
		t.Fatal("kein base64-Block in der @font-face-Regel")
	}
	if !strings.HasPrefix(m[1], "AAEAAA") { // 00 01 00 00 = TrueType
		t.Errorf("eingebettete Daten sind keine TrueType-Schrift: %.12s…", m[1])
	}
	if len(m[1]) < 100000 {
		t.Errorf("eingebettete Schrift zu klein (%d Zeichen) – abgeschnitten?", len(m[1]))
	}
}

// TestTextModeFontLicenseShipped prüft, dass die OFL-Lizenz neben der Schrift
// liegt (Pflicht laut SIL Open Font License beim Weitergeben).
func TestTextModeFontLicenseShipped(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("assets", "fonts", "OFL.txt"))
	if err != nil {
		t.Fatalf("OFL.txt fehlt: %v", err)
	}
	if !strings.Contains(string(data), "SIL Open Font License") {
		t.Error("OFL.txt enthält nicht die SIL Open Font License")
	}
}

// TestBodyTemplate_TextModeControls prüft Knopf und Farbfelder in der Toolbar.
func TestBodyTemplate_TextModeControls(t *testing.T) {
	for _, want := range []string{
		`id="btn-textmode"`,
		`id="tm-colors"`,
		`<input type="color" id="tm-fg"`,
		`<input type="color" id="tm-bg"`,
		`id="tm-filters"`,
	} {
		if !strings.Contains(htmlBodyHTML, want) {
			t.Errorf("%q fehlt im Body-Template", want)
		}
	}
}

// TestScripts_TextModeWired prüft, dass Knopf und Farbfelder per Listener
// verdrahtet sind (CSP verbietet Inline-Handler) und Go-seitig gespeichert wird.
func TestScripts_TextModeWired(t *testing.T) {
	if !strings.Contains(htmlJavaScript, "['btn-textmode', toggleTextMode]") {
		t.Error("btn-textmode wird in initUIListeners nicht verdrahtet")
	}
	for _, want := range []string{"'tm-fg'", "'tm-bg'", "window.persistTextMode("} {
		if !strings.Contains(textModeJS, want) {
			t.Errorf("%s fehlt in textmode.js", want)
		}
	}
}

// TestBuildInitialHTML_TextModeScriptBeforeMain prüft, dass das Text-Modus-Skript
// mit Nonce vor dem Hauptskript steht (das Hauptskript ruft initTextMode() sofort).
func TestBuildInitialHTML_TextModeScriptBeforeMain(t *testing.T) {
	html := BuildInitialHTML(textModeConfig())
	def := strings.Index(html, "function initTextMode(")
	call := strings.Index(html, "initTextMode();")
	if def < 0 || call < 0 {
		t.Fatalf("initTextMode fehlt (Definition %d, Aufruf %d)", def, call)
	}
	if def > call {
		t.Error("textmode.js steht hinter dem Hauptskript – initTextMode wäre beim Aufruf undefiniert")
	}
	if strings.Contains(html, "{{TEXTMODE_SCRIPT}}") {
		t.Error("Platzhalter des Text-Modus-Skripts nicht ersetzt")
	}
}

// TestTextModeCSS_Rules prüft zentrale Darstellungsregeln: Pixelschrift,
// gleiche Größe, 1,5-facher Zeilenabstand, fette Überschriften, Tabellenlinien,
// 1-Bit-Filter für Formeln und Diagramme, gerasterte Bilder ohne Glättung.
func TestTextModeCSS_Rules(t *testing.T) {
	for _, want := range []string{
		"body.textmode #content",
		"var(--tm-font)",
		"font-size: inherit !important",
		"line-height: 1.5",
		"calc(var(--tm-font-size) * 50)",
		"font-weight: bold !important",
		"border: var(--tm-px) solid var(--tm-fg)",
		"url(#tm-filter-alpha)",
		"url(#tm-filter-ink-dark)",
		"url(#tm-filter-ink-light)",
		"url(#tm-filter-photo)",
		"image-rendering: pixelated",
		"#btn-zoom-in",
	} {
		if !strings.Contains(textModeCSS, want) {
			t.Errorf("%q fehlt in textmode.css", want)
		}
	}
}

// TestTextModeJS_Node führt die JavaScript-Unit-Tests (Schriftgröße, Farben,
// 1-Bit-Rasterung) mit Node.js aus. Ohne Node wird der Test übersprungen.
func TestTextModeJS_Node(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node nicht installiert – JS-Tests übersprungen")
	}
	out, err := exec.Command(node, filepath.Join("testdata", "textmode_test.js"),
		filepath.Join("assets", "textmode.js")).CombinedOutput()
	if err != nil {
		t.Fatalf("JS-Tests fehlgeschlagen: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}

// TestTextModeToolbarStable prüft, dass Umschalten die Toolbar nicht verschiebt:
// Zoom-Elemente und Farbfelder werden per visibility versteckt, nicht per display
// (sonst rutscht der Text-Modus-Knopf unter dem Mauszeiger weg, Klicktest Build 45).
func TestTextModeToolbarStable(t *testing.T) {
	zoomRule := regexp.MustCompile(`body\.textmode #zoom-label \{([^}]*)\}`).FindStringSubmatch(textModeCSS)
	if zoomRule == nil {
		t.Fatal("Regel für die Zoom-Elemente im Text-Modus fehlt")
	}
	if strings.Contains(zoomRule[1], "display") || !strings.Contains(zoomRule[1], "visibility: hidden") {
		t.Errorf("Zoom-Elemente müssen per visibility versteckt werden, nicht per display: %q", zoomRule[1])
	}
	if strings.Contains(htmlCSS, "body.textmode #tm-colors { display") {
		t.Error("Farbfelder werden beim Umschalten per display ein-/ausgeblendet")
	}
	if !strings.Contains(htmlCSS, "body.textmode #tm-colors { visibility: visible; }") {
		t.Error("Farbfelder werden im Text-Modus nicht per visibility sichtbar")
	}
}
