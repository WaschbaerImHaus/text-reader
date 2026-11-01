// Tests für die Temp-Datei-Navigation bei großen HTML-Dokumenten.
//
// Hintergrund (Bug #012): WebView2 (Windows) begrenzt NavigateToString auf
// 2 MB. Größere Dokumente müssen als temporäre Datei gespeichert und per
// file://-URL geladen werden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (privates Temp-Verzeichnis, zufälliger Name, N1)
package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// TestCleanupTempViewFile prüft, dass die Temp-Datei samt privatem
// Verzeichnis gelöscht wird und danach keine Temp-URL mehr gilt (der
// Navigationsschutz gibt dann keine file://-Adresse mehr frei, N1).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (Verzeichnis und leere URL geprüft, N1)
func TestCleanupTempViewFile(t *testing.T) {
	if _, err := ui.WriteTempViewFile("wegwerfen"); err != nil {
		t.Fatalf("WriteTempViewFile() Fehler: %v", err)
	}
	path := ui.TempViewFilePath()
	ui.CleanupTempViewFile()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("CleanupTempViewFile() hat die Datei nicht gelöscht")
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("CleanupTempViewFile() hat das private Verzeichnis %s nicht gelöscht", filepath.Dir(path))
	}
	if got := ui.TempViewFileURL(); got != "" {
		t.Errorf("TempViewFileURL() nach Cleanup = %q, erwartet leer", got)
	}
	if got := ui.TempViewFilePath(); got != "" {
		t.Errorf("TempViewFilePath() nach Cleanup = %q, erwartet leer", got)
	}
}

// useIsolatedTempDir lenkt os.TempDir() für einen Test auf ein eigenes,
// leeres Verzeichnis um (Linux/macOS: TMPDIR, Windows: TMP/TEMP) und räumt
// vorher und nachher die Temp-Ansichtsdatei auf.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @param t Laufender Test.
// @return Pfad des umgelenkten Temp-Verzeichnisses.
func useIsolatedTempDir(t *testing.T) string {
	t.Helper()
	ui.CleanupTempViewFile()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
	t.Cleanup(ui.CleanupTempViewFile)
	return dir
}

// requireStrictlyInside bricht den Test ab, wenn path nicht echt unterhalb
// von dir liegt. Schutz vor Schäden durch fehlerhaften Code: Ohne diese
// Prüfung würde z. B. ein RemoveAll auf das Elternverzeichnis einer Datei
// direkt in /tmp das ganze gemeinsame Temp-Verzeichnis leeren.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @param t    Laufender Test.
// @param dir  Isoliertes Temp-Verzeichnis des Tests.
// @param path Pfad, der gelöscht werden soll.
func requireStrictlyInside(t *testing.T, dir, path string) {
	t.Helper()
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(rel) {
		t.Fatalf("Pfad %s liegt nicht im isolierten Temp-Verzeichnis %s – Löschen verweigert", path, dir)
	}
}

