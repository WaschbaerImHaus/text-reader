// Package ui stellt die HTML/CSS/JS-Oberfläche der MD-Reader-App bereit.
//
// Das vollständige HTML-Dokument wird aus drei Teilen zusammengebaut:
//   - styles.go   → CSS-Stile (htmlCSS)
//   - html_body.go → HTML-Grundstruktur (htmlBodyHTML)
//   - scripts.go  → JavaScript-Logik (htmlJavaScript)
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

import "fmt"

// UIConfig enthält alle Werte die das initiale HTML beeinflussen.
type UIConfig struct {
	// FontSize ist die Schriftgröße in Pixeln.
	FontSize int
	// Theme ist das Farbschema: "light", "dark" oder "retro".
	Theme string
	// IsPortrait gibt an ob der Hochformat-Modus aktiv ist.
	IsPortrait bool
}

// htmlDocHead ist der statische Dokumentkopf (DOCTYPE bis <style>-Öffnung).
const htmlDocHead = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>MD Reader</title>
`

// BuildInitialHTML erstellt das vollständige HTML-Dokument für den WebView.
//
// Setzt die gespeicherte Konfiguration (Theme, Layout, Schriftgröße) direkt
// in das HTML ein. Die drei Teilbereiche (CSS, HTML-Struktur, JavaScript)
// werden jeweils mit fmt.Sprintf und den passenden Konfigurationswerten
// zu einem vollständigen HTML-Dokument zusammengesetzt.
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

	return htmlDocHead +
		// CSS mit Schriftgröße einbetten
		fmt.Sprintf(htmlCSS, cfg.FontSize) +
		// HTML-Struktur mit Theme-Klasse
		fmt.Sprintf(htmlBodyHTML, themeClass) +
		// JavaScript mit Schriftgröße und Layout-Modus
		fmt.Sprintf(htmlJavaScript, cfg.FontSize, cfg.FontSize, portraitStr)
}
