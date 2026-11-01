// Package renderer stellt Funktionen zum Parsen von Dokumentdateien bereit.
//
// Unterstützte Formate:
//   - Markdown (.md, .markdown) → goldmark mit GFM + Syntax-Highlighting
//   - Plaintext (.txt)          → <pre>-Block mit HTML-Escape
//   - HTML (.html, .htm)        → Body-Extraktion und Bereinigung
//   - FB2 (.fb2)                → XML-Token-Parser
//   - EPUB (.epub)              → ZIP + XHTML-Kapitel-Extraktion
//   - LaTeX (.tex)              → eigener LaTeX-Parser
//   - PDF (.pdf)                → base64-Daten-URI in <embed>, nativ im WebView
//   - PostScript (.ps)          → gs-Konvertierung zu PDF, Fallback: <pre>-Block
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (Link-/Bild-Erlaubnisliste per AST-Transformer, F2)
package renderer

import (
	"bytes"
	stdhtml "html"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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
		// Sicherheit (F2): Link- und Bildziele nach Erlaubnisliste prüfen,
		// unabhängig von goldmarks eigener Sperrliste (CVE-2026-5160)
		parser.WithASTTransformers(util.Prioritized(linkAllowListTransformer{}, 100)),
	),
	// Sicherheit: Eingebettetes HTML in Markdown wird bewusst nicht zugelassen (RISK-001 behoben)
	// <script>, <iframe> etc. aus Markdown werden escaped statt gerendert.
)

// LoadFile lädt eine unterstützte Datei und konvertiert sie zu HTML.
//
// Erkennt das Format anhand der Dateiendung und leitet an den
// entsprechenden Parser weiter. Binärdateien (EPUB, PDF, PS) werden
// als []byte gelesen; alle anderen Formate als UTF-8-Text.
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
	// PDF benötigt Binärdaten (base64-Einbettung)
	if IsPDFFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParsePDF(data, filepath.Base(filePath))
	}
	// PostScript benötigt Binärdaten (gs-Konvertierung oder Text-Fallback)
	if IsPSFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParsePS(data, filepath.Base(filePath))
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
// Unterstützte Endungen: .md, .markdown, .txt, .fb2, .epub, .tex, .pdf, .ps
//
// @param path Dateipfad der zu prüfenden Datei.
// @return true wenn das Format unterstützt wird.
func IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".fb2", ".epub", ".tex", ".pdf", ".ps":
		return true
	}
	return false
}

// IsTextOnlyFile prüft, ob für eine Datei der Text-Modus gilt.
//
// Text-Modus heißt: alles in der Pixel-Monospace-Schrift, Bilder 1-Bit-gerastert.
// Das ergibt nur bei Textformaten Sinn (auch EPUB). PDF und PostScript werden
// als Seitenbilder angezeigt und bleiben deshalb unverändert.
//
// @param path Dateipfad oder Dateiname (Groß-/Kleinschreibung der Endung egal).
// @return true für .md, .markdown, .txt, .fb2, .epub, .tex, .html und .htm.
// @lastModified 2026-10-24
func IsTextOnlyFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".fb2", ".epub", ".tex", ".html", ".htm":
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

// isSafeMarkdownImageURL prüft ein Bildziel aus Markdown (Erlaubnisliste, F2).
//
// Erlaubt sind nur Ziele, die ResolveImagePaths später als lokale Datei
// einbettet, oder bereits eingebettete Bilder:
//   - relative Pfade und absolute Unix-Pfade (kein Schema, wie isSafeLinkURL),
//   - Windows-Laufwerkspfade "C:\…" bzw. "C:/…" (sonst als Schema "c" erkannt),
//   - "data:image/…".
//
// Alles andere (javascript:, file:, data:text/html, http(s) …) wird verworfen.
// Externe Bilder würde die CSP (img-src data:) ohnehin blockieren.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F2)
//
// @param value Entschlüsseltes Bildziel.
// @return true, wenn das Ziel eingebettet werden darf.
func isSafeMarkdownImageURL(value string) bool {
	if isDataImageURI(value) {
		return true
	}
	normalized := normalizeURLForCheck(value)
	colon := strings.Index(normalized, ":")
	if colon < 0 || strings.ContainsAny(normalized[:colon], "/?#") {
		return true // kein Schema → relativer oder absoluter lokaler Pfad
	}
	// Windows-Laufwerk: genau ein Buchstabe, dann ":" und ein Trenner
	if colon == 1 && normalized[0] >= 'a' && normalized[0] <= 'z' &&
		len(normalized) > 2 && (normalized[2] == '\\' || normalized[2] == '/') {
		return true
	}
	return false
}

