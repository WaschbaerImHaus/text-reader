// Tests für die Bildpfad-Auflösung in HTML.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestResolveImagePathsExternal prüft dass externe URLs unverändert bleiben.
func TestResolveImagePathsExternal(t *testing.T) {
	html := `<img src="https://example.com/image.png">`
	result := renderer.ResolveImagePaths(html, "/some/dir")
	if result != html {
		t.Errorf("Externe URL wurde verändert: %s", result)
	}
}

// TestResolveImagePathsDataURI prüft dass Data-URIs unverändert bleiben.
func TestResolveImagePathsDataURI(t *testing.T) {
	html := `<img src="data:image/png;base64,abc123">`
	result := renderer.ResolveImagePaths(html, "/some/dir")
	if result != html {
		t.Errorf("Data-URI wurde verändert: %s", result)
	}
}

// TestResolveImagePathsFileURI prüft dass file://-URIs unverändert bleiben.
func TestResolveImagePathsFileURI(t *testing.T) {
	html := `<img src="file:///home/user/image.png">`
	result := renderer.ResolveImagePaths(html, "/some/dir")
	if result != html {
		t.Errorf("file://-URI wurde verändert: %s", result)
	}
}

// TestResolveImagePathsAbsolute prüft dass absolute Pfade unverändert bleiben.
func TestResolveImagePathsAbsolute(t *testing.T) {
	html := `<img src="/absolute/path/image.png">`
	result := renderer.ResolveImagePaths(html, "/some/dir")
	if result != html {
		t.Errorf("Absoluter Pfad wurde verändert: %s", result)
	}
}

// TestResolveImagePathsMissingFile prüft Verhalten wenn Bilddatei nicht existiert.
func TestResolveImagePathsMissingFile(t *testing.T) {
	html := `<img src="nicht-vorhanden.png">`
	result := renderer.ResolveImagePaths(html, "/does/not/exist")
	// Unverändert lassen wenn Datei nicht lesbar
	if result != html {
		t.Errorf("Nicht vorhandene Datei wurde verändert: %s", result)
	}
}

// TestResolveImagePathsLocalFile prüft Auflösung eines tatsächlich existierenden Bildes.
func TestResolveImagePathsLocalFile(t *testing.T) {
	// Temp-Verzeichnis mit Testbild anlegen
	tmpDir := t.TempDir()

	// Minimales 1x1 PNG erstellen (gültiges PNG binary)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG Signatur
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR Chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT Chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND Chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	imgPath := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(imgPath, pngData, 0644); err != nil {
		t.Fatalf("Testbild konnte nicht erstellt werden: %v", err)
	}

	html := `<p><img src="test.png" alt="Test"></p>`
	result := renderer.ResolveImagePaths(html, tmpDir)

	// src sollte durch data:image/png;base64,... ersetzt worden sein
	if !strings.Contains(result, "data:image/png;base64,") {
		t.Errorf("Relativer Pfad wurde nicht durch Data-URI ersetzt: %s", result)
	}
	// Alt-Attribut sollte erhalten sein
	if !strings.Contains(result, `alt="Test"`) {
		t.Error("Alt-Attribut wurde entfernt")
	}
}

// TestResolveImagePathsHTTPUppercase prüft case-insensitive URL-Erkennung.
func TestResolveImagePathsHTTPUppercase(t *testing.T) {
	html := `<img src="HTTP://example.com/img.jpg">`
	result := renderer.ResolveImagePaths(html, "/dir")
	if result != html {
		t.Errorf("HTTP (Großbuchstaben) wurde nicht erkannt: %s", result)
	}
}

// TestResolveImagePathsNoImages prüft HTML ohne img-Tags.
func TestResolveImagePathsNoImages(t *testing.T) {
	html := `<p>Text ohne Bilder</p><h1>Überschrift</h1>`
	result := renderer.ResolveImagePaths(html, "/dir")
	if result != html {
		t.Errorf("HTML ohne Bilder wurde verändert: %s", result)
	}
}

// TestResolveImagePathsSingleQuotes prüft src mit einfachen Anführungszeichen.
func TestResolveImagePathsSingleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	// Minimale Bilddatei (leere Datei - mime/type fallback)
	imgPath := filepath.Join(tmpDir, "bild.png")
	if err := os.WriteFile(imgPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	html := `<img src='bild.png'>`
	result := renderer.ResolveImagePaths(html, tmpDir)
	if !strings.Contains(result, "data:image/png;base64,") {
		t.Errorf("Einfache Anführungszeichen werden nicht verarbeitet: %s", result)
	}
}
