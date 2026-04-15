// Package renderer – PostScript-Datei-Rendering.
//
// Konvertiert PostScript-Dateien zu PDF via Ghostscript (gs).
// Ist gs nicht verfügbar oder schlägt die Konvertierung fehl,
// wird der PS-Quelltext als <pre>-Block dargestellt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IsPSFile prüft ob ein Dateipfad auf eine PostScript-Datei (.ps) zeigt.
//
// @param path Dateipfad (mit oder ohne Verzeichnis).
// @return true wenn die Datei .ps Endung hat (Groß-/Kleinschreibung ignoriert).
func IsPSFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".ps"
}

// ParsePS konvertiert PostScript-Daten zu HTML.
//
// Ablauf:
//  1. Ghostscript (gs) wird aufgerufen: PS-Daten über stdin, PDF-Bytes aus stdout.
//  2. Erfolg → ParsePDF(pdfBytes, filename) → natives PDF-Rendering im WebView.
//  3. Fehler (gs nicht vorhanden, Timeout, Konvertierungsfehler)
//     → ParseTextContent(string(data), filename) → <pre>-Block Fallback.
//
// @param data     Rohe PostScript-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Inhalt und Metadaten, oder Fehler.
func ParsePS(data []byte, filename string) (*Result, error) {
	// Ghostscript-Konvertierung mit 30-Sekunden-Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// gs liest PS von stdin und schreibt PDF nach stdout
	cmd := exec.CommandContext(ctx, "gs",
		"-sDEVICE=pdfwrite", // Ausgabe als PDF
		"-sOutputFile=-",    // - bedeutet stdout
		"-dBATCH",           // Kein interaktiver Modus
		"-dNOPAUSE",         // Nicht auf Tastendruck warten
		"-dQUIET",           // Keine Fortschrittsausgabe
		"-q",                // Noch weniger Ausgabe
		"-",                 // Eingabe von stdin lesen
	)
	cmd.Stdin = bytes.NewReader(data)

	pdfBytes, err := cmd.Output()
	if err == nil && len(pdfBytes) > 4 {
		// Prüfen ob die Ausgabe wirklich eine PDF-Signatur hat (%PDF)
		if bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
			return ParsePDF(pdfBytes, filename)
		}
	}

	// Fallback: PS als Quelltext anzeigen
	if err != nil {
		log.Printf("ParsePS: gs nicht verfügbar oder Fehler: %v – zeige Text-Fallback", err)
	}
	return ParseTextContent(string(data), filename)
}
