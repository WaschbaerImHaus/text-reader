// Package ui – HTML-Grundstruktur der MD-Reader-Oberfläche.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14 (Bug #013: Inhalt nicht mehr im Kommentar)
package ui

import _ "embed"

// htmlBodyHTML enthält die HTML-Grundstruktur (aus assets/body.html).
//
//go:embed assets/body.html
var htmlBodyHTML string
