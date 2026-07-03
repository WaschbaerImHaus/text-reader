//go:build ignore

// Debugging-Skript: Prüft ob nach ParseEpub noch rohe Datei-Links
// (href="*.xhtml" u.ä.) im gerenderten HTML übrig sind.
//
// Aufruf: go run ../debugging/epub-link-check.go <datei.epub>  (aus src/)
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-07-03
package main

import (
	"fmt"
	"os"
	"regexp"

	"md-reader/renderer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Aufruf: go run epub-link-check.go <datei.epub>")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Lesefehler:", err)
		os.Exit(1)
	}
	result, err := renderer.ParseEpub(data, os.Args[1])
	if err != nil {
		fmt.Println("Parse-Fehler:", err)
		os.Exit(1)
	}
	fmt.Println("Titel:", result.Title)
	fmt.Println("HTML-Größe:", len(result.HTML))

	// Alle href-Attribute einsammeln und klassifizieren
	re := regexp.MustCompile(`(?i)<a[^>]*?\shref\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(result.HTML, -1)
	anchor, external, raw := 0, 0, 0
	for _, m := range matches {
		href := m[1]
		switch {
		case len(href) > 0 && href[0] == '#':
			anchor++
		case regexp.MustCompile(`(?i)^(https?:|mailto:|data:|javascript:)`).MatchString(href):
			external++
		default:
			raw++
			if raw <= 10 {
				fmt.Println("ROHER LINK ÜBRIG:", href)
			}
		}
	}
	fmt.Printf("Links gesamt: %d | Anker: %d | Extern: %d | Roh (Bug!): %d\n",
		len(matches), anchor, external, raw)
	if raw > 0 {
		os.Exit(2)
	}
	fmt.Println("OK: Keine rohen Datei-Links mehr vorhanden.")
}
