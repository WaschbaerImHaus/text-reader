// MD Reader - Betrachter für Markdown, EPUB, FB2 und TXT mit GitHub-ähnlicher Darstellung.
//
// Unterstützte Dateiformate: .md, .markdown, .epub, .fb2, .txt
// Features: Drag & Drop, Zoom, Vollbild, Dark/Retro-Mode, TOC, Suche,
// lokale Bilder, persistente Einstellungen, letzte Datei öffnen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-28 (Programm-Identität, Abbruch ohne Display)
package main

import (
	"log"
	"md-reader/renderer"
	"md-reader/ui"
	"os"
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
	// FileHash ist der FNV-64a-Hash des Dateiinhalts (für Scroll-History).
	FileHash string `json:"fileHash"`
	// ScrollPos ist die gespeicherte Scroll-Position für diesen Dateiinhalt.
	ScrollPos int `json:"scrollPos"`
}

func main() {
	// Sicherstellen dass der main-Thread für UI genutzt wird (macOS/Windows)
	runtime.LockOSThread()

	// Gespeicherte Konfiguration laden (letzte Datei, Zoom, Theme, Layout)
	cfg := loadConfig()

	// Feste Programm-Identität "md-reader" (Linux: Fensterklasse und
	// Prozessname, auch bei Start aus dem memfd des Starters), danach prüfen,
	// ob eine grafische Oberfläche erreichbar ist (sonst Meldung + Exit 1
	// statt SIGSEGV in webview_go). Windows/macOS: No-Op.
	setupPlatformIdentity()
	ensureDisplayAvailable()

	// WebView-Instanz erstellen
	debug := false
	w := webview.New(debug)
	// Hinweis: webview_go gibt nie nil zurück, auch wenn das Fenster nicht
	// angelegt werden konnte – den Display-Fall fängt ensureDisplayAvailable ab.
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

	// App-Icon setzen (Titelleiste, Taskleiste, Alt+Tab)
	setAppIcon(w)

	// Nativen GTK-Drag&Drop-Handler einrichten (Linux: WebKitGTK liefert
	// im JS-Drop-Event keine Dateipfade – GTK-Signal direkt abfangen)
	setupNativeFileDrop(w)

	// Menüleiste (nur macOS: ohne sie wirken ⌘C, ⌘V, ⌘A und ⌘Q nicht)
	setupPlatformMenu(w)

	// ----------------------------------------------------------------
	// Go→JS Bindings registrieren (aus bindings.go)
	// ----------------------------------------------------------------
	registerBindings(w)

	// ----------------------------------------------------------------
	// Initiales HTML rendern mit gespeicherter Konfiguration
	// ----------------------------------------------------------------
	uiCfg := ui.UIConfig{
		FontSize:        app.config.FontSize,
		DefaultFontSize: defaultFontSize, // Konstante Basis für Zoom-Prozentanzeige
		Theme:           app.config.Theme,
		IsPortrait:      app.config.Layout == "portrait",
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
		loadFileNative(w, startFile)
	}

	// Hauptschleife (blockiert bis Fenster geschlossen wird)
	w.Run()

	// Temp-Ansichtsdatei (große Dokumente, siehe displayFullHTML) aufräumen
	ui.CleanupTempViewFile()
}
