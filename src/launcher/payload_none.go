//go:build !embedgui

// Leere Nutzlast für Builds ohne Tag embedgui (go vet, go test, go run für
// --print-desktop-entry). Ein so gebauter Starter kann die Oberfläche nicht
// starten und meldet das ("Starter ohne eingebettete App gebaut").
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6: Starter-Programm)
package main

import "md-reader/linuxsetup"

// guiBinary ist ohne embedgui leer.
var guiBinary []byte

// iconFiles ist ohne embedgui leer.
var iconFiles linuxsetup.Icons
