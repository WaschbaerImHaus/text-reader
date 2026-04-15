// Package renderer – PDF-Datei-Einbettung für den WebView.
//
// Bettet PDF-Dateien als base64-Daten-URI in ein <embed>-Element ein.
// WebKitGTK (Linux) und Edge WebView2 (Windows) rendern PDFs nativ.
// Kein externes Tool benötigt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

// IsPDFFile prüft ob ein Dateipfad auf eine PDF-Datei (.pdf) zeigt.
//
// @param path Dateipfad (mit oder ohne Verzeichnis).
// @return true wenn die Datei .pdf Endung hat (Groß-/Kleinschreibung ignoriert).
func IsPDFFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".pdf"
}

// ParsePDF bettet PDF-Binärdaten als base64-Daten-URI in einen <embed>-Tag ein.
//
// Das erzeugte HTML nutzt position:fixed um unterhalb der Toolbar (44px)
// die gesamte Bildschirmfläche zu belegen – unabhängig vom max-width des
// #content-Containers.
//
// @param data     Rohe PDF-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Einbettung und Metadaten, oder Fehler.
func ParsePDF(data []byte, filename string) (*Result, error) {
	// PDF als base64 kodieren
	// Hinweis: Leere data-Slices werden nicht als Fehler behandelt –
	// die Validierung des Dateiinhalts liegt beim Aufrufer (LoadFile).
	b64 := base64.StdEncoding.EncodeToString(data)

	// Embed-Element: position:fixed bricht aus dem #content max-width heraus
	// und belegt die gesamte Fläche unterhalb der Toolbar (--toolbar-height = 44px).
	html := fmt.Sprintf(
		`<div class="pdf-viewer" style="position:fixed;top:44px;left:0;right:0;bottom:0;overflow:hidden;">`+
			`<embed src="data:application/pdf;base64,%s" type="application/pdf" width="100%%" height="100%%">`+
			`</div>`,
		b64,
	)

	return &Result{
		HTML:       html,
		Title:      fileBaseName(filename),
		RawContent: "", // Binärdaten werden nicht als RawContent gespeichert
	}, nil
}
