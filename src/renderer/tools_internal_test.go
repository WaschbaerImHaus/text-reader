// Interne Tests für die Suche nach externen Programmen (pdftoppm, gs)
// und die plattformabhängigen Entscheidungen beim PDF/PS-Rendering.
//
// Hintergrund (macOS-Port, Build 46): Apps, die aus dem Finder starten,
// bekommen nur PATH=/usr/bin:/bin:/usr/sbin:/sbin. Homebrew installiert
// aber nach /opt/homebrew/bin (Apple Silicon) bzw. /usr/local/bin (Intel).
// Ohne zusätzliche Suchorte fände die App poppler/ghostscript nie.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24
package renderer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestToolSearchDirs_Darwin prüft die Homebrew-/MacPorts-Orte unter macOS.
func TestToolSearchDirs_Darwin(t *testing.T) {
	dirs := toolSearchDirs("darwin")
	want := []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin"}
	if strings.Join(dirs, ",") != strings.Join(want, ",") {
		t.Errorf("toolSearchDirs(darwin) = %v, erwartet %v", dirs, want)
	}
}

// TestToolSearchDirs_OtherSystems prüft, dass Linux und Windows nur PATH nutzen.
func TestToolSearchDirs_OtherSystems(t *testing.T) {
	for _, goos := range []string{"linux", "windows", "freebsd", ""} {
		if dirs := toolSearchDirs(goos); len(dirs) != 0 {
			t.Errorf("toolSearchDirs(%q) = %v, erwartet keine Zusatzorte", goos, dirs)
		}
	}
}

// TestFindToolInDirs_Found prüft, dass ein ausführbares Programm gefunden wird,
// auch wenn es erst im zweiten Verzeichnis liegt.
func TestFindToolInDirs_Found(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ausführbarkeitsbit gibt es unter Windows nicht")
	}
	empty := t.TempDir()
	dir := t.TempDir()
	tool := filepath.Join(dir, "pdftoppm")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findToolInDirs("pdftoppm", []string{empty, dir}); got != tool {
		t.Errorf("findToolInDirs = %q, erwartet %q", got, tool)
	}
}

// TestFindToolInDirs_NotExecutable prüft, dass eine nicht ausführbare Datei
// gleichen Namens ignoriert wird.
func TestFindToolInDirs_NotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ausführbarkeitsbit gibt es unter Windows nicht")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gs"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findToolInDirs("gs", []string{dir}); got != "" {
		t.Errorf("nicht ausführbare Datei gefunden: %q", got)
	}
}

// TestFindToolInDirs_DirectoryIgnored prüft, dass ein Verzeichnis mit dem
// Programmnamen nicht als Programm gilt.
func TestFindToolInDirs_DirectoryIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "gs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findToolInDirs("gs", []string{dir}); got != "" {
		t.Errorf("Verzeichnis als Programm gefunden: %q", got)
	}
}

// TestFindToolInDirs_Missing prüft leere Rückgabe ohne Treffer und ohne Orte.
func TestFindToolInDirs_Missing(t *testing.T) {
	if got := findToolInDirs("pdftoppm", []string{t.TempDir()}); got != "" {
		t.Errorf("findToolInDirs ohne Datei = %q", got)
	}
	if got := findToolInDirs("pdftoppm", nil); got != "" {
		t.Errorf("findToolInDirs ohne Orte = %q", got)
	}
}

// TestFindToolInDirs_RejectsPathInName prüft, dass der Name keinen Pfad
// enthalten darf (sonst ließe sich aus dem Suchort ausbrechen).
func TestFindToolInDirs_RejectsPathInName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ausführbarkeitsbit gibt es unter Windows nicht")
	}
	base := t.TempDir()
	sub := filepath.Join(base, "bin")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "gs"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../gs", "", ".", ".."} {
		if got := findToolInDirs(name, []string{sub}); got != "" {
			t.Errorf("findToolInDirs(%q) = %q, erwartet leer", name, got)
		}
	}
}

// TestPDFUsesPageImages prüft, auf welchen Systemen pdftoppm versucht wird:
// überall außer Windows (dort zeigt WebView2 das PDF selbst an).
func TestPDFUsesPageImages(t *testing.T) {
	cases := map[string]bool{"linux": true, "darwin": true, "freebsd": true, "windows": false}
	for goos, want := range cases {
		if got := pdfUsesPageImages(goos); got != want {
			t.Errorf("pdfUsesPageImages(%q) = %v, erwartet %v", goos, got, want)
		}
	}
}

// TestGhostscriptInstallHint prüft den passenden Installationshinweis je System.
func TestGhostscriptInstallHint(t *testing.T) {
	cases := map[string]string{
		"darwin":  "brew install ghostscript",
		"linux":   "sudo apt-get install ghostscript",
		"windows": "MD Reader Setup",
	}
	for goos, want := range cases {
		if got := ghostscriptInstallHint(goos); !strings.Contains(got, want) {
			t.Errorf("ghostscriptInstallHint(%q) = %q, erwartet Teil %q", goos, got, want)
		}
	}
	if strings.Contains(ghostscriptInstallHint("darwin"), "apt-get") {
		t.Error("macOS-Hinweis nennt apt-get")
	}
}
