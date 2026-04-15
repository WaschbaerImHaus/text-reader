// Package renderer – PostScript-Datei-Rendering.
//
// Konvertiert PostScript-Dateien direkt zu PNG-Bildern via Ghostscript (gs).
// Jede PS-Seite wird als 144-DPI-PNG gerendert und als base64-Data-URI
// in einem HTML-Bild-Element eingebettet.
//
// Vorteile gegenüber dem PS→PDF→embed-Ansatz:
//   - Kein Browser-Plugin für PDF-Anzeige nötig (WebKitGTK, WebView2)
//   - Funktioniert auf Linux und Windows identisch
//   - Kein X11-Fenster-Flash (kein Display-Device-Initialisierung)
//
// Ist gs nicht verfügbar, wird der PS-Quelltext als <pre>-Block dargestellt
// mit einem Installationshinweis.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	htmlpkg "html"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

	// Windows: Standardinstallationspfad durchsuchen.
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
//  1. Ghostscript (gs) wird aufgerufen: PS-Daten von Temp-Datei, PNG-Bilder in Temp-Dir.
//     -sDEVICE=png16m rendert direkt zu Bildern – kein PDF-Zwischenschritt, kein Embed-Tag.
//  2. Erfolg → PNG-Bilder als base64-Data-URI in HTML einbetten.
//  3. gs nicht gefunden → Installationshinweis + PS als Quelltext.
//  4. gs-Fehler → PS als Quelltext (Fallback).
//
// @param data     Rohe PostScript-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Inhalt und Metadaten, oder Fehler.
func ParsePS(data []byte, filename string) (*Result, error) {
	// Ghostscript-Executable suchen (plattformübergreifend)
	gsExe := findGhostscriptExecutable()
	if gsExe == "" {
		log.Printf("ParsePS: Ghostscript nicht gefunden – zeige Hinweis + Text-Fallback")
		return parsePSNoGhostscript(data, filename)
	}

	// PS → PNG rendering mit Ghostscript
	result, err := renderPSToImages(gsExe, data, filename)
	if err != nil {
		log.Printf("ParsePS: Ghostscript-Rendering fehlgeschlagen: %v – zeige Text-Fallback", err)
		return ParseTextContent(string(data), filename)
	}
	return result, nil
}