// TestWriteTempViewFile_IgnoresPlantedFile prüft Fund N1 (Build 47): Ein
// anderer lokaler Nutzer kennt den alten, vorhersagbaren Namen
// md-reader-view-<pid>.html im gemeinsamen Temp-Verzeichnis und legt dort
// vorab eine Datei oder einen Symlink ab. WriteTempViewFile darf diesen
// Pfad nie benutzen und keinem Symlink folgen: Weder das Symlink-Ziel noch
// die vorab angelegte Datei dürfen verändert werden.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
func TestWriteTempViewFile_IgnoresPlantedFile(t *testing.T) {
	dir := useIsolatedTempDir(t)
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(dir, fmt.Sprintf("md-reader-view-%d.html", os.Getpid()))
	usedSymlink := os.Symlink(victim, planted) == nil
	if !usedSymlink {
		// z. B. Windows ohne Symlink-Recht: normale fremde Datei ablegen
		if err := os.WriteFile(planted, []byte("planted"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := ui.WriteTempViewFile("<html>geheim</html>"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	if data, _ := os.ReadFile(victim); string(data) != "original" {
		t.Errorf("Symlink-Ziel wurde überschrieben: %q", data)
	}
	if !usedSymlink {
		if data, _ := os.ReadFile(planted); string(data) != "planted" {
			t.Errorf("vorab angelegte Datei wurde überschrieben: %q", data)
		}
	}
	if got := ui.TempViewFilePath(); got == planted {
		t.Errorf("Temp-Ansichtsdatei nutzt den vorhersagbaren Pfad %s", got)
	}
}

// TestWriteTempViewFile_PrivateAndUnpredictable prüft Fund N1: Die Datei
// liegt in einem eigenen Unterverzeichnis des Temp-Verzeichnisses (nur für
// den Besitzer, 0700), ist selbst 0600, und ihr Pfad ist nicht vorhersagbar
// (zwei Sitzungen nacheinander ergeben verschiedene Pfade, die Prozess-ID
// steht nicht im Namen).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
func TestWriteTempViewFile_PrivateAndUnpredictable(t *testing.T) {
	dir := useIsolatedTempDir(t)
	if _, err := ui.WriteTempViewFile("eins"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	first := ui.TempViewFilePath()
	parent := filepath.Dir(first)
	if filepath.Clean(parent) == filepath.Clean(dir) {
		t.Fatalf("Temp-Ansichtsdatei liegt direkt im gemeinsamen Temp-Verzeichnis: %s", first)
	}
	if filepath.Dir(parent) != filepath.Clean(dir) {
		t.Errorf("privates Verzeichnis %s liegt nicht im Temp-Verzeichnis %s", parent, dir)
	}
	if strings.Contains(filepath.Base(first), fmt.Sprint(os.Getpid())) ||
		strings.Contains(filepath.Base(parent), fmt.Sprint(os.Getpid())) {
		t.Errorf("Pfad enthält die vorhersagbare Prozess-ID: %s", first)
	}
	if runtime.GOOS != "windows" {
		// Unter Windows schützt die Nutzer-ACL von %TEMP%, Unix-Rechte gibt es dort nicht
		if st, err := os.Stat(parent); err != nil || st.Mode().Perm() != 0700 {
			t.Errorf("privates Verzeichnis: Rechte %v (Fehler %v), erwartet 0700", st.Mode().Perm(), err)
		}
		if st, err := os.Stat(first); err != nil || st.Mode().Perm() != 0600 {
			t.Errorf("Temp-Ansichtsdatei: Rechte %v (Fehler %v), erwartet 0600", st.Mode().Perm(), err)
		}
	}
	ui.CleanupTempViewFile()
	if _, err := ui.WriteTempViewFile("zwei"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	if second := ui.TempViewFilePath(); second == first {
		t.Errorf("zwei Sitzungen ergeben denselben Pfad %s – Name ist vorhersagbar", first)
	}
}

// TestWriteTempViewFile_RecreatesMissingFile prüft, dass das Schreiben auch
// dann klappt, wenn ein Aufräumdienst (z. B. systemd-tmpfiles) die Datei oder
// das ganze private Verzeichnis gelöscht hat: Dann entsteht eine neue Datei,
// und TempViewFileURL liefert genau deren URL (Navigationsschutz).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
func TestWriteTempViewFile_RecreatesMissingFile(t *testing.T) {
	dir := useIsolatedTempDir(t)
	if _, err := ui.WriteTempViewFile("eins"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	// Erst prüfen, dass Datei UND Verzeichnis im isolierten Bereich liegen
	requireStrictlyInside(t, dir, filepath.Dir(ui.TempViewFilePath()))
	if err := os.Remove(ui.TempViewFilePath()); err != nil {
		t.Fatal(err)
	}
	fileURL, err := ui.WriteTempViewFile("zwei")
	if err != nil {
		t.Fatalf("Schreiben nach gelöschter Datei: %v", err)
	}
	if data, _ := os.ReadFile(ui.TempViewFilePath()); string(data) != "zwei" {
		t.Errorf("Inhalt nach Neuanlage = %q, erwartet \"zwei\"", data)
	}
	if fileURL != ui.TempViewFileURL() {
		t.Errorf("URL %q passt nicht zu TempViewFileURL() %q", fileURL, ui.TempViewFileURL())
	}
	requireStrictlyInside(t, dir, filepath.Dir(ui.TempViewFilePath()))
	if err := os.RemoveAll(filepath.Dir(ui.TempViewFilePath())); err != nil {
		t.Fatal(err)
	}
	if _, err := ui.WriteTempViewFile("drei"); err != nil {
		t.Fatalf("Schreiben nach gelöschtem Verzeichnis: %v", err)
	}
	if data, _ := os.ReadFile(ui.TempViewFilePath()); string(data) != "drei" {
		t.Errorf("Inhalt nach neuem Verzeichnis = %q, erwartet \"drei\"", data)
	}
}

// TestWriteTempViewFile_DoesNotFollowSwappedSymlink prüft, dass ein späteres
// Überschreiben keinem Symlink folgt, der die eigene Datei ersetzt hat: Das
// Ziel bleibt unverändert, es entsteht eine neue eigene Datei.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
func TestWriteTempViewFile_DoesNotFollowSwappedSymlink(t *testing.T) {
	dir := useIsolatedTempDir(t)
	if _, err := ui.WriteTempViewFile("eins"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	path := ui.TempViewFilePath()
	requireStrictlyInside(t, dir, filepath.Dir(path))
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, path); err != nil {
		t.Skipf("Symlink nicht möglich: %v", err)
	}
	if _, err := ui.WriteTempViewFile("zwei"); err != nil {
		t.Fatalf("WriteTempViewFile: %v", err)
	}
	if data, _ := os.ReadFile(victim); string(data) != "original" {
		t.Errorf("Symlink-Ziel wurde überschrieben: %q", data)
	}
	if data, _ := os.ReadFile(ui.TempViewFilePath()); string(data) != "zwei" {
		t.Errorf("neue Temp-Ansichtsdatei enthält %q, erwartet \"zwei\"", data)
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
