// Tests für den Starter (Paket main in src/launcher/).
//
// Geprüft werden:
//   - parseCommand (nur das erste Argument entscheidet),
//   - das Abbilden der eingebetteten Icons auf die Icons-Schlüssel,
//   - der Pfad der laufenden Datei (Symlinks, HOME hinter einem Symlink),
//   - runArgv (echter Exit-Code, Startfehler → -1),
//   - der gesamte Ablauf von launcher.run mit Fakes (Befehle, Fehlerwege,
//     fehlende Bibliotheken, Installationsangebot, Start der Oberfläche),
//   - unter Linux der echte memfd-Weg: Ladeprotokoll und exec aus dem Speicher.
//
// Die Datei muss auch für GOOS=windows übersetzbar sein (go vet), deshalb
// keine Linux-eigenen syscall-Konstanten; Linux-Tests überspringen sich
// auf anderen Systemen selbst.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6, Korrekturrunde 1: Angebotsprozess, Fehlermeldungen)
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"md-reader/linuxsetup"
)

// TestMain dient zugleich als Unterprozess für den exec-Test: Ist
// GO_TEST_EXEC=1 gesetzt, ersetzt sich der Testprozess per execGUI durch das
// in GO_TEST_EXEC_BIN genannte Programm (aus einem memfd) mit den Argumenten
// aus GO_TEST_EXEC_ARGS (durch "\x1f" getrennt). Kehrt execGUI zurück, ist
// das ein Fehler (Exit 3).
func TestMain(m *testing.M) {
	if os.Getenv("GO_TEST_EXEC") == "1" {
		os.Exit(execHelper())
	}
	if os.Getenv("GO_TEST_PRINT") == "1" {
		// Als Programm aus dem memfd gestartet: argv und Prozessnamen ausgeben.
		comm, _ := os.ReadFile("/proc/self/comm")
		os.Stdout.WriteString(strings.Join(os.Args, "\x1f") + "\x1fLD=" + os.Getenv("LD_TRACE_LOADED_OBJECTS") + "\n")
		os.Stderr.WriteString("comm=" + strings.TrimSpace(string(comm)) + "\n")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// execHelper führt den Unterprozess-Teil von TestMemExec_Exec aus.
func execHelper() int {
	data, err := os.ReadFile(os.Getenv("GO_TEST_EXEC_BIN"))
	if err != nil {
		os.Stderr.WriteString("lesen: " + err.Error() + "\n")
		return 2
	}
	fd, err := createMemFile(data)
	if err != nil {
		os.Stderr.WriteString("memfd: " + err.Error() + "\n")
		return 2
	}
	// Auf Wunsch vorher ein Ladeprotokoll im selben Prozess (mit /bin/true):
	// LD_TRACE_LOADED_OBJECTS darf danach nicht in der Umgebung der GUI stehen.
	if os.Getenv("GO_TEST_TRACE_FIRST") == "1" {
		trueData, err := os.ReadFile("/bin/true")
		if err != nil {
			os.Stderr.WriteString("lesen /bin/true: " + err.Error() + "\n")
			return 2
		}
		trueFD, err := createMemFile(trueData)
		if err != nil {
			os.Stderr.WriteString("memfd: " + err.Error() + "\n")
			return 2
		}
		if _, err := traceMissing(trueFD); err != nil {
			os.Stderr.WriteString("traceMissing: " + err.Error() + "\n")
			return 2
		}
	}
	// Die Testumgebung soll im neuen Programm nicht wieder den Helfer auslösen.
	os.Unsetenv("GO_TEST_EXEC")
	var args []string
	if a := os.Getenv("GO_TEST_EXEC_ARGS"); a != "" {
		args = strings.Split(a, "\x1f")
	}
	err = execGUI(fd, args)
	os.Stderr.WriteString("execGUI kehrte zurück: " + errString(err) + "\n")
	return 3
}

// errString liefert den Fehlertext oder "<nil>".
func errString(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// sameStrings vergleicht zwei Listen; nil und leer gelten als gleich.
func sameStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

// TestParseCommand prüft die Auswertung der Befehlszeile.
func TestParseCommand(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		cmd   string
		force bool
		rest  []string
	}{
		{"leer", []string{}, "", false, nil},
		{"nil", nil, "", false, nil},
		{"dokument", []string{"doc.md"}, "", false, []string{"doc.md"}},
		{"install", []string{"--install"}, "install", false, nil},
		{"install force", []string{"--install", "--force"}, "install", true, nil},
		{"uninstall", []string{"--uninstall"}, "uninstall", false, nil},
		{"version", []string{"--launcher-version"}, "version", false, nil},
		{"desktop", []string{"--print-desktop-entry", "/usr/bin/md-reader"}, "print-desktop-entry", false, []string{"/usr/bin/md-reader"}},
		{"desktop ohne pfad", []string{"--print-desktop-entry"}, "print-desktop-entry", false, nil},
		// Nur das erste Argument zählt: alles geht unverändert an die GUI.
		{"befehl nicht vorn", []string{"doc.md", "--install"}, "", false, []string{"doc.md", "--install"}},
		{"force allein", []string{"--force"}, "", false, []string{"--force"}},
		{"force bei uninstall", []string{"--uninstall", "--force"}, "uninstall", false, []string{"--force"}},
		{"unbekannte option", []string{"--zoom=2", "a b.md"}, "", false, []string{"--zoom=2", "a b.md"}},
		{"leeres argument", []string{""}, "", false, []string{""}},
		// Interner Befehl für den abgetrennten Angebotsprozess.
		{"offer-install", []string{"--offer-install"}, "offer-install", false, nil},
		{"offer-install detach", []string{"--offer-install", "--detach"}, "offer-install", false, []string{"--detach"}},
		{"offer-install nicht vorn", []string{"a.md", "--offer-install"}, "", false, []string{"a.md", "--offer-install"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd, force, rest := parseCommand(c.args)
			if cmd != c.cmd || force != c.force || !sameStrings(rest, c.rest) {
				t.Errorf("parseCommand(%q) = (%q, %v, %q), erwartet (%q, %v, %q)",
					c.args, cmd, force, rest, c.cmd, c.force, c.rest)
			}
		})
	}
}

// TestParseCommand_RestIsCopy prüft, dass rest eine Kopie ist (spätere
// Änderungen am Original wirken nicht auf die GUI-Argumente).
func TestParseCommand_RestIsCopy(t *testing.T) {
	args := []string{"a.md", "b.md"}
	_, _, rest := parseCommand(args)
	args[0] = "x"
	if len(rest) != 2 || rest[0] != "a.md" {
		t.Errorf("rest teilt sich den Speicher mit args: %q", rest)
	}
}

// TestBuildInt prüft das Umwandeln der Build-Nummer aus den ldflags.
func TestBuildInt(t *testing.T) {
	cases := map[string]int{"47": 47, "0": 0, "": 0, "abc": 0, "-3": 0, " 12 ": 12}
	for in, want := range cases {
		if got := buildInt(in); got != want {
			t.Errorf("buildInt(%q) = %d, erwartet %d", in, got, want)
		}
	}
}

// TestIconsFromFS prüft die Abbildung der Payload-Dateien auf die Schlüssel.
func TestIconsFromFS(t *testing.T) {
	fsys := fstest.MapFS{
		"payload/icons/favicon-16.png":  {Data: []byte("p16")},
		"payload/icons/favicon-32.png":  {Data: []byte("p32")},
		"payload/icons/favicon-48.png":  {Data: []byte("p48")},
		"payload/icons/favicon-256.png": {Data: []byte("p256")},
		"payload/icons/favicon.svg":     {Data: []byte("<svg/>")},
		"payload/icons/favicon.ico":     {Data: []byte("ico")}, // wird ignoriert
		"payload/icons/README":          {Data: []byte("x")},   // wird ignoriert
	}
	got := iconsFromFS(fsys, "payload/icons")
	want := linuxsetup.Icons{
		"16x16":    []byte("p16"),
		"32x32":    []byte("p32"),
		"48x48":    []byte("p48"),
		"256x256":  []byte("p256"),
		"scalable": []byte("<svg/>"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("iconsFromFS = %v, erwartet %v", got, want)
	}
}

// TestIconsFromFS_Partial prüft, dass fehlende Dateien einfach fehlen und ein
// fehlendes Verzeichnis eine leere Abbildung ergibt.
func TestIconsFromFS_Partial(t *testing.T) {
	got := iconsFromFS(fstest.MapFS{"payload/icons/favicon-48.png": {Data: []byte("p48")}}, "payload/icons")
	if len(got) != 1 || string(got["48x48"]) != "p48" {
		t.Errorf("Teilmenge falsch: %v", got)
	}
	if got := iconsFromFS(fstest.MapFS{}, "payload/icons"); len(got) != 0 {
		t.Errorf("leeres FS ergibt %v", got)
	}
}

// TestResolveExePath prüft den Pfad der laufenden Datei: Symlinks werden
// aufgelöst; zeigt der aufgelöste Pfad auf dieselbe Datei wie BinPath
// (HOME hinter einem Symlink), gilt BinPath.
func TestResolveExePath(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real", "md-reader")
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	realDir, _ := filepath.EvalSymlinks(filepath.Dir(real))
	realResolved := filepath.Join(realDir, "md-reader")

	// 1. Ein Symlink wird aufgelöst; BinPath ist eine andere Datei.
	other := filepath.Join(dir, "bin", "md-reader")
	exe, self, err := resolveExePath(func() (string, error) { return link, nil }, other)
	if err != nil || exe != realResolved || self != realResolved {
		t.Errorf("Symlink: (%q, %q, %v), erwartet %q", exe, self, err, realResolved)
	}

	// 2. HOME hinter einem Symlink: BinPath nennt dieselbe Datei über einen
	// anderen Weg → ExePath = BinPath, selfPath bleibt der aufgelöste Pfad.
	homeLink := filepath.Join(dir, "home")
	if err := os.Symlink(filepath.Join(dir, "real"), homeLink); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(homeLink, "md-reader")
	exe, self, err = resolveExePath(func() (string, error) { return real, nil }, binPath)
	if err != nil || exe != binPath || self != realResolved {
		t.Errorf("HOME-Symlink: (%q, %q, %v), erwartet (%q, %q)", exe, self, err, binPath, realResolved)
	}

	// 3. os.Executable scheitert → Fehler.
	if _, _, err := resolveExePath(func() (string, error) { return "", errors.New("kaputt") }, binPath); err == nil {
		t.Error("Fehler von executable wurde verschluckt")
	}
}

// TestRunArgv prüft, dass runArgv den echten Exit-Code liefert und ein
// nicht startbares Programm (-1, Fehler) ergibt.
func TestRunArgv(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("nur Linux")
	}
	if code, err := runArgv([]string{"true"}); code != 0 || err != nil {
		t.Errorf("true: (%d, %v)", code, err)
	}
	if code, err := runArgv([]string{"sh", "-c", "exit 126"}); code != 126 || err == nil {
		t.Errorf("exit 126: (%d, %v)", code, err)
	}
	if code, err := runArgv([]string{"/nicht/vorhanden/md-reader-xyz"}); code != -1 || err == nil {
		t.Errorf("nicht startbar: (%d, %v)", code, err)
	}
	if code, err := runArgv(nil); code != -1 || err == nil {
		t.Errorf("leere Liste: (%d, %v)", code, err)
	}
}

// ---------------------------------------------------------------------------
// Fakes für den Ablauf
// ---------------------------------------------------------------------------

// fakeAsker beantwortet Fragen der Reihe nach und zeichnet alles auf.
type fakeAsker struct {
	answers []string
	asks    []string
	informs []string
}

// Ask liefert die nächste vorgegebene Antwort ("" wenn aufgebraucht).
func (a *fakeAsker) Ask(title, body string, buttons []linuxsetup.Button, timeout time.Duration) (string, error) {
	a.asks = append(a.asks, title)
	if len(a.answers) == 0 {
		return "", nil
	}
	k := a.answers[0]
	a.answers = a.answers[1:]
	return k, nil
}

// Inform zeichnet Titel und Text auf.
func (a *fakeAsker) Inform(title, body string) {
	a.informs = append(a.informs, title+"\n"+body)
}

// fakeMem ersetzt memfd, Ladeprotokoll und exec.
type fakeMem struct {
	createErr  error
	created    []byte
	traces     [][]string // Ergebnis je Aufruf; danach leer
	traceErrs  []error
	traceCalls int
	execErr    error
	execCalls  int
	execFD     int
	execRest   []string
}

func (m *fakeMem) create(data []byte) (int, error) {
	if m.createErr != nil {
		return -1, m.createErr
	}
	m.created = data
	return 42, nil
}

func (m *fakeMem) trace(fd int) ([]string, error) {
	i := m.traceCalls
	m.traceCalls++
	var missing []string
	var err error
	if i < len(m.traces) {
		missing = m.traces[i]
	}
	if i < len(m.traceErrs) {
		err = m.traceErrs[i]
	}
	return missing, err
}

func (m *fakeMem) exec(fd int, rest []string) error {
	m.execCalls++
	m.execFD = fd
	m.execRest = rest
	return m.execErr
}

// testRig bündelt einen Launcher mit Fakes und seinen Ausgaben.
type testRig struct {
	l          *launcher
	home       string
	exe        string
	sysDesktop string
	stdout     bytes.Buffer
	stderr     bytes.Buffer
	asker      *fakeAsker
	askerMade  int
	mem        *fakeMem
	runs       [][]string
	runCode    int
	tools      map[string]bool
	// spawned zählt gestartete Angebotsprozesse; spawnBeforeExec hält fest,
	// ob der Start vor execGUI geschah.
	spawned         int
	spawnErr        error
	spawnBeforeExec bool
	detached        int
	detachErr       error
}

// newRig baut einen Launcher mit eigenem HOME im Temp-Verzeichnis, deutscher
// Sprache, einer Fake-Programmdatei außerhalb von ~/.local/bin und Fakes für
// Asker, memfd und Befehle.
func newRig(t *testing.T) *testRig {
	t.Helper()
	dir := t.TempDir()
	r := &testRig{
		home:       filepath.Join(dir, "home"),
		exe:        filepath.Join(dir, "Downloads", "md-reader"),
		sysDesktop: filepath.Join(dir, "usr-share", "md-reader.desktop"),
		asker:      &fakeAsker{},
		mem:        &fakeMem{},
		tools:      map[string]bool{},
	}
	if err := os.MkdirAll(filepath.Dir(r.exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.exe, []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"HOME": r.home, "LANG": "de_DE.UTF-8"}
	r.l = &launcher{
		stdout: &r.stdout,
		stderr: &r.stderr,
		getenv: func(k string) string { return env[k] },
		lookPath: func(name string) (string, error) {
			if r.tools[name] {
				return "/usr/bin/" + name, nil
			}
			return "", exec.ErrNotFound
		},
		newAsker: func() linuxsetup.Asker {
			r.askerMade++
			return r.asker
		},
		newLayout: func() (linuxsetup.Layout, error) {
			lay, err := linuxsetup.NewLayout(func(k string) string { return env[k] })
			lay.SystemDesktopPath = r.sysDesktop
			return lay, err
		},
		executable: func() (string, error) { return r.exe, nil },
		runCommand: func(argv []string) (int, error) {
			r.runs = append(r.runs, argv)
			if r.runCode != 0 {
				return r.runCode, errors.New("exit " + strconv.Itoa(r.runCode))
			}
			return 0, nil
		},
		spawnOffer: func() error {
			r.spawned++
			r.spawnBeforeExec = r.mem.execCalls == 0
			return r.spawnErr
		},
		detachOffer: func() error {
			r.detached++
			return r.detachErr
		},
		mem:   r.mem,
		gui:   []byte("\x7fELF-gui"),
		icons: linuxsetup.Icons{"48x48": []byte("p48"), "scalable": []byte("<svg/>")},
		build: "47",
	}
	return r
}

// layout liefert die Zielpfade des Rigs.
func (r *testRig) layout(t *testing.T) linuxsetup.Layout {
	t.Helper()
	lay, err := r.l.newLayout()
	if err != nil {
		t.Fatal(err)
	}
	return lay
}

// fileExists meldet, ob ein Pfad existiert.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ---------------------------------------------------------------------------
// Befehle
// ---------------------------------------------------------------------------

// TestRun_Version prüft --launcher-version: nur die Build-Nummer.
func TestRun_Version(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--launcher-version"}); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if r.stdout.String() != "47\n" {
		t.Errorf("Ausgabe %q, erwartet \"47\\n\"", r.stdout.String())
	}
	if r.mem.execCalls != 0 || r.askerMade != 0 {
		t.Error("Version darf weder starten noch fragen")
	}
}

// TestRun_PrintDesktopEntry prüft, dass genau DesktopEntry ausgegeben wird.
func TestRun_PrintDesktopEntry(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--print-desktop-entry", "/usr/bin/md-reader"}); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if want := linuxsetup.DesktopEntry("/usr/bin/md-reader", 47); r.stdout.String() != want {
		t.Errorf("Ausgabe:\n%s\nerwartet:\n%s", r.stdout.String(), want)
	}
	if r.mem.execCalls != 0 {
		t.Error("darf die GUI nicht starten")
	}
}

// TestRun_PrintDesktopEntry_NoPath prüft den Aufruf ohne Pfad: Exit 2, Hinweis.
func TestRun_PrintDesktopEntry_NoPath(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--print-desktop-entry"}); code != 2 {
		t.Errorf("Exit %d, erwartet 2", code)
	}
	if r.stdout.Len() != 0 || !strings.Contains(r.stderr.String(), "--print-desktop-entry") {
		t.Errorf("stdout %q, stderr %q", r.stdout.String(), r.stderr.String())
	}
}

