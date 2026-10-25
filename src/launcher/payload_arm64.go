//go:build embedgui

// Eingebettete Nutzlast für GOARCH=arm64: das GUI-Programm und die Icons.
// Beide Dateien kopiert der Build (Makefile) vorher nach payload/ (gitignoriert):
//   - payload/arm64/md-reader-gui
//   - payload/icons/favicon-{16,32,48,256}.png und favicon.svg
//
// Fehlen sie, scheitert der Build mit Tag embedgui – gewollt, damit nie ein
// Starter ohne Oberfläche veröffentlicht wird.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6: Starter-Programm)
package main

import "embed"

// guiBinary ist das eingebettete GUI-Programm.
//
//go:embed payload/arm64/md-reader-gui
var guiBinary []byte

// embeddedIcons enthält die Icon-Dateien aus payload/icons.
//
//go:embed payload/icons
var embeddedIcons embed.FS

// iconFiles sind die Icons, abgebildet auf die Schlüssel von linuxsetup.Icons.
var iconFiles = iconsFromFS(embeddedIcons, "payload/icons")
