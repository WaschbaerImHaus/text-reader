// Tests für deps.go – Auswertung des Ladeprotokolls, Paketmanager-Erkennung
// und die festen Installationsbefehle.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 1: Ladeprotokoll und Paketmanager)
package linuxsetup

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestParseMissingLibraries prüft, dass nur fehlende Bibliotheken
// herauskommen – sortiert und ohne Duplikate.
func TestParseMissingLibraries(t *testing.T) {
	trace := "\tlinux-vdso.so.1 (0x00007ffd)\n" +
		"\tlibwebkit2gtk-4.1.so.0 => not found\n" +
		"\tlibgtk-3.so.0 => /lib/x86_64-linux-gnu/libgtk-3.so.0 (0x7f)\n" +
		"\tlibjavascriptcoregtk-4.1.so.0 => not found\n" +
		"\tlibwebkit2gtk-4.1.so.0 => not found\n"
	got := ParseMissingLibraries(trace)
	want := []string{"libjavascriptcoregtk-4.1.so.0", "libwebkit2gtk-4.1.so.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMissingLibraries = %v, erwartet %v", got, want)
	}
}

// TestParseMissingLibraries_None prüft leere und vollständige Protokolle.
func TestParseMissingLibraries_None(t *testing.T) {
	complete := "\tlinux-vdso.so.1 (0x00007ffd)\n" +
		"\tlibgtk-3.so.0 => /lib/x86_64-linux-gnu/libgtk-3.so.0 (0x7f)\n"
	for name, trace := range map[string]string{"leer": "", "vollständig": complete} {
		if got := ParseMissingLibraries(trace); len(got) != 0 {
			t.Errorf("%s: erwartet keine fehlenden Bibliotheken, bekam %v", name, got)
		}
	}
}

// TestParseMissingLibraries_Whitespace prüft Windows-Zeilenenden und
// zusätzlichen Leerraum.
func TestParseMissingLibraries_Whitespace(t *testing.T) {
	trace := "  libfoo.so.1 => not found  \r\n\tlibbar.so.2   =>   not found\r\n"
	got := ParseMissingLibraries(trace)
	want := []string{"libbar.so.2", "libfoo.so.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMissingLibraries = %v, erwartet %v", got, want)
	}
}

// TestDetectPackageManager prüft die Reihenfolge apt-get, dnf, pacman, zypper.
func TestDetectPackageManager(t *testing.T) {
	// fakeLookPath findet nur die genannten Programme.
	fakeLookPath := func(present ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			for _, p := range present {
				if p == name {
					return "/usr/bin/" + name, nil
				}
			}
			return "", errors.New("nicht gefunden")
		}
	}
	cases := []struct {
		name    string
		present []string
		want    PackageManager
	}{
		{"nur dnf", []string{"dnf"}, PMDnf},
		{"apt-get und dnf", []string{"dnf", "apt-get"}, PMApt},
		{"pacman", []string{"pacman"}, PMPacman},
		{"zypper", []string{"zypper"}, PMZypper},
		{"dnf vor pacman", []string{"pacman", "dnf"}, PMDnf},
		{"pacman vor zypper", []string{"zypper", "pacman"}, PMPacman},
		{"nichts", nil, PMNone},
	}
	for _, c := range cases {
		if got := DetectPackageManager(fakeLookPath(c.present...)); got != c.want {
			t.Errorf("%s: %v, erwartet %v", c.name, got, c.want)
		}
	}
}

// TestInstallHint prüft die Installationshinweise je Paketmanager.
func TestInstallHint(t *testing.T) {
	cases := map[PackageManager]string{
		PMApt:    "sudo apt install libwebkit2gtk-4.1-0",
		PMDnf:    "sudo dnf install webkit2gtk4.1",
		PMPacman: "sudo pacman -S webkit2gtk-4.1",
		PMZypper: "sudo zypper install libwebkit2gtk-4_1-0",
	}
	for pm, want := range cases {
		if got := InstallHint(pm); got != want {
			t.Errorf("InstallHint(%v) = %q, erwartet %q", pm, got, want)
		}
	}
	if got := InstallHint(PMNone); !strings.Contains(got, "WebKitGTK 4.1") {
		t.Errorf("InstallHint(PMNone) = %q, erwartet Text mit \"WebKitGTK 4.1\"", got)
	}
}

// TestAptCommandsAreFixed prüft die festen Befehle und dass Aufrufer
// keine gemeinsame Liste verändern können.
func TestAptCommandsAreFixed(t *testing.T) {
	if WebKitPackage != "libwebkit2gtk-4.1-0" {
		t.Fatalf("WebKitPackage = %q", WebKitPackage)
	}
	wantInstall := []string{"pkexec", "apt-get", "install", "-y", "libwebkit2gtk-4.1-0"}
	wantUpdate := []string{"pkexec", "apt-get", "update"}

	first := AptInstallCommand()
	if !reflect.DeepEqual(first, wantInstall) {
		t.Errorf("AptInstallCommand = %v, erwartet %v", first, wantInstall)
	}
	first[0] = "evil"
	if second := AptInstallCommand(); !reflect.DeepEqual(second, wantInstall) {
		t.Errorf("AptInstallCommand nach Änderung der Kopie = %v, erwartet %v", second, wantInstall)
	}

	firstUp := AptUpdateCommand()
	if !reflect.DeepEqual(firstUp, wantUpdate) {
		t.Errorf("AptUpdateCommand = %v, erwartet %v", firstUp, wantUpdate)
	}
	firstUp[1] = "evil"
	if secondUp := AptUpdateCommand(); !reflect.DeepEqual(secondUp, wantUpdate) {
		t.Errorf("AptUpdateCommand nach Änderung der Kopie = %v, erwartet %v", secondUp, wantUpdate)
	}
}
