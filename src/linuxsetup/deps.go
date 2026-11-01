// Package linuxsetup – Starter, Selbst-Installation und Abhängigkeitsprüfung
// von MD Reader unter Linux.
//
// Dieses Paket ist reines Go ohne cgo und ohne Build-Tags, damit es auf allen
// Systemen übersetzt und getestet werden kann. Die Datei deps.go wertet das
// Ladeprotokoll des dynamischen Linkers aus (fehlende Bibliotheken, fehlende
// Symbolversionen wie GLIBC_2.38 bei einem zu alten System) und kennt die
// festen Befehle zum Nachinstallieren von WebKitGTK.
//
// Sicherheit: Befehle sind feste Argumentlisten für exec.Command – nie über
// eine Shell und nie mit Inhalten aus Dokumenten, Dateinamen oder Umgebung.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (fehlende Symbolversionen erkennen – System zu alt, Bug-Runde)
package linuxsetup

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// WebKitPackage ist der Debian/Ubuntu-Paketname der benötigten WebKitGTK-Bibliothek.
const WebKitPackage = "libwebkit2gtk-4.1-0"

// PackageManager bezeichnet den auf dem System gefundenen Paketmanager.
type PackageManager int

const (
	// PMNone: kein bekannter Paketmanager gefunden.
	PMNone PackageManager = iota
	// PMApt: Debian, Ubuntu, Linux Mint (apt-get).
	PMApt
	// PMDnf: Fedora, RHEL (dnf).
	PMDnf
	// PMPacman: Arch Linux (pacman).
	PMPacman
	// PMZypper: openSUSE (zypper).
	PMZypper
)

