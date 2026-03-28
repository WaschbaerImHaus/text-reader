// GTK-Drag&Drop-Callback für Linux.
//
// Diese Datei ist bewusst von platform_linux.go getrennt, weil CGo verlangt
// dass Dateien mit //export im C-Preamble nur Deklarationen (keine Definitionen)
// enthalten dürfen. Alle C-Definitionen für Drag&Drop stehen in platform_linux.go.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-28

//go:build linux

package main

import "C"

// goFileDropCallback wird vom GTK-Signal-Handler onDragDataReceived aufgerufen
// wenn der Nutzer eine Datei auf das Fenster ablegt.
//
// Läuft auf dem GTK-Hauptthread (daher go-Goroutine für loadFileNative, damit
// der GTK-Thread nicht blockiert wird).
//
// @param cPath C-String mit dem absoluten Dateipfad.
//
//export goFileDropCallback
func goFileDropCallback(cPath *C.char) {
	path := C.GoString(cPath)
	if path == "" || app == nil || app.webview == nil {
		return
	}
	// Datei in separatem Goroutine laden damit GTK-Hauptthread frei bleibt
	go loadFileNative(app.webview, path)
}
