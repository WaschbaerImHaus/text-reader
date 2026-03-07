// MD Reader - Betrachter für Markdown, EPUB, FB2 und TXT mit GitHub-ähnlicher Darstellung.
//
// Unterstützte Dateiformate: .md, .markdown, .epub, .fb2, .txt
// Features: Drag & Drop, Zoom, Vollbild, Dark/Retro-Mode, TOC, Suche,
//           lokale Bilder, persistente Einstellungen, letzte Datei öffnen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"md-reader/renderer"
	"md-reader/ui"
	"os"
	"path/filepath"
	"runtime"

	webview "github.com/webview/webview_go"
)

// defaultFontSize ist die anfängliche Schriftgröße in Pixeln.
const defaultFontSize = 16

// windowWidth ist die anfängliche Fensterbreite in Pixeln.
const windowWidth = 1000

// windowHeight ist die anfängliche Fensterhöhe in Pixeln.
const windowHeight = 720

// app enthält den globalen Anwendungszustand.
var app *AppState

// AppState hält den aktuellen Zustand der Anwendung.
type AppState struct {
	// webview ist die WebView-Instanz.
	webview webview.WebView
	// isFullscreen gibt an ob nativer Vollbild-Modus aktiv ist.
	isFullscreen bool
	// config enthält die persistenten Benutzereinstellungen.
	config AppConfig
}

// RenderResult ist das JSON-Ergebnis der Datei-Konvertierung.
type RenderResult struct {
	// HTML enthält das gerenderte HTML.
	HTML string `json:"html"`
	// Title ist der extrahierte Titel.
	Title string `json:"title"`
	// Error enthält eine Fehlermeldung falls die Konvertierung fehlschlug.
	Error string `json:"error,omitempty"`
}

func main() {
	// Sicherstellen dass der main-Thread für UI genutzt wird (macOS/Windows)
	runtime.LockOSThread()

	// Build-Nummer ausgeben
	buildNum := readBuildNumber()
	fmt.Printf("MD Reader - Build %s\n", buildNum)

	// Gespeicherte Konfiguration laden (letzte Datei, Zoom, Theme, Layout)
	cfg := loadConfig()

	// WebView-Instanz erstellen
	debug := false
	w := webview.New(debug)
	if w == nil {
		log.Fatal("WebView konnte nicht erstellt werden. Prüfen Sie die WebKit/WebView2-Installation.")
	}
	defer w.Destroy()

	// Anwendungszustand initialisieren
	app = &AppState{
		webview: w,
		config:  cfg,
	}

	// Fenstertitel und Größe setzen
	w.SetTitle("MD Reader")
	w.SetSize(windowWidth, windowHeight, webview.HintNone)

	// ----------------------------------------------------------------
	// Go→JS Bindings
	// ----------------------------------------------------------------

	// processMarkdown: Konvertiert Text-Formate (MD, TXT, FB2) zu HTML.
	w.Bind("processMarkdown", func(content string, filename string) RenderResult {
		result, err := renderer.ParseContent(content, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		return RenderResult{
			HTML:  result.HTML,
			Title: result.Title,
		}
	})

	// processEpub: Konvertiert eine EPUB-Datei (Base64-kodiert) zu HTML.
	// EPUB ist ein ZIP-Archiv und muss binär übertragen werden.
	w.Bind("processEpub", func(base64Data string, filename string) RenderResult {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return RenderResult{Error: "Base64-Dekodierung fehlgeschlagen: " + err.Error()}
		}
		result, err := renderer.ParseEpub(data, filename)
		if err != nil {
			return RenderResult{Error: err.Error()}
		}
		return RenderResult{
			HTML:  result.HTML,
			Title: result.Title,
		}
	})

	// persistState: Speichert Zoom, Theme und Layout in der Konfigurationsdatei.
	// Wird von JS nach jeder Einstellungsänderung aufgerufen.
	w.Bind("persistState", func(fontSize float64, theme string, layout string) {
		app.config.FontSize = int(fontSize)
		app.config.Theme = theme
		app.config.Layout = layout
		saveConfig(app.config)
	})

	// closeApp: Beendet die Anwendung sauber.
	w.Bind("closeApp", func() {
		w.Dispatch(func() {
			w.Terminate()
		})
	})

	// nativeFullscreen: Plattformspezifischer Vollbild-Modus.
	// Gibt den neuen Vollbild-Zustand (true = Vollbild) zurück.
	// Bug #002 Fix: Nur native Vollbild-API verwenden statt HTML5 Fullscreen,
	// damit der OS-Fensterrahmen korrekt entfernt wird.
	w.Bind("nativeFullscreen", func() bool {
		newState := !app.isFullscreen
		w.Dispatch(func() {
			toggleNativeFullscreen(w)
		})
		return newState
	})

	// ----------------------------------------------------------------
	// Initiales HTML rendern mit gespeicherter Konfiguration
	// ----------------------------------------------------------------
	uiCfg := ui.UIConfig{
		FontSize:   app.config.FontSize,
		Theme:      app.config.Theme,
		IsPortrait: app.config.Layout == "portrait",
	}
	initialHTML := ui.BuildInitialHTML(uiCfg)
	w.SetHtml(initialHTML)

	// ----------------------------------------------------------------
	// Datei beim Start laden (CLI-Argument hat Vorrang vor letzter Datei)
	// ----------------------------------------------------------------
	startFile := ""
	if len(os.Args) > 1 && renderer.IsSupportedFile(os.Args[1]) {
		startFile = os.Args[1]
	} else if app.config.LastFile != "" {
		// Letzte Datei wiederherstellen wenn sie noch existiert
		if _, err := os.Stat(app.config.LastFile); err == nil {
			if renderer.IsSupportedFile(app.config.LastFile) {
				startFile = app.config.LastFile
			}
		}
	}
	if startFile != "" {
		loadFileOnStartup(w, startFile)
	}

	// Hauptschleife (blockiert bis Fenster geschlossen wird)
	w.Run()
}

