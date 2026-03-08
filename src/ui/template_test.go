// Tests für das HTML-Template der MD-Reader-Oberfläche.
//
// Prüft ob BuildInitialHTML ein gültiges, vollständiges HTML-Dokument
// mit korrekten Konfigurationswerten erzeugt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08 (Fix: DefaultFontSize-Feld in UIConfig)
package ui

import (
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
