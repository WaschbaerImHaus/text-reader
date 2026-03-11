// Tests für das EPUB-Datei-Rendering.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsEpubFile prüft die Erkennung von EPUB-Dateien.
func TestIsEpubFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"book.epub", true},
		{"novel.EPUB", true},
		{"doc.md", false},
		{"text.txt", false},
		{"page.html", false},
		{"book.fb2", false},
		{"book.mobi", false},
		{"/pfad/roman.epub", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsEpubFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsEpubFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}

// buildTestEPUB erstellt ein minimales gültiges EPUB-Archiv als Byte-Slice.
//
// Struktur:
//   - mimetype
//   - META-INF/container.xml
//   - OEBPS/content.opf (Manifest + Spine)
//   - OEBPS/chapter1.xhtml
//   - OEBPS/chapter2.xhtml
func buildTestEPUB(title, chapter1, chapter2 string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// mimetype (muss erster Eintrag sein, unkomprimiert)
	mw, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	container := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(container))

	// OEBPS/content.opf
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>` + title + `</dc:title>
    <dc:creator>Test Autor</dc:creator>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(opf))

	// OEBPS/chapter1.xhtml
	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 1</title></head>
<body>` + chapter1 + `</body>
</html>`
	c1w, _ := w.Create("OEBPS/chapter1.xhtml")
	c1w.Write([]byte(ch1))

	// OEBPS/chapter2.xhtml
	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 2</title></head>
<body>` + chapter2 + `</body>
</html>`
	c2w, _ := w.Create("OEBPS/chapter2.xhtml")
	c2w.Write([]byte(ch2))

	w.Close()
	return buf.Bytes()
}

// TestParseEpubBasic prüft grundlegendes EPUB-Parsing.
func TestParseEpubBasic(t *testing.T) {
	data := buildTestEPUB(
		"Mein Testbuch",
		"<h1>Kapitel 1</h1><p>Inhalt des ersten Kapitels.</p>",
		"<h1>Kapitel 2</h1><p>Inhalt des zweiten Kapitels.</p>",
	)

	result, err := renderer.ParseEpub(data, "testbuch.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}

	if result.Title != "Mein Testbuch" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "Mein Testbuch")
	}
	if !strings.Contains(result.HTML, "Kapitel 1") {
		t.Error("HTML enthält nicht den Inhalt von Kapitel 1")
	}
	if !strings.Contains(result.HTML, "Kapitel 2") {
		t.Error("HTML enthält nicht den Inhalt von Kapitel 2")
	}
}

// TestParseEpubChapterSeparator prüft den Trennstrich zwischen Kapiteln.
func TestParseEpubChapterSeparator(t *testing.T) {
	data := buildTestEPUB(
		"Roman",
		"<p>Erstes Kapitel.</p>",
		"<p>Zweites Kapitel.</p>",
	)

	result, err := renderer.ParseEpub(data, "roman.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "epub-chapter-separator") {
		t.Error("Kein Kapitel-Trenner zwischen den Kapiteln gefunden")
	}
}

// TestParseEpubTitleFallback prüft Titel-Fallback auf Dateinamen.
func TestParseEpubTitleFallback(t *testing.T) {
	// EPUB mit leerem Titel
	data := buildTestEPUB("", "<p>Inhalt.</p>", "<p>Mehr Inhalt.</p>")

	result, err := renderer.ParseEpub(data, "mein-buch.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	if result.Title != "mein-buch" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "mein-buch")
	}
}

// TestParseEpubInvalidData prüft Fehlerbehandlung bei ungültigen Daten.
func TestParseEpubInvalidData(t *testing.T) {
	// Kein gültiges ZIP/EPUB
	_, err := renderer.ParseEpub([]byte("das ist kein epub"), "ungueltig.epub")
	if err == nil {
		t.Error("ParseEpub() sollte Fehler bei ungültigen Daten zurückgeben")
	}
}

