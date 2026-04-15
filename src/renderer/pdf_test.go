// Tests für den PDF-Renderer.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsPDFFile prüft die Erkennung von PDF-Dateien anhand der Endung.
func TestIsPDFFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.pdf", true},
		{"Doc.PDF", true},
		{"BERICHT.Pdf", true},
		{"doc.md", false},
		{"doc.ps", false},
		{"book.epub", false},
		{"/home/user/datei.pdf", true},
		{"C:/Users/user/datei.pdf", true},
	}
	for _, tt := range tests {
		got := renderer.IsPDFFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsPDFFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}

// TestParsePDF prüft dass eine base64-Einbettung erzeugt wird.
func TestParsePDF(t *testing.T) {
	data := []byte("%PDF-1.4 test")
	result, err := renderer.ParsePDF(data, "test.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() HTML enthält keine base64 PDF-Einbettung")
	}
	if !strings.Contains(result.HTML, `type="application/pdf"`) {
		t.Error("ParsePDF() HTML enthält keinen MIME-Typ application/pdf")
	}
	if !strings.Contains(result.HTML, "<embed") {
		t.Error("ParsePDF() HTML enthält kein <embed>-Element")
	}
	if result.Title != "test" {
		t.Errorf("ParsePDF() Title = %q, erwartet %q", result.Title, "test")
	}
}

// TestParsePDFEmpty prüft Verhalten bei leeren PDF-Daten.
func TestParsePDFEmpty(t *testing.T) {
	result, err := renderer.ParsePDF([]byte{}, "empty.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() sollte keinen Fehler für leere Daten liefern: %v", err)
	}
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() HTML fehlt auch bei leeren Daten")
	}
}

// TestParsePDFTitleFromFilename prüft Titelextraktion aus Dateiname.
func TestParsePDFTitleFromFilename(t *testing.T) {
	result, err := renderer.ParsePDF([]byte("%PDF"), "jahresbericht-2025.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() unerwarteter Fehler: %v", err)
	}
	if result.Title != "jahresbericht-2025" {
		t.Errorf("ParsePDF() Title = %q, erwartet %q", result.Title, "jahresbericht-2025")
	}
}