// renderPSToImages rendert alle Seiten einer PS-Datei zu PNG-Bildern via Ghostscript.
//
// Schreibt PS in eine temporäre Datei, ruft gs mit dem png16m-Device auf
// und liest die erzeugten PNG-Dateien ein. Alle Bilder werden als
// base64-Data-URI in HTML eingebettet (gleiche Struktur wie renderPDFWithPdftoppm).
//
// Timeout: 60 Sekunden (für große PS-Dokumente).
//
// @param gsExe    Vollständiger Pfad zur Ghostscript-Executable.
// @param data     Rohe PostScript-Binärdaten.
// @param filename Dateiname für den Titel und ALT-Text.
// @return Result mit Bild-HTML oder Fehler.
func renderPSToImages(gsExe string, data []byte, filename string) (*Result, error) {
	// PS in temporäre Datei schreiben (gs liest besser von Datei als stdin bei Multi-Page)
	tmpPS, err := os.CreateTemp("", "mdreader-*.ps")
	if err != nil {
		return nil, fmt.Errorf("temp-Datei erstellen: %w", err)
	}
	defer os.Remove(tmpPS.Name())

	if _, err := tmpPS.Write(data); err != nil {
		tmpPS.Close()
		return nil, fmt.Errorf("PS in temp-Datei schreiben: %w", err)
	}
	tmpPS.Close()

	// Ausgabeverzeichnis für PNG-Seiten anlegen
	tmpDir, err := os.MkdirTemp("", "mdreader-ps-pages-*")
	if err != nil {
		return nil, fmt.Errorf("temp-Verzeichnis erstellen: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// gs aufrufen: PS → PNG-Bilder (eine Datei pro Seite).
	// -sDEVICE=png16m: 24-Bit-Farb-PNG (kein Display-Device, kein X11-Fenster).
	// -r144:           144 DPI – gute Bildschirmqualität (Standard 72 DPI = zu niedrig).
	// -sOutputFile=...: Ausgabemuster mit %d für Seitennummer.
	// -dBATCH:          Kein interaktiver Modus.
	// -dNOPAUSE:        Nicht auf Tastendruck zwischen Seiten warten.
	// -dSAFER:          Sicherer Modus: keine Datei-I/O aus PS-Code.
	// -q:               Keine Fortschrittsausgabe.
	outPattern := filepath.Join(tmpDir, "page%d.png")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, gsExe,
		"-sDEVICE=png16m",
		"-r144",
		fmt.Sprintf("-sOutputFile=%s", outPattern),
		"-dBATCH",
		"-dNOPAUSE",
		"-dSAFER",
		"-q",
		tmpPS.Name(),
	)
	cmd.Stderr = &stderr
	cmd.WaitDelay = 5 * time.Second

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gs Fehler: %v – stderr: %s", err, stderr.String())
	}

	// Erzeugte PNG-Dateien einlesen und alphabetisch sortieren
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("PNG-Dateien lesen: %w", err)
	}

	var pngFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			pngFiles = append(pngFiles, entry.Name())
		}
	}
	sort.Strings(pngFiles)

	if len(pngFiles) == 0 {
		return nil, fmt.Errorf("gs erzeugte keine PNG-Ausgabe (stderr: %s)", stderr.String())
	}

	// HTML mit eingebetteten Seiten-Bildern aufbauen
	var htmlBuf strings.Builder
	htmlBuf.WriteString(`<div class="pdf-pages">`)

	for i, pngName := range pngFiles {
		imgData, err := os.ReadFile(filepath.Join(tmpDir, pngName))
		if err != nil {
			log.Printf("ParsePS: PNG-Datei %s konnte nicht gelesen werden: %v", pngName, err)
			continue
		}
		b64 := base64.StdEncoding.EncodeToString(imgData)
		fmt.Fprintf(&htmlBuf,
			`<div class="pdf-page-wrapper"><img src="data:image/png;base64,%s" class="pdf-page" alt="%s Seite %d"></div>`,
			b64, htmlpkg.EscapeString(fileBaseName(filename)), i+1,
		)
	}

	htmlBuf.WriteString(`</div>`)

	return &Result{
		HTML:       htmlBuf.String(),
		Title:      fileBaseName(filename),
		RawContent: "",
	}, nil
}

// parsePSNoGhostscript gibt einen Installationshinweis und den PS-Quelltext zurück.
//
// Wird aufgerufen wenn Ghostscript nicht installiert ist. Zeigt einen
// erklärenden Hinweisblock gefolgt vom PS-Quelltext.
//
// @param data     Rohe PostScript-Binärdaten.
// @param filename Dateiname für den Titel.
// @return Result mit Hinweis-HTML.
func parsePSNoGhostscript(data []byte, filename string) (*Result, error) {
	// Betriebssystem-spezifische Installationsanweisung
	var installHint string
	switch runtime.GOOS {
	case "windows":
		installHint = `Install Ghostscript via the MD Reader Setup (optional component "Ghostscript") ` +
			`or download from <b>ghostscript.com</b>.`
	default:
		installHint = `Install Ghostscript: <code>sudo apt-get install ghostscript</code>`
	}

	hintHTML := fmt.Sprintf(
		`<div class="ps-no-gs-hint" style="background:var(--blockquote-bg,#f6f8fa);border-left:4px solid #e36209;`+
			`padding:12px 16px;margin:16px 0;border-radius:0 4px 4px 0;font-family:inherit;">`+
			`<strong>PostScript-Rendering nicht möglich</strong><br>`+
			`Ghostscript ist nicht installiert oder wurde nicht gefunden.<br>`+
			`%s<br><br>`+
			`<small>Die Datei wird unten als Quelltext angezeigt.</small>`+
			`</div>`,
		installHint,
	)

	// PS-Quelltext als <pre>-Block anhängen
	textResult, err := ParseTextContent(string(data), filename)
	if err != nil {
		return &Result{HTML: hintHTML, Title: fileBaseName(filename)}, nil
	}

	return &Result{
		HTML:       hintHTML + textResult.HTML,
		Title:      textResult.Title,
		RawContent: textResult.RawContent,
	}, nil
}
