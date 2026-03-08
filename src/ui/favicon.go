// Package ui – Favicon-Einbettung für den MD Reader.
//
// Das SVG-Favicon wird als go:embed eingebettet und beim Aufbau des
// HTML-Dokuments als base64-Data-URI in den <head>-Bereich eingefügt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

import (
	_ "embed"
	"encoding/base64"
)

//go:embed assets/favicon.svg
var faviconSVG []byte

// faviconDataURI gibt die base64-kodierte Data-URI des SVG-Favicons zurück.
//
// Format: data:image/svg+xml;base64,<base64-Daten>
//
// @return Vollständige Data-URI als String.
func faviconDataURI() string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(faviconSVG)
}
