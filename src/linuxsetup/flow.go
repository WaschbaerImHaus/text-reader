// Ablaufsteuerung des Starters: verbindet Abhängigkeitsprüfung, Nachfragen,
// Installation und Antwortspeicher zu zwei Abläufen.
//
//  1. HandleMissingLibraries: WebKitGTK fehlt → per apt nachinstallieren
//     (mit höchstens einem Wiederholungsversuch nach "apt-get update") oder
//     den Befehl für die eigene Installation nennen.
//  2. HandleInstallOffer: MD Reader ins Menü aufnehmen bzw. eine ältere
//     Nutzerinstallation ersetzen.
//
// Für alle Fragen gilt die Asker-Konvention: Der letzte Knopf ist der harmlose;
// "" (Zeitablauf, geschlossen) und jeder unbekannte Schlüssel gelten wie er.
// Nur ein ausdrücklich gewählter Schlüssel löst eine Aktion aus.
//
// Sicherheit: Es laufen nur die festen Befehle AptInstallCommand und
// AptUpdateCommand; Bibliotheksnamen landen nur in Texten, nie in Befehlen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 5: Ablaufsteuerung)
package linuxsetup

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// askTimeout ist das Zeitlimit jeder Frage; danach gilt der harmlose Knopf.
const askTimeout = 120 * time.Second

// Exit-Codes von pkexec, bei denen nie weiter gefragt wird.
const (
	// exitAuthAborted: Passwortabfrage abgebrochen oder abgelehnt.
	exitAuthAborted = 126
	// exitNotFound: pkexec (oder das Zielprogramm) nicht gefunden.
	exitNotFound = 127
)

// missingLabels ist die Überschrift der Bibliotheksliste auf stderr.
var missingLabels = map[Lang]string{
	LangDE: "Fehlende Bibliotheken",
	LangEN: "Missing libraries",
}

// errNoRun meldet eine Env ohne Run-Funktion (es kann nichts installiert werden).
var errNoRun = errors.New("keine Run-Funktion gesetzt")

// Env bündelt alles, was die Abläufe von außen brauchen. Im Betrieb setzt der
// Starter echte Umsetzungen ein, in Tests Fakes.
type Env struct {
	// Asker stellt Fragen und zeigt Hinweise. nil: Ausgabe nur auf Stderr,
	// Fragen bleiben unbeantwortet (gilt als harmloser Knopf).
	Asker Asker
	// Lang ist die Sprache aller Texte.
	Lang Lang
	// Run führt eine feste Argumentliste ohne Shell aus und liefert den
	// Exit-Code. err != nil mit exitCode <= 0 bedeutet „ließ sich nicht
	// starten“ und wird wie Exit 127 behandelt. nil: nichts wird ausgeführt.
	Run func(argv []string) (exitCode int, err error)
	// LookPath sucht Programme (exec.LookPath oder Test-Fake). nil: exec.LookPath.
	LookPath func(string) (string, error)
	// Stderr erhält den Hinweis für Starts im Terminal. nil: verworfen.
	Stderr io.Writer
}

// stderr liefert das Ausgabeziel für Terminal-Hinweise (nie nil).
func (e Env) stderr() io.Writer {
	if e.Stderr == nil {
		return io.Discard
	}
	return e.Stderr
}

// asker liefert den Asker; ohne Asker eine reine Textausgabe auf Stderr.
func (e Env) asker() Asker {
	if e.Asker == nil {
		return &stderrAsker{w: e.stderr()}
	}
	return e.Asker
}

// lookPath liefert die Suchfunktion für Programme (nie nil).
func (e Env) lookPath() func(string) (string, error) {
	if e.LookPath == nil {
		return exec.LookPath
	}
	return e.LookPath
}

// run führt einen Befehl über Env.Run aus; ohne Run gilt er als nicht startbar.
func (e Env) run(argv []string) (int, error) {
	if e.Run == nil {
		return -1, errNoRun
	}
	return e.Run(argv)
}

// ask stellt eine Frage mit den gegebenen Knöpfen und dem festen Zeitlimit.
// Ein technischer Fehler wird zurückgegeben; der Schlüssel ist dann "".
func (e Env) ask(id MessageID, buttons []Button) (string, error) {
	title, body := Text(e.Lang, id)
	key, err := e.asker().Ask(title, body, buttons, askTimeout)
	if err != nil {
		return "", err
	}
	return key, nil
}

