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
		{"C:/Users/user/datei.ps", true},
	}
	for _, tt := range tests {
		got := renderer.IsPSFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsPSFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}

// TestParsePSOutputFormat prüft dass ParsePS immer ein nutzbares HTML-Ergebnis liefert.
//
// Akzeptiert alle drei möglichen Ausgaben:
//   - PNG-Bilder (gs vorhanden, Konvertierung erfolgreich)
//   - PS-Quelltext als <pre>-Block (gs vorhanden, Konvertierung fehlgeschlagen)
//   - Hinweismeldung + Quelltext (gs nicht installiert)
func TestParsePSOutputFormat(t *testing.T) {
	psContent := "%!PS-Adobe-3.0\n%%Title: Testdokument\n%%Pages: 1\n%%EndComments\n" +
		"/Helvetica findfont 12 scalefont setfont\n100 700 moveto (Hallo Welt) show\nshowpage\n"

	result, err := renderer.ParsePS([]byte(psContent), "test.ps")
	if err != nil {
		t.Fatalf("ParsePS() sollte keinen Fehler zurückgeben: %v", err)
	}
	if result == nil {
		t.Fatal("ParsePS() sollte kein nil Result zurückgeben")
	}
	if result.HTML == "" {
		t.Error("ParsePS() HTML darf nicht leer sein")
	}
	if result.Title == "" {
		t.Error("ParsePS() Title darf nicht leer sein")
	}

	// Mindestens eine der drei Ausgabeformen muss vorhanden sein:
	hasPNGPages := strings.Contains(result.HTML, "class=\"pdf-pages\"")
	hasText := strings.Contains(result.HTML, "PS-Adobe") || strings.Contains(result.HTML, "Hallo Welt")
	hasHint := strings.Contains(result.HTML, "Ghostscript")

	if !hasPNGPages && !hasText && !hasHint {
		t.Errorf("ParsePS() HTML hat unerwartetes Format (erste 200 Zeichen): %.200s", result.HTML)
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
// Hinweis: Dieser Test schlägt bis Task 3 fehl, da IsSupportedFile
// noch keine .ps-Endung kennt.
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
