// Webview-Bindings und natives Dateiladen für den MD Reader.
//
// Kernprinzip: Go liest und rendert Dateien, baut das vollständige HTML
// (inkl. Toolbar, CSS, JS) und übergibt es direkt an w.SetHtml().
// Kein JS-Binding-Roundtrip mehr für Dateiinhalte.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-28
package main

import (
	"encoding/base64"
	"hash/fnv"
	"log"
	"md-reader/renderer"
	"md-reader/ui"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	webview "github.com/webview/webview_go"
)

// maxScrollHistory ist die maximale Anzahl von Einträgen in der Scroll-History
// bevor gekürzt wird.
const maxScrollHistory = 200

// trimScrollHistoryKept ist die Zielgröße nach dem Kürzen.
const trimScrollHistoryKept = 150

// computeHash berechnet einen FNV-64a-Hash der gegebenen Bytes.
//
// FNV-64a ist schnell, deterministisch und ausreichend kollisionsresistent
// für Datei-Identifikation (kein kryptographischer Einsatz).
//
// @param data Zu hashende Daten.
// @return Hash als hexadezimaler String.
func computeHash(data []byte) string {
	h := fnv.New64a()
	h.Write(data)
	return strconv.FormatUint(h.Sum64(), 16)
}

// trimScrollHistory kürzt die Scroll-History wenn sie maxScrollHistory überschreitet.
//
// Entfernt die lexikographisch kleinsten Schlüssel (deterministisch)
// bis nur noch trimScrollHistoryKept Einträge übrig sind.
//
// @param cfg Pointer auf die Konfiguration (wird in-place modifiziert).
func trimScrollHistory(cfg *AppConfig) {
	if len(cfg.ScrollHistory) <= maxScrollHistory {
		return
	}
	keys := make([]string, 0, len(cfg.ScrollHistory))
	for k := range cfg.ScrollHistory {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	toDelete := len(keys) - trimScrollHistoryKept
	for i := 0; i < toDelete; i++ {
		delete(cfg.ScrollHistory, keys[i])
	}
}

// scrollPosForHash gibt die gespeicherte Scroll-Position für den gegebenen Hash zurück.
//
// @param hash FNV-64a-Hash des Dateiinhalts.
// @return Scroll-Position in Pixeln, 0 wenn nicht gespeichert.
func scrollPosForHash(hash string) int {
	if app.config.ScrollHistory == nil {
		return 0
	}
	return app.config.ScrollHistory[hash]
}

// renderAndDisplay baut das vollständige HTML und übergibt es an den WebView.
//
// Go rendert Dateien zu HTML und ruft diese Funktion auf. Sie baut das
// komplette Seiten-HTML (Toolbar, CSS, JS + gerenderter Inhalt) und
// übergibt es via w.SetHtml() direkt an den WebView.
// Kein JavaScript-Roundtrip nötig.
//
// Muss NICHT auf dem GTK-Hauptthread aufgerufen werden – w.Dispatch übernimmt das.
//
// @param w          Die WebView-Instanz.
// @param contentHTML Gerenderter HTML-Inhalt der Datei.
// @param title      Dokumenttitel.
// @param hash       FNV-64a-Hash des Dateiinhalts (für Scroll-History).
// @param scrollPos  Gespeicherte Scroll-Position in Pixeln.
func renderAndDisplay(w webview.WebView, contentHTML, title, hash string, scrollPos int) {
	uiCfg := ui.UIConfig{
		FontSize:        app.config.FontSize,
		DefaultFontSize: defaultFontSize,
		Theme:           app.config.Theme,
		IsPortrait:      app.config.Layout == "portrait",
		ContentHTML:     contentHTML,
		PageTitle:       title,
		FileHash:        hash,
		ScrollPos:       scrollPos,
	}
	fullHTML := ui.BuildInitialHTML(uiCfg)
	// w.SetHtml muss auf dem GTK/WebView-Hauptthread ausgeführt werden
	w.Dispatch(func() {
		w.SetHtml(fullHTML)
	})
}

// loadFileNative liest eine Datei vom Dateisystem, rendert sie und zeigt sie an.
//
// Dies ist der zentrale native Dateilademechanismus. Go liest die Datei,
// rendert den Inhalt zu HTML und übergibt das vollständige Seiten-HTML
// via w.SetHtml() an den WebView – ohne JavaScript-Binding-Roundtrip.
//
// @param w        Die WebView-Instanz.
// @param filePath Vollständiger Pfad zur Datei.
func loadFileNative(w webview.WebView, filePath string) {
	// Sicherheitsprüfung: nur unterstützte Formate laden
	if !renderer.IsSupportedFile(filePath) {
		log.Printf("Nicht unterstütztes Format: %s", filePath)
		return
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		log.Printf("Datei nicht gefunden oder Verzeichnis: %s", filePath)
		return
	}

	// Rohdaten lesen (für Hash-Berechnung)
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht gelesen werden: %v", err)
		return
	}

	// Dateiinhalt rendern
	result, err := renderer.LoadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht gerendert werden: %v", err)
		return
	}

	// Relative Bild-Pfade in absolute konvertieren (nicht bei EPUB)
	if !renderer.IsEpubFile(filePath) {
		result.HTML = renderer.ResolveImagePaths(result.HTML, filepath.Dir(filePath))
	}

	// Hash berechnen und gespeicherte Scroll-Position nachschlagen
	hash := computeHash(rawBytes)
	scrollPos := scrollPosForHash(hash)

	// Zuletzt geöffnete Datei und Konfiguration persistieren
	app.config.LastFile = filePath
	saveConfig(app.config)

	// Vollständiges HTML bauen und an WebView übergeben
	renderAndDisplay(w, result.HTML, result.Title, hash, scrollPos)
}

