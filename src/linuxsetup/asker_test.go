// Tests für asker.go, asker_dialog.go und asker_dbus.go – Auswahl der
// Nachfragemethode (Rückfallkette), kdialog/zenity-Argumente und die
// D-Bus-Benachrichtigung gegen einen privaten Session-Bus.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Korrekturrunde 2: Knopfprüfung, kdialog-Maskierung, Zeitwerte)
package linuxsetup

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// fakeLookPath liefert eine lookPath-Funktion, die nur die genannten Programme kennt.
func fakeLookPath(programs ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		for _, p := range programs {
			if p == name {
				return "/usr/bin/" + name, nil
			}
		}
		return "", exec.ErrNotFound
	}
}

// noSessionBus sperrt jeden Session-Bus für den Test (keine Adresse,
// leeres Laufzeitverzeichnis ohne "bus"-Socket).
func noSessionBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
}

// TestNewAsker_Fallback prüft die Rückfallkette ohne D-Bus: kdialog, zenity, stderr.
func TestNewAsker_Fallback(t *testing.T) {
	noSessionBus(t)

	a := NewAsker(fakeLookPath())
	if _, ok := a.(*stderrAsker); !ok {
		t.Errorf("ohne Programme: %T, erwartet *stderrAsker", a)
	}

	a = NewAsker(fakeLookPath("zenity"))
	d, ok := a.(*dialogAsker)
	if !ok || d.tool != "zenity" || d.path != "/usr/bin/zenity" {
		t.Errorf("nur zenity: %#v, erwartet dialogAsker{tool:\"zenity\"}", a)
	}

	a = NewAsker(fakeLookPath("kdialog", "zenity"))
	d, ok = a.(*dialogAsker)
	if !ok || d.tool != "kdialog" {
		t.Errorf("kdialog und zenity: %#v, erwartet kdialog", a)
	}

	// notify-send allein reicht nicht (keine verlässliche Antwort) → stderr.
	a = NewAsker(fakeLookPath("notify-send"))
	if _, ok := a.(*stderrAsker); !ok {
		t.Errorf("nur notify-send: %T, erwartet *stderrAsker", a)
	}
}

