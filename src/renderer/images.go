// Package renderer - Auflösung relativer Bildpfade in HTML.
//
// Ersetzt lokale <img src="..."> Pfade (relativ und absolut) durch
// base64-kodierte Data-URIs, damit Bilder in der WebView angezeigt werden
// können. Die CSP erlaubt nur noch data:-Bilder (keine file:-URLs).
//
// Nur anwendbar wenn der vollständige Dateipfad bekannt ist (CLI-Argument,
// nicht Drag & Drop - Browser-Sicherheitsbeschränkung verhindert vollen Pfad).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14 (Pfadprüfung als Erlaubnisliste, fail closed)
package renderer

import (
	"encoding/base64"
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

// embeddableImageTypes ordnet Bild-Dateiendungen ihren MIME-Typ zu.
// Nur diese Dateien werden eingebettet – ein Dokument soll keine beliebigen
// Dateien (z. B. Schlüssel) als "Bild" in die Seite laden können. Feste
// Tabelle statt mime.TypeByExtension: unter Windows liest diese die Registry
// und liefert je nach System andere Ergebnisse.
var embeddableImageTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".avif": "image/avif",
}

// reservedDeviceNames sind Windows-Gerätenamen, die auch mit Dateiendung
// (z. B. "COM1.png") auf ein Gerät zeigen. Lesen könnte die App blockieren.
var reservedDeviceNames = toSet(
	"con", "prn", "aux", "nul",
	"com0", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9",
	"lpt0", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9",
)

// isReadableImagePath entscheidet nach dem Erlaubnislisten-Prinzip (fail closed),
// ob ein Bildpfad aus dem Dokument gelesen werden darf.
//
// Erlaubt sind nur relative Pfade, Pfade mit Laufwerksbuchstaben ("C:\…") und
// Pfade mit einfachem Wurzel-Trenner ("/home/…", "\Users\…"). Abgelehnt werden
// alle Netzwerk-, NT- und Gerätepfade ("\\server\…", "//server/…",
// "\??\UNC\…", "\\?\…", "\\.\…"): Unter Windows löst schon das Öffnen
// einen SMB-Zugriff aus, der den NTLM-Hash preisgeben kann. Ebenso abgelehnt:
// laufwerksrelative Pfade ("C:bild.png"), NTFS-Datenströme ("bild.png:x")
// und reservierte Gerätenamen ("COM1.png").
//
// Die Prüfung ist bewusst plattformunabhängig ("/" und "\" gelten beide als
// Trenner), damit sie überall gleich wirkt und testbar ist.
//
// @param src Bildpfad aus dem src-Attribut.
// @return true wenn der Pfad gelesen werden darf.
func isReadableImagePath(src string) bool {
	norm := strings.ReplaceAll(src, "/", `\`)
	if norm == "" {
		return false
	}
	// Netzwerk-, NT-Objekt- und Gerätepfade
	if strings.HasPrefix(norm, `\\`) || strings.HasPrefix(norm, `\??\`) {
		return false
	}
	// Laufwerksbuchstabe: muss von einem Trenner gefolgt sein
	rest := norm
	if len(norm) >= 2 && norm[1] == ':' && isASCIILetter(norm[0]) {
		if len(norm) < 3 || norm[2] != '\\' {
			return false
		}
		rest = norm[3:]
	}
	// Weitere Doppelpunkte: NTFS-Datenströme oder ungültige Pfade
	if strings.Contains(rest, ":") {
		return false
	}
	// Reservierte Gerätenamen (Name vor dem ersten Punkt zählt)
	base := rest[strings.LastIndex(rest, `\`)+1:]
	if dot := strings.Index(base, "."); dot >= 0 {
		base = base[:dot]
	}
	return !reservedDeviceNames[strings.ToLower(base)]
}

// isASCIILetter prüft, ob ein Byte ein ASCII-Buchstabe ist.
//
// @param b Zu prüfendes Byte.
// @return true bei A–Z oder a–z.
func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// ResolveImagePaths ersetzt lokale Bildpfade in HTML durch base64-Data-URIs.
//
// Liest die Bilddatei von Disk und bettet sie als base64-kodierte Data-URI ein.
// Dies ist notwendig damit Bilder in der WebView funktionieren, wenn HTML via
// SetHtml() geladen wird (kein Basis-URI gesetzt).
//
// Folgende src-Werte werden NICHT verändert (die CSP blockiert sie ohnehin):
//   - http:// und https:// URLs
//   - data: URIs (bereits eingebettet)
//   - file:// URIs
//   - Netzwerk-, Geräte- und sonstige nicht erlaubte Pfade (werden nie
//     gelesen, siehe isReadableImagePath)
//   - Dateien ohne Bild-Endung (siehe embeddableImageTypes)
//   - nicht lesbare Dateien
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
			!isReadableImagePath(src) {
			return match
		}

		// Nur echte Bilddateien einbetten (MIME-Typ aus fester Tabelle)
		mimeType, isImage := embeddableImageTypes[strings.ToLower(filepath.Ext(src))]
		if !isImage {
			return match
		}

		// Absolute Pfade direkt verwenden, relative an baseDir anhängen
		absPath := filepath.FromSlash(src)
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(baseDir, absPath)
		}

		// Bilddatei lesen
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Datei nicht lesbar → src unverändert lassen (kein Abbruch)
			return match
		}

		// Base64-Data-URI zusammenbauen
		dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
		return prefix + dataURI + suffix
	})
}