// ParseMissingLibraries wertet die Ausgabe von "ldd" (bzw. LD_TRACE_LOADED_OBJECTS)
// aus und liefert die Namen aller Bibliotheken, die mit "=> not found" gemeldet werden.
//
// Zeilen haben die Form "<name> => not found"; Leerraum und Windows-Zeilenenden
// werden ignoriert. Die Liste ist sortiert und ohne Duplikate.
//
// @param trace Rohausgabe des Ladeprotokolls.
// @return Fehlende Bibliotheken (leer, wenn alles gefunden wurde).
func ParseMissingLibraries(trace string) []string {
	seen := make(map[string]bool)
	var missing []string
	for _, line := range strings.Split(trace, "\n") {
		// Fields trennt an beliebigem Leerraum (Tabs, \r, Mehrfach-Leerzeichen).
		fields := strings.Fields(line)
		// Erwartet genau: <name> => not found
		if len(fields) != 4 || fields[1] != "=>" || fields[2] != "not" || fields[3] != "found" {
			continue
		}
		name := fields[0]
		if !seen[name] {
			seen[name] = true
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

// missingVersionPattern erkennt die Meldung des Laders für eine fehlende
// Symbolversion, z. B. "version `GLIBC_2.38' not found (required by …)".
// Ältere glibc schreiben `…', neuere womöglich '…' oder typografische
// Anführungszeichen; erlaubt sind alle drei Formen.
// Zeilen mit „weak version `X' not found“ (nur bei LD_VERBOSE) zählen nicht:
// Eine fehlende schwache Version ist nicht tödlich (siehe weakVersionMarker).
var missingVersionPattern = regexp.MustCompile("version [`'‘]([^`'‘’\\s]+)['’] not found")

// ParseMissingVersions wertet die Meldungen des Laders (stderr des
// Ladeprotokolls) aus und liefert alle fehlenden Symbolversionen.
//
// Nur Zeilen der Form „version `X' not found“ zählen – sie bedeuten, dass
// eine Bibliothek des Systems (meist die glibc) älter ist als verlangt; dann
// kann das Programm nie starten. Harmlose Meldungen wie „no version
// information available“ und „weak version `X' not found“ werden ignoriert.
// Eine Version ist nie fest
// eingebaut, sondern stammt immer aus dem Protokoll.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (weak version ausgenommen, Nachrunde)
//
// @param loader Rohausgabe des Laders (stderr).
// @return Fehlende Versionen, sortiert und ohne Duplikate (leer, wenn keine).
func ParseMissingVersions(loader string) []string {
	seen := make(map[string]bool)
	var versions []string
	// Zeilenweise, damit eine „weak version“-Zeile nur sich selbst ausschließt.
	for _, line := range strings.Split(loader, "\n") {
		if strings.Contains(line, weakVersionMarker) {
			continue
		}
		for _, m := range missingVersionPattern.FindAllStringSubmatch(line, -1) {
			if v := m[1]; !seen[v] {
				seen[v] = true
				versions = append(versions, v)
			}
		}
	}
	sort.Strings(versions)
	return versions
}

// weakVersionMarker kennzeichnet die harmlose Lader-Meldung für eine fehlende
// schwache Symbolversion (glibc gibt sie nur mit LD_VERBOSE aus).
const weakVersionMarker = "weak version "

// glibcVersionPrefix ist der Name der Symbolversionen der glibc ("GLIBC_2.38").
const glibcVersionPrefix = "GLIBC_"

// DescribeRequiredVersions macht aus fehlenden Symbolversionen einen kurzen
// Text für die Meldung „System zu alt“.
//
// Alle GLIBC_x.y werden zur höchsten Version „glibc x.y“ zusammengefasst
// (numerischer Vergleich je Teil: 2.38 ist neuer als 2.9). Andere Versionen
// (z. B. GLIBCXX_3.4.30 oder GLIBC_PRIVATE) folgen unverändert, mit ", "
// getrennt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, Bug-Runde Build 47: System zu alt)
//
// @param versions Ergebnis von ParseMissingVersions.
// @return Text wie "glibc 2.38" oder "glibc 2.38, GLIBCXX_3.4.30"; "" ohne Versionen.
func DescribeRequiredVersions(versions []string) string {
	var glibc []int
	glibcText := ""
	var others []string
	for _, v := range versions {
		number, ok := strings.CutPrefix(v, glibcVersionPrefix)
		parts, valid := parseVersionNumber(number)
		if !ok || !valid {
			others = append(others, v)
			continue
		}
		if glibc == nil || compareVersionParts(parts, glibc) > 0 {
			glibc, glibcText = parts, number
		}
	}
	var out []string
	if glibc != nil {
		out = append(out, "glibc "+glibcText)
	}
	sort.Strings(others)
	return strings.Join(append(out, others...), ", ")
}

// parseVersionNumber zerlegt "2.38" in [2 38].
//
// @return Teile und true; false, wenn ein Teil keine Zahl ist.
func parseVersionNumber(s string) ([]int, bool) {
	if s == "" {
		return nil, false
	}
	fields := strings.Split(s, ".")
	parts := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}

// compareVersionParts vergleicht zwei Versionen teilweise numerisch
// (fehlende Teile zählen als 0).
//
// @return <0, 0 oder >0 wie strings.Compare.
func compareVersionParts(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x - y
		}
	}
	return 0
}

// DetectPackageManager sucht den Paketmanager in der Reihenfolge
// apt-get, dnf, pacman, zypper.
//
// @param lookPath Suchfunktion (im Betrieb exec.LookPath, im Test ein Fake).
// @return Gefundener Paketmanager oder PMNone.
func DetectPackageManager(lookPath func(string) (string, error)) PackageManager {
	candidates := []struct {
		binary string
		pm     PackageManager
	}{
		{"apt-get", PMApt},
		{"dnf", PMDnf},
		{"pacman", PMPacman},
		{"zypper", PMZypper},
	}
	for _, c := range candidates {
		if _, err := lookPath(c.binary); err == nil {
			return c.pm
		}
	}
	return PMNone
}

// InstallHint liefert den Befehl, mit dem der Nutzer WebKitGTK 4.1 selbst
// installieren kann (für Systeme ohne automatische Installation).
//
// @param pm Erkannter Paketmanager.
// @return Befehlszeile bzw. allgemeiner Hinweis bei PMNone.
func InstallHint(pm PackageManager) string {
	switch pm {
	case PMApt:
		return "sudo apt install " + WebKitPackage
	case PMDnf:
		return "sudo dnf install webkit2gtk4.1"
	case PMPacman:
		return "sudo pacman -S webkit2gtk-4.1"
	case PMZypper:
		return "sudo zypper install libwebkit2gtk-4_1-0"
	}
	return "Install WebKitGTK 4.1 with your package manager"
}

// PkexecPath ist der feste Pfad von pkexec (Härtung H3, Build 47).
// Keine Suche über PATH: Ein ~/.local/bin/pkexec oder ~/.local/bin/apt-get
// vor /usr/bin liefe sonst nach der Passworteingabe (bzw. mit dem
// Vertrauen des Nutzers) statt des echten Programms.
const PkexecPath = "/usr/bin/pkexec"

// AptGetPath ist der feste Pfad von apt-get (Härtung H3, Build 47).
const AptGetPath = "/usr/bin/apt-get"

// AptInstallCommand liefert den festen Befehl zum Installieren von WebKitGTK.
// Jeder Aufruf gibt eine neue Kopie zurück, damit Aufrufer die Liste ändern dürfen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (absolute Pfade, H3)
//
// @return Argumentliste für exec.Command: /usr/bin/pkexec /usr/bin/apt-get install -y libwebkit2gtk-4.1-0.
func AptInstallCommand() []string {
	return []string{PkexecPath, AptGetPath, "install", "-y", WebKitPackage}
}

// AptUpdateCommand liefert den festen Befehl zum Aktualisieren der Paketlisten.
// Jeder Aufruf gibt eine neue Kopie zurück.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (absolute Pfade, H3)
//
// @return Argumentliste für exec.Command: /usr/bin/pkexec /usr/bin/apt-get update.
func AptUpdateCommand() []string {
	return []string{PkexecPath, AptGetPath, "update"}
}
