// Package ui stellt die HTML/CSS/JS-Oberfläche der MD-Reader-App bereit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

import (
	"strconv"
	"strings"
)

// UIConfig enthält alle Werte die das initiale HTML beeinflussen.
type UIConfig struct {
	// FontSize ist die Schriftgröße in Pixeln.
	FontSize int
	// Theme ist das Farbschema: "light", "dark" oder "retro".
	Theme string
	// IsPortrait gibt an ob der Hochformat-Modus aktiv ist.
	IsPortrait bool
}

// htmlDocHead ist der statische Dokumentkopf.
const htmlDocHead = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: blob:; connect-src 'none';">
<title>MD Reader</title>
`

// BuildInitialHTML erstellt das vollständige HTML-Dokument für den WebView.
//
// Ersetzt Platzhalter ({{FONT_SIZE}}, {{DEFAULT_FONT_SIZE}}, {{IS_PORTRAIT}}, {{THEME_CLASS}})
// in den eingebetteten Asset-Dateien durch die Konfigurationswerte.
//
// @param cfg UIConfig mit den initialen Einstellungswerten.
// @return Vollständiges HTML-Dokument als String.
func BuildInitialHTML(cfg UIConfig) string {
	// Theme-Klasse für das <body>-Tag bestimmen
	themeClass := ""
	if cfg.Theme == "dark" || cfg.Theme == "retro" {
		themeClass = cfg.Theme
	}

	// isPortrait als JS-Boolean-String
	portraitStr := "false"
	if cfg.IsPortrait {
		portraitStr = "true"
	}

	fontSizeStr := strconv.Itoa(cfg.FontSize)

	// Platzhalter in CSS, HTML und JavaScript ersetzen
	r := strings.NewReplacer(
		"{{FONT_SIZE}}", fontSizeStr,
		"{{DEFAULT_FONT_SIZE}}", fontSizeStr,
		"{{IS_PORTRAIT}}", portraitStr,
		"{{THEME_CLASS}}", themeClass,
	)

	return htmlDocHead +
		r.Replace(htmlCSS) +
		r.Replace(htmlBodyHTML) +
		r.Replace(htmlJavaScript)
}
