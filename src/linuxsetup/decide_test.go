// Tests für decide.go – wann der Starter Installieren oder Aktualisieren anbietet.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 3: Installieren, Entfernen, Entscheiden)
package linuxsetup

import "testing"

// TestDecide ist tabellengesteuert; Layout.BinPath ist /h/.local/bin/md-reader.
func TestDecide(t *testing.T) {
	layout := Layout{BinPath: "/h/.local/bin/md-reader"}
	tests := []struct {
		name string
		ctx  StartContext
		want Offer
	}{
		{"frisch, nichts installiert", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout}, OfferInstall},
		{"Antwort never", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Prefs: Prefs{InstallPrompt: "never"}}, OfferNothing},
		{"Antwort later fragt erneut", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Prefs: Prefs{InstallPrompt: "later"}}, OfferInstall},
		{"Paket installiert", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Package: true}, OfferNothing},
		{"Paket und ältere Nutzerinstallation", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Package: true, Installed: true, InstalledBuild: 40}, OfferNothing},
		{"läuft aus ~/.local/bin", StartContext{ExePath: "/h/.local/bin/md-reader", CurrentBuild: 47, Layout: layout}, OfferNothing},
		{"läuft aus ~/.local/bin (unbereinigter Pfad)", StartContext{ExePath: "/h/.local/bin/../bin/./md-reader", CurrentBuild: 47, Layout: layout}, OfferNothing},
		{"läuft aus /usr/bin", StartContext{ExePath: "/usr/bin/md-reader", CurrentBuild: 47, Layout: layout}, OfferNothing},
		{"installiert 46, aktuell 47", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Installed: true, InstalledBuild: 46}, OfferUpdate},
		{"Update auch bei never", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Installed: true, InstalledBuild: 46, Prefs: Prefs{InstallPrompt: "never"}}, OfferUpdate},
		{"installiert 48, aktuell 47: keine Herabstufung", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Installed: true, InstalledBuild: 48}, OfferNothing},
		{"installiert 47, aktuell 47", StartContext{ExePath: "/tmp/dl/md-reader", CurrentBuild: 47, Layout: layout, Installed: true, InstalledBuild: 47}, OfferNothing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Decide(tc.ctx); got != tc.want {
				t.Errorf("Decide = %v, erwartet %v", got, tc.want)
			}
		})
	}
}
