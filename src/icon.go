// Eingebettetes App-Icon (48x48 PNG) für die Fenster-Iconleiste.
//
// Das Icon wird via go:embed aus assets/favicon-48.png geladen und auf
// Linux per GTK, auf Windows via WM_SETICON gesetzt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package main

import _ "embed"

//go:embed ui/assets/favicon-48.png
var appIconPNG []byte
