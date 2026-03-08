// Webview-Bindings für den MD Reader.
//
// Enthält alle Go→JavaScript-Bindings: Dateiverarbeitung,
// Einstellungen, Vollbild und App-Steuerung.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"md-reader/renderer"

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
	// Schlüssel sortieren für deterministisches Kürzen
	keys := make([]string, 0, len(cfg.ScrollHistory))
	for k := range cfg.ScrollHistory {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Älteste (lexikographisch kleinste) Einträge entfernen
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

// registerBindings registriert alle Go→JS-Bindings am WebView.
//
// @param w Die WebView-Instanz.
func registerBindings(w webview.WebView) {
	// processMarkdown: Konvertiert Text-Formate (MD, TXT, FB2) zu HTML.
	// Berechnet FNV-64a-Hash des Inhalts und gibt gespeicherte Scroll-Position zurück.
	w.Bind("processMarkdown", func(content string, filename string) RenderResult {
		result, err := renderer.ParseContent(content, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		hash := computeHash([]byte(content))
		scrollPos := scrollPosForHash(hash)
		return RenderResult{
			HTML:      result.HTML,
			Title:     result.Title,
			FileHash:  hash,
			ScrollPos: scrollPos,
		}
	})

	// processEpub: Konvertiert eine EPUB-Datei (Base64-kodiert) zu HTML.
	// Berechnet FNV-64a-Hash der dekodierten Binärdaten.
	w.Bind("processEpub", func(base64Data string, filename string) RenderResult {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return RenderResult{Error: "Base64-Dekodierung fehlgeschlagen: " + err.Error()}
		}
		result, err := renderer.ParseEpub(data, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		hash := computeHash(data)
		scrollPos := scrollPosForHash(hash)
		return RenderResult{
			HTML:      result.HTML,
			Title:     result.Title,
			FileHash:  hash,
			ScrollPos: scrollPos,
		}
	})

	// persistState: Speichert Zoom, Theme und Layout.
	w.Bind("persistState", func(fontSize float64, theme string, layout string) {
		app.config.FontSize = int(fontSize)
		app.config.Theme = theme
		app.config.Layout = layout
		saveConfig(app.config)
	})

	// persistLastFile: Speichert den vollständigen Dateipfad in der Konfiguration.
	// Die Scroll-Position wird separat per saveScrollPos gespeichert.
	//
	// Sicherheit: Pfad-Validierung gegen Path-Traversal (RISK-004 Fix).
	//
	// @param path Vollständiger, validierter Dateipfad.
	w.Bind("persistLastFile", func(path string) {
		if path != "" {
			// Nur existierende, unterstützte Dateien als LastFile akzeptieren
			if renderer.IsSupportedFile(path) {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					app.config.LastFile = path
					saveConfig(app.config)
				}
			}
		}
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

	// openFilePicker: Öffnet den nativen Datei-Öffnen-Dialog (Strg+O).
	//
	// Lösung für #005: WebView2 exponiert aus Sicherheitsgründen keine Dateipfade
	// im DataTransfer. Der native Dialog gibt den vollständigen Pfad zurück.
	w.Bind("openFilePicker", func() {
		path := openFilePickerBlocking(w)
		if path != "" {
			loadFileOnStartup(w, path)
		}
	})
}

// loadFileOnStartup lädt eine Datei beim Programmstart (oder per Datei-Dialog)
// und zeigt sie im WebView an. Stellt die zuletzt gespeicherte Scroll-Position wieder her.
//
// @param w        Die WebView-Instanz.
// @param filePath Vollständiger Pfad zur Datei.
func loadFileOnStartup(w webview.WebView, filePath string) {
	// Rohe Bytes lesen (für Hash-Berechnung konsistent mit processMarkdown/processEpub)
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht gelesen werden: %v", err)
		return
	}

	result, err := renderer.LoadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht gerendert werden: %v", err)
		return
	}
	if !renderer.IsEpubFile(filePath) {
		result.HTML = renderer.ResolveImagePaths(result.HTML, filepath.Dir(filePath))
	}

	// Zuletzt geöffnete Datei persistieren
	app.config.LastFile = filePath
	saveConfig(app.config)

	// Hash berechnen und gespeicherte Scroll-Position nachschlagen
	hash := computeHash(rawBytes)
	scrollPos := scrollPosForHash(hash)

	// JSON-sichere Werte für JS-Eval vorbereiten
	htmlJSON, err := json.Marshal(result.HTML)
	if err != nil {
		return
	}
	titleJSON, err := json.Marshal(result.Title)
	if err != nil {
		return
	}
	hashJSON, err := json.Marshal(hash)
	if err != nil {
		return
	}

	w.Dispatch(func() {
		w.Eval(fmt.Sprintf(`
            setTimeout(function() {
                showContent(%s, %s, %d, %s);
            }, 150);
        `, string(htmlJSON), string(titleJSON), scrollPos, string(hashJSON)))
	})
}
