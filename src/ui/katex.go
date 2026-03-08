// Package ui – KaTeX-Integration (offline, eingebettet).
//
// Bettet katex.min.js, katex.min.css und alle Schriftdateien ein.
// Die CSS-Schrift-URLs werden zur Laufzeit durch base64-Data-URIs ersetzt,
// damit KaTeX ohne Webserver im eingebetteten WebView funktioniert.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

import (
	"embed"
	"encoding/base64"
	"io/fs"
	"strings"
	"sync"
)

// Eingebettete KaTeX-Dateien (JavaScript, CSS, Schriften).

//go:embed assets/katex/katex.min.js
var katexMinJS []byte

//go:embed assets/katex/auto-render.min.js
var katexAutoRenderJS []byte

//go:embed assets/katex/katex.min.css
var katexMinCSS []byte

//go:embed assets/katex/fonts
var katexFontsFS embed.FS

// katexOnce stellt sicher dass das CSS-Patching nur einmal ausgeführt wird.
var katexOnce sync.Once

// katexPatchedCSS enthält das KaTeX-CSS mit base64-eingebetteten Schriftdateien.
var katexPatchedCSS string

// KaTeXJS gibt das KaTeX-Haupt-JavaScript als String zurück.
//
// @return katex.min.js als String.
func KaTeXJS() string {
	return string(katexMinJS)
}

// KaTeXAutoRenderJS gibt das KaTeX-Auto-Render-JavaScript als String zurück.
//
// Ermöglicht automatisches Erkennen und Rendern von LaTeX-Formeln in HTML-Text
// (Inline-Formeln mit $...$ und Block-Formeln mit $$...$$).
//
// @return auto-render.min.js als String.
func KaTeXAutoRenderJS() string {
	return string(katexAutoRenderJS)
}

// KaTeXCSS gibt das KaTeX-CSS zurück, in dem alle Schrift-URLs durch
// base64-kodierte Data-URIs ersetzt wurden.
//
// Das Patching erfolgt einmalig lazy (sync.Once) und wird gecacht.
// Nötig weil der WebView keine Dateien vom Dateisystem laden kann.
//
// @return Gepatchtes katex.min.css als String.
func KaTeXCSS() string {
	katexOnce.Do(buildKaTeXCSS)
	return katexPatchedCSS
}

// buildKaTeXCSS patcht das KaTeX-CSS: ersetzt relative font-URLs durch base64 Data-URIs.
//
// Die CSS-Datei enthält Einträge wie:
//
//	src: url('fonts/KaTeX_Main-Regular.woff2') format('woff2')
//
// Diese werden ersetzt durch:
//
//	src: url('data:font/woff2;base64,...') format('woff2')
func buildKaTeXCSS() {
	css := string(katexMinCSS)

	// Alle woff2-Schriftdateien einlesen und als Base64-Map bereitstellen.
	// Schlüssel: "fonts/KaTeX_Name.woff2", Wert: base64-String.
	fontBase64 := make(map[string]string)
	err := fs.WalkDir(katexFontsFS, "assets/katex/fonts", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// Nur .woff2-Dateien werden benötigt
		if !strings.HasSuffix(path, ".woff2") {
			return nil
		}
		data, readErr := katexFontsFS.ReadFile(path)
		if readErr != nil {
			return nil // Schriftdatei übersprungen
		}
		// CSS-Key: "fonts/KaTeX_Name.woff2" (relativer Pfad wie in der CSS-Datei)
		fontName := "fonts/" + d.Name()
		fontBase64[fontName] = base64.StdEncoding.EncodeToString(data)
		return nil
	})
	if err != nil {
		// Fehler beim Walk: ungepatchtes CSS zurückgeben
		katexPatchedCSS = css
		return
	}

	// Alle font-Referenzen im CSS ersetzen.
	// Pattern im CSS: url(fonts/KaTeX_Name.woff2) oder url('fonts/KaTeX_Name.woff2')
	for fontPath, b64 := range fontBase64 {
		dataURI := "data:font/woff2;base64," + b64
		// Mit Anführungszeichen und ohne ersetzen
		css = strings.ReplaceAll(css, "url("+fontPath+")", "url("+dataURI+")")
		css = strings.ReplaceAll(css, "url('"+fontPath+"')", "url('"+dataURI+"')")
		css = strings.ReplaceAll(css, `url("`+fontPath+`")`, `url("`+dataURI+`")`)
	}

	katexPatchedCSS = css
}