// TestParseEpubStyleStripped prüft dass style-Tags entfernt werden.
func TestParseEpubStyleStripped(t *testing.T) {
	data := buildTestEPUB(
		"Stiltest",
		`<style>body { color: red; font-size: 99px; }</style><p>Text</p>`,
		"<p>Zweites Kapitel.</p>",
	)

	result, err := renderer.ParseEpub(data, "stiltest.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	// style-Tags dürfen nicht im HTML landen (würden App-UI überschreiben)
	if strings.Contains(result.HTML, "<style>") {
		t.Error("style-Tags wurden nicht aus EPUB-Kapiteln entfernt")
	}
	// Normaler Text muss noch vorhanden sein
	if !strings.Contains(result.HTML, "Text") {
		t.Error("Inhalt nach style-Tag fehlt")
	}
}

// buildTestEPUBWithImage erstellt ein Test-EPUB mit einem eingebetteten Bild.
//
// Enthält ein minimales 1x1 Pixel PNG als Testbild im Manifest.
func buildTestEPUBWithImage() []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// mimetype
	mw, _ := w.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	container := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(container))

	// OEBPS/content.opf (mit Bild im Manifest)
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Bildtest</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="Text/chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="img1" href="Images/test.png" media-type="image/png"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(opf))

	// OEBPS/Images/test.png (minimales 1x1 PNG)
	// Echte PNG-Bytes: 1x1 transparentes Pixel
	minimalPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG Signatur
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR Länge + Typ
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // Breite=1, Höhe=1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // Bit-Tiefe, Farbtyp, etc.
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT Länge + Typ
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, // IDAT Daten
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC, // IDAT Ende
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND Länge + Typ
		0x44, 0xAE, 0x42, 0x60, 0x82, // IEND
	}
	pw, _ := w.Create("OEBPS/Images/test.png")
	pw.Write(minimalPNG)

	// OEBPS/Text/chapter1.xhtml mit relativer Bildreferenz
	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 1</title></head>
<body>
<h1>Kapitel mit Bild</h1>
<p>Text vor dem Bild.</p>
<img src="../Images/test.png" alt="Testbild"/>
<p>Text nach dem Bild.</p>
</body>
</html>`
	c1w, _ := w.Create("OEBPS/Text/chapter1.xhtml")
	c1w.Write([]byte(ch1))

	w.Close()
	return buf.Bytes()
}

// TestParseEpubImagesEmbedded prüft dass Bilder als base64-Data-URIs eingebettet werden.
func TestParseEpubImagesEmbedded(t *testing.T) {
	data := buildTestEPUBWithImage()

	result, err := renderer.ParseEpub(data, "bildtest.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}

	// Das Bild darf NICHT mehr als relativen Pfad vorhanden sein
	if strings.Contains(result.HTML, `src="../Images/test.png"`) {
		t.Error("Bild wurde nicht eingebettet – relativer Pfad noch vorhanden")
	}

	// Das Bild MUSS als Data-URI vorhanden sein
	if !strings.Contains(result.HTML, "data:image/png;base64,") {
		t.Error("Bild wurde nicht als base64-Data-URI eingebettet")
	}

	// Sonstiger Inhalt muss noch vorhanden sein
	if !strings.Contains(result.HTML, "Kapitel mit Bild") {
		t.Error("Kapitelinhalt fehlt nach Bildeinbettung")
	}
}

// TestParseEpubImageNotFound prüft graceful Handling von fehlenden Bildern.
func TestParseEpubImageNotFound(t *testing.T) {
	// EPUB mit Bildreferenz auf nicht vorhandene Datei
	data := buildTestEPUB(
		"Fehlerbild",
		`<p>Text mit fehlendem Bild:</p><img src="../images/nichtvorhanden.png" alt="fehlt"/>`,
		"<p>Zweites Kapitel.</p>",
	)

	// Soll keinen Fehler werfen, sondern den src-Wert unverändert lassen
	result, err := renderer.ParseEpub(data, "fehlerbild.epub")
	if err != nil {
		t.Fatalf("ParseEpub() soll bei fehlendem Bild keinen Fehler werfen: %v", err)
	}
	// Inhalt muss noch da sein
	if !strings.Contains(result.HTML, "Text mit fehlendem Bild") {
		t.Error("Kapitelinhalt fehlt nach Verarbeitung mit fehlendem Bild")
	}
}

// TestParseEpubMissingContainer prüft Fehlerbehandlung ohne container.xml.
func TestParseEpubMissingContainer(t *testing.T) {
	// ZIP ohne META-INF/container.xml
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("mimetype")
	fw.Write([]byte("application/epub+zip"))
	w.Close()

	_, err := renderer.ParseEpub(buf.Bytes(), "kaputt.epub")
	if err == nil {
		t.Error("ParseEpub() sollte Fehler zurückgeben wenn container.xml fehlt")
	}
}
