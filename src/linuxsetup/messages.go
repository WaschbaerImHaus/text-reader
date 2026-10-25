// Texte des Starters: Spracherkennung, Nachrichten und Button-Beschriftungen.
//
// Es gibt nur Deutsch und Englisch. Deutsch gilt, wenn die erste gesetzte
// Variable aus LC_ALL, LC_MESSAGES, LANG mit "de" beginnt, sonst Englisch.
// Argumente (Bibliotheksnamen, Befehle, Pfade) werden per strings.Replace
// eingesetzt, nie als Formatzeichenkette – ein "%" in einem Pfad bleibt so,
// wie es ist.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 4: Texte und Nachfragen)
package linuxsetup

import "strings"

// Lang ist die Sprache der Texte ("de" oder "en").
type Lang string

const (
	// LangDE: Deutsch.
	LangDE Lang = "de"
	// LangEN: Englisch (Standard und Rückfall für alle anderen Sprachen).
	LangEN Lang = "en"
)

// DetectLang ermittelt die Sprache aus der Umgebung.
//
// Maßgeblich ist die erste nicht leere Variable in der Reihenfolge LC_ALL,
// LC_MESSAGES, LANG (wie bei gettext). Beginnt sie mit "de" (z. B.
// "de_DE.UTF-8", "de_AT", "de"), gilt Deutsch, sonst Englisch.
//
// @param env Funktion zum Lesen von Umgebungsvariablen (os.Getenv oder Test-Map).
// @return LangDE oder LangEN.
func DetectLang(env func(string) string) Lang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := env(key)
		if value == "" {
			continue
		}
		// Nur die erste gesetzte Variable zählt (LC_ALL=C schlägt LANG=de_DE).
		if strings.HasPrefix(value, "de") {
			return LangDE
		}
		return LangEN
	}
	return LangEN
}

// MessageID bezeichnet eine Nachricht des Starters.
type MessageID int

const (
	// MsgWebKitMissing: Frage, ob WebKitGTK per apt installiert werden soll.
	// Argument 1: fehlende Bibliotheken, mit ", " verbunden
	// (ohne Argument wird libwebkit2gtk-4.1.so.0 genannt).
	// Knöpfe: "install", "cancel".
	MsgWebKitMissing MessageID = iota
	// MsgWebKitHint: Hinweis, WebKitGTK selbst zu installieren.
	// Argument 1: Befehl aus InstallHint.
	MsgWebKitHint
	// MsgInstallFailed: apt-get install ist gescheitert; Angebot, die
	// Paketlisten zu aktualisieren. Keine Argumente. Knöpfe: "update", "cancel".
	MsgInstallFailed
	// MsgOfferInstall: Angebot, MD Reader ins Menü aufzunehmen. Keine Argumente.
	// Knöpfe: "yes", "later", "never".
	MsgOfferInstall
	// MsgOfferUpdate: Diese Datei ist neuer als die installierte; Angebot zu
	// ersetzen. Keine Argumente. Knöpfe: "replace", "notnow".
	MsgOfferUpdate
	// MsgInstalled: Bestätigung nach der Installation. Keine Argumente.
	MsgInstalled
	// MsgPackageInstalled: --install, obwohl das .deb-Paket installiert ist.
	// Keine Argumente.
	MsgPackageInstalled
	// MsgUninstalled: Bestätigung nach --uninstall.
	// Argument 1: behaltene Pfade, mit "\n" verbunden.
	MsgUninstalled

	// messageCount ist die Anzahl der Nachrichten (nur für Tests und Prüfungen).
	messageCount
)

// argPlaceholder wird im Text durch das erste Argument ersetzt.
const argPlaceholder = "{1}"

// message ist ein Paar aus Titel und Text; der Text darf argPlaceholder enthalten.
type message struct {
	title, body string
}

// messagesDE enthält alle deutschen Texte.
var messagesDE = map[MessageID]message{
	MsgWebKitMissing: {
		"WebKitGTK ist nicht installiert",
		"MD Reader braucht WebKitGTK 4.1, um Dokumente anzuzeigen. Es fehlt: {1}.\n\n" +
			"Soll das Paket " + WebKitPackage + " jetzt installiert werden? " +
			"Dafür wird das Administrator-Passwort abgefragt.",
	},
	MsgWebKitHint: {
		"WebKitGTK ist nicht installiert",
		"MD Reader braucht WebKitGTK 4.1. Bitte installieren Sie es selbst, zum Beispiel mit:\n\n" +
			"{1}\n\nStarten Sie MD Reader danach erneut.",
	},
	MsgInstallFailed: {
		"Installation fehlgeschlagen",
		"Das Paket " + WebKitPackage + " konnte nicht installiert werden. " +
			"Oft sind die Paketlisten veraltet. Sollen sie aktualisiert und die Installation erneut versucht werden?",
	},
	MsgOfferInstall: {
		"MD Reader ins Menü aufnehmen?",
		"MD Reader kann sich für Sie ins Startmenü (Büro) eintragen, mit Icon und als Programm " +
			"für Markdown, EPUB, PDF und weitere Formate. Administratorrechte sind dafür nicht nötig.",
	},
	MsgOfferUpdate: {
		"Neuere Version von MD Reader",
		"Diese Datei ist neuer als das installierte MD Reader. Soll die Installation im Menü ersetzt werden?",
	},
	MsgInstalled: {
		"MD Reader installiert",
		"MD Reader ist jetzt im Menü unter Büro.",
	},
	MsgPackageInstalled: {
		"MD Reader ist als Paket installiert",
		"MD Reader ist bereits als Paket (.deb) installiert. Bitte verwenden Sie die Paketinstallation. " +
			"Mit --install --force wird trotzdem zusätzlich in Ihr Home-Verzeichnis installiert.",
	},
	MsgUninstalled: {
		"MD Reader entfernt",
		"Programm, Menüeintrag und Icons wurden entfernt. Ihre Einstellungen und Daten bleiben erhalten:\n{1}",
	},
}

