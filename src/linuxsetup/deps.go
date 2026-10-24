// Package linuxsetup – Starter, Selbst-Installation und Abhängigkeitsprüfung
// von MD Reader unter Linux.
//
// Dieses Paket ist reines Go ohne cgo und ohne Build-Tags, damit es auf allen
// Systemen übersetzt und getestet werden kann. Die Datei deps.go wertet das
// Ladeprotokoll des dynamischen Linkers aus (fehlende Bibliotheken) und kennt
// die festen Befehle zum Nachinstallieren von WebKitGTK.
//
// Sicherheit: Befehle sind feste Argumentlisten für exec.Command – nie über
// eine Shell und nie mit Inhalten aus Dokumenten, Dateinamen oder Umgebung.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 1: Ladeprotokoll und Paketmanager)
package linuxsetup

import (
	"sort"
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

// AptInstallCommand liefert den festen Befehl zum Installieren von WebKitGTK.
// Jeder Aufruf gibt eine neue Kopie zurück, damit Aufrufer die Liste ändern dürfen.
//
// @return Argumentliste für exec.Command: pkexec apt-get install -y libwebkit2gtk-4.1-0.
func AptInstallCommand() []string {
	return []string{"pkexec", "apt-get", "install", "-y", WebKitPackage}
}

// AptUpdateCommand liefert den festen Befehl zum Aktualisieren der Paketlisten.
// Jeder Aufruf gibt eine neue Kopie zurück.
//
// @return Argumentliste für exec.Command: pkexec apt-get update.
func AptUpdateCommand() []string {
	return []string{"pkexec", "apt-get", "update"}
}