// loadFileOnStartup lädt eine Datei beim Programmstart und zeigt sie an.
//
// Löst relative Bildpfade in HTML auf (base64 Data-URI) und speichert
// den Dateipfad als "letzte Datei" in der Konfiguration.
//
// @param w        Die WebView-Instanz.
// @param filePath Vollständiger Pfad zur Datei.
func loadFileOnStartup(w webview.WebView, filePath string) {
	result, err := renderer.LoadFile(filePath)
	if err != nil {
		log.Printf("Datei konnte nicht geladen werden: %v", err)
		return
	}

	// Relative Bildpfade durch base64-Data-URIs ersetzen (nur bei Textdateien,
	// da EPUB Bilder bereits einbettet)
	if !renderer.IsEpubFile(filePath) {
		result.HTML = renderer.ResolveImagePaths(result.HTML, filepath.Dir(filePath))
	}

	// Dateipfad als zuletzt geöffnete Datei speichern
	app.config.LastFile = filePath
	saveConfig(app.config)

	// HTML und Titel JSON-sicher encodieren für die JS-Übergabe
	htmlJSON, err := json.Marshal(result.HTML)
	if err != nil {
		return
	}
	titleJSON, err := json.Marshal(result.Title)
	if err != nil {
		return
	}

	// Nach kurzem Delay anzeigen (Seite muss erst vollständig geladen sein)
	w.Dispatch(func() {
		w.Eval(fmt.Sprintf(`
			setTimeout(function() {
				showContent(%s, %s);
			}, 150);
		`, string(htmlJSON), string(titleJSON)))
	})
}

// readBuildNumber liest die aktuelle Build-Nummer aus der build.txt Datei.
//
// @return Build-Nummer als String, oder "unbekannt" bei Fehler.
func readBuildNumber() string {
	data, err := os.ReadFile("build.txt")
	if err != nil {
		return "unbekannt"
	}
	num := string(data)
	if len(num) > 0 && num[len(num)-1] == '\n' {
		num = num[:len(num)-1]
	}
	return num
}
