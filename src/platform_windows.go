// Plattformspezifische Implementierung für Windows (WinAPI + CGo).
//
// Enthält nativen Vollbild-Toggle (user32) und nativen Datei-Öffnen-Dialog
// (comdlg32 via GetOpenFileNameW). Der Datei-Dialog ist notwendig weil
// WebView2 aus Sicherheitsgründen keine Dateipfade im DataTransfer exponiert.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08

//go:build windows

package main

/*
#cgo LDFLAGS: -lcomdlg32
#include <windows.h>
#include <commdlg.h>
#include <string.h>

// showFileDialog öffnet den nativen Windows-Datei-Öffnen-Dialog.
//
// Gibt den ausgewählten Pfad als UTF-8-String zurück, oder NULL wenn
// der Benutzer abbricht. Statische Buffer werden intern verwaltet.
//
// @param hwndOwner Optionales Eltern-Fenster-Handle (kann NULL sein).
// @return UTF-8-Pfad oder NULL.
const char* showFileDialog(HWND hwndOwner) {
    static wchar_t pathBuf[32768];
    static char    utf8Buf[65536];
    pathBuf[0] = L'\0';

    OPENFILENAMEW ofn;
    memset(&ofn, 0, sizeof(ofn));
    ofn.lStructSize  = sizeof(OPENFILENAMEW);
    ofn.hwndOwner    = hwndOwner;
    // Dateifilter: Unterstützte Formate zuerst, dann Alle Dateien
    ofn.lpstrFilter  = L"Unterst\u00fctzte Dateien\0*.md;*.markdown;*.txt;*.epub;*.fb2;*.html;*.htm\0Alle Dateien (*.*)\0*.*\0";
    ofn.nFilterIndex = 1;
    ofn.lpstrFile    = pathBuf;
    ofn.nMaxFile     = 32768;
    ofn.lpstrTitle   = L"Datei \u00f6ffnen \u2013 MD Reader";
    // OFN_EXPLORER: Moderner Dialog-Stil; OFN_FILEMUSTEXIST: Nur existierende Dateien
    ofn.Flags        = OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST | OFN_EXPLORER;

    if (!GetOpenFileNameW(&ofn)) {
        return NULL; // Abbruch oder Fehler
    }

    // Wchar-Pfad nach UTF-8 konvertieren
    int n = WideCharToMultiByte(CP_UTF8, 0, pathBuf, -1, utf8Buf, sizeof(utf8Buf), NULL, NULL);
    if (n <= 0) return NULL;
    return utf8Buf;
}
*/
import "C"

import (
	"syscall"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// WinAPI-Konstanten für ShowWindow
const (
	swMaximize = 3 // Fenster maximieren (Vollbild-Näherung)
	swRestore  = 9 // Fenster wiederherstellen
)

// user32 ist das Windows-User32-DLL für Fensterverwaltung.
var user32 = syscall.MustLoadDLL("user32.dll")

// procShowWindow ist der ShowWindow-Prozessaufruf aus user32.
var procShowWindow = user32.MustFindProc("ShowWindow")

// toggleNativeFullscreen wechselt den Windows-Vollbild-Modus via WinAPI.
//
// @param w Die WebView-Instanz.
func toggleNativeFullscreen(w webview.WebView) {
	hwnd := w.Window()
	if hwnd == nil {
		return
	}
	if app.isFullscreen {
		// Vollbild verlassen: Fenster auf normale Größe zurücksetzen
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swRestore)
		app.isFullscreen = false
	} else {
		// Vollbild: Fenster maximieren
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swMaximize)
		app.isFullscreen = true
	}
}

// showOpenFileDialog öffnet den nativen Windows-Datei-Dialog.
//
// Kann direkt aus einem Goroutine aufgerufen werden – GetOpenFileNameW
// ist thread-sicher und hat eine eigene Nachrichtenschleife.
//
// @param parentWindow Fenster-Handle (wird als unsafe.Pointer übergeben, kann nil sein).
// @return Vollständiger UTF-8-Pfad der gewählten Datei, oder leer bei Abbruch.
func showOpenFileDialog(parentWindow unsafe.Pointer) string {
	var hwnd C.HWND
	if parentWindow != nil {
		hwnd = C.HWND(parentWindow)
	}
	cpath := C.showFileDialog(hwnd)
	if cpath == nil {
		return ""
	}
	return C.GoString(cpath)
}

// openFilePickerBlocking öffnet den Datei-Dialog und blockiert bis zur Auswahl.
//
// Auf Windows wird GetOpenFileNameW direkt aus dem Goroutine aufgerufen
// (keine Dispatch auf den Hauptthread nötig).
//
// @param w Die WebView-Instanz.
// @return Gewählter Dateipfad oder leer bei Abbruch.
func openFilePickerBlocking(w webview.WebView) string {
	return showOpenFileDialog(w.Window())
}

// setAppIcon setzt das Fenster-Icon via WM_SETICON.
//
// Das primäre Icon ist bereits über die eingebettete Ressource (rsrc_windows_amd64.syso)
// in der .exe-Datei enthalten und wird von Windows automatisch für Explorer,
// Taskleiste und Alt+Tab verwendet. Diese Funktion setzt zusätzlich das
// Titelleisten-Icon zur Laufzeit.
//
// @param w Die WebView-Instanz.
func setAppIcon(w webview.WebView) {
	hwnd := w.Window()
	if hwnd == nil {
		return
	}
	// LoadImage lädt das ICON aus den eingebetteten Ressourcen (IDI_ICON1 = 1)
	loadImage := user32.MustFindProc("LoadImageW")
	hInst, _ := syscall.LoadDLL("kernel32.dll")
	getModuleHandle, _ := hInst.FindProc("GetModuleHandleW")
	hMod, _, _ := getModuleHandle.Call(0)

	// IDI_ICON1 = 1 (aus app.rc), ICON_BIG = 1, ICON_SMALL = 0
	const imageIcon = 1
	const lrDefaultSize = 0x0040
	hIcon, _, _ := loadImage.Call(
		hMod,
		1, // IDI_ICON1
		imageIcon,
		0, 0,
		lrDefaultSize,
	)
	if hIcon == 0 {
		return
	}
	// WM_SETICON = 0x0080, ICON_BIG = 1, ICON_SMALL = 0
	sendMessage := user32.MustFindProc("SendMessageW")
	sendMessage.Call(uintptr(unsafe.Pointer(hwnd)), 0x0080, 1, hIcon) // ICON_BIG
	sendMessage.Call(uintptr(unsafe.Pointer(hwnd)), 0x0080, 0, hIcon) // ICON_SMALL
}