// TestRun_Install prüft --install: Dateien angelegt, MsgInstalled auf stdout,
// keine Dialoge.
func TestRun_Install(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--install"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	lay := r.layout(t)
	if !fileExists(lay.BinPath) || !fileExists(lay.DesktopPath) {
		t.Fatal("Programm oder Desktop-Datei fehlt")
	}
	if b, ok := linuxsetup.InstalledBuild(lay); !ok || b != 47 {
		t.Errorf("InstalledBuild = (%d, %v)", b, ok)
	}
	title, _ := linuxsetup.Text(linuxsetup.LangDE, linuxsetup.MsgInstalled)
	if !strings.Contains(r.stdout.String(), title) {
		t.Errorf("stdout ohne Bestätigung: %q", r.stdout.String())
	}
	if r.askerMade != 0 || r.mem.execCalls != 0 {
		t.Error("--install darf keine Dialoge zeigen und nicht starten")
	}
}

// TestRun_Install_Package prüft: Paketinstallation vorhanden, ohne --force →
// MsgPackageInstalled auf stderr, Exit 1, nichts installiert; mit --force
// wird installiert.
func TestRun_Install_Package(t *testing.T) {
	r := newRig(t)
	if err := os.MkdirAll(filepath.Dir(r.sysDesktop), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.sysDesktop, []byte("[Desktop Entry]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := r.l.run([]string{"--install"}); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	title, _ := linuxsetup.Text(linuxsetup.LangDE, linuxsetup.MsgPackageInstalled)
	if !strings.Contains(r.stderr.String(), title) {
		t.Errorf("stderr ohne Paket-Hinweis: %q", r.stderr.String())
	}
	if fileExists(r.layout(t).BinPath) {
		t.Fatal("ohne --force darf nichts installiert werden")
	}

	r.stderr.Reset()
	if code := r.l.run([]string{"--install", "--force"}); code != 0 {
		t.Fatalf("--force: Exit %d, stderr %q", code, r.stderr.String())
	}
	if !fileExists(r.layout(t).BinPath) {
		t.Error("--force hat nicht installiert")
	}
}

// TestRun_Install_NoPayload prüft: ohne eingebettete Icons/GUI klare Meldung,
// Exit 1, nichts installiert.
func TestRun_Install_NoPayload(t *testing.T) {
	for _, name := range []string{"ohne icons", "ohne gui"} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			if name == "ohne icons" {
				r.l.icons = nil
			} else {
				r.l.gui = nil
			}
			if code := r.l.run([]string{"--install"}); code != 1 {
				t.Errorf("Exit %d, erwartet 1", code)
			}
			if !strings.Contains(r.stderr.String(), "ohne eingebettete App") {
				t.Errorf("stderr: %q", r.stderr.String())
			}
			if fileExists(r.layout(t).BinPath) {
				t.Error("es wurde trotzdem installiert")
			}
		})
	}
}

