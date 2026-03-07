// Tests für das FB2-Datei-Rendering.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// minimalFB2 ist ein minimales gültiges FB2-Dokument für Tests.
const minimalFB2 = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Testbuch</book-title>
      <author>
        <first-name>Test</first-name>
        <last-name>Autor</last-name>
      </author>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Erstes Kapitel</p></title>
      <p>Dies ist der erste Absatz.</p>
      <p>Dies ist der zweite Absatz mit <emphasis>Kursivtext</emphasis>.</p>
    </section>
    <section>
      <title><p>Zweites Kapitel</p></title>
      <p>Inhalt des zweiten Kapitels mit <strong>Fettdruck</strong>.</p>
    </section>
  </body>
</FictionBook>`

// TestIsFB2File prüft die Erkennung von FB2-Dateien.
func TestIsFB2File(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"book.fb2", true},
		{"roman.FB2", true},
		{"doc.md", false},
		{"page.html", false},
		{"text.txt", false},
		{"novel.epub", false},
		{"/pfad/buch.fb2", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsFB2File(tt.path)
			if got != tt.expected {
				t.Errorf("IsFB2File(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestParseFB2ContentBasic prüft grundlegendes FB2-Parsing.
func TestParseFB2ContentBasic(t *testing.T) {
	result, err := renderer.ParseFB2Content(minimalFB2, "testbuch.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if result.Title != "Testbuch" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "Testbuch")
	}
	if !strings.Contains(result.HTML, "<h2>") {
		t.Error("HTML enthält keine Kapitelüberschrift <h2>")
	}
	if !strings.Contains(result.HTML, "<p>") {
		t.Error("HTML enthält keine Absätze <p>")
	}
}

// TestParseFB2ContentInlineFormatting prüft Inline-Formatierung.
func TestParseFB2ContentInlineFormatting(t *testing.T) {
	result, err := renderer.ParseFB2Content(minimalFB2, "testbuch.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	// emphasis → <em>
	if !strings.Contains(result.HTML, "<em>") {
		t.Error("HTML enthält kein <em> für <emphasis>")
	}
	// strong → <strong>
	if !strings.Contains(result.HTML, "<strong>") {
		t.Error("HTML enthält kein <strong>")
	}
}

// TestParseFB2ContentMultipleSections prüft Trennung mehrerer Kapitel.
func TestParseFB2ContentMultipleSections(t *testing.T) {
	result, err := renderer.ParseFB2Content(minimalFB2, "testbuch.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	// Zwischen Kapiteln muss ein Trenner sein
	if !strings.Contains(result.HTML, "fb2-chapter-separator") {
		t.Error("Kein Kapitel-Trenner zwischen mehreren Abschnitten gefunden")
	}
}

// TestParseFB2ContentTitleFallback prüft Titel-Fallback auf Dateinamen.
func TestParseFB2ContentTitleFallback(t *testing.T) {
	fb2NoTitle := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info></title-info></description>
  <body>
    <section><p>Text</p></section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2NoTitle, "mein-roman.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if result.Title != "mein-roman" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "mein-roman")
	}
}

// TestParseFB2ContentNotesSkipped prüft dass Fußnoten-Body übersprungen wird.
func TestParseFB2ContentNotesSkipped(t *testing.T) {
	fb2WithNotes := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info><book-title>Buch</book-title></title-info></description>
  <body>
    <section><p>Haupttext</p></section>
  </body>
  <body name="notes">
    <section><p>Dies ist eine Fußnote die NICHT angezeigt werden soll.</p></section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2WithNotes, "buch.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if strings.Contains(result.HTML, "Fußnote die NICHT angezeigt") {
		t.Error("Fußnoten-Body wurde fälschlicherweise angezeigt")
	}
	if !strings.Contains(result.HTML, "Haupttext") {
		t.Error("Haupttext fehlt im HTML")
	}
}

// TestParseFB2ContentStrikethrough prüft durchgestrichenen Text.
func TestParseFB2ContentStrikethrough(t *testing.T) {
	fb2 := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info><book-title>Test</book-title></title-info></description>
  <body>
    <section><p>Normal <strikethrough>gestrichen</strikethrough> Ende</p></section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2, "test.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<del>") {
		t.Error("HTML enthält kein <del> für <strikethrough>")
	}
}

// TestParseFB2ContentPoem prüft Gedicht-Rendering.
func TestParseFB2ContentPoem(t *testing.T) {
	fb2 := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info><book-title>Gedichte</book-title></title-info></description>
  <body>
    <section>
      <poem>
        <stanza>
          <v>Erste Zeile</v>
          <v>Zweite Zeile</v>
        </stanza>
      </poem>
    </section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2, "gedichte.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "fb2-poem") {
		t.Error("HTML enthält kein fb2-poem-Element für <poem>")
	}
}

// TestParseFB2ContentEpigraph prüft Epigraph-Rendering.
func TestParseFB2ContentEpigraph(t *testing.T) {
	fb2 := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info><book-title>Roman</book-title></title-info></description>
  <body>
    <section>
      <epigraph><p>Ein weiser Spruch.</p></epigraph>
      <p>Kapitelinhalt</p>
    </section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2, "roman.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "fb2-epigraph") {
		t.Error("HTML enthält kein fb2-epigraph für <epigraph>")
	}
}

// TestParseFB2ContentXSSEscape prüft HTML-Escaping im Textinhalt.
func TestParseFB2ContentXSSEscape(t *testing.T) {
	fb2 := `<?xml version="1.0"?>
<FictionBook>
  <description><title-info><book-title>Test</book-title></title-info></description>
  <body>
    <section><p>&lt;script&gt;alert('xss')&lt;/script&gt;</p></section>
  </body>
</FictionBook>`
	result, err := renderer.ParseFB2Content(fb2, "test.fb2")
	if err != nil {
		t.Fatalf("ParseFB2Content() Fehler: %v", err)
	}
	// Das XML-Decoder decoded &lt; automatisch zu <
	// Unser escapeHTMLText muss es wieder escapen
	if strings.Contains(result.HTML, "<script>") {
		t.Error("HTML enthält ungescapten <script>-Tag (XSS-Risiko!)")
	}
}