// messagesEN enthält alle englischen Texte.
var messagesEN = map[MessageID]message{
	MsgWebKitMissing: {
		"WebKitGTK is not installed",
		"MD Reader needs WebKitGTK 4.1 to display documents. Missing: {1}.\n\n" +
			"Install the package " + WebKitPackage + " now? " +
			"You will be asked for the administrator password.",
	},
	MsgWebKitHint: {
		"WebKitGTK is not installed",
		"MD Reader needs WebKitGTK 4.1. Please install it yourself, for example with:\n\n" +
			"{1}\n\nThen start MD Reader again.",
	},
	MsgInstallFailed: {
		"Installation failed",
		"The package " + WebKitPackage + " could not be installed. " +
			"The package lists are often out of date. Update them and try the installation again?",
	},
	MsgOfferInstall: {
		"Add MD Reader to the menu?",
		"MD Reader can add itself to your application menu (Office), with an icon and as a program " +
			"for Markdown, EPUB, PDF and other formats. No administrator rights are needed.",
	},
	MsgOfferUpdate: {
		"Newer version of MD Reader",
		"This file is newer than the installed MD Reader. Replace the installation in the menu?",
	},
	MsgInstalled: {
		"MD Reader installed",
		"MD Reader is now in the menu under Office.",
	},
	MsgPackageInstalled: {
		"MD Reader is installed as a package",
		"MD Reader is already installed as a package (.deb). Please use the package installation. " +
			"--install --force installs into your home directory anyway.",
	},
	MsgUninstalled: {
		"MD Reader removed",
		"The program, menu entry and icons were removed. Your settings and data are kept:\n{1}",
	},
}

// defaultArgs sind die Ersatzwerte, wenn der Aufrufer kein Argument übergibt.
var defaultArgs = map[MessageID]string{
	MsgWebKitMissing: "libwebkit2gtk-4.1.so.0",
	MsgWebKitHint:    "sudo apt install " + WebKitPackage,
	MsgUninstalled:   "-",
}

// Text liefert Titel und Text einer Nachricht in der gewünschten Sprache.
//
// Argumente je Nachricht: siehe Kommentar an der jeweiligen MessageID
// (MsgWebKitMissing: Bibliotheken mit ", " verbunden; MsgWebKitHint: Befehl
// aus InstallHint; MsgUninstalled: behaltene Pfade mit "\n" verbunden; alle
// anderen ohne Argument). Fehlt ein nötiges Argument, wird ein sinnvoller
// Ersatzwert eingesetzt; überzählige Argumente werden ignoriert.
// Unbekannte Sprachen fallen auf Englisch zurück; eine unbekannte ID liefert
// den Titel "MD Reader" und einen leeren Text.
//
// @param lang Sprache (LangDE oder LangEN).
// @param id Nachricht.
// @param args Optionales Argument (nur das erste wird verwendet).
// @return title Titel der Nachricht, body Nachrichtentext.
func Text(lang Lang, id MessageID, args ...string) (title, body string) {
	table := messagesEN
	if lang == LangDE {
		table = messagesDE
	}
	m, ok := table[id]
	if !ok {
		return "MD Reader", ""
	}
	arg := defaultArgs[id]
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		arg = args[0]
	}
	// Einfache Ersetzung statt fmt.Sprintf: Argumente werden nie interpretiert.
	return m.title, strings.Replace(m.body, argPlaceholder, arg, 1)
}

// buttonLabelsDE enthält die deutschen Button-Beschriftungen je Schlüssel.
var buttonLabelsDE = map[string]string{
	"install": "Jetzt installieren",
	"cancel":  "Abbrechen",
	"update":  "Paketlisten aktualisieren und erneut versuchen",
	"yes":     "Ja",
	"later":   "Später",
	"never":   "Nie",
	"replace": "Ersetzen",
	"notnow":  "Nicht jetzt",
}

// buttonLabelsEN enthält die englischen Button-Beschriftungen je Schlüssel.
var buttonLabelsEN = map[string]string{
	"install": "Install now",
	"cancel":  "Cancel",
	"update":  "Update package lists and try again",
	"yes":     "Yes",
	"later":   "Later",
	"never":   "Never",
	"replace": "Replace",
	"notnow":  "Not now",
}

// ButtonLabel liefert die Beschriftung eines Knopfes.
//
// Bekannte Schlüssel: "install", "cancel", "update", "yes", "later", "never",
// "replace", "notnow". Ein unbekannter Schlüssel wird unverändert
// zurückgegeben (nie leer); unbekannte Sprachen fallen auf Englisch zurück.
//
// @param lang Sprache.
// @param key Button-Schlüssel.
// @return Beschriftung.
func ButtonLabel(lang Lang, key string) string {
	table := buttonLabelsEN
	if lang == LangDE {
		table = buttonLabelsDE
	}
	if label, ok := table[key]; ok {
		return label
	}
	return key
}

// Buttons baut aus Schlüsseln die Knöpfe einer Frage mit Beschriftung.
//
// @param lang Sprache.
// @param keys Button-Schlüssel in der gewünschten Reihenfolge.
// @return Knöpfe für Asker.Ask.
func Buttons(lang Lang, keys ...string) []Button {
	buttons := make([]Button, 0, len(keys))
	for _, key := range keys {
		buttons = append(buttons, Button{Key: key, Label: ButtonLabel(lang, key)})
	}
	return buttons
}
