// Plattformspezifische Implementierung für Windows (WinAPI).
//
// Autor: Reisen macht Spass... mit Pia und Dirk e.Kfm.
// Letzte Änderung: 2026-03-05

//go:build windows

package main

import (
	"syscall"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// WinAPI-Konstanten für Fenster-Verwaltung
const (
	swMaximize = 3  // ShowWindow: maximieren
	swRestore  = 9  // ShowWindow: wiederherstellen
)

// user32 ist das Windows-User32-DLL für Fenster-Funktionen.
var user32 = syscall.MustLoadDLL("user32.dll")

// procShowWindow ist der ShowWindow-Prozessaufruf.
var procShowWindow = user32.MustFindProc("ShowWindow")

// toggleNativeFullscreen wechselt den Windows-Vollbild-Modus via WinAPI.
//
// @param w Die WebView-Instanz.
func toggleNativeFullscreen(w webview.WebView) {
	// HWND (Fenster-Handle) aus dem WebView holen
	hwnd := w.Window()
	if hwnd == nil {
		return
	}

	if app.isFullscreen {
		// Aus Vollbild zurück: Fenster wiederherstellen
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swRestore)
		app.isFullscreen = false
	} else {
		// Vollbild: Fenster maximieren (einfachste Windows-Implementierung)
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swMaximize)
		app.isFullscreen = true
	}
}
