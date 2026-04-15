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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// findGhostscriptExecutable sucht nach der Ghostscript-Executable im System.
//
// Auf Linux heißt die Executable "gs". Auf Windows installiert der offizielle
// GPL Ghostscript Installer "gswin64c.exe" (64-Bit Console) oder "gswin32c.exe"
// (32-Bit) – aber kein "gs.exe". Daher werden auf Windows mehrere Namen und
// der Standardinstallationspfad durchsucht.
//
// @return Vollständiger Pfad zur Ghostscript-Executable, oder "" wenn nicht gefunden.
func findGhostscriptExecutable() string {
	// Kandidaten in Prioritätsreihenfolge (gs zuerst für Linux-Kompatibilität)
	candidates := []string{"gs", "gswin64c", "gswin64", "gswin32c", "gswin32"}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	// Windows: Standardinstallationspfad durchsuchen
	// Der offizielle Ghostscript Installer legt Dateien nach
	// C:\Program Files\gs\gs<version>\bin\ ab und trägt diesen Pfad
	// nur in die System-PATH ein – laufende Prozesse sehen diese
	// Änderung erst nach einem Neustart. Daher direkter Dateisystem-Check.
	if runtime.GOOS == "windows" {
		basePaths := []string{
			`C:\Program Files\gs`,
			`C:\Program Files (x86)\gs`,
		}
		exeNames := []string{"gswin64c.exe", "gswin64.exe", "gswin32c.exe", "gs.exe"}
		for _, basePath := range basePaths {
			entries, err := os.ReadDir(basePath)
			if err != nil {
				continue
			}
			// Neueste Version zuerst (umgekehrte Sortierung, da Versionsordner
			// lexikographisch aufsteigend sortiert sind, z.B. gs10.05.0)
			for i := len(entries) - 1; i >= 0; i-- {
				entry := entries[i]
				if !entry.IsDir() {
					continue
				}
				for _, exeName := range exeNames {
					fullPath := filepath.Join(basePath, entry.Name(), "bin", exeName)
					if _, err := os.Stat(fullPath); err == nil {
						log.Printf("findGhostscriptExecutable: gefunden unter %s", fullPath)
						return fullPath
					}
				}
			}
		}
	}
	return ""
}

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
	// Ghostscript-Executable suchen (plattformübergreifend)
	gsExe := findGhostscriptExecutable()
	if gsExe == "" {
		log.Printf("ParsePS: Ghostscript nicht gefunden – zeige Text-Fallback")
		return ParseTextContent(string(data), filename)
	}

	// Ghostscript-Konvertierung mit 30-Sekunden-Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// gs liest PS von stdin und schreibt PDF nach stdout
	cmd := exec.CommandContext(ctx, gsExe,
		"-sDEVICE=pdfwrite", // Ausgabe als PDF
		"-sOutputFile=-",    // - bedeutet stdout
		"-dBATCH",           // Kein interaktiver Modus
		"-dNOPAUSE",         // Nicht auf Tastendruck warten
		"-dQUIET",           // Keine Fortschrittsausgabe
		"-q",                // Noch weniger Ausgabe
		"-",                 // Eingabe von stdin lesen
	)
	cmd.Stdin = bytes.NewReader(data)
	// WaitDelay stellt sicher dass Pipes vollständig geleert werden
	// bevor der Prozess nach einem Timeout als beendet gilt.
	cmd.WaitDelay = 5 * time.Second

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	pdfBytes := stdout.Bytes()

	if err == nil && len(pdfBytes) > 4 {
		// Prüfen ob die Ausgabe wirklich eine PDF-Signatur hat (%PDF)
		if bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
			return ParsePDF(pdfBytes, filename)
		}
		// gs lief erfolgreich aber erzeugte keine gültige PDF-Datei
		log.Printf("ParsePS: gs lieferte keine gültige PDF-Ausgabe (kein %%PDF-Header) – zeige Text-Fallback")
	}

	// Fallback: PS als Quelltext anzeigen
	if err != nil {
		log.Printf("ParsePS: gs nicht verfügbar oder Fehler: %v – zeige Text-Fallback", err)
	}
	return ParseTextContent(string(data), filename)
}
