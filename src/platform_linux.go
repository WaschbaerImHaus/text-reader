// Plattformspezifische Implementierung für Linux (GTK).
//
// Autor: Reisen macht Spass... mit Pia und Dirk e.Kfm.
// Letzte Änderung: 2026-03-05

//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>

// toggleWindowFullscreen wechselt den Vollbild-Modus des GTK-Fensters.
void toggleWindowFullscreen(void* window, int isFullscreen) {
    GtkWindow* win = GTK_WINDOW(window);
    if (isFullscreen) {
        gtk_window_unfullscreen(win);
    } else {
        gtk_window_fullscreen(win);
    }
}
*/
import "C"

import (
	"unsafe"

	webview "github.com/webview/webview_go"
)

// toggleNativeFullscreen wechselt den nativen GTK-Vollbild-Modus.
// Diese Funktion nutzt das GTK-Fenster-Handle des WebViews.
//
// @param w Die WebView-Instanz.
func toggleNativeFullscreen(w webview.WebView) {
	// Nativen GTK-Fenster-Zeiger aus dem WebView holen
	ptr := w.Window()
	if ptr == nil {
		return
	}
	// Vollbild-Zustand toggeln via CGo
	var fullscreenInt C.int
	if app.isFullscreen {
		fullscreenInt = 1
	} else {
		fullscreenInt = 0
	}
	C.toggleWindowFullscreen(unsafe.Pointer(ptr), fullscreenInt)
	app.isFullscreen = !app.isFullscreen
}
