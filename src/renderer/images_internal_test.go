// Interne Tests für die Pfadprüfung beim Einbetten lokaler Bilder.
//
// Security-Review zu Build 42: Die Erkennung von Netzwerkpfaden darf nicht
// auf einer Sperrliste beruhen (z. B. "\??\UNC\server\…" unter Windows).
// isReadableImagePath arbeitet daher als Erlaubnisliste und ist
// plattformunabhängig, damit die Windows-Fälle auch unter Linux testbar sind.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-22 (Gerätenamen-Umgehungen)
package renderer

import "testing"

// TestIsReadableImagePath_Allowed prüft erlaubte lokale Pfade.
func TestIsReadableImagePath_Allowed(t *testing.T) {
	for _, src := range []string{
		"bild.png",
		"img/bild.png",
		"../bilder/bild.png",
		`img\bild.png`,
		"/home/user/bild.png",
		`C:\Users\x\bild.png`,
		"C:/Users/x/bild.png",
		"d:/bild.png",
		`\Users\x\bild.png`,
	} {
		if !isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = false, erwartet true", src)
		}
	}
}

// TestIsReadableImagePath_NetworkAndDevicePathsRejected prüft, dass alle
// Netzwerk-, NT- und Gerätepfade abgelehnt werden (fail closed).
func TestIsReadableImagePath_NetworkAndDevicePathsRejected(t *testing.T) {
	for _, src := range []string{
		`\\server\share\x.png`,
		"//server/share/x.png",
		`/\server\share\x.png`,
		`\/server/share/x.png`,
		`\??\UNC\server\share\x.png`,
		`/??/UNC/server/share/x.png`,
		`\\?\UNC\server\share\x.png`,
		`\\?\C:\x.png`,
		`\\.\pipe\x.png`,
		`\\.\C:\x.png`,
		"",
		"C:",
		"C:bild.png",
		"file.png:stream",
		"bild.png:ads.png",
	} {
		if isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = true, erwartet false", src)
		}
	}
}

// TestIsReadableImagePath_DeviceNamesRejected prüft reservierte Windows-
// Gerätenamen (Lesen von COM1 o. ä. könnte die App blockieren).
func TestIsReadableImagePath_DeviceNamesRejected(t *testing.T) {
	for _, src := range []string{"CON.png", "nul.png", "img/COM1.png", `C:\x\lpt9.jpg`, "AUX.gif", "prn.svg", "com0.png"} {
		if isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = true, erwartet false", src)
		}
	}
	// Gegenprobe: Namen, die nur ähnlich sind, bleiben erlaubt
	for _, src := range []string{"console.png", "com10.png", "nulle.png"} {
		if !isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = false, erwartet true", src)
		}
	}
}

// TestIsReadableImagePath_DeviceNameEvasionsRejected prüft (Security-Review zu
// Build 43), dass Gerätenamen nicht über Windows-Namensnormalisierung umgangen
// werden: Windows entfernt Leerzeichen/Punkte am Segmentende ("COM1 .png" →
// COM1). Dazu die selteneren Gerätenamen und hochgestellte Ziffern.
func TestIsReadableImagePath_DeviceNameEvasionsRejected(t *testing.T) {
	for _, src := range []string{
		"COM1 .png",
		"nul  .gif",
		"CONIN$.png",
		"conout$.jpg",
		"CLOCK$.png",
		"COM¹.png",
		"lpt³.png",
		"CON/bild.png",
		`img\AUX\bild.png`,
		"img./bild.png",
		"img /bild.png",
		"bild.png.",
		"bild.png ",
	} {
		if isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = true, erwartet false", src)
		}
	}
	// Gegenprobe: Leerzeichen und Punkte INNERHALB von Namen bleiben erlaubt
	for _, src := range []string{"mein bild.png", "img/v1.2/bild.png", "../a b/c.d.png"} {
		if !isReadableImagePath(src) {
			t.Errorf("isReadableImagePath(%q) = false, erwartet true", src)
		}
	}
}