// TestRun_Uninstall prüft --uninstall: entfernte Pfade und behaltene Pfade
// (MsgUninstalled) auf stdout, Dateien weg, Nutzerdaten bleiben.
func TestRun_Uninstall(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--install"}); code != 0 {
		t.Fatalf("Installation: Exit %d", code)
	}
	lay := r.layout(t)
	if err := os.MkdirAll(lay.KeptPaths[0], 0o755); err != nil {
		t.Fatal(err)
	}
	r.stdout.Reset()
	if code := r.l.run([]string{"--uninstall"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	out := r.stdout.String()
	for _, p := range []string{lay.BinPath, lay.DesktopPath} {
		if fileExists(p) {
			t.Errorf("%s noch vorhanden", p)
		}
		if !strings.Contains(out, p) {
			t.Errorf("entfernter Pfad %s fehlt in der Ausgabe", p)
		}
	}
	for _, p := range lay.KeptPaths {
		if !strings.Contains(out, p) {
			t.Errorf("behaltener Pfad %s fehlt in der Ausgabe", p)
		}
	}
	if !fileExists(lay.KeptPaths[0]) {
		t.Error("Konfiguration wurde gelöscht")
	}
	title, _ := linuxsetup.Text(linuxsetup.LangDE, linuxsetup.MsgUninstalled)
	if !strings.Contains(out, title) || r.askerMade != 0 {
		t.Errorf("Ausgabe ohne MsgUninstalled oder mit Dialog: %q", out)
	}
}

// TestRun_Uninstall_NothingInstalled prüft --uninstall ohne Installation:
// kein Fehler, behaltene Pfade werden trotzdem genannt.
func TestRun_Uninstall_NothingInstalled(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--uninstall"}); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if !strings.Contains(r.stdout.String(), r.layout(t).KeptPaths[0]) {
		t.Errorf("Ausgabe: %q", r.stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Start der Oberfläche
// ---------------------------------------------------------------------------

// TestRun_NoGUI prüft: Starter ohne eingebettete App → Meldung, Exit 1.
func TestRun_NoGUI(t *testing.T) {
	r := newRig(t)
	r.l.gui = nil
	if code := r.l.run([]string{"doc.md"}); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if !strings.Contains(r.stderr.String(), "Starter ohne eingebettete App gebaut") {
		t.Errorf("stderr: %q", r.stderr.String())
	}
	if r.mem.execCalls != 0 || r.mem.created != nil {
		t.Error("ohne GUI darf nichts gestartet werden")
	}
}

// permErr ist ein Berechtigungsfehler wie EACCES/EPERM (errors.Is(…,
// os.ErrPermission) ist wahr), ohne Linux-eigene Konstanten im Test.
var permErr = fmt.Errorf("memfd_create: %w", os.ErrPermission)

// TestRun_MemfdFails prüft Review-Fokus 1: Scheitert memfd mit einem
// Berechtigungsfehler (z. B. vm.memfd_noexec=2 → EACCES), kommt eine klare
// Meldung mit Ursache und Rat auf stderr und als Hinweis, Exit 1.
func TestRun_MemfdFails(t *testing.T) {
	r := newRig(t)
	r.mem.createErr = permErr
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	for _, want := range []string{"permission denied", "Ausführungsrecht", "memfd_noexec"} {
		if !strings.Contains(r.stderr.String(), want) {
			t.Errorf("stderr ohne %q: %q", want, r.stderr.String())
		}
	}
	if len(r.asker.informs) != 1 || !strings.Contains(r.asker.informs[0], "permission denied") {
		t.Errorf("Hinweis fehlt: %q", r.asker.informs)
	}
	if r.mem.execCalls != 0 || r.mem.traceCalls != 0 || r.spawned != 0 {
		t.Error("nach gescheitertem memfd darf nichts mehr laufen")
	}
}

// TestRun_MemfdFails_Generic prüft: Andere Fehler (kein EACCES/EPERM)
// bekommen eine allgemeine Meldung ohne den memfd/noexec-Rat.
func TestRun_MemfdFails_Generic(t *testing.T) {
	r := newRig(t)
	r.mem.createErr = errors.New("kaputt")
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	out := r.stderr.String()
	if !strings.Contains(out, "konnte nicht starten: kaputt") {
		t.Errorf("allgemeine Meldung fehlt: %q", out)
	}
	if strings.Contains(out, "memfd_noexec") || strings.Contains(out, "Ausführungsrecht") {
		t.Errorf("noexec-Rat bei fremdem Fehler: %q", out)
	}
	if len(r.asker.informs) != 1 {
		t.Errorf("Hinweis fehlt: %q", r.asker.informs)
	}
}

// TestRun_TraceStartFails prüft: Lässt sich die GUI aus dem memfd wegen
// fehlender Berechtigung nicht starten, kommt die klare Meldung mit Rat.
func TestRun_TraceStartFails(t *testing.T) {
	r := newRig(t)
	r.mem.traceErrs = []error{&execStartError{err: fmt.Errorf("fork/exec /proc/self/fd/3: %w", os.ErrPermission)}}
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if !strings.Contains(r.stderr.String(), "Ausführungsrecht") || len(r.asker.informs) != 1 {
		t.Errorf("stderr %q, informs %q", r.stderr.String(), r.asker.informs)
	}
	if r.mem.execCalls != 0 || len(r.asker.asks) != 0 || r.spawned != 0 {
		t.Error("darf weder fragen noch starten")
	}
}

// TestRun_TraceOtherError prüft: Ein anderer Fehler des Ladeprotokolls (z. B.
// Exit-Code ungleich 0 oder Meldungen des Laders) wird nur protokolliert –
// samt Text des Laders –, die GUI startet trotzdem.
func TestRun_TraceOtherError(t *testing.T) {
	r := newRig(t)
	r.mem.traceErrs = []error{errors.New("exit status 1; Lader: version `GLIBC_2.40' not found")}
	if code := r.l.run([]string{"doc.md"}); code != 0 {
		t.Errorf("Exit %d", code)
	}
	if r.mem.execCalls != 1 || !strings.Contains(r.stderr.String(), "version `GLIBC_2.40' not found") {
		t.Errorf("execCalls %d, stderr %q", r.mem.execCalls, r.stderr.String())
	}
}

// TestRun_MissingLibs_NoApt prüft: Bibliotheken fehlen, kein apt/pkexec →
// Hinweis, Exit 1, kein Start, kein Angebotsprozess, nur ein Ladeprotokoll.
func TestRun_MissingLibs_NoApt(t *testing.T) {
	r := newRig(t)
	r.mem.traces = [][]string{{"libwebkit2gtk-4.1.so.0"}}
	if code := r.l.run([]string{"doc.md"}); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if r.mem.execCalls != 0 || r.mem.traceCalls != 1 || len(r.runs) != 0 || r.spawned != 0 {
		t.Errorf("exec %d, trace %d, runs %q, spawned %d", r.mem.execCalls, r.mem.traceCalls, r.runs, r.spawned)
	}
	if !strings.Contains(r.stderr.String(), "libwebkit2gtk-4.1.so.0") {
		t.Errorf("stderr ohne Bibliothek: %q", r.stderr.String())
	}
}

// TestRun_MissingLibs_InstalledThenStart prüft: Nutzer installiert per apt,
// zweites Ladeprotokoll ist leer → GUI startet mit den Argumenten.
func TestRun_MissingLibs_InstalledThenStart(t *testing.T) {
	r := newRig(t)
	r.tools["apt-get"], r.tools["pkexec"] = true, true
	r.mem.traces = [][]string{{"libwebkit2gtk-4.1.so.0"}, nil}
	r.asker.answers = []string{"install"}
	if code := r.l.run([]string{"doc.md"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	if want := [][]string{linuxsetup.AptInstallCommand()}; !reflect.DeepEqual(r.runs, want) {
		t.Errorf("Befehle %q, erwartet %q", r.runs, want)
	}
	if r.mem.traceCalls != 2 || r.mem.execCalls != 1 || !sameStrings(r.mem.execRest, []string{"doc.md"}) {
		t.Errorf("trace %d, exec %d, rest %q", r.mem.traceCalls, r.mem.execCalls, r.mem.execRest)
	}
}

// TestRun_MissingLibs_StillMissing prüft: Auch nach der Installation fehlt
// etwas → Exit 1, höchstens ein zweites Ladeprotokoll, keine Endlosschleife.
func TestRun_MissingLibs_StillMissing(t *testing.T) {
	r := newRig(t)
	r.tools["apt-get"], r.tools["pkexec"] = true, true
	r.mem.traces = [][]string{{"libwebkit2gtk-4.1.so.0"}, {"libjavascriptcoregtk-4.1.so.0"}, {"x"}}
	r.asker.answers = []string{"install", "install", "install"}
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if r.mem.traceCalls != 2 || r.mem.execCalls != 0 || len(r.runs) != 1 {
		t.Errorf("trace %d, exec %d, runs %q", r.mem.traceCalls, r.mem.execCalls, r.runs)
	}
	if !strings.Contains(r.stderr.String(), "libjavascriptcoregtk-4.1.so.0") {
		t.Errorf("stderr ohne die noch fehlende Bibliothek: %q", r.stderr.String())
	}
}

// TestRun_MissingLibs_SecondTraceStartFails prüft: Das zweite Ladeprotokoll
// lässt sich nicht starten → klare Meldung, Exit 1.
func TestRun_MissingLibs_SecondTraceStartFails(t *testing.T) {
	r := newRig(t)
	r.tools["apt-get"], r.tools["pkexec"] = true, true
	r.mem.traces = [][]string{{"libwebkit2gtk-4.1.so.0"}}
	r.mem.traceErrs = []error{nil, &execStartError{err: permErr}}
	r.asker.answers = []string{"install"}
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if r.mem.execCalls != 0 || !strings.Contains(r.stderr.String(), "Ausführungsrecht") {
		t.Errorf("exec %d, stderr %q", r.mem.execCalls, r.stderr.String())
	}
}

// TestRun_SpawnsOfferChild prüft die Vorgabe „Angebot verzögert die GUI
// nicht“: Start aus dem Download-Ordner ohne Installation → der Starter
// startet den Angebotsprozess (vor execGUI), fragt selbst nichts, baut nicht
// einmal einen Asker und erreicht execGUI mit den Argumenten.
func TestRun_SpawnsOfferChild(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"a.md", "--install"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	if r.spawned != 1 || !r.spawnBeforeExec {
		t.Errorf("spawned %d, vor exec %v", r.spawned, r.spawnBeforeExec)
	}
	if r.askerMade != 0 || len(r.asker.asks) != 0 {
		t.Errorf("Elternprozess hat gefragt: made %d, asks %q", r.askerMade, r.asker.asks)
	}
	if r.mem.execCalls != 1 || r.mem.execFD != 42 || !sameStrings(r.mem.execRest, []string{"a.md", "--install"}) {
		t.Errorf("exec %d fd %d rest %q", r.mem.execCalls, r.mem.execFD, r.mem.execRest)
	}
	if string(r.mem.created) != "\x7fELF-gui" {
		t.Errorf("memfd-Inhalt %q", r.mem.created)
	}
	// Der Elternprozess schreibt nichts: weder Programm noch Antwortspeicher.
	lay := r.layout(t)
	if fileExists(lay.BinPath) || fileExists(lay.PrefsPath) {
		t.Error("Elternprozess hat installiert oder eine Antwort gespeichert")
	}
}

// TestRun_SpawnFails_StillStarts prüft: Scheitert der Start des
// Angebotsprozesses, wird das nur protokolliert; die GUI startet.
func TestRun_SpawnFails_StillStarts(t *testing.T) {
	r := newRig(t)
	r.spawnErr = errors.New("fork kaputt")
	if code := r.l.run(nil); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if r.mem.execCalls != 1 || !strings.Contains(r.stderr.String(), "fork kaputt") {
		t.Errorf("exec %d, stderr %q", r.mem.execCalls, r.stderr.String())
	}
}

// TestRun_NoSpawnWhenNothingToOffer prüft: Kein Angebotsprozess bei Start aus
// ~/.local/bin, über einen Symlink darauf, bei einer Paketinstallation, bei
// der Antwort „Nie“ und bei einer gleich neuen Nutzerinstallation.
func TestRun_NoSpawnWhenNothingToOffer(t *testing.T) {
	cases := []string{"aus BinPath", "Symlink auf BinPath", "Paket", "Nie", "gleich neu installiert"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			lay := r.layout(t)
			switch name {
			case "aus BinPath", "Symlink auf BinPath":
				if err := os.MkdirAll(filepath.Dir(lay.BinPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(lay.BinPath, []byte("launcher"), 0o755); err != nil {
					t.Fatal(err)
				}
				r.exe = lay.BinPath
				if name == "Symlink auf BinPath" {
					link := filepath.Join(t.TempDir(), "md-reader-link")
					if err := os.Symlink(lay.BinPath, link); err != nil {
						t.Fatal(err)
					}
					r.exe = link
				}
			case "Paket":
				if err := os.MkdirAll(filepath.Dir(r.sysDesktop), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(r.sysDesktop, []byte("[Desktop Entry]\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "Nie":
				if err := linuxsetup.SavePrefs(lay.PrefsPath, linuxsetup.Prefs{InstallPrompt: linuxsetup.PromptNever}); err != nil {
					t.Fatal(err)
				}
			case "gleich neu installiert":
				if code := r.l.run([]string{"--install"}); code != 0 {
					t.Fatalf("Installation: Exit %d", code)
				}
			}
			if code := r.l.run([]string{"x.md"}); code != 0 {
				t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
			}
			if r.spawned != 0 || r.askerMade != 0 {
				t.Errorf("spawned %d, Asker gebaut %d", r.spawned, r.askerMade)
			}
			if r.mem.execCalls != 1 {
				t.Error("GUI wurde nicht gestartet")
			}
		})
	}
}

// TestRun_NoSpawnInCommands prüft: --install, --uninstall, Version,
// Desktop-Datei und ein Starter ohne GUI starten nie den Angebotsprozess.
func TestRun_NoSpawnInCommands(t *testing.T) {
	for _, args := range [][]string{{"--install"}, {"--uninstall"}, {"--launcher-version"}, {"--print-desktop-entry", "/x"}} {
		r := newRig(t)
		r.l.run(args)
		if r.spawned != 0 || r.detached != 0 {
			t.Errorf("%q: spawned %d, detached %d", args, r.spawned, r.detached)
		}
	}
	r := newRig(t)
	r.l.gui = nil
	r.l.run([]string{"a.md"})
	if r.spawned != 0 {
		t.Errorf("ohne GUI: spawned %d", r.spawned)
	}
}

// TestRun_ExecFails prüft: Scheitert exec mit fehlender Berechtigung, kommt
// die klare Meldung, Exit 1.
func TestRun_ExecFails(t *testing.T) {
	r := newRig(t)
	r.mem.execErr = fmt.Errorf("exec: %w", os.ErrPermission)
	if code := r.l.run(nil); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if !strings.Contains(r.stderr.String(), "Ausführungsrecht") {
		t.Errorf("stderr: %q", r.stderr.String())
	}
}

// ---------------------------------------------------------------------------
// Interner Befehl --offer-install (abgetrennter Angebotsprozess)
// ---------------------------------------------------------------------------

// TestRun_OfferInstallCmd_Later prüft: --offer-install stellt die
// Installationsfrage, speichert „Später“, startet keine GUI, Exit 0.
func TestRun_OfferInstallCmd_Later(t *testing.T) {
	r := newRig(t)
	r.asker.answers = []string{"later"}
	if code := r.l.run([]string{"--offer-install"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	title, _ := linuxsetup.Text(linuxsetup.LangDE, linuxsetup.MsgOfferInstall)
	if len(r.asker.asks) != 1 || r.asker.asks[0] != title {
		t.Errorf("Fragen %q", r.asker.asks)
	}
	if p := linuxsetup.LoadPrefs(r.layout(t).PrefsPath); p.InstallPrompt != linuxsetup.PromptLater {
		t.Errorf("Antwort nicht gespeichert: %+v", p)
	}
	if r.mem.execCalls != 0 || r.mem.created != nil || r.mem.traceCalls != 0 || r.spawned != 0 {
		t.Error("--offer-install darf weder memfd noch GUI noch Unterprozess starten")
	}
}

// TestRun_OfferInstallCmd_Yes prüft: „Ja“ installiert die laufende Datei –
// dieselbe, die der Elternprozess als ExePath sieht (executable) – samt
// Icons und Build-Nummer, Exit 0.
func TestRun_OfferInstallCmd_Yes(t *testing.T) {
	r := newRig(t)
	r.asker.answers = []string{"yes"}
	if code := r.l.run([]string{"--offer-install"}); code != 0 {
		t.Fatalf("Exit %d, stderr %q", code, r.stderr.String())
	}
	lay := r.layout(t)
	data, err := os.ReadFile(lay.BinPath)
	if err != nil || string(data) != "launcher" {
		t.Errorf("installiertes Programm: %q, %v", data, err)
	}
	if b, ok := linuxsetup.InstalledBuild(lay); !ok || b != 47 {
		t.Errorf("InstalledBuild = (%d, %v)", b, ok)
	}
	if !fileExists(filepath.Join(lay.IconBase, "48x48", "apps", "md-reader.png")) {
		t.Error("Icon fehlt")
	}
}

// TestRun_OfferInstallCmd_Fails prüft: Scheitert die Installation, steht der
// Grund auf stderr und der Exit-Code ist 1 (niemand wartet darauf).
func TestRun_OfferInstallCmd_Fails(t *testing.T) {
	r := newRig(t)
	r.asker.answers = []string{"yes"}
	// Eine Datei an der Stelle von ~/.local lässt MkdirAll scheitern.
	if err := os.MkdirAll(r.home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.home, ".local"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := r.l.run([]string{"--offer-install"}); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if r.stderr.Len() == 0 {
		t.Error("kein Grund auf stderr")
	}
}

// TestRun_OfferInstallCmd_NothingToOffer prüft: Hat sich die Lage geändert
// (inzwischen installiert), fragt der Angebotsprozess nicht, Exit 0.
func TestRun_OfferInstallCmd_NothingToOffer(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--install"}); code != 0 {
		t.Fatalf("Installation: Exit %d", code)
	}
	if code := r.l.run([]string{"--offer-install"}); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if r.askerMade != 0 {
		t.Error("Asker gebaut, obwohl nichts anzubieten ist")
	}
}

// TestRun_OfferInstallCmd_NoPayload prüft: Ohne eingebettete App wird nichts
// angeboten (eine Installation wäre nutzlos), Exit 1.
func TestRun_OfferInstallCmd_NoPayload(t *testing.T) {
	r := newRig(t)
	r.l.gui = nil
	if code := r.l.run([]string{"--offer-install"}); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if r.askerMade != 0 || fileExists(r.layout(t).PrefsPath) {
		t.Error("ohne Nutzlast gefragt oder gespeichert")
	}
}

// TestRun_OfferInstallCmd_Detach prüft die Zwischenstufe: --offer-install
// --detach startet nur den abgetrennten Angebotsprozess und endet mit 0,
// ohne selbst zu fragen; scheitert der Start, Exit 1.
func TestRun_OfferInstallCmd_Detach(t *testing.T) {
	r := newRig(t)
	if code := r.l.run([]string{"--offer-install", "--detach"}); code != 0 {
		t.Fatalf("Exit %d", code)
	}
	if r.detached != 1 || r.askerMade != 0 || r.spawned != 0 {
		t.Errorf("detached %d, Asker %d, spawned %d", r.detached, r.askerMade, r.spawned)
	}
	r.detachErr = errors.New("kein fork")
	if code := r.l.run([]string{"--offer-install", "--detach"}); code != 1 {
		t.Errorf("Startfehler: Exit %d, erwartet 1", code)
	}
}

// TestRun_ExtraArgs prüft: Unerwartete Argumente nach --install (außer
// --force), --uninstall, --launcher-version und --offer-install ergeben eine
// Warnung auf stderr und Exit 2; es passiert nichts.
func TestRun_ExtraArgs(t *testing.T) {
	cases := [][]string{
		{"--install", "x.md"},
		{"--install", "--force", "x.md"},
		{"--uninstall", "--force"},
		{"--launcher-version", "x"},
		{"--offer-install", "x"},
	}
	for _, args := range cases {
		r := newRig(t)
		if code := r.l.run(args); code != 2 {
			t.Errorf("%q: Exit %d, erwartet 2", args, code)
		}
		bad := args[len(args)-1]
		if !strings.Contains(r.stderr.String(), bad) || r.stdout.Len() != 0 {
			t.Errorf("%q: stderr %q, stdout %q", args, r.stderr.String(), r.stdout.String())
		}
		if fileExists(r.layout(t).BinPath) || r.askerMade != 0 || r.detached != 0 {
			t.Errorf("%q: es ist trotzdem etwas passiert", args)
		}
	}
}

// TestLimitedBuffer prüft den begrenzten Puffer für stderr des Laders.
func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{max: 8}
	n, err := b.Write([]byte("12345"))
	if n != 5 || err != nil {
		t.Errorf("Write = (%d, %v)", n, err)
	}
	// Überlauf: Write meldet trotzdem alles als geschrieben (kein Fehler für
	// den Kindprozess), behalten werden nur max Bytes.
	n, err = b.Write([]byte("67890abc"))
	if n != 8 || err != nil || b.String() != "12345678" {
		t.Errorf("Write = (%d, %v), Inhalt %q", n, err, b.String())
	}
	if n, _ := b.Write([]byte("x")); n != 1 || b.String() != "12345678" {
		t.Errorf("nach vollem Puffer: %q", b.String())
	}
}

// TestRun_English prüft die englische Meldung bei LANG=C.
func TestRun_English(t *testing.T) {
	r := newRig(t)
	r.l.getenv = func(k string) string {
		if k == "HOME" {
			return r.home
		}
		return ""
	}
	r.l.gui = nil
	r.l.run(nil)
	if !strings.Contains(r.stderr.String(), "without an embedded app") {
		t.Errorf("stderr: %q", r.stderr.String())
	}
}

// TestLazyAsker prüft, dass die Fabrik erst bei der ersten Nutzung und nur
// einmal aufgerufen wird.
func TestLazyAsker(t *testing.T) {
	made := 0
	inner := &fakeAsker{answers: []string{"a", "b"}}
	la := &lazyAsker{factory: func() linuxsetup.Asker { made++; return inner }}
	if made != 0 {
		t.Fatal("Fabrik zu früh aufgerufen")
	}
	k1, _ := la.Ask("t", "b", nil, time.Second)
	la.Inform("t", "b")
	k2, _ := la.Ask("t", "b", nil, time.Second)
	if made != 1 || k1 != "a" || k2 != "b" || len(inner.informs) != 1 {
		t.Errorf("made %d, keys %q %q, informs %d", made, k1, k2, len(inner.informs))
	}
}

// ---------------------------------------------------------------------------
// Echter memfd-Weg (nur Linux)
// ---------------------------------------------------------------------------

// requireLinux überspringt den Test außerhalb von Linux.
func requireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("memfd gibt es nur unter Linux")
	}
}

// TestCreateMemFile prüft Inhalt und CLOEXEC-Flag des memfd.
func TestCreateMemFile(t *testing.T) {
	requireLinux(t)
	data := []byte("Inhalt\x00mit Nullbyte und mehr als null Bytes")
	fd, err := createMemFile(data)
	if err != nil {
		t.Fatalf("createMemFile: %v", err)
	}
	path := "/proc/self/fd/" + strconv.Itoa(fd)
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) {
		t.Errorf("Inhalt %q, %v", got, err)
	}
	target, _ := os.Readlink(path)
	if !strings.HasPrefix(target, "/memfd:md-reader") {
		t.Errorf("Name des memfd: %q", target)
	}
	// flags in fdinfo ist oktal; O_CLOEXEC = 02000000.
	info, err := os.ReadFile("/proc/self/fdinfo/" + strconv.Itoa(fd))
	if err != nil {
		t.Fatal(err)
	}
	var flags int64 = -1
	for _, line := range strings.Split(string(info), "\n") {
		if v, ok := strings.CutPrefix(line, "flags:"); ok {
			flags, _ = strconv.ParseInt(strings.TrimSpace(v), 8, 64)
		}
	}
	if flags < 0 || flags&0o2000000 == 0 {
		t.Errorf("memfd ohne CLOEXEC (flags %o)", flags)
	}
}

// TestCreateMemFile_Large prüft, dass auch große Daten (mehrere MB, wie die
// echte GUI) vollständig geschrieben werden.
func TestCreateMemFile_Large(t *testing.T) {
	requireLinux(t)
	data := bytes.Repeat([]byte("0123456789abcdef"), 1<<20) // 16 MiB
	fd, err := createMemFile(data)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat("/proc/self/fd/" + strconv.Itoa(fd))
	if err != nil || fi.Size() != int64(len(data)) {
		t.Errorf("Größe %v, %v", fi, err)
	}
}

// TestMemExec_TraceAndExec prüft das Ladeprotokoll aus dem memfd mit
// /bin/true: keine fehlenden Bibliotheken, kein Fehler; zweimal hintereinander
// (der Starter prüft nach einer Installation erneut mit derselben fd).
func TestMemExec_TraceAndExec(t *testing.T) {
	requireLinux(t)
	data, err := os.ReadFile("/bin/true")
	if err != nil {
		t.Skip("/bin/true fehlt")
	}
	fd, err := createMemFile(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		missing, err := traceMissing(fd)
		if err != nil || len(missing) != 0 {
			t.Errorf("Lauf %d: traceMissing = (%q, %v)", i+1, missing, err)
		}
	}
	// Die fd des Starters bleibt danach gültig (für execGUI).
	if _, err := os.Stat("/proc/self/fd/" + strconv.Itoa(fd)); err != nil {
		t.Errorf("fd nach traceMissing geschlossen: %v", err)
	}
}

// TestTraceMissing_RealMissingLibrary baut mit gcc ein Programm gegen eine
// eigene Bibliothek, löscht die Bibliothek und prüft, dass traceMissing sie
// als fehlend meldet. Ohne gcc wird der Test übersprungen.
func TestTraceMissing_RealMissingLibrary(t *testing.T) {
	requireLinux(t)
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc fehlt")
	}
	dir := t.TempDir()
	lib := filepath.Join(dir, "libmdreaderfake.so")
	src := filepath.Join(dir, "lib.c")
	mainSrc := filepath.Join(dir, "main.c")
	bin := filepath.Join(dir, "prog")
	if err := os.WriteFile(src, []byte("int mdreader_fake(void){return 7;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainSrc, []byte("int mdreader_fake(void);\nint main(void){return mdreader_fake()-7;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(gcc, "-shared", "-fPIC", "-o", lib, src).CombinedOutput(); err != nil {
		t.Skipf("gcc -shared: %v %s", err, out)
	}
	if out, err := exec.Command(gcc, "-o", bin, mainSrc, "-L"+dir, "-lmdreaderfake").CombinedOutput(); err != nil {
		t.Skipf("gcc: %v %s", err, out)
	}
	if err := os.Remove(lib); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	fd, err := createMemFile(data)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := traceMissing(fd)
	if err != nil || !reflect.DeepEqual(missing, []string{"libmdreaderfake.so"}) {
		t.Errorf("traceMissing = (%q, %v), erwartet [libmdreaderfake.so]", missing, err)
	}
}

// TestTraceMissing_LoaderStderr baut mit gcc ein Programm, das eine
// Symbolversion MDR_2 verlangt, und ersetzt die Bibliothek durch eine mit
// MDR_1. Der Lader meldet im Ladeprotokoll (Exit 0!) auf stderr
// "version `MDR_2' not found" – wie bei einer zu alten glibc. traceMissing
// liefert dann keinen execStartError, sondern einen Fehler mit diesem Text.
func TestTraceMissing_LoaderStderr(t *testing.T) {
	requireLinux(t)
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc fehlt")
	}
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	libSrc := write("l.c", "int f(void){return 1;}\n")
	mainSrc := write("m.c", "int f(void);\nint main(void){return f()-1;}\n")
	v2 := write("v2.map", "MDR_2 { global: f; local: *; };\n")
	v1 := write("v1.map", "MDR_1 { global: f; local: *; };\n")
	lib := filepath.Join(dir, "libmdrv.so")
	bin := filepath.Join(dir, "prog")
	gccRun := func(args ...string) {
		if out, err := exec.Command(gcc, args...).CombinedOutput(); err != nil {
			t.Skipf("gcc %q: %v %s", args, err, out)
		}
	}
	gccRun("-shared", "-fPIC", "-Wl,--version-script="+v2, "-Wl,-soname,libmdrv.so", "-o", lib, libSrc)
	gccRun("-o", bin, mainSrc, "-L"+dir, "-lmdrv", "-Wl,-rpath,"+dir)
	gccRun("-shared", "-fPIC", "-Wl,--version-script="+v1, "-Wl,-soname,libmdrv.so", "-o", lib, libSrc)

	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	fd, err := createMemFile(data)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := traceMissing(fd)
	if len(missing) != 0 || err == nil || isStartError(err) {
		t.Fatalf("traceMissing = (%q, %v), erwartet Fehler mit Lader-Text", missing, err)
	}
	if !strings.Contains(err.Error(), "version `MDR_2' not found") {
		t.Errorf("Lader-Text fehlt: %v", err)
	}
}

// TestTraceMissing_NotExecutable prüft: Inhalt ohne gültiges Programmformat
// (der Kern verweigert exec) ergibt einen execStartError – denselben Weg, den
// ein memfd ohne Ausführungsrecht nimmt.
func TestTraceMissing_NotExecutable(t *testing.T) {
	requireLinux(t)
	fd, err := createMemFile([]byte("kein Programm"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = traceMissing(fd)
	var se *execStartError
	if !errors.As(err, &se) {
		t.Errorf("Fehler %v (%T), erwartet *execStartError", err, err)
	}
}

// TestMemExec_Exec startet den Testprozess erneut als Unterprozess, der sich
// per execGUI durch ein Programm aus einem memfd ersetzt:
//   - /bin/echo mit ["hallo"] → Ausgabe "hallo". uutils-coreutils (Rust, z. B.
//     Ubuntu 25.10) verweigert einen fremden argv[0] ("Requested utility
//     `md-reader` does not match executable name"); dann wird dieser Fall
//     übersprungen – die übrigen Fälle prüfen dasselbe ohne diese Eigenheit.
//   - /bin/sh -c 'echo "$0"' → argv[0] ist "md-reader".
//   - die Testdatei selbst (GO_TEST_PRINT=1) → argv vollständig und
//     unverändert, auch mit Leerzeichen; vorher lief im selben Prozess
//     traceMissing, trotzdem ist LD_TRACE_LOADED_OBJECTS in der GUI leer.
func TestMemExec_Exec(t *testing.T) {
	requireLinux(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, bin string
		args      []string
		env       string
		want      string
	}{
		{"echo", "/bin/echo", []string{"hallo"}, "", "hallo\n"},
		// sh -c ohne weitere Argumente: $0 ist argv[0] des Programms.
		{"argv0", "/bin/sh", []string{"-c", `echo "$0"`}, "", "md-reader\n"},
		// Vorher läuft traceMissing im selben Prozess; LD= muss leer bleiben.
		{"argv komplett", self, []string{"hallo", "a b.md", "--install"}, "GO_TEST_PRINT=1\x1eGO_TEST_TRACE_FIRST=1",
			"md-reader\x1fhallo\x1fa b.md\x1f--install\x1fLD=\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.bin); err != nil {
				t.Skipf("%s fehlt", c.bin)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^$")
			cmd.Env = append(os.Environ(),
				"GO_TEST_EXEC=1",
				"GO_TEST_EXEC_BIN="+c.bin,
				"GO_TEST_EXEC_ARGS="+strings.Join(c.args, "\x1f"))
			if c.env != "" {
				cmd.Env = append(cmd.Env, strings.Split(c.env, "\x1e")...)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if c.name == "echo" && strings.Contains(stderr.String(), "does not match executable name") {
				t.Skipf("uutils-echo prüft argv[0]: %s", strings.TrimSpace(stderr.String()))
			}
			if err != nil || string(out) != c.want {
				t.Errorf("Ausgabe %q, Fehler %v, stderr %q; erwartet %q", out, err, stderr.String(), c.want)
			}
			if c.env != "" {
				t.Logf("Prozess aus dem memfd: %s", strings.TrimSpace(stderr.String()))
			}
		})
	}
}
