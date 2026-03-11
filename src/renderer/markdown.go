// Package renderer stellt Funktionen zum Parsen von Dokumentdateien bereit.
//
// Unterstützte Formate:
//   - Markdown (.md, .markdown) → goldmark mit GFM + Syntax-Highlighting
//   - Plaintext (.txt)          → <pre>-Block mit HTML-Escape
//   - HTML (.html, .htm)        → Body-Extraktion und Bereinigung
//   - FB2 (.fb2)                → XML-Token-Parser
//   - EPUB (.epub)              → ZIP + XHTML-Kapitel-Extraktion
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package renderer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Result enthält das Ergebnis des Markdown-Parsings.
type Result struct {
	// HTML ist das gerenderte HTML aus dem Markdown-Inhalt.
	HTML string
	// Title ist der Titel (erste H1-Überschrift oder Dateiname).
	Title string
	// RawContent ist der ursprüngliche Markdown-Text.
	RawContent string
}

// md ist die goldmark-Instanz mit GFM-Unterstützung und Syntax-Highlighting.
var md = goldmark.New(
	goldmark.WithExtensions(
		// GFM-Erweiterungen für GitHub-kompatibles Markdown
		extension.GFM,
		// Syntax-Highlighting für Code-Blöcke (GitHub-Stil)
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
		),
	),
	goldmark.WithParserOptions(
		// Automatische Anker-IDs für Überschriften
		parser.WithAutoHeadingID(),
	),
	// Sicherheit: Eingebettetes HTML in Markdown wird bewusst nicht zugelassen (RISK-001 behoben)
	// <script>, <iframe> etc. aus Markdown werden escaped statt gerendert.
)

// LoadFile lädt eine unterstützte Datei und konvertiert sie zu HTML.
//
// Erkennt das Format anhand der Dateiendung und leitet an den
// entsprechenden Parser weiter. EPUB-Dateien werden als Binärdaten gelesen.
//
// @param filePath Absoluter oder relativer Pfad zur Datei.
// @return Result mit gerendertem HTML und Metadaten, oder Fehler.
func LoadFile(filePath string) (*Result, error) {
	// EPUB benötigt Binärdaten (ZIP-Archiv)
	if IsEpubFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParseEpub(data, filepath.Base(filePath))
	}
	// Alle anderen Formate als Text lesen
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseContent(string(data), filepath.Base(filePath))
}

// ParseContent konvertiert Text-Inhalt zu HTML.
//
// Erkennt das Format anhand der Dateiendung und leitet an den
// entsprechenden Parser weiter. Unterstützt: .md, .markdown, .txt, .html, .htm, .fb2, .tex
//
// @param content  Der Quelltext der Datei.
// @param filename Dateiname mit Erweiterung zur Formaterkennung.
// @return Result mit gerendertem HTML und Metadaten, oder Fehler.
func ParseContent(content string, filename string) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return ParseTextContent(content, filename)
	case ".fb2":
		return ParseFB2Content(content, filename)
	case ".tex":
		// LaTeX-Quelldateien mit eigenem Parser rendern
		return ParseLaTeXContent(content, filename)
	default:
		// Standard: Markdown-Rendering (.md, .markdown und unbekannte Typen)
		return parseMarkdown(content, filename)
	}
}

// parseMarkdown konvertiert Markdown-Text zu HTML (interner Parser).
//
// @param content  Der Markdown-Quelltext.
// @param filename Dateiname für den Fallback-Titel.
// @return Result mit gerendertem HTML und Metadaten, oder Fehler.
func parseMarkdown(content string, filename string) (*Result, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return nil, err
	}
	title := extractTitle(content, filename)
	return &Result{
		HTML:       buf.String(),
		Title:      title,
		RawContent: content,
	}, nil
}

// IsMarkdownFile prüft ob eine Datei eine gültige Markdown-Datei ist.
//
// @param path Dateipfad der zu prüfenden Datei.
// @return true wenn die Datei eine .md oder .markdown Erweiterung hat.
func IsMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// IsSupportedFile prüft ob eine Datei ein unterstütztes Format hat.
//
// Unterstützte Endungen: .md, .markdown, .txt, .fb2, .epub, .tex
//
// @param path Dateipfad der zu prüfenden Datei.
// @return true wenn das Format unterstützt wird.
func IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".fb2", ".epub", ".tex":
		return true
	}
	return false
}

// extractTitle ermittelt den Titel aus dem Markdown-Inhalt.
//
// @param content  Der Markdown-Quelltext.
// @param fallback Fallback-Titel falls kein H1 gefunden wird.
// @return Den ermittelten Titel als String.
func extractTitle(content, fallback string) string {
	// Zeilenweise nach erstem ATX-Stil H1-Header suchen
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	// Dateiname ohne Erweiterung als Fallback verwenden
	if ext := filepath.Ext(fallback); ext != "" {
		return strings.TrimSuffix(fallback, ext)
	}
	return fallback
}
