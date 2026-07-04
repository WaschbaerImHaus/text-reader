// Tests für die Temp-Datei-Navigation bei großen HTML-Dokumenten.
//
// Hintergrund (Bug #012): WebView2 (Windows) begrenzt NavigateToString auf
// 2 MB. Größere Dokumente müssen als temporäre Datei gespeichert und per
// file://-URL geladen werden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-07-04
package ui_test

import (
	"os"
	"strings"
	"testing"

	"md-reader/ui"
)

// TestNeedsFileNavigation prüft die Schwellwert-Entscheidung.
func TestNeedsFileNavigation(t *testing.T) {
	// Kleines Dokument → direktes SetHtml erlaubt
	if ui.NeedsFileNavigation(strings.Repeat("a", 1000)) {
		t.Error("1 KB HTML sollte kein file://-Fallback benötigen")
	}
	// Dokument über dem Schwellwert → file://-Navigation nötig
	if !ui.NeedsFileNavigation(strings.Repeat("a", ui.MaxInlineHTMLSize+1)) {
		t.Error("HTML über MaxInlineHTMLSize muss file://-Fallback benötigen")
	}
	// Der Schwellwert selbst muss sicher unter dem 2-MB-Limit von WebView2 liegen
	if ui.MaxInlineHTMLSize >= 2*1024*1024 {
		t.Errorf("MaxInlineHTMLSize (%d) muss unter dem 2-MB-Limit von WebView2 liegen", ui.MaxInlineHTMLSize)
	}
}

// TestWriteTempViewFile prüft Schreiben und URL-Bildung der Temp-Ansichtsdatei.
func TestWriteTempViewFile(t *testing.T) {
	html := "<h1>Größentest äöü</h1>" + strings.Repeat("x", 5000)

	fileURL, err := ui.WriteTempViewFile(html)
	if err != nil {
		t.Fatalf("WriteTempViewFile() Fehler: %v", err)
	}
	defer ui.CleanupTempViewFile()

	// URL muss eine absolute file://-URL sein
	if !strings.HasPrefix(fileURL, "file://") {
		t.Errorf("URL muss mit file:// beginnen, ist: %s", fileURL)
	}

	// Die Datei muss existieren und exakt den HTML-Inhalt enthalten
	path := ui.TempViewFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Temp-Ansichtsdatei nicht lesbar (%s): %v", path, err)
	}
	if string(data) != html {
		t.Error("Inhalt der Temp-Ansichtsdatei stimmt nicht mit dem HTML überein")
	}
}

// TestWriteTempViewFileOverwrite prüft, dass wiederholtes Schreiben dieselbe
// Datei überschreibt (keine Ansammlung von Temp-Dateien).
func TestWriteTempViewFileOverwrite(t *testing.T) {
	url1, err := ui.WriteTempViewFile("erste Version")
	if err != nil {
		t.Fatalf("erster Schreibvorgang: %v", err)
	}
	url2, err := ui.WriteTempViewFile("zweite Version")
	if err != nil {
		t.Fatalf("zweiter Schreibvorgang: %v", err)
	}
	defer ui.CleanupTempViewFile()

	if url1 != url2 {
		t.Errorf("wiederholtes Schreiben muss dieselbe Datei nutzen: %s != %s", url1, url2)
	}
	data, _ := os.ReadFile(ui.TempViewFilePath())
	if string(data) != "zweite Version" {
		t.Error("Temp-Ansichtsdatei wurde nicht überschrieben")
	}
}

// TestCleanupTempViewFile prüft, dass die Temp-Datei gelöscht wird.
func TestCleanupTempViewFile(t *testing.T) {
	if _, err := ui.WriteTempViewFile("wegwerfen"); err != nil {
		t.Fatalf("WriteTempViewFile() Fehler: %v", err)
	}
	ui.CleanupTempViewFile()
	if _, err := os.Stat(ui.TempViewFilePath()); !os.IsNotExist(err) {
		t.Error("CleanupTempViewFile() hat die Datei nicht gelöscht")
	}
}