// registerBindings registriert alle Go→JS-Bindings am WebView.
//
// @param w Die WebView-Instanz.
func registerBindings(w webview.WebView) {
	// openFilePicker: Öffnet den nativen Datei-Öffnen-Dialog.
	//
	// Go öffnet zenity/kdialog (Linux) bzw. den nativen Windows-Dialog,
	// liest die Datei und ruft w.SetHtml() auf. JS muss nichts weiter tun.
	w.Bind("openFilePicker", func() {
		path := openFilePickerBlocking(w)
		if path != "" {
			loadFileNative(w, path)
		}
	})

	// loadNativeFile: Lädt eine Datei anhand ihres vollständigen Pfads nativ.
	//
	// Wird vom Drag & Drop Handler aufgerufen wenn der Pfad aus text/uri-list
	// verfügbar ist (Linux + Windows Explorer). Go liest und rendert die Datei
	// direkt, kein FileReader in JS nötig.
	//
	// @param path Vollständiger Dateipfad.
	w.Bind("loadNativeFile", func(path string) {
		if path != "" {
			loadFileNative(w, path)
		}
	})

	// processMarkdown: Konvertiert Text-Inhalt (MD, TXT, FB2, HTML, TEX) zu HTML.
	//
	// Fallback für Drag & Drop auf Windows wenn kein Pfad aus URI-Liste verfügbar.
	// JS liest die Datei via FileReader und sendet den Inhalt hierher.
	// Go rendert und ruft w.SetHtml() auf – kein RenderResult-Roundtrip mehr.
	//
	// @param content  Dateiinhalt als UTF-8-String.
	// @param filename Dateiname (nur für Format-Erkennung, kein vollständiger Pfad).
	w.Bind("processMarkdown", func(content string, filename string) {
		result, err := renderer.ParseContent(content, filename)
		if err != nil {
			log.Printf("processMarkdown Fehler: %v", err)
			return
		}
		hash := computeHash([]byte(content))
		scrollPos := scrollPosForHash(hash)
		renderAndDisplay(w, result.HTML, result.Title, hash, scrollPos)
	})

	// processEpub: Konvertiert eine EPUB-Datei (Base64-kodiert) zu HTML.
	//
	// Fallback für Drag & Drop auf Windows. JS liest EPUB binär via FileReader,
	// kodiert als Base64 und sendet hierher. Go rendert und ruft w.SetHtml() auf.
	//
	// @param base64Data EPUB-Dateiinhalt als Base64-kodierter String.
	// @param filename   Dateiname für die Titelzeile.
	w.Bind("processEpub", func(base64Data string, filename string) {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			log.Printf("processEpub Base64-Fehler: %v", err)
			return
		}
		result, err := renderer.ParseEpub(data, filename)
		if err != nil {
			log.Printf("processEpub Render-Fehler: %v", err)
			return
		}
		hash := computeHash(data)
		scrollPos := scrollPosForHash(hash)
		renderAndDisplay(w, result.HTML, result.Title, hash, scrollPos)
	})

	// persistState: Speichert Zoom, Theme und Layout.
	w.Bind("persistState", func(fontSize float64, theme string, layout string) {
		app.config.FontSize = int(fontSize)
		app.config.Theme = theme
		app.config.Layout = layout
		saveConfig(app.config)
	})

	// saveScrollPos: Speichert die Scroll-Position für einen Dateiinhalt-Hash.
	//
	// Wird debounced beim Scrollen und synchron beim Beenden aufgerufen.
	// Die History wird auf maxScrollHistory Einträge begrenzt.
	//
	// @param hash      FNV-64a-Hash des Dateiinhalts.
	// @param scrollPos Aktuelle Scroll-Position in Pixeln.
	w.Bind("saveScrollPos", func(hash string, scrollPos float64) {
		if hash == "" {
			return
		}
		if app.config.ScrollHistory == nil {
			app.config.ScrollHistory = make(map[string]int)
		}
		app.config.ScrollHistory[hash] = int(scrollPos)
		trimScrollHistory(&app.config)
		saveConfig(app.config)
	})

	// _closeAppNative: Beendet die Anwendung sauber.
	w.Bind("_closeAppNative", func() {
		w.Dispatch(func() {
			w.Terminate()
		})
	})

	// nativeFullscreen: Plattformspezifischer Vollbild-Modus.
	w.Bind("nativeFullscreen", func() bool {
		newState := !app.isFullscreen
		w.Dispatch(func() {
			toggleNativeFullscreen(w)
		})
		return newState
	})
}
