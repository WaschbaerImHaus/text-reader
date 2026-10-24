// Package ui – Text-Modus (C64-ähnliche Monospace-Ansicht).
//
// Im Text-Modus zeigt MD Reader Textdokumente (auch EPUB) nur in einer
// Pixel-Monospace-Schrift ("Press Start 2P", SIL Open Font License) in
// zwei frei wählbaren Farben an: 50 Spalten, 32 Zeilen mit 1,5-fachem
// Zeilenabstand, Schriftgröße abhängig von der kürzeren Fensterseite.
// Bilder werden 1-Bit-gerastert, Formeln und Diagramme hart auf zwei
// Farben geschnitten. Die Darstellung selbst steckt in assets/textmode.css
// und assets/textmode.js; hier liegen die Einstellungen und die Schrift.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24
package ui

import (
	_ "embed"
	"encoding/base64"
	"regexp"
	"strings"
	"sync"
)

// Standardfarben des Text-Modus: die C64-Startfarben (Palette "Pepto").
const (
	// DefaultTextModeForeground ist C64-Hellblau (Schriftfarbe).
	DefaultTextModeForeground = "#7869c4"
	// DefaultTextModeBackground ist C64-Blau (Hintergrund).
	DefaultTextModeBackground = "#40318d"
)

// textModeFontFamily ist der interne Name der eingebetteten Schrift.
// Eigener Name, damit keine gleichnamige Systemschrift sie verdrängt.
const textModeFontFamily = "MDR Press Start 2P"

// hexColorPattern erlaubt ausschließlich "#rrggbb" (ASCII-Hexziffern).
// Die Farbe landet ungeschützt in CSS und JS – jede andere Form könnte
// dort als Code wirken.
var hexColorPattern = regexp.MustCompile(`\A#[0-9a-fA-F]{6}\z`)

// textModeFontTTF enthält die Pixelschrift (8x8-Raster, quadratische Zeichen).
//
//go:embed assets/fonts/PressStart2P-Regular.ttf
var textModeFontTTF []byte

// textModeCSS enthält die Darstellungsregeln des Text-Modus.
//
//go:embed assets/textmode.css
var textModeCSS string

// textModeJS enthält die Logik des Text-Modus (Größe, Farben, Rasterung).
//
//go:embed assets/textmode.js
var textModeJS string

// textModeFontOnce sorgt dafür, dass die Schrift nur einmal kodiert wird.
var textModeFontOnce sync.Once

// textModeFontFace ist die fertige @font-face-Regel (zwischengespeichert).
var textModeFontFace string

// TextMode bündelt die Einstellungen des Text-Modus (Wertobjekt).
type TextMode struct {
	// Enabled gibt an, ob der Nutzer den Modus eingeschaltet hat.
	Enabled bool
	// Foreground ist die Schriftfarbe als "#rrggbb".
	Foreground string
	// Background ist die Hintergrundfarbe als "#rrggbb".
	Background string
}

// IsHexColor prüft, ob ein Wert exakt die Form "#rrggbb" hat.
//
// @param value Zu prüfender Farbwert.
// @return true nur für "#" gefolgt von genau sechs Hexziffern.
// @lastModified 2026-10-24
func IsHexColor(value string) bool {
	return hexColorPattern.MatchString(value)
}

// NewTextMode erzeugt geprüfte Text-Modus-Einstellungen.
//
// Gültige Farben werden klein geschrieben übernommen, ungültige einzeln durch
// die C64-Standardfarben ersetzt.
//
// @param enabled    Modus eingeschaltet?
// @param foreground Gewünschte Schriftfarbe.
// @param background Gewünschte Hintergrundfarbe.
// @return Normalisierte Einstellungen.
// @lastModified 2026-10-24
func NewTextMode(enabled bool, foreground, background string) TextMode {
	return TextMode{
		Enabled:    enabled,
		Foreground: normalizeColor(foreground, DefaultTextModeForeground),
		Background: normalizeColor(background, DefaultTextModeBackground),
	}
}

// Normalized liefert eine geprüfte Kopie der Einstellungen.
//
// @return Einstellungen mit garantiert gültigen Farben.
// @lastModified 2026-10-24
func (m TextMode) Normalized() TextMode {
	return NewTextMode(m.Enabled, m.Foreground, m.Background)
}

// normalizeColor gibt eine gültige Farbe klein geschrieben zurück, sonst den Ersatz.
//
// @param value    Zu prüfende Farbe.
// @param fallback Ersatzfarbe bei ungültigem Wert.
// @return Gültige Farbe "#rrggbb".
// @lastModified 2026-10-24
func normalizeColor(value, fallback string) string {
	if IsHexColor(value) {
		return strings.ToLower(value)
	}
	return fallback
}

// TextModeFontFaceCSS liefert die @font-face-Regel mit der als data:-URI
// eingebetteten Pixelschrift (CSP erlaubt nur font-src data:).
//
// @return CSS-Regel (ohne <style>-Tags).
// @lastModified 2026-10-24
func TextModeFontFaceCSS() string {
	textModeFontOnce.Do(func() {
		textModeFontFace = `@font-face { font-family: "` + textModeFontFamily +
			`"; src: url(data:font/ttf;base64,` +
			base64.StdEncoding.EncodeToString(textModeFontTTF) +
			`) format("truetype"); font-display: block; }`
	})
	return textModeFontFace
}

// textModeStyleBlock baut den <style>-Block des Text-Modus (Schrift + Regeln).
//
// @return Vollständiger <style>-Block.
// @lastModified 2026-10-24
func textModeStyleBlock() string {
	return "<style>\n" + TextModeFontFaceCSS() + "\n" + textModeCSS + "</style>\n"
}
