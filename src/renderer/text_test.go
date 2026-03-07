// Tests für das Textdatei- und HTML-Datei-Rendering.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsTextFile prüft die Erkennung von Textdateien.
func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"readme.txt", true},
		{"README.TXT", true},
		{"notes.txt", true},
		{"doc.md", false},
		{"page.html", false},
		{"book.epub", false},
		{"/pfad/zur/datei.txt", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsTextFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsTextFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestParseTextContentBasic prüft grundlegendes Textdatei-Rendering.
func TestParseTextContentBasic(t *testing.T) {
	content := "Hallo Welt\nZweite Zeile"
	result, err := renderer.ParseTextContent(content, "test.txt")
	if err != nil {
		t.Fatalf("ParseTextContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<pre") {
		t.Error("HTML enthält kein <pre>-Element")
	}
	if !strings.Contains(result.HTML, "Hallo Welt") {
		t.Error("HTML enthält nicht den Originaltext")
	}
	if result.Title != "test" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "test")
	}
}

// TestParseTextContentHTMLEscape prüft HTML-Escaping für Sicherheit.
func TestParseTextContentHTMLEscape(t *testing.T) {
	// Kritische HTML-Zeichen müssen escaped werden
	content := "<script>alert('XSS')</script>"
	result, err := renderer.ParseTextContent(content, "xss.txt")
	if err != nil {
		t.Fatalf("ParseTextContent() Fehler: %v", err)
	}
	// Rohe script-Tags dürfen NICHT im HTML erscheinen
	if strings.Contains(result.HTML, "<script>") {
		t.Error("HTML enthält ungescapte <script>-Tags (XSS-Risiko!)")
	}
	// Escaped Version muss vorhanden sein
	if !strings.Contains(result.HTML, "&lt;script&gt;") {
		t.Error("HTML enthält die escaped Version nicht")
	}
}

// TestParseTextContentAmpersand prüft Escaping von Sonderzeichen.
func TestParseTextContentAmpersand(t *testing.T) {
	content := "Preis: 10 & 20 EUR"
	result, err := renderer.ParseTextContent(content, "text.txt")
	if err != nil {
		t.Fatalf("ParseTextContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "&amp;") {
		t.Error("Ampersand wurde nicht zu &amp; escaped")
	}
}

// TestParseTextContentEmpty prüft Verhalten bei leerem Inhalt.
func TestParseTextContentEmpty(t *testing.T) {
	result, err := renderer.ParseTextContent("", "empty.txt")
	if err != nil {
		t.Fatalf("ParseTextContent() Fehler bei leerem Inhalt: %v", err)
	}
	if !strings.Contains(result.HTML, "<pre") {
		t.Error("HTML enthält kein <pre>-Element auch bei leerem Inhalt")
	}
}

// TestIsSupportedFile prüft die Erkennung aller unterstützten Formate.
func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"README.md", true},
		{"doc.markdown", true},
		{"notes.txt", true},
		{"book.fb2", true},
		{"novel.epub", true},
		// Groß-/Kleinschreibung
		{"README.MD", true},
		{"BOOK.EPUB", true},
		// Nicht unterstützt (inkl. HTML - bewusst entfernt)
		{"page.html", false},
		{"page.htm", false},
		{"image.png", false},
		{"style.css", false},
		{"data.json", false},
		{"video.mp4", false},
		{"book.mobi", false},
		{"book.azw3", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsSupportedFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsSupportedFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}