// TestSessionBusAddress prüft die Ermittlung der Bus-Adresse ohne dbus-launch.
func TestSessionBusAddress(t *testing.T) {
	dir := t.TempDir()
	if got := sessionBusAddress(envFrom(map[string]string{"DBUS_SESSION_BUS_ADDRESS": "unix:path=/x"})); got != "unix:path=/x" {
		t.Errorf("aus Env: %q", got)
	}
	// "autolaunch:" würde einen neuen Bus starten – nicht verwenden.
	if got := sessionBusAddress(envFrom(map[string]string{"DBUS_SESSION_BUS_ADDRESS": "autolaunch:", "XDG_RUNTIME_DIR": dir})); got != "" {
		t.Errorf("autolaunch: %q, erwartet leer", got)
	}
	if got := sessionBusAddress(envFrom(map[string]string{"XDG_RUNTIME_DIR": dir})); got != "" {
		t.Errorf("ohne bus-Socket: %q, erwartet leer", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "bus"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	want := "unix:path=" + dbus.EscapeBusAddressValue(filepath.Join(dir, "bus"))
	if got := sessionBusAddress(envFrom(map[string]string{"XDG_RUNTIME_DIR": dir})); got != want {
		t.Errorf("mit bus-Socket: %q, erwartet %q", got, want)
	}
	// Relatives XDG_RUNTIME_DIR wird ignoriert.
	if got := sessionBusAddress(envFrom(map[string]string{"XDG_RUNTIME_DIR": "relative"})); strings.Contains(got, "relative") {
		t.Errorf("relatives XDG_RUNTIME_DIR verwendet: %q", got)
	}
}

// dialogCall merkt sich einen Aufruf des Dialogprogramms.
type dialogCall struct {
	name    string
	args    []string
	timeout time.Duration
}

// fakeRun liefert eine run-Funktion mit fester Antwort und protokolliert die Aufrufe.
func fakeRun(calls *[]dialogCall, code int, out string, err error) func(string, []string, time.Duration) (int, string, error) {
	return func(name string, args []string, timeout time.Duration) (int, string, error) {
		*calls = append(*calls, dialogCall{name, append([]string(nil), args...), timeout})
		return code, out, err
	}
}

// threeButtons sind die Knöpfe des Installationsangebots. Nach der
// Konvention steht der harmlose Knopf („Später“) zuletzt.
var threeButtons = []Button{{"yes", "Ja"}, {"never", "Nie"}, {"later", "Später"}}

// TestDialogAsker_ArgsAndMapping prüft Argumente und Zuordnung von Exit-Code
// bzw. Ausgabe zum Button-Schlüssel für kdialog und zenity. Konvention: Der
// letzte Knopf ist der harmlose; Schließen/Escape muss genau ihn liefern.
func TestDialogAsker_ArgsAndMapping(t *testing.T) {
	evil := "Text $(rm -rf ~) `id` ; | & \"x\""
	two := []Button{{"install", "Jetzt installieren"}, {"cancel", "Abbrechen"}}

	// kdialog, drei Knöpfe: --yesnocancel, yes=erster, no=mittlerer, cancel=letzter.
	var calls []dialogCall
	a := &dialogAsker{tool: "kdialog", path: "/usr/bin/kdialog", run: fakeRun(&calls, 0, "", nil)}
	key, err := a.Ask("Titel", evil, threeButtons, time.Minute)
	if err != nil || key != "yes" {
		t.Errorf("kdialog exit 0: (%q, %v), erwartet yes", key, err)
	}
	// kdialog liest den Text als Qt-Rich-Text: maskiert und in <qt> gefasst.
	evilKDE := "<qt>Text $(rm -rf ~) `id` ; | &amp; \"x\"</qt>"
	wantArgs := []string{"--title", "Titel", "--yesnocancel", evilKDE,
		"--yes-label", "Ja", "--no-label", "Nie", "--cancel-label", "Später"}
	if len(calls) != 1 || calls[0].name != "/usr/bin/kdialog" || !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("kdialog Aufruf = %#v,\nerwartet %q %#v", calls, "/usr/bin/kdialog", wantArgs)
	}
	if len(calls) == 1 && calls[0].timeout != time.Minute {
		t.Errorf("kdialog Zeitlimit = %v, erwartet 1m", calls[0].timeout)
	}
	// Exit 2 ist auch Escape/Schließen → letzter (harmloser) Knopf.
	for code, want := range map[int]string{0: "yes", 1: "never", 2: "later", 3: "", -1: "", 255: ""} {
		a.run = fakeRun(&calls, code, "", nil)
		if key, err := a.Ask("Titel", "Text", threeButtons, time.Minute); err != nil || key != want {
			t.Errorf("kdialog drei Knöpfe exit %d: (%q, %v), erwartet %q", code, key, err, want)
		}
	}

	// kdialog, zwei Knöpfe: --yesno, yes=erster, no=letzter (auch Schließen = Exit 1).
	calls = nil
	a.run = fakeRun(&calls, 1, "", nil)
	if key, _ := a.Ask("T", "B", two, time.Minute); key != "cancel" {
		t.Errorf("kdialog zwei Knöpfe exit 1 (geschlossen): %q, erwartet cancel", key)
	}
	wantArgs = []string{"--title", "T", "--yesno", "<qt>B</qt>", "--yes-label", "Jetzt installieren", "--no-label", "Abbrechen"}
	if len(calls) == 0 || !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("kdialog --yesno Argumente = %#v,\nerwartet %#v", calls, wantArgs)
	}
	for code, want := range map[int]string{0: "install", 1: "cancel", 2: "", -1: ""} {
		a.run = fakeRun(&calls, code, "", nil)
		if key, _ := a.Ask("T", "B", two, time.Minute); key != want {
			t.Errorf("kdialog zwei Knöpfe exit %d: %q, erwartet %q", code, key, want)
		}
	}

	// zenity, drei Knöpfe: ok=erster, cancel=letzter, extra=mittlerer.
	calls = nil
	z := &dialogAsker{tool: "zenity", path: "/usr/bin/zenity", run: fakeRun(&calls, 0, "", nil)}
	if key, err := z.Ask("Titel", evil, threeButtons, time.Minute); err != nil || key != "yes" {
		t.Errorf("zenity exit 0: (%q, %v)", key, err)
	}
	wantArgs = []string{"--question", "--no-markup", "--title", "Titel", "--text", evil,
		"--ok-label", "Ja", "--cancel-label", "Später", "--extra-button", "Nie"}
	if len(calls) != 1 || calls[0].name != "/usr/bin/zenity" || !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("zenity Aufruf = %#v,\nerwartet %#v", calls, wantArgs)
	}
	cases := []struct {
		code int
		out  string
		want string
	}{
		{0, "", "yes"},
		{1, "", "later"}, // Abbrechen-Knopf oder Fenster geschlossen → letzter Knopf
		{1, "\n", "later"},
		{1, "Nie\n", "never"}, // Extra-Knopf → mittlerer Knopf
		{1, "Nie", "never"},
		{1, "Später\n", ""}, // Ausgabe passt zu keinem Extra-Knopf
		{1, "Anderes\n", ""},
		{5, "", ""},  // zenity-Zeitablauf
		{-1, "", ""}, // abgebrochen (Zeitlimit des Aufrufers)
	}
	for _, c := range cases {
		z.run = fakeRun(&calls, c.code, c.out, nil)
		if key, err := z.Ask("T", "B", threeButtons, time.Minute); err != nil || key != c.want {
			t.Errorf("zenity drei Knöpfe exit %d, Ausgabe %q: (%q, %v), erwartet %q", c.code, c.out, key, err, c.want)
		}
	}

	// zenity, zwei Knöpfe: ok=erster, cancel=letzter, ohne --extra-button.
	calls = nil
	z.run = fakeRun(&calls, 0, "", nil)
	if key, _ := z.Ask("T", "B", two, time.Minute); key != "install" {
		t.Errorf("zenity zwei Knöpfe: %q", key)
	}
	wantArgs = []string{"--question", "--no-markup", "--title", "T", "--text", "B",
		"--ok-label", "Jetzt installieren", "--cancel-label", "Abbrechen"}
	if len(calls) == 0 || !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("zenity zwei Knöpfe Argumente = %#v,\nerwartet %#v", calls, wantArgs)
	}
	for _, c := range []struct {
		code int
		out  string
		want string
	}{{0, "", "install"}, {1, "", "cancel"}, {1, "Abbrechen\n", ""}, {5, "", ""}} {
		z.run = fakeRun(&calls, c.code, c.out, nil)
		if key, _ := z.Ask("T", "B", two, time.Minute); key != c.want {
			t.Errorf("zenity zwei Knöpfe exit %d, Ausgabe %q: %q, erwartet %q", c.code, c.out, key, c.want)
		}
	}
}

