//go:build ignore

// Debug-Skript: baut für eine Datei das vollständige Seiten-HTML genau so,
// wie die App es an den WebView übergibt, und schreibt es in eine Datei.
// Damit lässt sich prüfen, ob CSS/JS korrekt eingebettet sind und ob die
// Seite das WebView2-Größenlimit überschreitet.
//
// Aufruf (aus src/):
//
//	go run ../debugging/full-html-dump.go pfad/zur/datei.md ausgabe.html
//	go run ../debugging/full-html-dump.go pfad/zur/datei.md ausgabe.html textmode [#fg #bg]
//
// Mit "textmode" wird die Seite im Text-Modus gebaut (optional mit Farben).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Option textmode)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"md-reader/renderer"
	"md-reader/ui"
)

func main() {
	if len(os.Args) < 3 || (len(os.Args) > 3 && os.Args[3] != "textmode") {
		fmt.Fprintln(os.Stderr, "Aufruf: go run ../debugging/full-html-dump.go <datei> <ausgabe.html> [textmode [#fg #bg]]")
		os.Exit(1)
	}
	inputPath := os.Args[1]
	result, err := renderer.LoadFile(inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Rendern fehlgeschlagen:", err)
		os.Exit(1)
	}
	result.HTML = renderer.ResolveImagePaths(result.HTML, filepath.Dir(inputPath))
	// Optional Text-Modus mit Farben
	textMode := ui.NewTextMode(false, "", "")
	if len(os.Args) > 3 {
		fg, bg := "", ""
		if len(os.Args) > 5 {
			fg, bg = os.Args[4], os.Args[5]
		}
		textMode = ui.NewTextMode(true, fg, bg)
	}
	fullHTML := ui.BuildInitialHTML(ui.UIConfig{
		FontSize:          16,
		DefaultFontSize:   16,
		Theme:             "dark",
		ContentHTML:       result.HTML,
		PageTitle:         result.Title,
		FileHash:          "debug",
		TextMode:          textMode,
		TextModeAvailable: renderer.IsTextOnlyFile(inputPath),
	})
	if err := os.WriteFile(os.Args[2], []byte(fullHTML), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "Schreiben fehlgeschlagen:", err)
		os.Exit(1)
	}
	// Größe und Anzeigeweg ausgeben (SetHtml oder Temp-Datei per file://)
	fmt.Printf("Inhalt: %d Bytes, Seite: %d Bytes, Temp-Datei-Weg: %v\n",
		len(result.HTML), len(fullHTML), ui.NeedsFileNavigation(fullHTML))
}
