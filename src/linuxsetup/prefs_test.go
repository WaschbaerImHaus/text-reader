// Tests für prefs.go – Antwortspeicher des Starters (launcher.json).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 3: Installieren, Entfernen, Entscheiden)
package linuxsetup

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPrefs_RoundTrip: speichern und wieder laden ergibt denselben Wert.
func TestPrefs_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "md-reader", "launcher.json")
	if err := SavePrefs(path, Prefs{InstallPrompt: "never"}); err != nil {
		t.Fatal(err)
	}
	if got := LoadPrefs(path); got.InstallPrompt != "never" {
		t.Errorf("LoadPrefs = %+v", got)
	}
}

// TestPrefs_BrokenJSON: kaputter Inhalt ergibt leere Einstellungen.
func TestPrefs_BrokenJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher.json")
	if err := os.WriteFile(path, []byte("{kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadPrefs(path); got != (Prefs{}) {
		t.Errorf("LoadPrefs = %+v", got)
	}
}

// TestPrefs_Missing: eine fehlende Datei ergibt leere Einstellungen.
func TestPrefs_Missing(t *testing.T) {
	if got := LoadPrefs(filepath.Join(t.TempDir(), "nope.json")); got != (Prefs{}) {
		t.Errorf("LoadPrefs = %+v", got)
	}
}

// TestPrefs_UnknownValue: unbekannte Werte werden wie "nicht beantwortet" behandelt.
func TestPrefs_UnknownValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher.json")
	if err := os.WriteFile(path, []byte(`{"installPrompt":"sudo rm -rf"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadPrefs(path); got != (Prefs{}) {
		t.Errorf("LoadPrefs = %+v", got)
	}
}

// TestPrefs_Modes: Verzeichnis 0700, Datei 0600 – auch bei bestehender Datei mit anderem Modus.
func TestPrefs_Modes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "md-reader")
	path := filepath.Join(dir, "launcher.json")
	if err := SavePrefs(path, Prefs{InstallPrompt: "later"}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0o700 {
		t.Errorf("Verzeichnis = %o", info.Mode().Perm())
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("Datei = %o", info.Mode().Perm())
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SavePrefs(path, Prefs{InstallPrompt: "never"}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("Datei nach Überschreiben = %o", info.Mode().Perm())
	}
}
