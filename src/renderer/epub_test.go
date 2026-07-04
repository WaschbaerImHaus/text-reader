// Tests für das EPUB-Datei-Rendering.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-07-04
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
	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 1</title></head>
<body>` + chapter1 + `</body>
</html>`
	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 2</title></head>
<body>` + chapter2 + `</body>
</html>`
	return buildTestEPUBRaw(title, ch1, ch2)
}

// buildTestEPUBRaw erstellt ein Test-EPUB mit vollständigen XHTML-Dokumenten
// als Kapitel (inkl. eigenem <body>-Tag, z.B. für Tests mit <body id=...>).
func buildTestEPUBRaw(title, chapter1XHTML, chapter2XHTML string) []byte {
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
	c1w, _ := w.Create("OEBPS/chapter1.xhtml")
	c1w.Write([]byte(chapter1XHTML))

	// OEBPS/chapter2.xhtml
	c2w, _ := w.Create("OEBPS/chapter2.xhtml")
	c2w.Write([]byte(chapter2XHTML))

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

// TestParseEpubChapterAnchors prüft dass jedes Kapitel einen Anker mit
// pfadbasierter ID bekommt, damit interne Links darauf zeigen können.
func TestParseEpubChapterAnchors(t *testing.T) {
	data := buildTestEPUB(
		"Ankertest",
		"<p>Erstes Kapitel.</p>",
		"<p>Zweites Kapitel.</p>",
	)

	result, err := renderer.ParseEpub(data, "ankertest.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	// Jedes Kapitel muss einen Anker mit ID aus dem ZIP-Pfad haben
	if !strings.Contains(result.HTML, `id="epub-oebps-chapter1-xhtml"`) {
		t.Error("Kapitel 1 hat keinen Anker mit pfadbasierter ID")
	}
	if !strings.Contains(result.HTML, `id="epub-oebps-chapter2-xhtml"`) {
		t.Error("Kapitel 2 hat keinen Anker mit pfadbasierter ID")
	}
}

// TestParseEpubInternalLinksRewritten prüft dass Kapitel-Links auf andere
// EPUB-Dateien in interne Anker-Links umgeschrieben werden.
//
// Hintergrund (Bug): Links wie href="chapter2.xhtml" führen im WebView zu
// einer Navigation weg vom per SetHtml gesetzten Inhalt → leeres Fenster.
func TestParseEpubInternalLinksRewritten(t *testing.T) {
	data := buildTestEPUB(
		"Linktest",
		`<p>Siehe <a href="chapter2.xhtml">Kapitel 2</a> und <a href='./chapter2.xhtml'>nochmal</a>.</p>`,
		"<p>Zweites Kapitel.</p>",
	)

	result, err := renderer.ParseEpub(data, "linktest.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	// Der rohe Dateilink darf nicht mehr vorhanden sein
	if strings.Contains(result.HTML, `href="chapter2.xhtml"`) {
		t.Error("Kapitel-Link wurde nicht umgeschrieben (doppelte Anführungszeichen)")
	}
	if strings.Contains(result.HTML, `href='./chapter2.xhtml'`) {
		t.Error("Kapitel-Link wurde nicht umgeschrieben (einfache Anführungszeichen)")
	}
	// Stattdessen muss ein interner Anker-Link vorhanden sein
	if !strings.Contains(result.HTML, `href="#epub-oebps-chapter2-xhtml"`) {
		t.Error("Kein interner Anker-Link auf Kapitel 2 gefunden")
	}
}

// TestParseEpubLinkWithFragment prüft dass Links mit Fragment auf das
// Fragment umgeschrieben werden (Ziel-ID bleibt im Kapitelinhalt erhalten).
func TestParseEpubLinkWithFragment(t *testing.T) {
	data := buildTestEPUB(
		"Fragmenttest",
		`<p><a href="chapter2.xhtml#abschnitt2">Zu Abschnitt 2</a></p>`,
		`<h2 id="abschnitt2">Abschnitt 2</h2><p>Inhalt.</p>`,
	)

	result, err := renderer.ParseEpub(data, "fragmenttest.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	if strings.Contains(result.HTML, `href="chapter2.xhtml#abschnitt2"`) {
		t.Error("Fragment-Link wurde nicht umgeschrieben")
	}
	if !strings.Contains(result.HTML, `href="#abschnitt2"`) {
		t.Error("Fragment-Link zeigt nicht auf den internen Anker #abschnitt2")
	}
}

// TestParseEpubLinksUnchanged prüft dass reine Fragment-Links und externe
// URLs NICHT verändert werden.
func TestParseEpubLinksUnchanged(t *testing.T) {
	data := buildTestEPUB(
		"Unverändert",
		`<p><a href="#lokal">Lokal</a> <a href="https://example.org/seite.xhtml">Extern</a></p><p id="lokal">Ziel</p>`,
		"<p>Zweites Kapitel.</p>",
	)

	result, err := renderer.ParseEpub(data, "unveraendert.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, `href="#lokal"`) {
		t.Error("Reiner Fragment-Link wurde fälschlich verändert")
	}
	if !strings.Contains(result.HTML, `href="https://example.org/seite.xhtml"`) {
		t.Error("Externe URL wurde fälschlich verändert")
	}
}

// TestParseEpubBodyIDPreserved prüft dass eine ID auf dem <body>-Tag als
// Anker erhalten bleibt (Calibre setzt Kapitel-Sprungziele auf <body id=...>).
func TestParseEpubBodyIDPreserved(t *testing.T) {
	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 1</title></head>
<body><p><a href="chapter2.xhtml#zielk2">Zu Kapitel 2</a></p></body>
</html>`
	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 2</title></head>
<body id="zielk2" class="calibre"><p>Zweites Kapitel.</p></body>
</html>`
	data := buildTestEPUBRaw("BodyID-Test", ch1, ch2)

	result, err := renderer.ParseEpub(data, "bodyid.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}
	// Der Link muss auf das Fragment umgeschrieben sein
	if !strings.Contains(result.HTML, `href="#zielk2"`) {
		t.Error("Fragment-Link auf Body-ID wurde nicht umgeschrieben")
	}
	// Die Body-ID muss als Anker im Dokument existieren
	if !strings.Contains(result.HTML, `id="zielk2"`) {
		t.Error("Body-ID ging beim Extrahieren verloren – Linkziel fehlt")
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

// buildTestEPUBWithSVGCover erstellt ein Test-EPUB mit einer Calibre-artigen
// Titelseite: ein SVG mit <image xlink:href="cover.jpeg"> im EPUB-Root.
//
// Nachgebaut nach der Struktur echter Calibre-EPUBs (titlepage.xhtml + cover.jpeg
// direkt im Root, OPF ebenfalls im Root). Zusätzlich ein zweites Kapitel mit
// einem einfach gequoteten SVG-href (SVG2-Syntax ohne xlink-Namensraum).
func buildTestEPUBWithSVGCover() []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// mimetype (unkomprimiert, wie von der EPUB-Spezifikation verlangt)
	mw, _ := w.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml – OPF liegt im Root
	container := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(container))

	// content.opf mit Titelseite, Kapitel und Cover-Bild im Manifest
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>SVG-Cover-Test</dc:title>
  </metadata>
  <manifest>
    <item id="titlepage" href="titlepage.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch1" href="text/chapter1.html" media-type="application/xhtml+xml"/>
    <item id="cover" href="cover.jpeg" media-type="image/jpeg"/>
  </manifest>
  <spine>
    <itemref idref="titlepage"/>
    <itemref idref="ch1"/>
  </spine>
</package>`
	ow, _ := w.Create("content.opf")
	ow.Write([]byte(opf))

	// cover.jpeg (minimale JPEG-Signatur reicht für den Test)
	minimalJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0xFF, 0xD9}
	jw, _ := w.Create("cover.jpeg")
	jw.Write(minimalJPEG)

	// titlepage.xhtml – Calibre-Stil: SVG mit xlink:href auf das Cover
	titlepage := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Cover</title></head>
<body>
<div>
  <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
       version="1.1" width="100%" height="100%" viewBox="0 0 474 751" preserveAspectRatio="none">
    <image width="474" height="751" xlink:href="cover.jpeg"/>
  </svg>
</div>
</body>
</html>`
	tw, _ := w.Create("titlepage.xhtml")
	tw.Write([]byte(titlepage))

	// text/chapter1.html – SVG2-Syntax: href ohne xlink, einfach gequotet,
	// relativer Pfad aus einem Unterverzeichnis heraus
	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Kapitel 1</title></head>
<body>
<h1>Kapitel mit SVG-Bild</h1>
<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10">
  <image width="10" height="10" href='../cover.jpeg'/>
</svg>
</body>
</html>`
	c1w, _ := w.Create("text/chapter1.html")
	c1w.Write([]byte(ch1))

	w.Close()
	return buf.Bytes()
}

// TestParseEpubSVGCoverEmbedded prüft, dass SVG-<image>-Referenzen
// (xlink:href und href) als base64-Data-URIs eingebettet werden.
//
// Hintergrund (Bug #012): Calibre-Titelseiten referenzieren das Cover als
// <image xlink:href="cover.jpeg"> in einem SVG. Ohne Einbettung bleibt die
// erste Seite des Buchs komplett weiß.
func TestParseEpubSVGCoverEmbedded(t *testing.T) {
	data := buildTestEPUBWithSVGCover()

	result, err := renderer.ParseEpub(data, "svgcover.epub")
	if err != nil {
		t.Fatalf("ParseEpub() Fehler: %v", err)
	}

	// Die rohen Bildpfade dürfen nicht mehr vorhanden sein
	if strings.Contains(result.HTML, `xlink:href="cover.jpeg"`) {
		t.Error("SVG-Cover (xlink:href) wurde nicht eingebettet – roher Pfad noch vorhanden")
	}
	if strings.Contains(result.HTML, `href='../cover.jpeg'`) {
		t.Error("SVG-Bild (href, einfach gequotet) wurde nicht eingebettet – roher Pfad noch vorhanden")
	}

	// Beide müssen als JPEG-Data-URI vorhanden sein (2 Vorkommen)
	if strings.Count(result.HTML, "data:image/jpeg;base64,") < 2 {
		t.Errorf("SVG-Bilder nicht als base64-Data-URIs eingebettet (gefunden: %d von 2)",
			strings.Count(result.HTML, "data:image/jpeg;base64,"))
	}

	// Kapiteltext muss erhalten bleiben
	if !strings.Contains(result.HTML, "Kapitel mit SVG-Bild") {
		t.Error("Kapitelinhalt fehlt nach SVG-Bildeinbettung")
	}
}

// TestParseEpubSVGImageNotFound prüft graceful Handling eines fehlenden SVG-Bilds.
func TestParseEpubSVGImageNotFound(t *testing.T) {
	data := buildTestEPUBRaw(
		"SVG-Fehlt",
		`<?xml version="1.0" encoding="UTF-8"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>K1</title></head><body><p>Vor dem SVG.</p><svg xmlns="http://www.w3.org/2000/svg"><image xlink:href="gibtsnicht.jpeg"/></svg></body></html>`,
		`<?xml version="1.0" encoding="UTF-8"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>K2</title></head><body><p>Zweites Kapitel.</p></body></html>`,
	)

	result, err := renderer.ParseEpub(data, "svgfehlt.epub")
	if err != nil {
		t.Fatalf("ParseEpub() soll bei fehlendem SVG-Bild keinen Fehler werfen: %v", err)
	}
	if !strings.Contains(result.HTML, "Vor dem SVG.") {
		t.Error("Kapitelinhalt fehlt nach Verarbeitung mit fehlendem SVG-Bild")
	}
}
