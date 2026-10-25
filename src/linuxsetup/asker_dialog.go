// Nachfragen über die Dialogprogramme kdialog (KDE) und zenity (GTK).
//
// Beide werden als eigenständige Prozesse mit fester Argumentliste gestartet
// (exec.Command, nie über eine Shell). Titel und Text sind je genau ein
// Argument; zenity bekommt zusätzlich --no-markup, damit "<" und "&" in
// Pfaden nicht als Pango-Auszeichnung gelesen werden.
//
// Konvention (siehe Asker): Der LETZTE Knopf ist der harmlose. Schließen bzw.
// Escape liefert bei beiden Programmen genau diesen Knopf.
//
// Zuordnung der Antworten:
//   - kdialog, 2 Knöpfe (--yesno): yes = erster, no = letzter.
//     Exit 0 → erster, 1 → letzter (auch Schließen).
//   - kdialog, 3 Knöpfe (--yesnocancel): yes = erster, no = mittlerer,
//     cancel = letzter. Exit 0 → erster, 1 → mittlerer, 2 → letzter (auch Schließen).
//   - zenity (--question): ok = erster, cancel = letzter, bei 3 Knöpfen
//     --extra-button = mittlerer. Exit 0 → erster; Exit 1 mit Ausgabe gleich
//     der Extra-Beschriftung → mittlerer; Exit 1 ohne Ausgabe (Abbrechen oder
//     Schließen) → letzter.
//   - Alles andere (Zeitablauf, unbekannte Codes) → "" (keine Antwort).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Korrekturrunde: letzter Knopf ist der harmlose, Inform mit Zeitlimit)
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

// informTimeout begrenzt, wie lange ein Hinweis (Inform) höchstens offen
// bleibt; danach wird das Dialogprogramm beendet.
const informTimeout = 120 * time.Second

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

// Inform zeigt einen Hinweis mit OK-Knopf und wartet, bis er geschlossen
// wird – höchstens informTimeout (120 s), dann wird das Programm beendet.
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
	_, _, _ = a.run(a.path, args, informTimeout)
}

// questionArgs baut die feste Argumentliste der Frage (letzter Knopf = harmlos,
// er liegt auf der Taste, die auch Schließen/Escape auslöst).
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
	last := buttons[len(buttons)-1]
	args := []string{"--question", "--no-markup", "--title", title, "--text", body,
		"--ok-label", buttons[0].Label, "--cancel-label", last.Label}
	if len(buttons) == 3 {
		// Der mittlere Knopf wird zum Extra-Knopf (gibt seine Beschriftung aus).
		args = append(args, "--extra-button", buttons[1].Label)
	}
	return args
}

// mapAnswer ordnet Exit-Code und Ausgabe einem Button-Schlüssel zu.
func (a *dialogAsker) mapAnswer(code int, out string, buttons []Button) string {
	if a.tool == "kdialog" {
		// Exit-Code ist direkt der Index des Knopfes; der höchste Code
		// (no bei --yesno, cancel bei --yesnocancel) ist auch der des Schließens.
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
			// Abbrechen-Knopf oder Fenster geschlossen → letzter (harmloser) Knopf.
			return buttons[len(buttons)-1].Key
		}
		if len(buttons) == 3 && answer == buttons[1].Label {
			// Extra-Knopf → mittlerer Knopf.
			return buttons[1].Key
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