// TestKDialogRichText prüft die Maskierung für kdialog (Qt::AutoText):
// &, <, > werden maskiert, Zeilenumbrüche werden zu <br>, der Titel bleibt
// unverändert (Fenstertitel sind immer reiner Text).
func TestKDialogRichText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "<qt></qt>"},
		{"a & b", "<qt>a &amp; b</qt>"},
		{"<b>fett</b>", "<qt>&lt;b&gt;fett&lt;/b&gt;</qt>"},
		{"Zeile1\nZeile2", "<qt>Zeile1<br>Zeile2</qt>"},
		{"/home/a&b/<x>\n/home/c", "<qt>/home/a&amp;b/&lt;x&gt;<br>/home/c</qt>"},
		{"&amp;", "<qt>&amp;amp;</qt>"},
		{"a\r\nb", "<qt>a<br>b</qt>"},
	}
	for _, c := range cases {
		if got := kdialogText(c.in); got != c.want {
			t.Errorf("kdialogText(%q) = %q, erwartet %q", c.in, got, c.want)
		}
	}
	var calls []dialogCall
	k := &dialogAsker{tool: "kdialog", path: "/usr/bin/kdialog", run: fakeRun(&calls, 0, "", nil)}
	if _, err := k.Ask("A & <B>", "x < y\nz & w", []Button{{"install", "Jetzt installieren"}, {"cancel", "Abbrechen"}}, time.Minute); err != nil {
		t.Fatal(err)
	}
	k.Inform("Info & mehr", "Pfad:\n/home/a&b")
	want := [][]string{
		{"--title", "A & <B>", "--yesno", "<qt>x &lt; y<br>z &amp; w</qt>", "--yes-label", "Jetzt installieren", "--no-label", "Abbrechen"},
		{"--title", "Info & mehr", "--msgbox", "<qt>Pfad:<br>/home/a&amp;b</qt>"},
	}
	if len(calls) != 2 {
		t.Fatalf("kdialog Aufrufe = %#v, erwartet 2", calls)
	}
	if !reflect.DeepEqual([][]string{calls[0].args, calls[1].args}, want) {
		t.Errorf("kdialog Argumente = %#v,\nerwartet %#v", calls, want)
	}
	// zenity bleibt bei --no-markup und bekommt den Text unverändert.
	calls = nil
	z := &dialogAsker{tool: "zenity", path: "/usr/bin/zenity", run: fakeRun(&calls, 0, "", nil)}
	_, _ = z.Ask("A & <B>", "x < y\nz", []Button{{"ok", "OK"}, {"cancel", "Abbrechen"}}, time.Minute)
	if len(calls) != 1 || calls[0].args[5] != "x < y\nz" || calls[0].args[3] != "A & <B>" {
		t.Errorf("zenity Argumente = %#v", calls)
	}
}