// inform zeigt eine Nachricht ohne Argument bzw. mit einem Argument.
func (e Env) inform(id MessageID, args ...string) {
	title, body := Text(e.Lang, id, args...)
	e.asker().Inform(title, body)
}

// runOutcome ist das Ergebnis eines apt-Aufrufs.
type runOutcome int

const (
	// runOK: Exit 0.
	runOK runOutcome = iota
	// runAborted: Passwortabfrage abgebrochen, pkexec fehlt oder der Befehl
	// ließ sich nicht starten – nie weiter fragen.
	runAborted
	// runFailed: anderer Fehler (z. B. apt-get Exit 100) – Wiederholung möglich.
	runFailed
)

// classify ordnet Exit-Code und Fehler eines Aufrufs ein.
func classify(code int, err error) runOutcome {
	switch {
	case code == 0 && err == nil:
		return runOK
	case code == exitAuthAborted || code == exitNotFound:
		return runAborted
	case err != nil && code <= 0:
		// Ließ sich gar nicht starten (z. B. pkexec nicht gefunden).
		return runAborted
	}
	return runFailed
}

// HandleMissingLibraries kümmert sich um fehlende Bibliotheken (WebKitGTK).
//
// Ablauf:
//  1. Immer zuerst: Titel, fehlende Bibliotheken und InstallHint auf Stderr.
//  2. Ohne apt-get oder ohne pkexec: nur der Hinweis MsgWebKitHint, keine
//     Frage, false.
//  3. Mit apt-get: Frage MsgWebKitMissing mit "install" / "cancel" (120 s).
//     Bei "install" läuft AptInstallCommand. Exit 0 → true. Exit 126/127
//     oder nicht startbar → Hinweis, false, keine weitere Frage.
//  4. Anderer Fehler → Frage MsgInstallFailed mit "update" / "cancel".
//     Bei "update": AptUpdateCommand (bricht nur bei 126/127 ab), dann noch
//     einmal AptInstallCommand. Exit 0 → true, sonst Hinweis und false.
//     Höchstens dieser eine Wiederholungsversuch.
//
// "cancel", "" (Zeitablauf) und unbekannte Schlüssel ergeben false ohne
// Hinweis (er steht schon auf Stderr); ein technischer Fehler der Frage
// ergibt den Hinweis.
//
// @param e Umgebung (Asker, Sprache, Run, LookPath, Stderr).
// @param missing Fehlende Bibliotheken aus ParseMissingLibraries.
// @return true, wenn WebKitGTK jetzt installiert ist und der Start erneut versucht werden soll.
func HandleMissingLibraries(e Env, missing []string) (retry bool) {
	pm := DetectPackageManager(e.lookPath())
	hint := InstallHint(pm)
	libs := strings.Join(missing, ", ")
	writeTerminalHint(e, libs, hint)

	// Ohne apt oder ohne pkexec (grafische Passwortabfrage) kein automatisches
	// Nachinstallieren, nur der passende Befehl – eine Frage wäre zwecklos.
	if pm != PMApt || !hasProgram(e.lookPath(), "pkexec") {
		e.inform(MsgWebKitHint, hint)
		return false
	}

	title, body := Text(e.Lang, MsgWebKitMissing, libs)
	key, err := e.asker().Ask(title, body, Buttons(e.Lang, "install", "cancel"), askTimeout)
	if err != nil {
		// Die Frage kam womöglich nie an – wenigstens den Befehl zeigen.
		e.inform(MsgWebKitHint, hint)
		return false
	}
	if key != "install" {
		return false
	}

	switch classify(e.run(AptInstallCommand())) {
	case runOK:
		return true
	case runAborted:
		e.inform(MsgWebKitHint, hint)
		return false
	}

	// Erster Versuch gescheitert: einmal Paketlisten aktualisieren anbieten.
	key, err = e.ask(MsgInstallFailed, Buttons(e.Lang, "update", "cancel"))
	if err != nil {
		e.inform(MsgWebKitHint, hint)
		return false
	}
	if key != "update" {
		return false
	}
	// Ein teilweise gescheitertes update (z. B. eine Quelle offline) ist kein
	// Grund aufzugeben; nur Abbruch der Passwortabfrage beendet den Ablauf.
	if classify(e.run(AptUpdateCommand())) == runAborted {
		e.inform(MsgWebKitHint, hint)
		return false
	}
	if classify(e.run(AptInstallCommand())) == runOK {
		return true
	}
	e.inform(MsgWebKitHint, hint)
	return false
}

