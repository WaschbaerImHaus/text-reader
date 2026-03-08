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
	"log"
	"os"
	"path/filepath"

	"md-reader/renderer"

	webview "github.com/webview/webview_go"
)

// registerBindings registriert alle Go→JS-Bindings am WebView.
//
// @param w Die WebView-Instanz.
func registerBindings(w webview.WebView) {
	// processMarkdown: Konvertiert Text-Formate (MD, TXT, FB2) zu HTML.
	w.Bind("processMarkdown", func(content string, filename string) RenderResult {
		result, err := renderer.ParseContent(content, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		return RenderResult{HTML: result.HTML, Title: result.Title}
	})

	// processEpub: Konvertiert eine EPUB-Datei (Base64-kodiert) zu HTML.
	w.Bind("processEpub", func(base64Data string, filename string) RenderResult {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return RenderResult{Error: "Base64-Dekodierung fehlgeschlagen: " + err.Error()}
		}
		result, err := renderer.ParseEpub(data, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		return RenderResult{HTML: result.HTML, Title: result.Title}
	})

	// persistState: Speichert Zoom, Theme und Layout.
	w.Bind("persistState", func(fontSize float64, theme string, layout string) {
		app.config.FontSize = int(fontSize)
		app.config.Theme = theme
		app.config.Layout = layout
		saveConfig(app.config)
	})

	// persistLastFile: Speichert Dateipfad und Scroll-Position.
	// Sicherheit: Pfad-Validierung gegen path traversal (RISK-004 Fix).
	w.Bind("persistLastFile", func(path string, scrollPos float64) {
		if path != "" {
			// Nur existierende, unterstützte Dateien als LastFile akzeptieren
			if renderer.IsSupportedFile(path) {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					app.config.LastFile = path
				}
			}
		}
		app.config.LastFileScrollPos = int(scrollPos)
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

// loadFileOnStartup lädt eine Datei beim Programmstart und zeigt sie an.
//
// @param w        Die WebView-Instanz.
// @param filePath Vollständiger Pfad zur Datei.
func loadFileOnStartup(w webview.WebView, filePath string) {
	result, err := renderer.LoadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht geladen werden: %v", err)
		return
	}
	if !renderer.IsEpubFile(filePath) {
		result.HTML = renderer.ResolveImagePaths(result.HTML, filepath.Dir(filePath))
	}
	app.config.LastFile = filePath
	saveConfig(app.config)

	htmlJSON, err := json.Marshal(result.HTML)
	if err != nil {
		return
	}
	titleJSON, err := json.Marshal(result.Title)
	if err != nil {
		return
	}
	scrollPos := app.config.LastFileScrollPos

	w.Dispatch(func() {
		w.Eval(fmt.Sprintf(`
            setTimeout(function() {
                showContent(%s, %s);
                setTimeout(function() {
                    document.getElementById('main').scrollTop = %d;
                }, 80);
            }, 150);
        `, string(htmlJSON), string(titleJSON), scrollPos))
	})
}
