// Tests für deps.go – Auswertung des Ladeprotokolls, Paketmanager-Erkennung
// und die festen Installationsbefehle.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (fehlende Symbolversionen: ParseMissingVersions, DescribeRequiredVersions – Bug-Runde Build 47)
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
	// H3: absolute Pfade, keine Suche über PATH (ein ~/.local/bin/apt-get
	// liefe sonst nach der Passworteingabe als root)
	wantInstall := []string{"/usr/bin/pkexec", "/usr/bin/apt-get", "install", "-y", "libwebkit2gtk-4.1-0"}
	wantUpdate := []string{"/usr/bin/pkexec", "/usr/bin/apt-get", "update"}

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

// TestParseMissingVersions prüft die Erkennung fehlender Symbolversionen im
// Ladeprotokoll (stderr des Laders): nur Zeilen „version `…' not found“
// zählen, harmlose Meldungen wie „no version information available“ nicht.
// Ergebnis sortiert und ohne Duplikate (Bug-Runde Build 47, Fund 2).
func TestParseMissingVersions(t *testing.T) {
	loader := "md-reader: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.36' not found (required by md-reader)\n" +
		"md-reader: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.38' not found (required by md-reader)\n" +
		"md-reader: /usr/lib/libfoo.so.1: no version information available (required by md-reader)\n" +
		"md-reader: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.38' not found (required by /lib/libbar.so)\r\n" +
		"md-reader: /lib/libstdc++.so.6: version 'GLIBCXX_3.4.30' not found (required by md-reader)\n" +
		"md-reader: /lib/x86_64-linux-gnu/libc.so.6: weak version `GLIBC_2.40' not found (required by md-reader)\n"
	got := ParseMissingVersions(loader)
	want := []string{"GLIBCXX_3.4.30", "GLIBC_2.36", "GLIBC_2.38"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMissingVersions = %v, erwartet %v", got, want)
	}
	for _, harmless := range []string{
		"",
		"md-reader: /usr/lib/libfoo.so.1: no version information available (required by md-reader)\n",
		"\tlibwebkit2gtk-4.1.so.0 => not found\n",
		// Nur mit LD_VERBOSE: fehlende schwache Version ist nicht tödlich.
		"md-reader: /lib/x86_64-linux-gnu/libc.so.6: weak version `GLIBC_2.40' not found (required by md-reader)\n",
		"ERROR: ld.so: object 'x.so' from LD_PRELOAD cannot be preloaded: ignored.\n",
	} {
		if got := ParseMissingVersions(harmless); len(got) != 0 {
			t.Errorf("ParseMissingVersions(%q) = %v, erwartet leer", harmless, got)
		}
	}
}

// TestDescribeRequiredVersions prüft den Text für die Meldung „System zu
// alt“: GLIBC_x.y wird zur höchsten Version „glibc x.y“ (numerisch
// verglichen, 2.38 > 2.9), andere Versionen werden unverändert genannt.
func TestDescribeRequiredVersions(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"GLIBC_2.36", "GLIBC_2.38"}, "glibc 2.38"},
		{[]string{"GLIBC_2.38", "GLIBC_2.9"}, "glibc 2.38"},
		{[]string{"GLIBC_2.35.1", "GLIBC_2.35"}, "glibc 2.35.1"},
		{[]string{"GLIBCXX_3.4.30", "GLIBC_2.38"}, "glibc 2.38, GLIBCXX_3.4.30"},
		{[]string{"MDR_2"}, "MDR_2"},
		{[]string{"GLIBC_PRIVATE"}, "GLIBC_PRIVATE"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := DescribeRequiredVersions(c.in); got != c.want {
			t.Errorf("DescribeRequiredVersions(%v) = %q, erwartet %q", c.in, got, c.want)
		}
	}
}
