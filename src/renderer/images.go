// Package renderer - Auflösung relativer Bildpfade in HTML.
//
// Ersetzt relative <img src="..."> Pfade durch base64-kodierte Data-URIs,
// damit Bilder in der WebView angezeigt werden können ohne Basis-URI.
//
// Nur anwendbar wenn der vollständige Dateipfad bekannt ist (CLI-Argument,
// nicht Drag & Drop - Browser-Sicherheitsbeschränkung verhindert vollen Pfad).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer

import (
	"encoding/base64"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// imgSrcDouble erkennt <img>-Tags mit doppelt-gequotetem src-Attribut.
// Gruppe 1: Alles bis zum öffnenden Anführungszeichen (inkl.).
// Gruppe 2: Der src-Pfadwert.
// Gruppe 3: Alles ab dem schließenden Anführungszeichen.
var imgSrcDouble = regexp.MustCompile(`(?i)(<img[^>]*?\ssrc\s*=\s*")([^"]+)(")`)

// imgSrcSingle erkennt <img>-Tags mit einfach-gequotetem src-Attribut.
var imgSrcSingle = regexp.MustCompile(`(?i)(<img[^>]*?\ssrc\s*=\s*')([^']+)(')`)

// ResolveImagePaths ersetzt relative Bildpfade in HTML durch base64-Data-URIs.
//
// Liest die Bilddatei von Disk und bettet sie als base64-kodierte Data-URI ein.
// Dies ist notwendig damit Bilder in der WebView funktionieren, wenn HTML via
// SetHtml() geladen wird (kein Basis-URI gesetzt).
//
// Folgende src-Werte werden NICHT verändert:
//   - http:// und https:// URLs
//   - data: URIs (bereits eingebettet)
//   - file:// URIs
//   - Absolute Dateipfade
//
// @param html    HTML-Text mit möglicherweise relativen Bildpfaden.
// @param baseDir Verzeichnis der Quelldatei als Basis für relative Pfade.
// @return HTML mit aufgelösten Bildpfaden als Data-URIs.
func ResolveImagePaths(html string, baseDir string) string {
	// Doppelt- und einfach-gequotete src-Attribute separat verarbeiten
	// (Go regexp unterstützt keine Backreferences)
	html = resolveImgSrc(html, baseDir, imgSrcDouble)
	html = resolveImgSrc(html, baseDir, imgSrcSingle)
	return html
}

// resolveImgSrc ersetzt relative src-Werte die vom gegebenen Regex gefunden werden.
//
// @param html    HTML-Text.
// @param baseDir Basisverzeichnis für relative Pfade.
// @param re      Regex mit Gruppen: (prefix)(src)(suffix).
// @return HTML mit aufgelösten Bildpfaden.
func resolveImgSrc(html string, baseDir string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(html, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		prefix := parts[1] // z.B. `<img src="`
		src := parts[2]    // der Pfadwert
		suffix := parts[3] // schließendes Anführungszeichen

		// Externe und bereits eingebettete URLs unverändert lassen
		lower := strings.ToLower(src)
		if strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") ||
			strings.HasPrefix(lower, "data:") ||
			strings.HasPrefix(lower, "file://") ||
			filepath.IsAbs(src) {
			return match
		}

		// Absoluten Pfad aus baseDir + relativer src berechnen
		absPath := filepath.Join(baseDir, filepath.FromSlash(src))

		// Bilddatei lesen
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Datei nicht lesbar → src unverändert lassen (kein Abbruch)
			return match
		}

		// MIME-Typ anhand der Dateiendung bestimmen
		ext := strings.ToLower(filepath.Ext(absPath))
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "image/png" // Fallback
		}
		// Nur den Typ-Teil verwenden (ohne charset= o.ä. Parameter)
		if idx := strings.Index(mimeType, ";"); idx >= 0 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}

		// Base64-Data-URI zusammenbauen
		dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
		return prefix + dataURI + suffix
	})
}
