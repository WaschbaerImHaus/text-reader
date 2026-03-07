// Tests für das renderer-Package.
//
// Autor: Reisen macht Spass... mit Pia und Dirk e.Kfm.
// Letzte Änderung: 2026-03-05
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsMarkdownFile prüft die Erkennung von Markdown-Dateien.
func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Gültige Markdown-Dateien
		{"README.md", true},
		{"doc.MD", true},
		{"file.markdown", true},
		{"file.MARKDOWN", true},
		// Keine Markdown-Dateien
		{"document.txt", false},
		{"image.png", false},
		{"style.css", false},
		{"file.html", false},
		// Mit Pfad-Präfix
		{"/home/user/notes/README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := renderer.IsMarkdownFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsMarkdownFile(%q) = %v, erwartet %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestParseContentBasic prüft grundlegendes Markdown-Parsing.
func TestParseContentBasic(t *testing.T) {
	content := "# Hallo Welt\n\nDies ist ein **Test**."

	result, err := renderer.ParseContent(content, "test.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<h1") {
		t.Error("HTML enthält keine H1-Überschrift")
	}
	if !strings.Contains(result.HTML, "<strong>") {
		t.Error("HTML enthält kein <strong>-Element für **Text**")
	}
	if result.Title != "Hallo Welt" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "Hallo Welt")
	}
	if result.RawContent != content {
		t.Error("RawContent weicht vom Original ab")
	}
}

// TestParseContentHeadings prüft alle Überschriftsebenen.
func TestParseContentHeadings(t *testing.T) {
	content := "# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6"

	result, err := renderer.ParseContent(content, "headings.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	for _, tag := range []string{"<h1", "<h2", "<h3", "<h4", "<h5", "<h6"} {
		if !strings.Contains(result.HTML, tag) {
			t.Errorf("HTML enthält kein %s-Element", tag)
		}
	}
}

// TestParseContentCodeBlock prüft Code-Block-Rendering.
func TestParseContentCodeBlock(t *testing.T) {
	content := "```go\nfmt.Println(\"Hallo\")\n```"

	result, err := renderer.ParseContent(content, "code.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<pre") {
		t.Error("HTML enthält kein <pre>-Element für Code-Block")
	}
	if !strings.Contains(result.HTML, "<code") {
		t.Error("HTML enthält kein <code>-Element für Code-Block")
	}
}

// TestParseContentLists prüft Listen-Rendering.
func TestParseContentLists(t *testing.T) {
	content := "- Eintrag 1\n- Eintrag 2\n\n1. Geordnet 1\n2. Geordnet 2"

	result, err := renderer.ParseContent(content, "lists.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<ul") {
		t.Error("HTML enthält keine <ul>-Liste")
	}
	if !strings.Contains(result.HTML, "<ol") {
		t.Error("HTML enthält keine <ol>-Liste")
	}
}

// TestParseContentBlockquote prüft Blockquote-Rendering.
func TestParseContentBlockquote(t *testing.T) {
	content := "> Dies ist ein Zitat"

	result, err := renderer.ParseContent(content, "quote.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<blockquote") {
		t.Error("HTML enthält kein <blockquote>-Element")
	}
}

// TestParseContentTable prüft Tabellen-Rendering (GFM-Erweiterung).
func TestParseContentTable(t *testing.T) {
	content := "| Sp1 | Sp2 |\n|-----|-----|\n| A   | B   |"

	result, err := renderer.ParseContent(content, "table.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<table") {
		t.Error("HTML enthält keine <table>-Element für GFM-Tabelle")
	}
}

// TestParseContentLinks prüft Link-Rendering.
func TestParseContentLinks(t *testing.T) {
	content := "[GitHub](https://github.com)"

	result, err := renderer.ParseContent(content, "links.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<a ") {
		t.Error("HTML enthält keinen <a>-Link")
	}
}

// TestParseContentEmpty prüft Handling leerer Dateien.
func TestParseContentEmpty(t *testing.T) {
	result, err := renderer.ParseContent("", "empty.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler bei leerem Content: %v", err)
	}
	_ = result
}

// TestParseContentTitleFallback prüft Titel-Fallback auf Dateinamen.
func TestParseContentTitleFallback(t *testing.T) {
	content := "Kein H1-Header vorhanden."

	result, err := renderer.ParseContent(content, "mein-dokument.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if result.Title != "mein-dokument" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "mein-dokument")
	}
}

// TestParseContentStrikethrough prüft Durchstreichungs-Rendering (GFM).
func TestParseContentStrikethrough(t *testing.T) {
	content := "~~durchgestrichen~~"

	result, err := renderer.ParseContent(content, "strike.md")
	if err != nil {
		t.Fatalf("ParseContent() Fehler: %v", err)
	}

	if !strings.Contains(result.HTML, "<del") {
		t.Error("HTML enthält kein <del>-Element für ~~text~~")
	}
}

// TestLoadFileNotFound prüft Fehlerbehandlung bei nicht existierenden Dateien.
func TestLoadFileNotFound(t *testing.T) {
	_, err := renderer.LoadFile("/nicht/vorhandene/datei.md")
	if err == nil {
		t.Error("LoadFile() sollte Fehler zurückgeben für nicht existierende Datei")
	}
}

// TestLoadFileValid prüft das Laden einer gültigen Markdown-Datei.
func TestLoadFileValid(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	content := "# Test\n\nHallo Welt!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Temp-Datei konnte nicht erstellt werden: %v", err)
	}

	result, err := renderer.LoadFile(testFile)
	if err != nil {
		t.Fatalf("LoadFile() Fehler: %v", err)
	}

	if result.Title != "Test" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "Test")
	}
	if !strings.Contains(result.HTML, "<h1") {
		t.Error("HTML enthält keine H1-Überschrift")
	}
}
