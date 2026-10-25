// Nachfragen über die Dialogprogramme kdialog (KDE) und zenity (GTK).
//
// Beide werden als eigenständige Prozesse mit fester Argumentliste gestartet
// (exec.Command, nie über eine Shell). Titel und Text sind je genau ein
// Argument; zenity bekommt zusätzlich --no-markup, damit "<" und "&" in
// Pfaden nicht als Pango-Auszeichnung gelesen werden.
//
// Zuordnung der Antworten:
//   - kdialog: Exit 0 → 1. Knopf, 1 → 2. Knopf, 2 → 3. Knopf (nur bei drei Knöpfen).
//     Achtung: kdialog meldet auch Escape/Schließen bei --yesnocancel als 2.
//   - zenity: Exit 0 → 1. Knopf; Exit 1 mit Ausgabe gleich der Extra-Beschriftung
//     → 3. Knopf; Exit 1 ohne Ausgabe (auch Schließen) → 2. Knopf.
//   - Alles andere (Zeitablauf, unbekannte Codes) → "" (keine Antwort).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 4: Texte und Nachfragen)
package linuxsetup

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// maxDialogButtons ist die höchste Zahl an Knöpfen, die kdialog und zenity
// zuverlässig darstellen.
const maxDialogButtons = 3

// dialogAsker fragt über kdialog oder zenity.
type dialogAsker struct {
	// tool ist "kdialog" oder "zenity".
	tool string
	// path ist der per lookPath gefundene Pfad des Programms.
	path string
	// run startet das Programm (im Betrieb runDialog, im Test ein Fake).
	// Rückgabe: Exit-Code (-1 bei Zeitablauf), Standardausgabe, Startfehler.
	run func(name string, args []string, timeout time.Duration) (exitCode int, out string, err error)
}

// newDialogAsker liefert einen dialogAsker für das angegebene Programm.
//
// @param tool "kdialog" oder "zenity".
// @param path Pfad des Programms.
// @return dialogAsker mit echtem Programmaufruf.
func newDialogAsker(tool, path string) *dialogAsker {
	return &dialogAsker{tool: tool, path: path, run: runDialog}
}

// Ask zeigt die Frage im Dialogprogramm und ordnet die Antwort einem Knopf zu.
//
// @param title Titel des Fensters.
// @param body Fragetext.
// @param buttons 1 bis 3 Knöpfe.
// @param timeout Höchste Wartezeit (0 = ohne Zeitlimit); danach wird das
// Programm beendet und "" geliefert.
// @return key Gewählter Schlüssel oder ""; err bei ungültigen Knöpfen oder
// wenn das Programm nicht startet.
func (a *dialogAsker) Ask(title, body string, buttons []Button, timeout time.Duration) (string, error) {
	if len(buttons) == 0 || len(buttons) > maxDialogButtons {
		return "", errors.New("dialogAsker: 1 bis 3 Knöpfe erlaubt")
	}
	code, out, err := a.run(a.path, a.questionArgs(title, body, buttons), timeout)
	if err != nil {
		return "", err
	}
	return a.mapAnswer(code, out, buttons), nil
}

// Inform zeigt einen Hinweis mit OK-Knopf und wartet, bis er geschlossen wird.
//
// @param title Titel des Fensters.
// @param body Hinweistext.
func (a *dialogAsker) Inform(title, body string) {
	var args []string
	if a.tool == "kdialog" {
		args = []string{"--title", title, "--msgbox", body}
	} else {
		args = []string{"--info", "--no-markup", "--title", title, "--text", body}
	}
	_, _, _ = a.run(a.path, args, 0)
}

// questionArgs baut die feste Argumentliste der Frage.
func (a *dialogAsker) questionArgs(title, body string, buttons []Button) []string {
	if a.tool == "kdialog" {
		switch len(buttons) {
		case 1:
			return []string{"--title", title, "--msgbox", body}
		case 2:
			return []string{"--title", title, "--yesno", body,
				"--yes-label", buttons[0].Label, "--no-label", buttons[1].Label}
		default:
			return []string{"--title", title, "--yesnocancel", body,
				"--yes-label", buttons[0].Label, "--no-label", buttons[1].Label,
				"--cancel-label", buttons[2].Label}
		}
	}
	// zenity
	if len(buttons) == 1 {
		return []string{"--info", "--no-markup", "--title", title, "--text", body,
			"--ok-label", buttons[0].Label}
	}
	args := []string{"--question", "--no-markup", "--title", title, "--text", body,
		"--ok-label", buttons[0].Label, "--cancel-label", buttons[1].Label}
	if len(buttons) == 3 {
		args = append(args, "--extra-button", buttons[2].Label)
	}
	return args
}

// mapAnswer ordnet Exit-Code und Ausgabe einem Button-Schlüssel zu.
func (a *dialogAsker) mapAnswer(code int, out string, buttons []Button) string {
	if a.tool == "kdialog" {
		// Exit-Code ist direkt der Index des Knopfes.
		if code >= 0 && code < len(buttons) {
			return buttons[code].Key
		}
		return ""
	}
	// zenity
	switch code {
	case 0:
		return buttons[0].Key
	case 1:
		if len(buttons) < 2 {
			// Nur ein Knopf: Exit 1 heißt „Fenster geschlossen“.
			return ""
		}
		answer := strings.TrimSpace(out)
		if answer == "" {
			return buttons[1].Key
		}
		if len(buttons) == 3 && answer == buttons[2].Label {
			return buttons[2].Key
		}
		return ""
	}
	return ""
}

// runDialog startet ein Programm ohne Shell, liest die Standardausgabe und
// beendet es nach timeout (0 = ohne Zeitlimit). Fehlerausgaben des Programms
// (z. B. GTK-Warnungen) werden verworfen.
//
// @param name Pfad des Programms.
// @param args Argumente (jedes genau ein Argument, keine Shell-Auswertung).
// @param timeout Zeitlimit.
// @return exitCode Exit-Code bzw. -1 bei Zeitablauf; out Standardausgabe;
// err nur, wenn das Programm nicht gestartet werden konnte.
func runDialog(name string, args []string, timeout time.Duration) (int, string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// Nach dem Beenden nicht auf Kindprozesse warten, die die Ausgabe offen halten.
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if ctx.Err() != nil {
		// Zeitablauf: gilt als „keine Antwort“, nicht als Fehler.
		return -1, stdout.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), nil
	}
	if err != nil {
		return -1, "", err
	}
	return 0, stdout.String(), nil
}
