// Package ui stellt die HTML/CSS/JS-Oberfläche der MD-Reader-App bereit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-28
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

	// ContentHTML enthält den gerenderten Dateiinhalt (leer = Drop-Zone anzeigen).
	// Wenn gesetzt, wird der Inhalt direkt im WebView angezeigt ohne JS-Binding-Umweg.
	ContentHTML string
	// PageTitle ist der Dokumenttitel (ohne " – MD Reader" Suffix, wird von Go ergänzt).
	PageTitle string
	// FileHash ist der FNV-64a-Hash des Dateiinhalts für die JS-Scroll-History.
	FileHash string
	// ScrollPos ist die Anfangs-Scroll-Position in Pixeln (0 = Anfang).
	ScrollPos int
}

// htmlDocHeadTemplate ist der Dokumentkopf mit Platzhaltern für Favicon, Titel und KaTeX-CSS.
// font-src data: ist nötig damit die eingebetteten KaTeX-Schriften (base64 data URIs) geladen werden.
const htmlDocHeadTemplate = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: blob: file:; font-src data:; connect-src 'none';">
<link rel="icon" type="image/svg+xml" href="{{FAVICON_URI}}">
<title>{{PAGE_TITLE}}</title>
<style>
{{KATEX_CSS}}
</style>
`

// BuildInitialHTML erstellt das vollständige HTML-Dokument für den WebView.
//
// Wenn cfg.ContentHTML gesetzt ist, wird der gerenderte Inhalt direkt eingebettet
// und die Drop-Zone ausgeblendet. Go ruft diese Funktion nach jedem Dateiladen auf
// und übergibt das Ergebnis an w.SetHtml() – kein JS-Binding-Roundtrip nötig.
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
	defaultFontSizeStr := strconv.Itoa(cfg.DefaultFontSize)

	// Seitentitel: "Dokumentname – MD Reader" oder nur "MD Reader" bei Drop-Zone
	pageTitle := "MD Reader"
	if cfg.PageTitle != "" {
		pageTitle = cfg.PageTitle + " \u2013 MD Reader"
	}

	// Drop-Zone und Inhaltsbereich abhängig davon ob Inhalt vorhanden ist ein-/ausblenden
	dropZoneDisplay := "flex"
	contentWrapperDisplay := "none"
	if cfg.ContentHTML != "" {
		dropZoneDisplay = "none"
		contentWrapperDisplay = "block"
	}

	// Alle Platzhalter in CSS, HTML und JavaScript ersetzen
	r := strings.NewReplacer(
		"{{FONT_SIZE}}", fontSizeStr,
		"{{DEFAULT_FONT_SIZE}}", defaultFontSizeStr,
		"{{IS_PORTRAIT}}", portraitStr,
		"{{THEME_CLASS}}", themeClass,
		"{{PAGE_TITLE}}", pageTitle,
		"{{DROP_ZONE_DISPLAY}}", dropZoneDisplay,
		"{{CONTENT_WRAPPER_DISPLAY}}", contentWrapperDisplay,
		"{{CONTENT_HTML}}", cfg.ContentHTML,
		"{{INITIAL_SCROLL_POS}}", strconv.Itoa(cfg.ScrollPos),
		"{{INITIAL_FILE_HASH}}", cfg.FileHash,
	)

	// Favicon und KaTeX-CSS in den Dokumentkopf einsetzen
	head := strings.ReplaceAll(htmlDocHeadTemplate, "{{FAVICON_URI}}", faviconDataURI())
	head = strings.ReplaceAll(head, "{{KATEX_CSS}}", KaTeXCSS())
	// PAGE_TITLE im Head-Template ersetzen
	head = r.Replace(head)

	// KaTeX JS + Auto-Render JS als Inline-Skripte
	katexScripts := "<script>\n" + KaTeXJS() + "\n</script>\n" +
		"<script>\n" + KaTeXAutoRenderJS() + "\n</script>"

	mainJS := r.Replace(htmlJavaScript)
	mainJS = strings.ReplaceAll(mainJS, "{{KATEX_SCRIPTS}}", katexScripts)

	return head +
		r.Replace(htmlCSS) +
		r.Replace(htmlBodyHTML) +
		mainJS
}
