// Plattformspezifische Implementierung für Windows (WinAPI + CGo).
//
// Enthält:
//   - Nativen Vollbild-Toggle (user32 ShowWindow)
//   - Nativen Datei-Öffnen-Dialog (comdlg32 GetOpenFileNameW)
//   - Icon-Setzung aus go:embed (CreateIconFromResourceEx + SetClassLongPtrW)
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08

//go:build windows

package main

/*
#cgo LDFLAGS: -lcomdlg32 -luser32

#include <windows.h>
#include <commdlg.h>
#include <string.h>

// ICO-Dateiformat-Strukturen (Little-Endian, Windows-Standard).
#pragma pack(push, 1)
typedef struct {
    WORD reserved;
    WORD type;
    WORD count;
} IcoDir;

typedef struct {
    BYTE  width;
    BYTE  height;
    BYTE  colorCount;
    BYTE  reserved;
    WORD  planes;
    WORD  bitCount;
    DWORD bytesInRes;
    DWORD imageOffset;
} IcoDirEntry;
#pragma pack(pop)

// setWindowIconFromICO setzt das Fenster-Icon direkt aus ICO-Rohdaten.
//
// Wählt die beste Größe für ICON_BIG (32 px) und ICON_SMALL (16 px),
// erzeugt HICON via CreateIconFromResourceEx und setzt sie über
// WM_SETICON und SetClassLongPtrW auf das Top-Level-Fenster (GetAncestor).
//
// @param hwnd    Fenster-Handle (kann inneres WebView2-Control sein).
// @param data    Rohdaten der .ico-Datei.
// @param dataLen Länge der Rohdaten in Bytes.
void setWindowIconFromICO(HWND hwnd, const BYTE* data, int dataLen) {
    if (!hwnd || !data || dataLen < (int)sizeof(IcoDir)) return;

    const IcoDir* dir = (const IcoDir*)data;
    if (dir->type != 1 || dir->count == 0) return;

    const IcoDirEntry* entries = (const IcoDirEntry*)(data + sizeof(IcoDir));

    // Beste Eintraege fuer grosses (<=32px) und kleines (kleinste >=16px) Icon suchen
    int bigIdx = 0, smallIdx = 0;
    int bigBest = 0, smallBest = 9999;

    for (int i = 0; i < (int)dir->count; i++) {
        int w = entries[i].width == 0 ? 256 : (int)entries[i].width;
        if (w <= 32 && w > bigBest)   { bigBest = w;   bigIdx = i;   }
        if (w >= 16 && w < smallBest) { smallBest = w; smallIdx = i; }
    }

    // Sicherheitsprüfung: Bilddaten müssen im Puffer liegen
    if ((int)(entries[bigIdx].imageOffset   + entries[bigIdx].bytesInRes)   > dataLen) return;
    if ((int)(entries[smallIdx].imageOffset + entries[smallIdx].bytesInRes) > dataLen) return;

    const BYTE* bigData   = data + entries[bigIdx].imageOffset;
    const BYTE* smallData = data + entries[smallIdx].imageOffset;

    // HICON aus ICO-Bilddaten erzeugen (0x00030000 = ICO Version 3.0)
    HICON hIconBig = CreateIconFromResourceEx(
        (PBYTE)bigData,   entries[bigIdx].bytesInRes,
        TRUE, 0x00030000, 32, 32, LR_DEFAULTCOLOR
    );
    HICON hIconSmall = CreateIconFromResourceEx(
        (PBYTE)smallData, entries[smallIdx].bytesInRes,
        TRUE, 0x00030000, 16, 16, LR_DEFAULTCOLOR
    );

    // Fallback: fehlende Größe durch die andere ersetzen
    if (!hIconBig)   hIconBig   = hIconSmall;
    if (!hIconSmall) hIconSmall = hIconBig;
    if (!hIconBig)   return;

    // Top-Level-Fenster ermitteln (GA_ROOT=2): w.Window() kann WebView2-Control sein
    HWND top = GetAncestor(hwnd, 2);
    if (!top) top = hwnd;

    // WM_SETICON: Titelleisten-Icon setzen
    SendMessageW(top, WM_SETICON, ICON_BIG,   (LPARAM)hIconBig);
    SendMessageW(top, WM_SETICON, ICON_SMALL, (LPARAM)hIconSmall);

    // SetClassLongPtrW: Icon der Fensterklasse setzen (persistenter, auch fuer Alt+Tab)
    // GCLP_HICON = -14, GCLP_HICONSM = -34
    SetClassLongPtrW(top, -14, (LONG_PTR)hIconBig);
    SetClassLongPtrW(top, -34, (LONG_PTR)hIconSmall);
}

// showFileDialog öffnet den nativen Windows-Datei-Öffnen-Dialog.
//
// @param hwndOwner Optionales Eltern-Fenster-Handle (kann NULL sein).
// @return UTF-8-Pfad oder NULL bei Abbruch.
const char* showFileDialog(HWND hwndOwner) {
    static wchar_t pathBuf[32768];
    static char    utf8Buf[65536];
    pathBuf[0] = L'\0';

    OPENFILENAMEW ofn;
    memset(&ofn, 0, sizeof(ofn));
    ofn.lStructSize  = sizeof(OPENFILENAMEW);
    ofn.hwndOwner    = hwndOwner;
    ofn.lpstrFilter  = L"Unterst\u00fctzte Dateien\0*.md;*.markdown;*.txt;*.epub;*.fb2;*.html;*.htm;*.tex\0Alle Dateien (*.*)\0*.*\0";
    ofn.nFilterIndex = 1;
    ofn.lpstrFile    = pathBuf;
    ofn.nMaxFile     = 32768;
    ofn.lpstrTitle   = L"Datei \u00f6ffnen \u2013 MD Reader";
    ofn.Flags        = OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST | OFN_EXPLORER;

    if (!GetOpenFileNameW(&ofn)) return NULL;

    int n = WideCharToMultiByte(CP_UTF8, 0, pathBuf, -1, utf8Buf, sizeof(utf8Buf), NULL, NULL);
    if (n <= 0) return NULL;
    return utf8Buf;
}
*/
import "C"

