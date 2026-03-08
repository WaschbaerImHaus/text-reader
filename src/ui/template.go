// Package ui stellt die HTML/CSS/JS-Oberfläche der MD-Reader-App bereit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08 (KaTeX offline eingebettet)
package ui

import (
	"strconv"
	"strings"
)

// UIConfig enthält alle Werte die das initiale HTML beeinflussen.
type UIConfig struct {
	// FontSize ist die gespeicherte Schriftgröße in Pixeln.
	FontSize int
	// DefaultFontSize ist die unveränderliche Standard-Schriftgröße (Basis für Zoom-Prozent).
	DefaultFontSize int
	// Theme ist das Farbschema: "light", "dark" oder "retro".
	Theme string
	// IsPortrait gibt an ob der Hochformat-Modus aktiv ist.
	IsPortrait bool
}

// htmlDocHeadTemplate ist der Dokumentkopf mit Platzhaltern für Favicon und KaTeX-CSS.
// font-src data: ist nötig damit die eingebetteten KaTeX-Schriften (base64 data URIs) geladen werden.
const htmlDocHeadTemplate = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none';">
<link rel="icon" type="image/svg+xml" href="{{FAVICON_URI}}">
<title>MD Reader</title>
<style>
{{KATEX_CSS}}
</style>
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
	// defaultFontSize ist die konstante Basis für die Zoom-Prozentanzeige (z.B. 16px = 100%).
	// Dieser Wert darf NICHT mit dem gespeicherten fontSize überschrieben werden.
	defaultFontSizeStr := strconv.Itoa(cfg.DefaultFontSize)

	// Platzhalter in CSS, HTML und JavaScript ersetzen
	r := strings.NewReplacer(
		"{{FONT_SIZE}}", fontSizeStr,
		"{{DEFAULT_FONT_SIZE}}", defaultFontSizeStr,
		"{{IS_PORTRAIT}}", portraitStr,
		"{{THEME_CLASS}}", themeClass,
	)

	// Favicon und KaTeX-CSS in den Dokumentkopf einsetzen.
	// KaTeX-CSS wird eingebettet, damit Formeln offline ohne Webserver rendern.
	head := strings.ReplaceAll(htmlDocHeadTemplate, "{{FAVICON_URI}}", faviconDataURI())
	head = strings.ReplaceAll(head, "{{KATEX_CSS}}", KaTeXCSS())

	// KaTeX JS + Auto-Render JS als Inline-Skripte (direkt nach den App-Skripten).
	// Der Platzhalter {{KATEX_SCRIPTS}} wird im htmlJavaScript-Template erwartet.
	katexScripts := "<script>\n" + KaTeXJS() + "\n</script>\n" +
		"<script>\n" + KaTeXAutoRenderJS() + "\n</script>"

	mainJS := r.Replace(htmlJavaScript)
	mainJS = strings.ReplaceAll(mainJS, "{{KATEX_SCRIPTS}}", katexScripts)

	return head +
		r.Replace(htmlCSS) +
		r.Replace(htmlBodyHTML) +
		mainJS
}
