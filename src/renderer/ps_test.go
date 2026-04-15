// Tests für den PostScript-Renderer.
//
// ParsePS produziert entweder eine PDF-Einbettung (wenn gs vorhanden)
// oder einen Text-Fallback. Beide Pfade werden getestet.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsPSFile prüft die Erkennung von PostScript-Dateien anhand der Endung.
func TestIsPSFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.ps", true},
		{"Doc.PS", true},
		{"Bericht.Ps", true},
		{"doc.pdf", false},
		{"doc.md", false},
		{"/home/user/datei.ps", true},
	}
	for _, tt := range tests {
		got := renderer.IsPSFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsPSFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}

// TestParsePSFallback prüft den Text-Fallback wenn gs nicht verfügbar ist
// oder die Konvertierung fehlschlägt.
func TestParsePSFallback(t *testing.T) {
	// Einfaches PS-Dokument (kein echtes PostScript, gs schlägt fehl → Fallback)
	psContent := `%!PS-Adobe-3.0
%%Title: Testdokument
%%Pages: 1
%%EndComments
/Helvetica findfont 12 scalefont setfont
100 700 moveto (Hallo Welt) show
showpage`

	result, err := renderer.ParsePS([]byte(psContent), "test.ps")
	if err != nil {
		t.Fatalf("ParsePS() sollte keinen Fehler zurückgeben: %v", err)
	}
	// Entweder PDF-Einbettung oder Text-Fallback – beides ist korrekt
	hasPDF := strings.Contains(result.HTML, "data:application/pdf;base64,")
	hasText := strings.Contains(result.HTML, "PS-Adobe") || strings.Contains(result.HTML, "Hallo Welt")
	if !hasPDF && !hasText {
		t.Error("ParsePS() HTML enthält weder PDF-Einbettung noch PS-Quelltext")
	}
}

// TestParsePSTitle prüft Titelextraktion aus dem Dateinamen.
func TestParsePSTitle(t *testing.T) {
	result, err := renderer.ParsePS([]byte("%!PS"), "mein-dokument.ps")
	if err != nil {
		t.Fatalf("ParsePS() Fehler: %v", err)
	}
	if result.Title != "mein-dokument" {
		t.Errorf("ParsePS() Title = %q, erwartet %q", result.Title, "mein-dokument")
	}
}

// TestParsePSEmpty prüft Verhalten bei leeren Daten.
func TestParsePSEmpty(t *testing.T) {
	result, err := renderer.ParsePS([]byte{}, "empty.ps")
	if err != nil {
		t.Fatalf("ParsePS() sollte keinen Fehler für leere Daten liefern: %v", err)
	}
	if result == nil {
		t.Fatal("ParsePS() sollte kein nil Result zurückgeben")
	}
}

// TestIsSupportedFileWithPS prüft dass .ps als unterstützt erkannt wird.
func TestIsSupportedFileWithPS(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.ps", true},
		{"doc.pdf", true},
		{"README.md", true},
		{"image.png", false},
	}
	for _, tt := range tests {
		got := renderer.IsSupportedFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsSupportedFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}