import (
	_ "embed"
	"syscall"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// appIconICO enthält die eingebettete favicon.ico (256/48/32/16 px).
// Wird in setAppIcon direkt via CreateIconFromResourceEx in HICON umgewandelt.
//
//go:embed ui/assets/favicon.ico
var appIconICO []byte

// WinAPI-Konstanten für ShowWindow
const (
	swMaximize = 3 // Fenster maximieren
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
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swRestore)
		app.isFullscreen = false
	} else {
		procShowWindow.Call(uintptr(unsafe.Pointer(hwnd)), swMaximize)
		app.isFullscreen = true
	}
}

// showOpenFileDialog öffnet den nativen Windows-Datei-Dialog.
//
// @param parentWindow Fenster-Handle (als unsafe.Pointer, kann nil sein).
// @return UTF-8-Pfad der gewählten Datei, oder leer bei Abbruch.
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
// @param w Die WebView-Instanz.
// @return Gewählter Dateipfad oder leer bei Abbruch.
func openFilePickerBlocking(w webview.WebView) string {
	return showOpenFileDialog(w.Window())
}

// setAppIcon setzt das Titelleisten-Icon aus dem eingebetteten ICO-Byte-Slice.
//
// Verwendet CreateIconFromResourceEx (unabhängig von .syso-Ressourcen) und
// setzt das Icon per WM_SETICON sowie SetClassLongPtrW auf dem Top-Level-Fenster.
//
// @param w Die WebView-Instanz.
func setAppIcon(w webview.WebView) {
	hwnd := w.Window()
	if hwnd == nil {
		return
	}
	// ICO-Bytes in CGo-Speicher kopieren (C.CBytes alloziert und kopiert)
	data := C.CBytes(appIconICO)
	defer C.free(data)
	C.setWindowIconFromICO(
		C.HWND(hwnd),
		(*C.BYTE)(data),
		C.int(len(appIconICO)),
	)
}

// setupNativeFileDrop ist auf Windows ein No-Op.
//
// Windows WebView2 liefert e.dataTransfer.files korrekt im Drop-Event,
// daher ist kein nativer GTK-Level-Handler nötig.
//
// @param w Die WebView-Instanz (nicht verwendet).
func setupNativeFileDrop(w webview.WebView) {}