// linkAllowListTransformer entfernt Links und Bilder mit nicht erlaubten
// Zielen aus dem Markdown-Syntaxbaum (Fund F2, Build 47).
//
// Hintergrund: goldmark filtert selbst nur per Sperrliste (javascript:,
// vbscript:, file:, data:) und ließ bis 1.7.16 über Entities wie
// "javascript&colon;" sogar javascript:-Links durch (CVE-2026-5160).
// Schemata wie smb: (NTLM-Leck unter Windows) oder ms-settings: kommen auch
// in neueren Versionen durch. Hier gilt deshalb dieselbe Erlaubnisliste wie
// in SanitizeHTML: Links nur http/https/mailto/#/relativ (isSafeLinkURL),
// Bilder nur lokal oder data:image/ (isSafeMarkdownImageURL).
//
// Abgelehnte Links und Bilder werden „ausgepackt“: Der Linktext bzw. der
// Alternativtext bleibt als normaler Text stehen, nur der Verweis entfällt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F2)
type linkAllowListTransformer struct{}

// Transform prüft alle Link-, Bild- und Autolink-Knoten des Dokuments.
//
// Die Knoten werden erst gesammelt und danach umgebaut, weil ein Umbau
// während ast.Walk die Durchlaufreihenfolge stören würde.
//
// @param doc    Wurzel des Syntaxbaums.
// @param reader Quelltext-Leser (für Autolink-Adressen).
// @param pc     Parser-Kontext (nicht verwendet).
func (linkAllowListTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var unwrap []ast.Node
	var autoLinks []*ast.AutoLink
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Link:
			if !isSafeDestination(node.Destination, isSafeLinkURL) {
				unwrap = append(unwrap, node)
			}
		case *ast.Image:
			if !isSafeDestination(node.Destination, isSafeMarkdownImageURL) {
				unwrap = append(unwrap, node)
			}
		case *ast.AutoLink:
			// E-Mail-Autolinks rendert goldmark als mailto: – immer erlaubt
			if node.AutoLinkType != ast.AutoLinkEmail && !isSafeLinkURL(string(node.URL(source))) {
				autoLinks = append(autoLinks, node)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, n := range unwrap {
		unwrapNode(n)
	}
	for _, a := range autoLinks {
		// Adresse als reinen Text stehen lassen (der Renderer escaped ihn)
		if parent := a.Parent(); parent != nil {
			parent.ReplaceChild(parent, a, ast.NewString(a.Label(source)))
		}
	}
}

// isSafeDestination prüft ein Ziel roh UND nach dem Entschlüsseln von
// HTML-Entities – beide Formen müssen die Prüfung bestehen (fail closed).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F2)
//
// @param dest  Ziel aus dem Syntaxbaum.
// @param check Prüffunktion (isSafeLinkURL bzw. isSafeMarkdownImageURL).
// @return true, wenn das Ziel erlaubt ist.
func isSafeDestination(dest []byte, check func(string) bool) bool {
	raw := string(dest)
	return check(raw) && check(stdhtml.UnescapeString(raw))
}

// unwrapNode ersetzt einen Knoten durch seine Kinder (Linktext bzw.
// Alternativtext eines Bildes bleibt erhalten, der Verweis entfällt).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F2)
//
// @param n Zu entfernender Link- oder Bildknoten.
func unwrapNode(n ast.Node) {
	parent := n.Parent()
	if parent == nil {
		return
	}
	for child := n.FirstChild(); child != nil; {
		next := child.NextSibling()
		// InsertBefore löst das Kind selbst aus dem alten Elternknoten
		parent.InsertBefore(parent, n, child)
		child = next
	}
	parent.RemoveChild(parent, n)
}
