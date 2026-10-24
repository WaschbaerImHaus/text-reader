// Tests für desktop.go – Inhalt der Desktop-Datei, Quoting der Exec-Zeile
// und Auslesen der Build-Nummer.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 2: Desktop-Datei)
package linuxsetup

import (
	"strings"
	"testing"
)

// hasLine prüft, ob der Inhalt genau diese Zeile enthält.
func hasLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if l == line {
			return true
		}
	}
	return false
}

// TestDesktopEntry_Fields prüft alle Pflichtzeilen und deren Reihenfolge.
func TestDesktopEntry_Fields(t *testing.T) {
	got := DesktopEntry("/home/u/.local/bin/md-reader", 47)
	wantLines := []string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=MD Reader",
		"GenericName=Document Viewer",
		"GenericName[de]=Dokumentbetrachter",
		"Comment=View Markdown, EPUB, FB2, LaTeX, PDF and more",
		"Comment[de]=Markdown, EPUB, FB2, LaTeX, PDF und mehr anzeigen",
		`Exec="/home/u/.local/bin/md-reader" %f`,
		"Icon=md-reader",
		"Terminal=false",
		"Categories=Office;Viewer;",
		"MimeType=text/markdown;text/x-markdown;text/plain;application/epub+zip;application/x-fictionbook+xml;text/x-tex;text/html;application/pdf;application/postscript;",
		"StartupWMClass=md-reader",
		"X-MD-Reader-Build=47",
	}
	prev := -1
	for _, line := range wantLines {
		if !hasLine(got, line) {
			t.Errorf("Zeile fehlt: %q", line)
			continue
		}
		idx := strings.Index(got, line+"\n")
		if idx < prev {
			t.Errorf("Zeile in falscher Reihenfolge: %q", line)
		}
		prev = idx
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("Inhalt muss mit Zeilenende enden")
	}
	if strings.Contains(got, "\r") || strings.Contains(got, "\t") {
		t.Error("Inhalt darf weder CR noch Tabs enthalten")
	}
}

// TestDesktopEntry_QuotesSpecialPaths prüft das Quoting nach der
// Desktop-Entry-Spezifikation (Anführungszeichen, Backtick, Dollar, Backslash).
func TestDesktopEntry_QuotesSpecialPaths(t *testing.T) {
	got := DesktopEntry(`/home/a b/$x/"q"/back\slash/md-reader`, 47)
	want := `Exec="/home/a b/\\$x/\\"q\\"/back\\\\slash/md-reader" %f`
	if !hasLine(got, want) {
		t.Errorf("Exec-Zeile falsch.\nerwartet: %s\ninhalt:\n%s", want, got)
	}
	got = DesktopEntry("/home/a`b/md-reader", 47)
	if want := "Exec=\"/home/a\\\\`b/md-reader\" %f"; !hasLine(got, want) {
		t.Errorf("Backtick nicht escaped. erwartet: %s\ninhalt:\n%s", want, got)
	}
}

// TestDesktopEntry_PercentInPath prüft, dass % verdoppelt wird
// (sonst wäre es ein Feldcode).
func TestDesktopEntry_PercentInPath(t *testing.T) {
	got := DesktopEntry("/home/100%/md-reader", 47)
	if want := `Exec="/home/100%%/md-reader" %f`; !hasLine(got, want) {
		t.Errorf("Prozentzeichen nicht verdoppelt. erwartet: %s\ninhalt:\n%s", want, got)
	}
}

// TestQuoteExec prüft die Hilfsfunktion direkt.
func TestQuoteExec(t *testing.T) {
	if got, want := quoteExec("/usr/bin/md-reader"), `"/usr/bin/md-reader"`; got != want {
		t.Errorf("quoteExec = %s, erwartet %s", got, want)
	}
}

// TestBuildFromDesktopEntry prüft Rundreise, fehlenden und ungültigen Schlüssel.
func TestBuildFromDesktopEntry(t *testing.T) {
	if n, ok := BuildFromDesktopEntry(DesktopEntry("/x/md-reader", 47)); !ok || n != 47 {
		t.Errorf("Rundreise = (%d, %v), erwartet (47, true)", n, ok)
	}
	if n, ok := BuildFromDesktopEntry("[Desktop Entry]\nName=MD Reader\n"); ok || n != 0 {
		t.Errorf("ohne Schlüssel = (%d, %v), erwartet (0, false)", n, ok)
	}
	if n, ok := BuildFromDesktopEntry("[Desktop Entry]\nX-MD-Reader-Build=abc\n"); ok || n != 0 {
		t.Errorf("ungültig = (%d, %v), erwartet (0, false)", n, ok)
	}
	// Windows-Zeilenenden und Leerraum werden toleriert.
	if n, ok := BuildFromDesktopEntry("[Desktop Entry]\r\nX-MD-Reader-Build=12\r\n"); !ok || n != 12 {
		t.Errorf("CRLF = (%d, %v), erwartet (12, true)", n, ok)
	}
}