// hasProgram meldet, ob lookPath das Programm findet.
func hasProgram(lookPath func(string) (string, error), name string) bool {
	path, err := lookPath(name)
	return err == nil && path != ""
}

// writeTerminalHint schreibt Titel, fehlende Bibliotheken und den
// Installationsbefehl auf Stderr (sichtbar bei Starts im Terminal, auch wenn
// kein Desktop-Dialog erscheint). Schreibfehler werden ignoriert.
func writeTerminalHint(e Env, libs, hint string) {
	title, _ := Text(e.Lang, MsgWebKitMissing)
	if strings.TrimSpace(libs) == "" {
		libs = defaultArgs[MsgWebKitMissing]
	}
	label, ok := missingLabels[e.Lang]
	if !ok {
		label = missingLabels[LangEN]
	}
	_, hintBody := Text(e.Lang, MsgWebKitHint, hint)
	_, _ = fmt.Fprintf(e.stderr(), "%s\n%s: %s\n\n%s\n", title, label, libs, hintBody)
}

// HandleInstallOffer bietet abhängig von Decide die Installation ins Menü
// oder das Ersetzen einer älteren Nutzerinstallation an.
//
//   - OfferNothing: keine Frage, nichts wird geschrieben.
//   - OfferInstall: Frage MsgOfferInstall mit "yes" / "never" / "later" (120 s).
//     "yes" → Install und Bestätigung MsgInstalled (Antwortspeicher bleibt
//     unberührt). "never" → Prefs "never". "later", "", unbekannt oder ein
//     technischer Fehler der Frage → Prefs "later".
//   - OfferUpdate: Frage MsgOfferUpdate mit "replace" ("Ersetzen") /
//     "later" ("Nicht jetzt"). "replace" → Install und MsgInstalled.
//     Der Antwortspeicher wird hier nie geschrieben.
//
// @param e Umgebung (Asker, Sprache).
// @param c Lage beim Start (enthält Layout und aktuelle Build-Nummer).
// @param selfPath Pfad der laufenden Datei (Quelle der Kopie).
// @param icons Icon-Inhalte für Install.
// @return Fehler von Install, SavePrefs oder der Frage; der Starter
// protokolliert ihn und startet die Oberfläche trotzdem.
func HandleInstallOffer(e Env, c StartContext, selfPath string, icons Icons) error {
	switch Decide(c) {
	case OfferInstall:
		key, askErr := e.ask(MsgOfferInstall, Buttons(e.Lang, "yes", "never", "later"))
		switch key {
		case "yes":
			return installAndConfirm(e, c, selfPath, icons)
		case "never":
			return savePrompt(c, PromptNever)
		}
		// Harmloser Fall: beim nächsten Start erneut fragen.
		return errors.Join(askErr, savePrompt(c, PromptLater))
	case OfferUpdate:
		buttons := []Button{
			{Key: "replace", Label: ButtonLabel(e.Lang, "replace")},
			{Key: "later", Label: ButtonLabel(e.Lang, "notnow")},
		}
		key, askErr := e.ask(MsgOfferUpdate, buttons)
		if key == "replace" {
			return installAndConfirm(e, c, selfPath, icons)
		}
		return askErr
	}
	return nil
}

// installAndConfirm installiert und bestätigt den Erfolg mit MsgInstalled.
func installAndConfirm(e Env, c StartContext, selfPath string, icons Icons) error {
	if err := Install(c.Layout, selfPath, c.CurrentBuild, icons); err != nil {
		return fmt.Errorf("Installation gescheitert: %w", err)
	}
	e.inform(MsgInstalled)
	return nil
}

// savePrompt speichert die Antwort auf die Installationsfrage.
func savePrompt(c StartContext, prompt string) error {
	if err := SavePrefs(c.Layout.PrefsPath, Prefs{InstallPrompt: prompt}); err != nil {
		return fmt.Errorf("Antwort nicht gespeichert: %w", err)
	}
	return nil
}
