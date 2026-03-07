// Package renderer - FB2 (FictionBook 2) Parsing und Rendering.
//
// Liest FB2-Dateien (XML-basiertes russisches E-Book-Format) und wandelt
// sie in anzeigbares HTML um.
//
// FB2-Struktur:
//   - <description><title-info> → Metadaten (Titel, Autor)
//   - <body> → Buchinhalt (Kapitel, Abschnitte, Absätze)
//   - <body name="notes"> → Fußnoten (werden übersprungen)
//
// Mapping FB2 → HTML:
//   - <section>    → <div class="fb2-section">
//   - <title>      → <h2>
//   - <subtitle>   → <h3>
//   - <p>          → <p>
//   - <emphasis>   → <em>
//   - <strong>     → <strong>
//   - <strikethrough> → <del>
//   - <epigraph>   → <blockquote class="fb2-epigraph">
//   - <poem>       → <pre class="fb2-poem">
//   - <v>          → Verszeile mit <br>
//   - <cite>       → <blockquote class="fb2-cite">
//   - <empty-line> → <br>
//   - <code>       → <code>
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// ParseFB2Content konvertiert eine FB2-Datei in anzeigbares HTML.
//
// Verwendet einen XML-Token-Parser um die FB2-Elemente in HTML-Tags
// umzuwandeln. Fußnoten-Bodies (name="notes") werden übersprungen.
//
// @param content  Der FB2-XML-Quelltext.
// @param filename Dateiname für den Titel-Fallback.
// @return Result mit gerendertem HTML und Metadaten, oder Fehler.
func ParseFB2Content(content string, filename string) (*Result, error) {
	// Buchtitel aus dem description-Block extrahieren
	title := extractFB2Title(content)

	// Buchinhalt aus dem body-Block in HTML umwandeln
	htmlContent, chapterCount := convertFB2BodyToHTML(content)

	// Titel-Fallback auf Dateinamen
	if title == "" {
		title = fileBaseName(filename)
	}
	// Sicherstellen dass Inhalt vorhanden ist
	if chapterCount == 0 && strings.TrimSpace(htmlContent) == "" {
		return nil, fmt.Errorf("keine Kapitel in FB2-Datei gefunden")
	}

	return &Result{
		HTML:       htmlContent,
		Title:      title,
		RawContent: content,
	}, nil
}

// IsFB2File prüft ob ein Dateipfad auf eine FB2-Datei zeigt.
//
// @param filePath Dateipfad.
// @return true wenn die Datei .fb2 Endung hat.
func IsFB2File(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".fb2"
}

// extractFB2Title liest den Buchtitel aus dem <book-title>-Tag.
//
// Durchsucht den Quelltext nach dem book-title-Element das im
// description/title-info-Block des FB2-Formats steht.
//
// @param content FB2-XML-Quelltext.
// @return Buchtitel oder leerer String wenn nicht gefunden.
func extractFB2Title(content string) string {
	lower := strings.ToLower(content)
	// Nach <book-title> suchen (mit oder ohne Attribute)
	markers := []string{"<book-title>", "<book-title "}
	for _, marker := range markers {
		idx := strings.Index(lower, marker)
		if idx == -1 {
			continue
		}
		start := idx + len(marker)
		if marker == "<book-title " {
			// Ende des öffnenden Tags finden (überspringt Attribute)
			end := strings.Index(lower[start:], ">")
			if end == -1 {
				continue
			}
			start = start + end + 1
		}
		end := strings.Index(lower[start:], "</book-title>")
		if end == -1 {
			continue
		}
		return strings.TrimSpace(content[start : start+end])
	}
	return ""
}

