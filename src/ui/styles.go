// Package ui – CSS-Stile der MD-Reader-Oberfläche.
//
// Enthält alle CSS-Definitionen für Light-, Dark- und Retro-Mode.
// Die CSS-Datei wird per go:embed eingebunden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

import _ "embed"

// htmlCSS enthält die CSS-Stile (aus assets/styles.css).
//
//go:embed assets/styles.css
var htmlCSS string
