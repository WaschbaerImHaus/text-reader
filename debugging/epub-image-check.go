//go:build ignore

// Debug-Skript: prüft ein EPUB auf nicht eingebettete Bildreferenzen und
// auf Überschreitung des WebView2-2-MB-Limits (Bug #012).
//
// Aufruf (aus src/):
//   go run ../debugging/epub-image-check.go pfad/zur/datei.epub
//
// Exit-Codes: 0 = alles ok, 2 = rohe Bildpfade gefunden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-07-04
package main

import (
	"fmt"
	"os"
	"regexp"

	"md-reader/renderer"
	"md-reader/ui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Aufruf: go run epub-image-check.go <datei.epub>")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Lesefehler:", err)
		os.Exit(1)
	}
	result, err := renderer.ParseEpub(data, os.Args[1])
	if err != nil {
		fmt.Println("ParseEpub-Fehler:", err)
		os.Exit(1)
	}

	// Rohe (nicht eingebettete) Bildreferenzen suchen: alles außer data:-URIs
	imgSrc := regexp.MustCompile(`(?i)<img[^>]*?\ssrc\s*=\s*["']([^"']+)["']`)
	svgHref := regexp.MustCompile(`(?i)<image[^>]*?\s(?:xlink:)?href\s*=\s*["']([^"']+)["']`)
	raw := 0
	for _, re := range []*regexp.Regexp{imgSrc, svgHref} {
		for _, m := range re.FindAllStringSubmatch(result.HTML, -1) {
			src := m[1]
			if len(src) >= 5 && src[:5] == "data:" {
				continue
			}
			fmt.Println("ROH (nicht eingebettet):", src)
			raw++
		}
	}

	fmt.Printf("Titel: %s\n", result.Title)
	fmt.Printf("Inhalts-HTML: %d Bytes\n", len(result.HTML))
	fmt.Printf("Rohe Bildreferenzen: %d\n", raw)
	if ui.NeedsFileNavigation(result.HTML) {
		fmt.Println("Hinweis: Dokument > MaxInlineHTMLSize → wird per file://-Temp-Datei geladen")
	}
	if raw > 0 {
		os.Exit(2)
	}
}
