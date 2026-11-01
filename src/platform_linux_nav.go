// Go-Callback für den nativen Navigationsschutz unter Linux (Fund F1, Build 47).
//
// Eigene Datei aus demselben Grund wie platform_linux_drop.go: Dateien mit
// //export dürfen im CGo-Preamble nur Deklarationen enthalten. Der C-Handler
// onDecidePolicy steht in platform_linux.go.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F1)

//go:build linux

package main

import "C"

import "md-reader/ui"

// goDecideNavigation wird vom WebKitGTK-Signal decide-policy für jede
// Navigation des Hauptfensters aufgerufen (GTK-Hauptthread).
//
// Die Entscheidung trifft handleNavigationRequest: nur die eigene Seite und
// die eigene Temp-Ansichtsdatei sind erlaubt; fremde http/https/mailto-Links
// öffnet bei Nutzeraktion extlink im Standardprogramm.
//
// @param cURI        Ziel-URI der Navigation (C-String, nie NULL).
// @param userAction  1 bei Nutzeraktion (Geste oder Linkklick, z. B. Kontextmenü).
// @return 1 = Navigation erlauben, 0 = ablehnen.
//
//export goDecideNavigation
func goDecideNavigation(cURI *C.char, userAction C.int) C.int {
	if handleNavigationRequest(C.GoString(cURI), userAction != 0, ui.TempViewFileURL(), externalOpener.Open) {
		return 1
	}
	return 0
}