// TestDialogAsker_Errors prüft Fehlerfälle: zu viele oder keine Knöpfe,
// Startfehler des Programms.
func TestDialogAsker_Errors(t *testing.T) {
	var calls []dialogCall
	a := &dialogAsker{tool: "kdialog", path: "/usr/bin/kdialog", run: fakeRun(&calls, 0, "", nil)}
	four := append(append([]Button(nil), threeButtons...), Button{"x", "X"})
	if _, err := a.Ask("T", "B", four, time.Minute); err == nil {
		t.Error("vier Knöpfe: kein Fehler")
	}
	if _, err := a.Ask("T", "B", nil, time.Minute); err == nil {
		t.Error("keine Knöpfe: kein Fehler")
	}
	if len(calls) != 0 {
		t.Errorf("bei ungültigen Knöpfen wurde das Programm gestartet: %#v", calls)
	}
	startErr := errors.New("start fehlgeschlagen")
	a.run = fakeRun(&calls, -1, "", startErr)
	if key, err := a.Ask("T", "B", threeButtons, time.Minute); key != "" || !errors.Is(err, startErr) {
		t.Errorf("Startfehler: (%q, %v)", key, err)
	}
}

// TestDialogAsker_OneButtonAndInform prüft einen einzelnen Knopf und Inform.
func TestDialogAsker_OneButtonAndInform(t *testing.T) {
	var calls []dialogCall
	one := []Button{{"ok", "OK"}}
	k := &dialogAsker{tool: "kdialog", path: "/usr/bin/kdialog", run: fakeRun(&calls, 0, "", nil)}
	if key, _ := k.Ask("T", "B", one, time.Minute); key != "ok" {
		t.Errorf("kdialog ein Knopf: %q", key)
	}
	k.Inform("Info", "Text")
	z := &dialogAsker{tool: "zenity", path: "/usr/bin/zenity", run: fakeRun(&calls, 1, "", nil)}
	if key, _ := z.Ask("T", "B", one, time.Minute); key != "" {
		t.Errorf("zenity ein Knopf, geschlossen: %q", key)
	}
	z.Inform("Info", "Text")
	want := [][]string{
		{"--title", "T", "--msgbox", "<qt>B</qt>"},
		{"--title", "Info", "--msgbox", "<qt>Text</qt>"},
		{"--info", "--no-markup", "--title", "T", "--text", "B", "--ok-label", "OK"},
		{"--info", "--no-markup", "--title", "Info", "--text", "Text"},
	}
	if len(calls) != len(want) {
		t.Fatalf("Aufrufe = %#v", calls)
	}
	for i := range want {
		if !reflect.DeepEqual(calls[i].args, want[i]) {
			t.Errorf("Aufruf %d = %#v, erwartet %#v", i, calls[i].args, want[i])
		}
	}
	// Inform blockiert höchstens informTimeout (120 s), nie ohne Grenze.
	if informTimeout != 120*time.Second {
		t.Errorf("informTimeout = %v, erwartet 120s", informTimeout)
	}
	for _, i := range []int{1, 3} {
		if calls[i].timeout != informTimeout {
			t.Errorf("Inform-Aufruf %d: Zeitlimit %v, erwartet %v", i, calls[i].timeout, informTimeout)
		}
	}
}

