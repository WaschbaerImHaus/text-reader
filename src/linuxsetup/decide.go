// Entscheidung des Starters: Soll er das Installieren oder Aktualisieren anbieten?
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 3: Installieren, Entfernen, Entscheiden)
package linuxsetup

import "path/filepath"

// systemBinPath ist der Programmpfad einer Paketinstallation.
const systemBinPath = "/usr/bin/md-reader"

// Offer ist das Ergebnis von Decide.
type Offer int

const (
	// OfferNothing: nichts anbieten, einfach starten.
	OfferNothing Offer = iota
	// OfferInstall: Installation anbieten.
	OfferInstall
	// OfferUpdate: Aktualisierung der Nutzerinstallation anbieten.
	OfferUpdate
)

// StartContext beschreibt die Lage beim Start.
type StartContext struct {
	// ExePath ist der Pfad der laufenden Datei.
	ExePath string
	// CurrentBuild ist die Build-Nummer der laufenden Datei.
	CurrentBuild int
	// Layout enthält die Zielpfade.
	Layout Layout
	// Installed: es gibt eine Nutzerinstallation (siehe InstalledBuild).
	Installed bool
	// InstalledBuild ist deren Build-Nummer.
	InstalledBuild int
	// Package: MD Reader ist als .deb-Paket installiert.
	Package bool
	// Prefs sind die gespeicherten Antworten.
	Prefs Prefs
}

// Decide entscheidet, ob eine Frage gestellt wird.
//
// Regeln in dieser Reihenfolge:
//  1. Paketinstallation vorhanden: nichts anbieten (das Paket verwaltet sich selbst).
//  2. Läuft schon aus ~/.local/bin oder /usr/bin: nichts anbieten.
//  3. Nutzerinstallation ist älter: Update anbieten – auch bei "Nie", denn der
//     Nutzer hat bereits installiert.
//  4. Nutzerinstallation ist gleich neu oder neuer: nichts (keine Herabstufung).
//  5. Antwort "Nie": nichts. Sonst (leer oder "Später"): Installation anbieten.
//
// Pfade werden vor dem Vergleich mit filepath.Clean bereinigt.
func Decide(c StartContext) Offer {
	if c.Package {
		return OfferNothing
	}
	exe := filepath.Clean(c.ExePath)
	if exe == filepath.Clean(c.Layout.BinPath) || exe == systemBinPath {
		return OfferNothing
	}
	if c.Installed {
		if c.InstalledBuild < c.CurrentBuild {
			return OfferUpdate
		}
		return OfferNothing
	}
	if c.Prefs.InstallPrompt == PromptNever {
		return OfferNothing
	}
	return OfferInstall
}
