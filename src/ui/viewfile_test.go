// Tests für die Temp-Datei-Navigation bei großen HTML-Dokumenten.
//
// Hintergrund (Bug #012): WebView2 (Windows) begrenzt NavigateToString auf
// 2 MB. Größere Dokumente müssen als temporäre Datei gespeichert und per
// file://-URL geladen werden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (macOS ohne Temp-Datei)
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

// TestNeedsFileNavigationFor prüft die Entscheidung je Betriebssystem:
// Linux und Windows laden große Seiten per Temp-Datei, macOS (WKWebView,
// kein 2-MB-Limit, file:// per loadRequest unzuverlässig) immer direkt.
func TestNeedsFileNavigationFor(t *testing.T) {
	big := strings.Repeat("a", ui.MaxInlineHTMLSize+1)
	small := strings.Repeat("a", 1000)
	for _, goos := range []string{"linux", "windows"} {
		if !ui.NeedsFileNavigationFor(goos, big) {
			t.Errorf("%s: großes HTML muss per Temp-Datei geladen werden", goos)
		}
		if ui.NeedsFileNavigationFor(goos, small) {
			t.Errorf("%s: kleines HTML braucht keine Temp-Datei", goos)
		}
	}
	if ui.NeedsFileNavigationFor("darwin", big) {
		t.Error("darwin: großes HTML muss direkt per SetHtml geladen werden")
	}
	if ui.NeedsFileNavigationFor("darwin", small) {
		t.Error("darwin: kleines HTML braucht keine Temp-Datei")
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

// TestTempViewFileURL prüft, dass TempViewFileURL genau die URL liefert, die
// WriteTempViewFile zurückgibt (beide müssen für den Navigationsschutz
// übereinstimmen, sonst würde die eigene große Seite blockiert).
func TestTempViewFileURL(t *testing.T) {
	defer ui.CleanupTempViewFile()
	written, err := ui.WriteTempViewFile("<html></html>")
	if err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	if got := ui.TempViewFileURL(); got != written {
		t.Errorf("TempViewFileURL() = %q, WriteTempViewFile lieferte %q", got, written)
	}
	if !strings.HasPrefix(written, "file:///") {
		t.Errorf("Temp-URL %q beginnt nicht mit file:///", written)
	}
}

// TestAllowNavigation prüft die Erlaubnisliste für Navigationen des
// App-Fensters (F1, Build 47): Nur die eigene Seite (SetHtml lädt unter
// WebKitGTK als about:blank, Basis-URI NULL) und genau die eigene
// Temp-Ansichtsdatei dürfen geladen werden. Alles andere – fremde Websites
// (sonst Go-Bindings auf fremder Seite), lokale Dateien, javascript:, data: –
// wird abgelehnt (fail closed).
func TestAllowNavigation(t *testing.T) {
	tempURL := "file:///tmp/md-reader-view-4711.html"
	cases := []struct {
		uri  string
		want bool
	}{
		// Eigene Seite (SetHtml) und Anker innerhalb dieser Seite
		{"about:blank", true},
		{"about:blank#kapitel-2", true},
		// Eigene Temp-Ansichtsdatei (große Dokumente) samt Anker
		{tempURL, true},
		{tempURL + "#abschnitt", true},
		// Fremde Websites und Mail: nie im App-Fenster
		{"http://x", false},
		{"https://example.com/index.html", false},
		{"http://127.0.0.1:8765/index.html", false},
		{"mailto:a@b.c", false},
		// Andere lokale Dateien, auch ähnlich benannte
		{"file:///etc/passwd", false},
		{"file:///tmp/md-reader-view-4711.html.evil", false},
		{"file:///tmp/md-reader-view-4711.html?x=1", false},
		{"file://server/tmp/md-reader-view-4711.html", false},
		{"file:///tmp/../tmp/md-reader-view-4711.htm", false},
		// Skript- und Daten-URLs (SetHtml nutzt unter WebKitGTK kein data:)
		{"javascript:alert(1)", false},
		{"data:text/html,<h1>x</h1>", false},
		{"data:text/html;charset=utf-8;base64,PGgxPng8L2gxPg==", false},
		// Sonstiges: leer, andere about:-Seiten, Großschreibung, Leerzeichen
		{"", false},
		{"about:srcdoc", false},
		{"about:blankx", false},
		{"ABOUT:BLANK", false},
		{" about:blank", false},
		{"blob:about:blank/1234", false},
	}
	for _, c := range cases {
		if got := ui.AllowNavigation(c.uri, tempURL); got != c.want {
			t.Errorf("AllowNavigation(%q) = %v, erwartet %v", c.uri, got, c.want)
		}
	}
}

// TestAllowNavigation_NoTempViewURL prüft, dass eine leere Temp-URL nie eine
// file://-Navigation freigibt (z. B. wenn noch keine Temp-Datei existiert).
func TestAllowNavigation_NoTempViewURL(t *testing.T) {
	for _, uri := range []string{"", "file:///", "file:///tmp/x.html"} {
		if ui.AllowNavigation(uri, "") {
			t.Errorf("AllowNavigation(%q, \"\") = true, erwartet false", uri)
		}
	}
	if !ui.AllowNavigation("about:blank", "") {
		t.Error("about:blank muss auch ohne Temp-URL erlaubt sein")
	}
}
