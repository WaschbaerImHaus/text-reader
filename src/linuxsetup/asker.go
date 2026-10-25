// Nachfragen an den Nutzer: gemeinsame Schnittstelle, Auswahl der Methode
// (Rückfallkette) und die einfachste Methode, die Textausgabe auf stderr.
//
// Rückfallkette von NewAsker:
//  1. Desktop-Benachrichtigung über D-Bus (asker_dbus.go) – nur, wenn ein
//     Session-Bus erreichbar ist und der Dienst Aktionsknöpfe ("actions") kann.
//  2. kdialog (asker_dialog.go)
//  3. zenity (asker_dialog.go)
//  4. Text auf stderr (diese Datei) – ohne Antwortmöglichkeit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Korrekturrunde: Knopf-Konvention, Inform-Zeitlimit)
package linuxsetup

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// probeTimeout begrenzt die Abfrage des Benachrichtigungsdienstes beim Start,
// damit ein hängender Dienst den Programmstart nicht aufhält.
const probeTimeout = 2 * time.Second

// Button ist ein Knopf einer Frage.
type Button struct {
	// Key ist der Schlüssel, den Ask bei Auswahl zurückgibt (z. B. "install").
	Key string
	// Label ist die angezeigte Beschriftung (siehe ButtonLabel).
	Label string
}

// Asker stellt dem Nutzer Fragen oder zeigt ihm Hinweise.
//
// Konvention für die Knöpfe: Der LETZTE Knopf der Liste ist der harmlose
// (z. B. „Später“, „Abbrechen“, „Nicht jetzt“). Schließen, Escape oder
// Zeitablauf liefern bei jeder Umsetzung entweder genau diesen letzten Key
// oder "" – nie einen anderen Knopf. Aufrufer müssen "" und den letzten Key
// gleich behandeln.
type Asker interface {
	// Ask zeigt eine Frage mit 1 bis 3 Knöpfen und wartet höchstens timeout
	// (0 = ohne Zeitlimit; Aufrufer sollten immer ein Zeitlimit setzen).
	// Rückgabe ist der Key des gewählten Knopfes; ein leerer Key bedeutet
	// „keine Antwort“ (Zeitablauf, Nachricht geschlossen, keine
	// Antwortmöglichkeit). kdialog/zenity liefern beim Schließen des Fensters
	// den letzten Key. err nur bei technischen Fehlern.
	Ask(title, body string, buttons []Button, timeout time.Duration) (key string, err error)
	// Inform zeigt einen Hinweis; Fehler werden ignoriert. Bei kdialog/zenity
	// blockiert Inform, bis der Nutzer OK drückt, höchstens jedoch 120 s
	// (informTimeout). Die D-Bus-Benachrichtigung und stderr kehren sofort zurück.
	Inform(title, body string)
}

// NewAsker wählt die beste verfügbare Methode für Nachfragen.
//
// Reihenfolge: D-Bus-Benachrichtigung (Session-Bus erreichbar und Fähigkeit
// "actions"), kdialog, zenity, stderr. Die Bus-Adresse stammt aus
// DBUS_SESSION_BUS_ADDRESS bzw. $XDG_RUNTIME_DIR/bus; ein Bus wird nie
// selbst gestartet (kein dbus-launch). Der D-Bus-Weg erhält die nächste
// Methode der Kette als Rückfall für spätere Verbindungsfehler.
//
// @param lookPath Suchfunktion für Programme (exec.LookPath oder Test-Fake).
// @return Asker; nie nil.
func NewAsker(lookPath func(string) (string, error)) Asker {
	fallback := newDialogOrStderrAsker(lookPath)

	// D-Bus nur, wenn der Dienst Knöpfe darstellen kann (sonst käme nie eine Antwort).
	address := sessionBusAddress(os.Getenv)
	if address != "" {
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		caps, ok := probeNotifications(ctx, address)
		cancel()
		if ok && hasCapability(caps, "actions") {
			return &dbusAsker{
				address:    address,
				bodyMarkup: hasCapability(caps, "body-markup"),
				fallback:   fallback,
			}
		}
	}
	return fallback
}

// newDialogOrStderrAsker liefert kdialog, zenity oder – wenn keins vorhanden
// ist – die Ausgabe auf stderr.
func newDialogOrStderrAsker(lookPath func(string) (string, error)) Asker {
	for _, tool := range []string{"kdialog", "zenity"} {
		if path, err := lookPath(tool); err == nil && path != "" {
			return newDialogAsker(tool, path)
		}
	}
	return newStderrAsker()
}

// hasCapability prüft, ob eine Fähigkeit in der Liste steht.
func hasCapability(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

// stderrAsker gibt Titel und Text nur aus und liefert nie eine Antwort.
// Letzte Stufe der Rückfallkette (z. B. Start im Terminal ohne Desktop).
type stderrAsker struct {
	// w ist das Ausgabeziel (im Betrieb os.Stderr, im Test ein Puffer).
	w io.Writer
}

// newStderrAsker liefert einen stderrAsker mit Ausgabe nach os.Stderr.
func newStderrAsker() *stderrAsker {
	return &stderrAsker{w: os.Stderr}
}

// Ask gibt Titel und Text aus und liefert "" (keine Antwort möglich).
//
// @param title Titel.
// @param body Text.
// @param buttons Wird ignoriert.
// @param timeout Wird ignoriert.
// @return Immer "" und nil.
func (a *stderrAsker) Ask(title, body string, buttons []Button, timeout time.Duration) (string, error) {
	a.write(title, body)
	return "", nil
}

// Inform gibt Titel und Text aus.
//
// @param title Titel.
// @param body Text.
func (a *stderrAsker) Inform(title, body string) {
	a.write(title, body)
}

// write schreibt "Titel", Leerzeile, Text; Schreibfehler werden ignoriert.
func (a *stderrAsker) write(title, body string) {
	_, _ = fmt.Fprintf(a.w, "%s\n\n%s\n", title, body)
}
