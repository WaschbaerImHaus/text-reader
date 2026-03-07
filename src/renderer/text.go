// Package renderer - Textdatei- und HTML-Datei-Rendering.
//
// Wandelt einfache Textdateien (.txt) und HTML-Dateien (.html, .htm)
// in anzeigbares HTML um.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer

import (
	"html"
	"path/filepath"
	"strings"
)

// ParseTextContent wandelt eine einfache Textdatei in darstellbares HTML um.
//
// Der Text wird HTML-kodiert und in einem <pre>-Block dargestellt,
// damit Formatierung (Zeilenumbrüche, Einrückungen) erhalten bleibt.
//
// @param content  Der Rohtext der Datei.
// @param filename Dateiname für den Titel-Fallback.
// @return Result mit HTML und Metadaten, oder Fehler.
func ParseTextContent(content string, filename string) (*Result, error) {
	// HTML-Sonderzeichen escapen um XSS-Angriffe zu verhindern
	escaped := html.EscapeString(content)
	// Text in pre-Block einbetten (monospace, Whitespace erhalten)
	htmlOut := `<pre class="txt-content" style="white-space:pre-wrap;word-wrap:break-word;">` +
		escaped + `</pre>`
	return &Result{
		HTML:       htmlOut,
		Title:      fileBaseName(filename),
		RawContent: content,
	}, nil
}

// IsTextFile prüft ob ein Dateipfad auf eine Textdatei (.txt) zeigt.
//
// @param filePath Dateipfad.
// @return true wenn die Datei .txt Endung hat.
func IsTextFile(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".txt"
}

// fileBaseName gibt den Dateinamen ohne Erweiterung zurück.
//
// @param filename Dateiname (mit oder ohne Pfad).
// @return Dateiname ohne Erweiterung.
func fileBaseName(filename string) string {
	base := filepath.Base(filename)
	if ext := filepath.Ext(base); ext != "" {
		return strings.TrimSuffix(base, ext)
	}
	return base
}

// extractHTMLBody extrahiert den Inhalt des <body>-Tags aus HTML.
//
// Gibt den vollständigen Quelltext zurück wenn kein body-Tag gefunden wurde.
//
// @param content Vollständiger HTML-Text.
// @return Body-Inhalt oder der gesamte Content als Fallback.
func extractHTMLBody(content string) string {
	lower := strings.ToLower(content)
	// Öffnendes <body>-Tag suchen (ggf. mit Attributen)
	start := strings.Index(lower, "<body")
	if start == -1 {
		return content
	}
	// Ende des öffnenden Tags finden (inklusive möglicher Attribute)
	tagEnd := strings.Index(lower[start:], ">")
	if tagEnd == -1 {
		return content
	}
	bodyStart := start + tagEnd + 1
	// Schließendes </body>-Tag suchen
	end := strings.LastIndex(lower, "</body>")
	if end == -1 {
		return content[bodyStart:]
	}
	return content[bodyStart:end]
}

// extractHTMLTitle extrahiert den Inhalt des <title>-Tags.
//
// @param content Vollständiger HTML-Text.
// @return Titel-Text oder leerer String wenn kein title-Tag vorhanden.
func extractHTMLTitle(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "<title>")
	if start == -1 {
		return ""
	}
	start += 7 // len("<title>")
	end := strings.Index(lower[start:], "</title>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(content[start : start+end])
}

// stripContentTags entfernt Inhalt und Tags der angegebenen HTML-Elemente.
//
// Wird verwendet um style/script/link-Tags aus fremden HTML-Inhalten
// zu entfernen, damit sie die App-UI nicht beeinflussen.
//
// @param content HTML-Text der bereinigt werden soll.
// @param tags    Liste der zu entfernenden Tag-Namen (z.B. ["style", "script"]).
// @return Bereinigter HTML-Text.
func stripContentTags(content string, tags []string) string {
	for _, tag := range tags {
		open := "<" + tag
		closeTag := "</" + tag + ">"
		for {
			lower := strings.ToLower(content)
			start := strings.Index(lower, open)
			if start == -1 {
				break
			}
			// Selbst-schließendes Tag? (z.B. <link ... />)
			tagEnd := strings.Index(lower[start:], ">")
			if tagEnd == -1 {
				break
			}
			if strings.HasSuffix(strings.TrimSpace(lower[start:start+tagEnd+1]), "/>") {
				// Selbst-schließendes Tag entfernen
				content = content[:start] + content[start+tagEnd+1:]
				continue
			}
			// Mit Inhalt: bis zum schließenden Tag entfernen
			end := strings.Index(lower[start:], closeTag)
			if end == -1 {
				// Kein schließendes Tag - nur das öffnende Tag entfernen
				content = content[:start] + content[start+tagEnd+1:]
			} else {
				content = content[:start] + content[start+end+len(closeTag):]
			}
		}
	}
	return content
}

// escapeHTMLText ersetzt HTML-Sonderzeichen in Text durch HTML-Entities.
//
// Wird verwendet wenn Text-Inhalte sicher in HTML eingebettet werden sollen.
//
// @param s Eingabetext.
// @return HTML-sicherer Text.
func escapeHTMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