// TestRunDialog prüft den echten Programmaufruf ohne Shell: Argumente kommen
// einzeln an, Exit-Code und Ausgabe werden geliefert, das Zeitlimit greift.
func TestRunDialog(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("kein sh vorhanden")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")
	// Das Skript gibt sein erstes Argument aus und endet mit Exit 3.
	script := filepath.Join(dir, "tool")
	if err := os.WriteFile(script, []byte("#!"+sh+"\nprintf '%s' \"$1\"\nexit 3\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	arg := "$(touch " + marker + ")"
	code, out, err := runDialog(script, []string{arg}, 10*time.Second)
	if err != nil || code != 3 || out != arg {
		t.Errorf("runDialog = (%d, %q, %v), erwartet (3, %q, nil)", code, out, err, arg)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("Argument wurde von einer Shell ausgewertet")
	}

	// Zeitlimit: Programm wird beendet, Exit-Code -1, kein Fehler.
	slow := filepath.Join(dir, "slow")
	if err := os.WriteFile(slow, []byte("#!"+sh+"\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	code, _, err = runDialog(slow, nil, 200*time.Millisecond)
	if err != nil || code != -1 {
		t.Errorf("Zeitlimit: (%d, %v), erwartet (-1, nil)", code, err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("Zeitlimit wirkte nicht: %v", time.Since(start))
	}

	// Startfehler: Programm fehlt.
	if _, _, err := runDialog(filepath.Join(dir, "missing"), nil, time.Second); err == nil {
		t.Error("fehlendes Programm: kein Fehler")
	}
}

// TestStderrAsker prüft die reine Textausgabe ohne Antwort.
func TestStderrAsker(t *testing.T) {
	var buf bytes.Buffer
	a := &stderrAsker{w: &buf}
	key, err := a.Ask("Titel", "Text", threeButtons, time.Minute)
	if key != "" || err != nil {
		t.Errorf("Ask = (%q, %v), erwartet leer", key, err)
	}
	if !strings.Contains(buf.String(), "Titel") || !strings.Contains(buf.String(), "Text") {
		t.Errorf("Ausgabe = %q", buf.String())
	}
	buf.Reset()
	a.Inform("Info", "Zeile1\nZeile2")
	if !strings.Contains(buf.String(), "Info") || !strings.Contains(buf.String(), "Zeile1\nZeile2") {
		t.Errorf("Inform-Ausgabe = %q", buf.String())
	}
	// Standard-Ziel ist os.Stderr.
	if w := newStderrAsker().w; w != os.Stderr {
		t.Errorf("Standard-Writer = %v, erwartet os.Stderr", w)
	}
}

// --- D-Bus ---------------------------------------------------------------

// fakeNotifyServer spielt org.freedesktop.Notifications nach.
type fakeNotifyServer struct {
	conn *dbus.Conn
	// impostor ist eine zweite Verbindung, die Antworten fälschen will.
	impostor *dbus.Conn

	mu sync.Mutex
	// caps ist die Antwort auf GetCapabilities.
	caps []string
	// reply bestimmt, welches Signal nach Notify gesendet wird:
	// "action:<key>", "closed", "impostor:<key>" (erst gefälschte Antwort
	// "install" von fremder Verbindung, dann die echte) oder "" (keins).
	reply string
	// capsDelay verzögert GetCapabilities (simuliert einen hängenden Dienst).
	capsDelay time.Duration
	// delay verzögert das Signal (simuliert einen Nutzer, der sich Zeit lässt).
	delay time.Duration
	// notifies und closed protokollieren die Aufrufe.
	notifies []fakeNotify
	closed   []uint32
}

// fakeNotify ist ein protokollierter Notify-Aufruf.
type fakeNotify struct {
	appName, appIcon, summary, body string
	actions                         []string
	expire                          int32
}

// GetCapabilities liefert die eingestellten Fähigkeiten.
func (s *fakeNotifyServer) GetCapabilities() ([]string, *dbus.Error) {
	s.mu.Lock()
	caps, delay := s.caps, s.capsDelay
	s.mu.Unlock()
	time.Sleep(delay)
	return caps, nil
}

// Notify protokolliert den Aufruf, liefert ID 7 und sendet je nach Einstellung
// zuerst ein Signal mit fremder ID (muss ignoriert werden) und dann das passende.
func (s *fakeNotifyServer) Notify(sender dbus.Sender, appName string, replaces uint32, appIcon, summary, body string,
	actions []string, hints map[string]dbus.Variant, expire int32) (uint32, *dbus.Error) {
	s.mu.Lock()
	s.notifies = append(s.notifies, fakeNotify{appName, appIcon, summary, body, actions, expire})
	reply, delay := s.reply, s.delay
	s.mu.Unlock()

	const path = dbus.ObjectPath("/org/freedesktop/Notifications")
	go func() {
		time.Sleep(delay)
		switch {
		case strings.HasPrefix(reply, "action:"):
			_ = s.conn.Emit(path, "org.freedesktop.Notifications.ActionInvoked", uint32(99), "cancel")
			_ = s.conn.Emit(path, "org.freedesktop.Notifications.ActionInvoked", uint32(7), strings.TrimPrefix(reply, "action:"))
		case strings.HasPrefix(reply, "impostor:"):
			// Gezieltes Signal einer fremden Verbindung an den Fragenden – muss ignoriert werden.
			fake := &dbus.Message{
				Type: dbus.TypeSignal,
				Headers: map[dbus.HeaderField]dbus.Variant{
					dbus.FieldPath:        dbus.MakeVariant(path),
					dbus.FieldInterface:   dbus.MakeVariant("org.freedesktop.Notifications"),
					dbus.FieldMember:      dbus.MakeVariant("ActionInvoked"),
					dbus.FieldDestination: dbus.MakeVariant(string(sender)),
					dbus.FieldSignature:   dbus.MakeVariant(dbus.SignatureOf(uint32(0), "")),
				},
				Body: []any{uint32(7), "install"},
			}
			s.impostor.Send(fake, nil)
			time.Sleep(150 * time.Millisecond)
			_ = s.conn.Emit(path, "org.freedesktop.Notifications.ActionInvoked", uint32(7), strings.TrimPrefix(reply, "impostor:"))
		case reply == "closed":
			_ = s.conn.Emit(path, "org.freedesktop.Notifications.NotificationClosed", uint32(99), uint32(2))
			_ = s.conn.Emit(path, "org.freedesktop.Notifications.NotificationClosed", uint32(7), uint32(2))
		}
	}()
	return 7, nil
}

// CloseNotification protokolliert das Schließen.
func (s *fakeNotifyServer) CloseNotification(id uint32) *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = append(s.closed, id)
	return nil
}

// GetServerInformation gehört zur Schnittstelle (wird nicht ausgewertet).
func (s *fakeNotifyServer) GetServerInformation() (string, string, string, string, *dbus.Error) {
	return "fake", "md-reader-test", "1", "1.2", nil
}

// set stellt Fähigkeiten und Antwort ein und löscht das Protokoll.
func (s *fakeNotifyServer) set(caps []string, reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.caps, s.reply, s.delay, s.capsDelay, s.notifies, s.closed = caps, reply, 0, 0, nil, nil
}

// snapshot liefert eine Kopie des Protokolls.
func (s *fakeNotifyServer) snapshot() ([]fakeNotify, []uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]fakeNotify(nil), s.notifies...), append([]uint32(nil), s.closed...)
}

// startPrivateBus startet einen eigenen dbus-daemon unter t.TempDir() und
// liefert dessen Adresse. Der Daemon wird am Testende beendet.
func startPrivateBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon nicht vorhanden")
	}
	dir := t.TempDir()
	// Minimale Konfiguration: nur dieser Socket, keine Dienst-Aktivierung.
	conf := `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
	"http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
	<type>session</type>
	<listen>unix:path=` + filepath.Join(dir, "bus") + `</listen>
	<auth>EXTERNAL</auth>
	<policy context="default">
		<allow send_destination="*" eavesdrop="true"/>
		<allow eavesdrop="true"/>
		<allow own="*"/>
	</policy>
</busconfig>
`
	confPath := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(confPath, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(daemon, "--config-file="+confPath, "--print-address", "--nofork")
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("dbus-daemon: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lines <- strings.TrimSpace(line)
		// Restliche Ausgabe verwerfen, damit der Daemon nie blockiert.
		_, _ = io.Copy(io.Discard, stdout)
	}()
	select {
	case addr := <-lines:
		if addr == "" {
			t.Fatal("dbus-daemon lieferte keine Adresse")
		}
		return addr
	case <-time.After(10 * time.Second):
		t.Fatal("dbus-daemon startete nicht")
	}
	return ""
}

// startFakeNotifyServer verbindet sich mit dem Bus, übernimmt den Namen
// org.freedesktop.Notifications und exportiert den Fake-Server.
func startFakeNotifyServer(t *testing.T, addr string) *fakeNotifyServer {
	t.Helper()
	conn, err := dbus.Connect(addr)
	if err != nil {
		t.Fatalf("Verbindung zum privaten Bus: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	impostor, err := dbus.Connect(addr)
	if err != nil {
		t.Fatalf("zweite Verbindung: %v", err)
	}
	t.Cleanup(func() { _ = impostor.Close() })
	s := &fakeNotifyServer{conn: conn, impostor: impostor}
	if err := conn.Export(s, "/org/freedesktop/Notifications", "org.freedesktop.Notifications"); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("RequestName: %v, %v", reply, err)
	}
	return s
}

// TestDBusAsker prüft Notify, Antwort per ActionInvoked, Schließen und
// Zeitablauf gegen einen Fake-Benachrichtigungsdienst auf einem privaten Bus.
func TestDBusAsker(t *testing.T) {
	addr := startPrivateBus(t)
	server := startFakeNotifyServer(t, addr)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	buttons := []Button{{"install", "Jetzt installieren"}, {"cancel", "Abbrechen"}}

	t.Run("ActionInvoked", func(t *testing.T) {
		server.set([]string{"body", "actions"}, "action:install")
		a := NewAsker(fakeLookPath("kdialog", "zenity"))
		d, ok := a.(*dbusAsker)
		if !ok {
			t.Fatalf("NewAsker = %T, erwartet *dbusAsker", a)
		}
		if _, ok := d.fallback.(*dialogAsker); !ok {
			t.Errorf("Rückfall = %T, erwartet *dialogAsker", d.fallback)
		}
		key, err := a.Ask("WebKitGTK fehlt", "Text <b>&</b>", buttons, 5*time.Second)
		if err != nil || key != "install" {
			t.Fatalf("Ask = (%q, %v), erwartet install", key, err)
		}
		notifies, closed := server.snapshot()
		if len(notifies) != 1 {
			t.Fatalf("Notify-Aufrufe = %d", len(notifies))
		}
		n := notifies[0]
		if n.appName != "MD Reader" || n.appIcon != "md-reader" || n.summary != "WebKitGTK fehlt" {
			t.Errorf("Notify = %#v", n)
		}
		// Ohne "body-markup" bleibt der Text unverändert.
		if n.body != "Text <b>&</b>" {
			t.Errorf("Text = %q", n.body)
		}
		if want := []string{"install", "Jetzt installieren", "cancel", "Abbrechen"}; !reflect.DeepEqual(n.actions, want) {
			t.Errorf("Aktionen = %#v, erwartet %#v", n.actions, want)
		}
		if n.expire != 0 {
			t.Errorf("expire_timeout = %d, erwartet 0 (nie von selbst ausblenden)", n.expire)
		}
		if len(closed) != 0 {
			t.Errorf("nach Antwort CloseNotification gerufen: %v", closed)
		}
	})

	t.Run("BodyMarkupEscaped", func(t *testing.T) {
		server.set([]string{"actions", "body-markup"}, "action:cancel")
		key, err := NewAsker(fakeLookPath()).Ask("T", "a < b & c > d", buttons, 5*time.Second)
		if err != nil || key != "cancel" {
			t.Fatalf("Ask = (%q, %v)", key, err)
		}
		notifies, _ := server.snapshot()
		if len(notifies) != 1 || notifies[0].body != "a &lt; b &amp; c &gt; d" {
			t.Errorf("Text nicht maskiert: %#v", notifies)
		}
	})

	t.Run("UnknownActionIsNoAnswer", func(t *testing.T) {
		server.set([]string{"actions"}, "action:default")
		key, err := NewAsker(fakeLookPath()).Ask("T", "B", buttons, 5*time.Second)
		if err != nil || key != "" {
			t.Errorf("Ask = (%q, %v), erwartet leer", key, err)
		}
	})

	t.Run("NotificationClosed", func(t *testing.T) {
		server.set([]string{"actions"}, "closed")
		key, err := NewAsker(fakeLookPath()).Ask("T", "B", buttons, 5*time.Second)
		if err != nil || key != "" {
			t.Errorf("Ask = (%q, %v), erwartet leer", key, err)
		}
		if _, closed := server.snapshot(); len(closed) != 0 {
			t.Errorf("bereits geschlossene Nachricht erneut geschlossen: %v", closed)
		}
	})

	t.Run("NoActionsCapability", func(t *testing.T) {
		server.set([]string{"body", "body-markup"}, "")
		if a := NewAsker(fakeLookPath("zenity")); func() bool { _, ok := a.(*dbusAsker); return ok }() {
			t.Errorf("ohne actions wurde D-Bus gewählt")
		} else if d, ok := a.(*dialogAsker); !ok || d.tool != "zenity" {
			t.Errorf("ohne actions: %#v, erwartet zenity", a)
		}
		if _, ok := NewAsker(fakeLookPath()).(*stderrAsker); !ok {
			t.Error("ohne actions und ohne Programme: kein stderrAsker")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		server.set([]string{"actions"}, "")
		a := NewAsker(fakeLookPath())
		start := time.Now()
		key, err := a.Ask("T", "B", buttons, 200*time.Millisecond)
		if err != nil || key != "" {
			t.Errorf("Ask = (%q, %v), erwartet leer", key, err)
		}
		if d := time.Since(start); d < 150*time.Millisecond || d > 5*time.Second {
			t.Errorf("Dauer %v passt nicht zum Zeitlimit 200 ms", d)
		}
		if _, closed := server.snapshot(); !reflect.DeepEqual(closed, []uint32{7}) {
			t.Errorf("CloseNotification-Aufrufe = %v, erwartet [7]", closed)
		}
	})

	t.Run("ButtonCount", func(t *testing.T) {
		// Wie dialogAsker: 0 oder mehr als 3 Knöpfe sind ein Fehler, ohne Notify.
		server.set([]string{"actions"}, "action:install")
		a := NewAsker(fakeLookPath())
		four := []Button{{"a", "A"}, {"b", "B"}, {"c", "C"}, {"d", "D"}}
		for _, bs := range [][]Button{nil, four} {
			if key, err := a.Ask("T", "B", bs, time.Second); key != "" || !errors.Is(err, errButtonCount) {
				t.Errorf("%d Knöpfe: (%q, %v), erwartet errButtonCount", len(bs), key, err)
			}
		}
		if notifies, _ := server.snapshot(); len(notifies) != 0 {
			t.Errorf("bei ungültigen Knöpfen wurde Notify gerufen: %#v", notifies)
		}
		d := &dialogAsker{tool: "zenity", path: "/usr/bin/zenity", run: runDialog}
		if _, err := d.Ask("T", "B", four, time.Second); !errors.Is(err, errButtonCount) {
			t.Errorf("dialogAsker: %v, erwartet errButtonCount", err)
		}
	})

	t.Run("ImpostorSignalIgnored", func(t *testing.T) {
		// Nur der Eigentümer von org.freedesktop.Notifications darf antworten.
		server.set([]string{"actions"}, "impostor:cancel")
		key, err := NewAsker(fakeLookPath()).Ask("T", "B", buttons, 5*time.Second)
		if err != nil || key != "cancel" {
			t.Errorf("Ask = (%q, %v), erwartet cancel (gefälschtes install ignoriert)", key, err)
		}
	})

	t.Run("ProbeHangingService", func(t *testing.T) {
		// Ein hängender Dienst darf den Start nicht aufhalten.
		server.set([]string{"actions"}, "")
		server.mu.Lock()
		server.capsDelay = 3 * time.Second
		server.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		start := time.Now()
		caps, ok := probeNotifications(ctx, addr)
		if ok || caps != nil {
			t.Errorf("probeNotifications = (%v, %v), erwartet (nil, false)", caps, ok)
		}
		if d := time.Since(start); d >= time.Second {
			t.Errorf("probeNotifications dauerte %v, erwartet < 1 s", d)
		}
	})

	t.Run("LateAnswerAfterCallTimeout", func(t *testing.T) {
		// Die Verbindung darf nicht mit dem Zeitlimit der Einzelaufrufe enden:
		// Antwortet der Nutzer nach callTimeout, muss sie noch ankommen.
		old := callTimeout
		callTimeout = 250 * time.Millisecond
		t.Cleanup(func() { callTimeout = old })
		server.set([]string{"actions"}, "action:install")
		server.mu.Lock()
		server.delay = 600 * time.Millisecond
		server.mu.Unlock()
		key, err := NewAsker(fakeLookPath()).Ask("T", "B", buttons, 5*time.Second)
		if err != nil || key != "install" {
			t.Errorf("Ask = (%q, %v), erwartet install nach später Antwort", key, err)
		}
	})

	t.Run("Inform", func(t *testing.T) {
		server.set([]string{"actions"}, "")
		NewAsker(fakeLookPath()).Inform("Fertig", "MD Reader ist jetzt im Menü unter Büro.")
		notifies, closed := server.snapshot()
		if len(notifies) != 1 || len(notifies[0].actions) != 0 || notifies[0].summary != "Fertig" || notifies[0].expire != -1 {
			t.Errorf("Inform-Notify = %#v", notifies)
		}
		if len(closed) != 0 {
			t.Errorf("Inform schloss die Nachricht: %v", closed)
		}
	})

	t.Run("ServerGoneFallsBack", func(t *testing.T) {
		server.set([]string{"actions"}, "")
		a := NewAsker(fakeLookPath())
		d, ok := a.(*dbusAsker)
		if !ok {
			t.Fatalf("NewAsker = %T", a)
		}
		var buf bytes.Buffer
		d.fallback = &stderrAsker{w: &buf}
		// Bus-Adresse ungültig machen: Ask muss auf den Rückfall wechseln.
		d.address = "unix:path=" + filepath.Join(t.TempDir(), "gone")
		key, err := d.Ask("Rückfall", "Text", buttons, time.Second)
		if key != "" || err != nil || !strings.Contains(buf.String(), "Rückfall") {
			t.Errorf("Ask = (%q, %v), Ausgabe %q", key, err, buf.String())
		}
		buf.Reset()
		d.Inform("Info-Rückfall", "Text")
		if !strings.Contains(buf.String(), "Info-Rückfall") {
			t.Errorf("Inform-Rückfall: %q", buf.String())
		}
	})
}

// TestProbeNotifications_Timeout prüft, dass ein hängender Dienst den Start
// nicht blockiert (Zeitlimit beim Abfragen der Fähigkeiten).
func TestProbeNotifications_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if caps, ok := probeNotifications(ctx, "unix:path=/nonexistent/bus"); ok || caps != nil {
		t.Errorf("probeNotifications = (%v, %v), erwartet (nil, false)", caps, ok)
	}
	if _, ok := probeNotifications(context.Background(), ""); ok {
		t.Error("leere Adresse: ok")
	}
}