// convertFB2BodyToHTML konvertiert den <body>-Block einer FB2-Datei zu HTML.
//
// Verarbeitet FB2-XML-Tokens sequenziell und erzeugt entsprechendes HTML.
// Der Fußnoten-Body (name="notes") wird dabei übersprungen.
//
// @param content FB2-XML-Quelltext.
// @return Erzeugtes HTML und Anzahl der verarbeiteten Kapitel.
func convertFB2BodyToHTML(content string) (string, int) {
	var htmlBuilder strings.Builder

	// XML-Decoder erstellen (nicht-strikt für bessere Kompatibilität)
	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.Strict = false

	// Zustandsvariablen für den Token-Parser
	var (
		inBody      bool   // Sind wir im Haupt-Body-Block?
		inNotes     bool   // Sind wir im Fußnoten-Body?
		inPoem      bool   // Sind wir in einem <poem>-Block?
		chapterCount int   // Zähler für Kapitel (Sections)
		firstSect   = true // Erster Abschnitt → kein Trennstrich davor
	)

	for {
		tok, err := decoder.Token()
		if err != nil {
			// EOF oder nicht behebbarer Fehler → Ende der Verarbeitung
			break
		}

		switch t := tok.(type) {

		// --- Öffnende Tags ---
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			switch name {

			case "body":
				// Prüfen ob es ein Fußnoten-Body ist (name="notes")
				inNotes = false
				for _, attr := range t.Attr {
					if strings.ToLower(attr.Name.Local) == "name" &&
						strings.ToLower(attr.Value) == "notes" {
						inNotes = true
					}
				}
				if !inNotes {
					inBody = true
				}

			case "section":
				if inBody && !inNotes {
					// Kapitel-Trennlinie (nicht vor dem ersten Kapitel)
					if !firstSect {
						htmlBuilder.WriteString(`<hr class="fb2-chapter-separator">`)
					}
					firstSect = false
					chapterCount++
					htmlBuilder.WriteString(`<div class="fb2-section">`)
				}

			case "title":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<h2>")
				}

			case "subtitle":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<h3>")
				}

			case "p":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<p>")
				}

			case "emphasis":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<em>")
				}

			case "strong":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<strong>")
				}

			case "strikethrough":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<del>")
				}

			case "epigraph":
				if inBody && !inNotes {
					htmlBuilder.WriteString(`<blockquote class="fb2-epigraph">`)
				}

			case "poem":
				if inBody && !inNotes {
					inPoem = true
					htmlBuilder.WriteString(`<pre class="fb2-poem">`)
				}

			case "v":
				// Verszeile in einem Gedicht - Inhalt wird als Text gelesen
				// das <br> kommt beim schließenden Tag

			case "cite":
				if inBody && !inNotes {
					htmlBuilder.WriteString(`<blockquote class="fb2-cite">`)
				}

			case "code":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<code>")
				}

			case "empty-line":
				if inBody && !inNotes {
					htmlBuilder.WriteString("<br>")
				}
			}

		// --- Schließende Tags ---
		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			switch name {

			case "body":
				inBody = false
				inNotes = false

			case "section":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</div>")
				}

			case "title":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</h2>")
				}

			case "subtitle":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</h3>")
				}

			case "p":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</p>")
				}

			case "emphasis":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</em>")
				}

			case "strong":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</strong>")
				}

			case "strikethrough":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</del>")
				}

			case "epigraph":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</blockquote>")
				}

			case "poem":
				if inBody && !inNotes {
					inPoem = false
					htmlBuilder.WriteString("</pre>")
				}

			case "v":
				// Verszeile abschließen mit Zeilenumbruch
				if inBody && !inNotes && inPoem {
					htmlBuilder.WriteString("\n")
				}

			case "stanza":
				// Gedichtstrophe: Leerzeile als Trenner einfügen
				if inBody && !inNotes && inPoem {
					htmlBuilder.WriteString("\n")
				}

			case "cite":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</blockquote>")
				}

			case "code":
				if inBody && !inNotes {
					htmlBuilder.WriteString("</code>")
				}
			}

		// --- Text-Inhalte ---
		case xml.CharData:
			if inBody && !inNotes {
				text := string(t)
				// Leerzeilen und reine Whitespace-Texte überspringen
				if strings.TrimSpace(text) != "" {
					htmlBuilder.WriteString(escapeHTMLText(text))
				}
			}
		}
	}

	return htmlBuilder.String(), chapterCount
}
