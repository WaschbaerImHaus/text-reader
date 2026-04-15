// Tests für den PDF-Renderer.
//
// Testet sowohl den pdftoppm-Pfad (Linux mit installiertem poppler-utils)
// als auch den embed-Fallback-Pfad (Windows / ungültige PDF-Daten).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer_test

import (
	"os/exec"
	"runtime"
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

// TestParsePDFInvalidData prüft dass ungültige PDF-Daten sicher verarbeitet werden.
//
// Ungültige Daten → pdftoppm schlägt fehl → Embed-Fallback wird verwendet.
func TestParsePDFInvalidData(t *testing.T) {
	data := []byte("%PDF-1.4 ungültiger inhalt")
	result, err := renderer.ParsePDF(data, "test.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() Fehler: %v", err)
	}
	// Bei ungültigen Daten wird immer der Embed-Fallback verwendet
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() Fallback-HTML enthält keine base64-Einbettung")
	}
	if !strings.Contains(result.HTML, "<embed") {
		t.Error("ParsePDF() Fallback-HTML enthält kein <embed>-Element")
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
	// Leere Daten → pdftoppm schlägt fehl → Embed-Fallback
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() Fallback-HTML fehlt auch bei leeren Daten")
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

// TestParsePDFOutputFormat prüft dass das HTML entweder Bilder (pdftoppm) oder embed enthält.
//
// Der Test akzeptiert beide Ausgabepfade:
//   - Linux mit pdftoppm und gültigem PDF → pdf-pages Div mit Bild-Tags
//   - Alle anderen Fälle → embed-Tag mit base64-Daten-URI
func TestParsePDFOutputFormat(t *testing.T) {
	data := []byte("%PDF-1.4 test")
	result, err := renderer.ParsePDF(data, "test.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() Fehler: %v", err)
	}

	// Mindestanforderung: HTML ist nicht leer und enthält einen Dokumenttitel
	if result.HTML == "" {
		t.Error("ParsePDF() HTML darf nicht leer sein")
	}
	if result.Title == "" {
		t.Error("ParsePDF() Title darf nicht leer sein")
	}

	// Format-Prüfung: entweder Bild-HTML (pdftoppm) oder embed-HTML
	hasPDFPages := strings.Contains(result.HTML, "class=\"pdf-pages\"")
	hasEmbed := strings.Contains(result.HTML, "<embed") && strings.Contains(result.HTML, "application/pdf")

	if !hasPDFPages && !hasEmbed {
		t.Error("ParsePDF() HTML enthält weder pdf-pages noch embed-Element")
	}
}

// TestParsePDFPdftoppmUsedOnLinux prüft dass pdftoppm auf Linux versucht wird.
//
// Wird auf Nicht-Linux-Systemen oder ohne pdftoppm übersprungen.
// Für einen echten Test wird ein minimales gültiges PDF erzeugt.
func TestParsePDFPdftoppmUsedOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Nur auf Linux relevant")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm nicht installiert – Test übersprungen")
	}

	// Minimales gültiges PDF (1 leere Seite) erzeugen.
	// Dieses PDF wurde mit qpdf verifiziert und hat korrekte xref-Offsets.
	minimalPDF := buildMinimalPDF()
	if len(minimalPDF) == 0 {
		t.Skip("Minimales Test-PDF konnte nicht erzeugt werden")
	}

	result, err := renderer.ParsePDF(minimalPDF, "minimal.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() Fehler: %v", err)
	}

	// Auf Linux mit pdftoppm und gültigem PDF: Bild-HTML erwartet
	if !strings.Contains(result.HTML, "class=\"pdf-pages\"") {
		t.Error("ParsePDF() auf Linux sollte pdf-pages HTML erzeugen")
	}
	if !strings.Contains(result.HTML, "data:image/png;base64,") {
		t.Error("ParsePDF() auf Linux sollte base64 PNG-Bilder enthalten")
	}
	if result.Title != "minimal" {
		t.Errorf("ParsePDF() Title = %q, erwartet %q", result.Title, "minimal")
	}
}

// buildMinimalPDF erstellt ein minimales gültiges 1-Seiten-PDF für Tests.
//
// Erzeugt das PDF durch Aufruf von gs (falls verfügbar) aus einer minimalen
// PostScript-Eingabe. Gibt nil zurück wenn gs nicht verfügbar ist.
//
// @return PDF-Binärdaten oder nil.
func buildMinimalPDF() []byte {
	gsExe, err := exec.LookPath("gs")
	if err != nil {
		// Alternativ gswin64c auf Windows
		gsExe, err = exec.LookPath("gswin64c")
		if err != nil {
			return nil
		}
	}

	// Minimales PostScript-Dokument (1 leere Seite A4)
	psContent := []byte("%!PS-Adobe-3.0\n%%Pages: 1\n%%Page: 1 1\nshowpage\n%%EOF\n")

	cmd := exec.Command(gsExe,
		"-sDEVICE=pdfwrite",
		"-sOutputFile=-",
		"-dBATCH",
		"-dNOPAUSE",
		"-dSAFER",
		"-q",
		"-",
	)
	cmd.Stdin = strings.NewReader(string(psContent))

	out, err := cmd.Output()
	if err != nil || len(out) < 4 {
		return nil
	}
	if !strings.HasPrefix(string(out[:4]), "%PDF") {
		return nil
	}
	return out
}
