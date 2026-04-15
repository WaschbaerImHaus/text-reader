// Package renderer – PDF-Datei-Rendering für den WebView.
//
// Strategie nach Plattform:
//   - Linux: pdftoppm (aus poppler-utils) rendert jede Seite als PNG.
//            Seiten werden als base64-Data-URI-Bilder eingebettet.
//            Kein Browser-Plugin nötig, funktioniert in WebKitGTK.
//   - Windows: <embed type="application/pdf"> mit base64-Data-URI.
//              Edge WebView2 rendert PDFs nativ.
//   - Fallback (kein pdftoppm, kein Embed-Support): Fehlermeldung + Text.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// IsPDFFile prüft ob ein Dateipfad auf eine PDF-Datei (.pdf) zeigt.
//
// @param path Dateipfad (mit oder ohne Verzeichnis).
// @return true wenn die Datei .pdf Endung hat (Groß-/Kleinschreibung ignoriert).
func IsPDFFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".pdf"
}

// ParsePDF konvertiert PDF-Binärdaten zu HTML.
//
// Ablauf:
//  1. Linux: pdftoppm-Konvertierung (PNG-Bilder pro Seite) → Bild-HTML
//  2. Windows: <embed type="application/pdf"> mit base64-Data-URI
//  3. Fallback: Fehlermeldung mit Installationshinweis
//
// @param data     Rohe PDF-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Inhalt und Metadaten, oder Fehler.
func ParsePDF(data []byte, filename string) (*Result, error) {
	// Linux-Pfad: pdftoppm rendert Seiten zu PNG-Bildern
	if runtime.GOOS == "linux" {
		if result, err := renderPDFWithPdftoppm(data, filename); err == nil {
			return result, nil
		} else {
			log.Printf("ParsePDF: pdftoppm fehlgeschlagen (%v), versuche Embed-Fallback", err)
		}
	}

	// Windows-Pfad und Fallback: <embed> mit base64-Data-URI
	// Edge WebView2 unterstützt PDFs nativ via embed-Tag.
	return renderPDFAsEmbed(data, filename)
}

// renderPDFWithPdftoppm rendert PDF-Seiten mit pdftoppm zu PNG-Bildern.
//
// Schreibt das PDF in eine temporäre Datei, ruft pdftoppm auf und
// liest die erzeugten PNG-Dateien ein. Alle Bilder werden als
// base64-Data-URI in einem HTML-Dokument eingebettet.
//
// Timeout: 60 Sekunden (für große PDFs).
//
// @param data     Rohe PDF-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit Bild-HTML oder Fehler wenn pdftoppm nicht verfügbar.
func renderPDFWithPdftoppm(data []byte, filename string) (*Result, error) {
	// Verfügbarkeit von pdftoppm prüfen
	pdftoppmPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf("pdftoppm nicht gefunden: %w", err)
	}

	// PDF in temporäre Datei schreiben
	tmpPDF, err := os.CreateTemp("", "mdreader-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("temp-Datei erstellen: %w", err)
	}
	defer os.Remove(tmpPDF.Name())

	if _, err := tmpPDF.Write(data); err != nil {
		tmpPDF.Close()
		return nil, fmt.Errorf("PDF in temp-Datei schreiben: %w", err)
	}
	tmpPDF.Close()

	// Ausgabeverzeichnis für PNG-Seiten anlegen
	tmpDir, err := os.MkdirTemp("", "mdreader-pdf-pages-*")
	if err != nil {
		return nil, fmt.Errorf("temp-Verzeichnis erstellen: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// pdftoppm ausführen: alle Seiten als PNG mit 144 DPI rendern
	// -r 144: gute Bildschirmqualität (Standard = 72 DPI, zu niedrig)
	// -png: PNG-Ausgabe statt PPM
	outPrefix := filepath.Join(tmpDir, "page")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, pdftoppmPath, "-r", "144", "-png", tmpPDF.Name(), outPrefix)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm Fehler: %v – stderr: %s", err, stderr.String())
	}

	// Erzeugte PNG-Dateien einlesen und alphabetisch sortieren
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("PNG-Dateien lesen: %w", err)
	}

	// Nur .png-Dateien berücksichtigen, sortiert nach Name
	var pngFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			pngFiles = append(pngFiles, entry.Name())
		}
	}
	sort.Strings(pngFiles)

	if len(pngFiles) == 0 {
		return nil, fmt.Errorf("pdftoppm erzeugte keine PNG-Ausgabe")
	}

	// HTML mit eingebetteten Seiten-Bildern aufbauen
	var htmlBuf strings.Builder
	htmlBuf.WriteString(`<div class="pdf-pages">`)

	for i, pngName := range pngFiles {
		imgData, err := os.ReadFile(filepath.Join(tmpDir, pngName))
		if err != nil {
			log.Printf("ParsePDF: PNG-Datei %s konnte nicht gelesen werden: %v", pngName, err)
			continue
		}
		b64 := base64.StdEncoding.EncodeToString(imgData)
		fmt.Fprintf(&htmlBuf,
			`<div class="pdf-page-wrapper"><img src="data:image/png;base64,%s" class="pdf-page" alt="%s Seite %d"></div>`,
			b64, html.EscapeString(fileBaseName(filename)), i+1,
		)
	}

	htmlBuf.WriteString(`</div>`)

	return &Result{
		HTML:       htmlBuf.String(),
		Title:      fileBaseName(filename),
		RawContent: "",
	}, nil
}

// renderPDFAsEmbed bettet PDF als base64-Data-URI in einen <embed>-Tag ein.
//
// Funktioniert mit Edge WebView2 auf Windows. Auf Linux ohne pdftoppm
// wird diese Methode als Fallback verwendet.
//
// @param data     Rohe PDF-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit Embed-HTML und Metadaten.
func renderPDFAsEmbed(data []byte, filename string) (*Result, error) {
	b64 := base64.StdEncoding.EncodeToString(data)

	// position:fixed bricht aus dem #content max-width heraus und belegt
	// die gesamte Fläche unterhalb der Toolbar.
	embedHTML := fmt.Sprintf(
		`<div class="pdf-viewer" style="position:fixed;top:var(--toolbar-height);left:0;right:0;bottom:0;overflow:hidden;">`+
			`<embed src="data:application/pdf;base64,%s" type="application/pdf" width="100%%" height="100%%">`+
			`</div>`,
		b64,
	)

	return &Result{
		HTML:       embedHTML,
		Title:      fileBaseName(filename),
		RawContent: "",
	}, nil
}
