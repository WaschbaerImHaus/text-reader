// Starter von MD Reader unter Linux (eigenes Programm, reines Go, CGO_ENABLED=0).
//
// Der Starter ist die Datei, die der Nutzer herunterlädt. Er enthält die
// eigentliche Oberfläche (GUI-Programm mit WebKitGTK) eingebettet und
//  1. legt sie in einen memfd (Datei nur im Arbeitsspeicher, siehe
//     memexec_linux.go) – so startet sie auch von einem Datenträger ohne
//     Ausführungsrecht (noexec, z. B. USB-Stick) und unter beliebigem Namen,
//  2. prüft mit dem Ladeprotokoll des dynamischen Laders, ob Bibliotheken
//     (WebKitGTK) fehlen, und bietet das Nachinstallieren an,
//  3. bietet an, sich ins Startmenü einzutragen (oder eine ältere
//     Nutzerinstallation zu ersetzen),
//  4. ersetzt sich per exec durch die Oberfläche (argv[0] = "md-reader").
//
// Befehle (nur das erste Argument zählt, alles andere geht an die Oberfläche):
//
//	--install [--force]                 ins Home-Verzeichnis installieren
//	--uninstall                         Nutzerinstallation entfernen
//	--launcher-version                  Build-Nummer ausgeben
//	--print-desktop-entry <Programmpfad> Desktop-Datei ausgeben (für das .deb)
//
// Die Fachlogik liegt im Paket linuxsetup; diese Datei verbindet sie nur.
// Alle Abhängigkeiten nach außen (Umgebung, Ausgaben, Fragen, memfd, Befehle)
// stecken im Typ launcher und werden in Tests durch Fakes ersetzt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6: Starter-Programm)
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"md-reader/linuxsetup"
)

// buildNumber ist die Build-Nummer des Starters. Gesetzt beim Bauen über
// -ldflags "-X main.buildNumber=<n>"; ohne Angabe "0".
var buildNumber = "0"

// Befehle, die parseCommand erkennt.
const (
	cmdNone          = ""
	cmdInstall       = "install"
	cmdUninstall     = "uninstall"
	cmdVersion       = "version"
	cmdDesktopEntry  = "print-desktop-entry"
	optionForce      = "--force"
	desktopEntryFlag = "--print-desktop-entry"
)

// iconFileKeys ordnet den Dateinamen in payload/icons die Icons-Schlüssel zu.
var iconFileKeys = map[string]string{
	"favicon-16.png":  "16x16",
	"favicon-32.png":  "32x32",
	"favicon-48.png":  "48x48",
	"favicon-256.png": "256x256",
	"favicon.svg":     "scalable",
}

// main startet den Starter mit den echten Abhängigkeiten.
func main() {
	os.Exit(newLauncher().run(os.Args[1:]))
}

// ---------------------------------------------------------------------------
// Befehlszeile und kleine Helfer
// ---------------------------------------------------------------------------

// parseCommand wertet die Befehlszeile aus. Nur args[0] entscheidet über den
// Befehl; steht dort keiner, gehen alle Argumente unverändert an die GUI.
//
//   - "--install" [--force]: cmd "install"; force nur, wenn args[1] "--force" ist.
//   - "--uninstall", "--launcher-version": cmd ohne Wirkung der übrigen Argumente.
//   - "--print-desktop-entry" <Pfad>: rest enthält den Pfad.
//
// @param args Argumente ohne Programmnamen (os.Args[1:]).
// @return cmd Befehl oder ""; force bei "--install --force"; rest übrige
// Argumente als Kopie (nie derselbe Speicher wie args).
func parseCommand(args []string) (cmd string, force bool, rest []string) {
	if len(args) == 0 {
		return cmdNone, false, nil
	}
	switch args[0] {
	case "--install":
		tail := args[1:]
		if len(tail) > 0 && tail[0] == optionForce {
			force = true
			tail = tail[1:]
		}
		return cmdInstall, force, copyStrings(tail)
	case "--uninstall":
		return cmdUninstall, false, copyStrings(args[1:])
	case "--launcher-version":
		return cmdVersion, false, copyStrings(args[1:])
	case desktopEntryFlag:
		return cmdDesktopEntry, false, copyStrings(args[1:])
	}
	return cmdNone, false, copyStrings(args)
}

// copyStrings liefert eine Kopie der Liste (nil bei leerer Liste).
func copyStrings(s []string) []string {
	return append([]string(nil), s...)
}

// buildInt wandelt die Build-Nummer aus den ldflags in eine Zahl um.
// Ungültige oder negative Werte ergeben 0 (gilt als „unbekannt, sehr alt“).
func buildInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// iconsFromFS liest die eingebetteten Icons aus dir und bildet sie auf die
// Icons-Schlüssel ab. Unbekannte Dateien werden ignoriert, fehlende fehlen.
//
// @param fsys Dateisystem (embed.FS oder Test-Map).
// @param dir Verzeichnis darin (z. B. "payload/icons").
// @return Icons; leer, wenn nichts gefunden wurde.
func iconsFromFS(fsys fs.FS, dir string) linuxsetup.Icons {
	icons := linuxsetup.Icons{}
	for name, key := range iconFileKeys {
		data, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			continue
		}
		icons[key] = data
	}
	return icons
}

// resolveExePath ermittelt den Pfad der laufenden Datei.
//
// Symlinks werden aufgelöst (selfPath ist die echte Datei, Quelle einer
// Installation). Ist sie dieselbe Datei wie binPath – etwa weil HOME hinter
// einem Symlink liegt –, wird binPath als exePath geliefert, damit Decide den
// Start aus ~/.local/bin erkennt.
//
// @param executable os.Executable oder Test-Fake.
// @param binPath Layout.BinPath.
// @return exePath für StartContext, selfPath für Install; Fehler, wenn der
// Pfad nicht ermittelt werden kann.
func resolveExePath(executable func() (string, error), binPath string) (exePath, selfPath string, err error) {
	exe, err := executable()
	if err != nil {
		return "", "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", "", err
	}
	exePath = resolved
	if a, errA := os.Stat(resolved); errA == nil {
		if b, errB := os.Stat(binPath); errB == nil && os.SameFile(a, b) {
			exePath = filepath.Clean(binPath)
		}
	}
	return exePath, resolved, nil
}

// runArgv führt eine feste Argumentliste ohne Shell aus (für pkexec apt-get).
// Ein- und Ausgaben gehen ans Terminal (stdout auf stderr, damit stdout des
// Starters sauber bleibt).
//
// @param argv Programm und Argumente.
// @return Exit-Code und Fehler; (0, nil) bei Erfolg, (Code, Fehler) bei
// Exit-Code ungleich 0, (-1, Fehler), wenn das Programm nicht startet.
func runArgv(argv []string) (int, error) {
	if len(argv) == 0 {
		return -1, errors.New("leere Argumentliste")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), err
	}
	return -1, err
}

// execStartError meldet, dass die GUI aus dem memfd gar nicht erst starten
// konnte (z. B. memfd ohne Ausführungsrecht). Andere Fehler des
// Ladeprotokolls (Exit-Code, Zeitlimit) sind keine execStartError.
type execStartError struct {
	err error
}

// Error liefert den Text des ursprünglichen Fehlers.
func (e *execStartError) Error() string { return e.err.Error() }

// Unwrap liefert den ursprünglichen Fehler (für errors.Is/As).
func (e *execStartError) Unwrap() error { return e.err }

// isStartError meldet, ob err ein execStartError ist.
func isStartError(err error) bool {
	var se *execStartError
	return errors.As(err, &se)
}

// ---------------------------------------------------------------------------
// Fragen erst bei Bedarf vorbereiten
// ---------------------------------------------------------------------------

// lazyAsker baut den echten Asker erst bei der ersten Frage. NewAsker fragt
// den Benachrichtigungsdienst ab (bis zu 2 s); ohne Frage soll der normale
// Start dadurch nicht langsamer werden.
type lazyAsker struct {
	factory func() linuxsetup.Asker
	once    sync.Once
	inner   linuxsetup.Asker
}

// get liefert den echten Asker (beim ersten Aufruf gebaut).
func (a *lazyAsker) get() linuxsetup.Asker {
	a.once.Do(func() { a.inner = a.factory() })
	return a.inner
}

// Ask reicht die Frage an den echten Asker weiter.
func (a *lazyAsker) Ask(title, body string, buttons []linuxsetup.Button, timeout time.Duration) (string, error) {
	return a.get().Ask(title, body, buttons, timeout)
}

// Inform reicht den Hinweis an den echten Asker weiter.
func (a *lazyAsker) Inform(title, body string) {
	a.get().Inform(title, body)
}

// ---------------------------------------------------------------------------
// Texte des Starters (die Texte der Abläufe stehen in linuxsetup)
// ---------------------------------------------------------------------------

// launcherMsg bezeichnet einen Text des Starters.
type launcherMsg int

const (
	// lmNoPayload: Starter ohne eingebettete GUI gebaut.
	lmNoPayload launcherMsg = iota
	// lmMemExecTitle: Überschrift, wenn die GUI nicht aus dem Speicher startet.
	lmMemExecTitle
	// lmMemExecBody: Text dazu; %s = Ursache.
	lmMemExecBody
	// lmStillMissing: nach der Installation fehlen weiter Bibliotheken; %s = Liste.
	lmStillMissing
	// lmUsageDesktop: Aufruf von --print-desktop-entry.
	lmUsageDesktop
	// lmRemoved: eine entfernte Datei; %s = Pfad.
	lmRemoved
	// lmUninstallFailed: Entfernen gescheitert; %s = Ursache.
	lmUninstallFailed
	// lmInstallFailed: Installieren gescheitert; %s = Ursache.
	lmInstallFailed
	// lmWarning: Warnung, der Start geht weiter; %s = Ursache.
	lmWarning
)

// launcherTexts enthält alle Starter-Texte auf Deutsch und Englisch.
var launcherTexts = map[launcherMsg][2]string{
	lmNoPayload: {
		"MD Reader: Starter ohne eingebettete App gebaut (Build-Tag embedgui fehlt). Bitte die veröffentlichte Datei verwenden.",
		"MD Reader: launcher built without an embedded app (build tag embedgui missing). Please use the released file.",
	},
	lmMemExecTitle: {
		"MD Reader kann nicht starten",
		"MD Reader cannot start",
	},
	lmMemExecBody: {
		"Die Oberfläche ließ sich nicht aus dem Arbeitsspeicher starten. Ursache: %s\n\n" +
			"Häufiger Grund: Das System verbietet ausführbaren Speicher (vm.memfd_noexec). " +
			"Liegt die Datei auf einem Datenträger ohne Ausführungsrecht (noexec, z. B. USB-Stick), " +
			"kopieren Sie sie in einen Ordner mit Ausführungsrecht (z. B. ~/.local/bin) und starten Sie sie dort. " +
			"Hilft das nicht, installieren Sie das .deb-Paket.",
		"The user interface could not be started from memory. Cause: %s\n\n" +
			"Common reason: the system forbids executable memory (vm.memfd_noexec). " +
			"If the file is on a drive without execute permission (noexec, e.g. a USB stick), " +
			"copy it to a folder with execute permission (e.g. ~/.local/bin) and start it from there. " +
			"If that does not help, install the .deb package.",
	},
	lmStillMissing: {
		"MD Reader: Es fehlen weiterhin Bibliotheken: %s",
		"MD Reader: libraries are still missing: %s",
	},
	lmUsageDesktop: {
		"Aufruf: md-reader --print-desktop-entry <Programmpfad>",
		"usage: md-reader --print-desktop-entry <program path>",
	},
	lmRemoved: {
		"Entfernt: %s",
		"Removed: %s",
	},
	lmUninstallFailed: {
		"MD Reader: Entfernen gescheitert: %s",
		"MD Reader: uninstall failed: %s",
	},
	lmInstallFailed: {
		"MD Reader: Installation gescheitert: %s",
		"MD Reader: installation failed: %s",
	},
	lmWarning: {
		"MD Reader: Warnung: %s",
		"MD Reader: warning: %s",
	},
}

// ---------------------------------------------------------------------------
// Der Starter
// ---------------------------------------------------------------------------

// memExecutor kapselt memfd, Ladeprotokoll und exec (Linux: memexec_linux.go).
type memExecutor interface {
	// create legt die Daten in einen memfd und liefert dessen fd.
	create(data []byte) (int, error)
	// trace liefert die fehlenden Bibliotheken des Programms in fd.
	trace(fd int) ([]string, error)
	// exec ersetzt den Prozess durch das Programm in fd; kehrt nur bei Fehler zurück.
	exec(fd int, rest []string) error
}

// systemMem ist die echte Umsetzung von memExecutor.
type systemMem struct{}

func (systemMem) create(data []byte) (int, error)  { return createMemFile(data) }
func (systemMem) trace(fd int) ([]string, error)   { return traceMissing(fd) }
func (systemMem) exec(fd int, rest []string) error { return execGUI(fd, rest) }

// launcher bündelt alle Abhängigkeiten des Starters nach außen.
type launcher struct {
	// stdout und stderr sind die Ausgaben (im Betrieb os.Stdout/os.Stderr).
	stdout, stderr io.Writer
	// getenv liest Umgebungsvariablen (Sprache, HOME, XDG-Pfade).
	getenv func(string) string
	// lookPath sucht Programme (apt-get, pkexec, …).
	lookPath func(string) (string, error)
	// newAsker baut den Asker für Fragen (erst bei Bedarf aufgerufen).
	newAsker func() linuxsetup.Asker
	// newLayout berechnet die Zielpfade.
	newLayout func() (linuxsetup.Layout, error)
	// executable liefert den Pfad der laufenden Datei (os.Executable).
	executable func() (string, error)
	// runCommand führt pkexec/apt-get aus (runArgv).
	runCommand func(argv []string) (int, error)
	// mem kapselt memfd, Ladeprotokoll und exec.
	mem memExecutor
	// gui ist das eingebettete GUI-Programm (leer ohne Build-Tag embedgui).
	gui []byte
	// icons sind die eingebetteten Icons.
	icons linuxsetup.Icons
	// build ist die Build-Nummer als Text (aus den ldflags).
	build string
}

// newLauncher liefert einen Starter mit den echten Abhängigkeiten.
func newLauncher() *launcher {
	return &launcher{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		getenv:     os.Getenv,
		lookPath:   exec.LookPath,
		newAsker:   func() linuxsetup.Asker { return linuxsetup.NewAsker(exec.LookPath) },
		newLayout:  func() (linuxsetup.Layout, error) { return linuxsetup.NewLayout(os.Getenv) },
		executable: os.Executable,
		runCommand: runArgv,
		mem:        systemMem{},
		gui:        guiBinary,
		icons:      iconFiles,
		build:      buildNumber,
	}
}

// lang liefert die Sprache aus LC_ALL, LC_MESSAGES bzw. LANG.
func (l *launcher) lang() linuxsetup.Lang {
	return linuxsetup.DetectLang(l.getenv)
}

// text liefert einen Starter-Text in der Sprache des Nutzers.
func (l *launcher) text(id launcherMsg) string {
	pair := launcherTexts[id]
	if l.lang() == linuxsetup.LangDE {
		return pair[0]
	}
	return pair[1]
}

// say schreibt einen Text mit genau einem Argument und Zeilenende.
func (l *launcher) say(w io.Writer, id launcherMsg, arg string) {
	_, _ = fmt.Fprintf(w, l.text(id)+"\n", arg)
}

// warn protokolliert einen Fehler auf stderr; der Start geht weiter.
func (l *launcher) warn(err error) {
	l.say(l.stderr, lmWarning, err.Error())
}

// env baut die Umgebung für die Abläufe aus linuxsetup.
func (l *launcher) env(asker linuxsetup.Asker) linuxsetup.Env {
	return linuxsetup.Env{
		Asker:    asker,
		Lang:     l.lang(),
		Run:      l.runCommand,
		LookPath: l.lookPath,
		Stderr:   l.stderr,
	}
}

// hasPayload meldet, ob GUI und Icons eingebettet sind.
func (l *launcher) hasPayload() bool {
	return len(l.gui) > 0 && len(l.icons) > 0
}

// run führt den Starter aus.
//
// @param args Argumente ohne Programmnamen.
// @return Exit-Code (kehrt nur zurück, wenn die GUI nicht per exec startet).
func (l *launcher) run(args []string) int {
	cmd, force, rest := parseCommand(args)
	switch cmd {
	case cmdVersion:
		_, _ = fmt.Fprintln(l.stdout, l.build)
		return 0
	case cmdDesktopEntry:
		return l.printDesktopEntry(rest)
	case cmdUninstall:
		return l.uninstall()
	case cmdInstall:
		return l.install(force)
	}
	return l.startGUI(rest)
}

// printDesktopEntry gibt die Desktop-Datei für den genannten Programmpfad aus.
func (l *launcher) printDesktopEntry(rest []string) int {
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		_, _ = fmt.Fprintln(l.stderr, l.text(lmUsageDesktop))
		return 2
	}
	_, _ = io.WriteString(l.stdout, linuxsetup.DesktopEntry(rest[0], buildInt(l.build)))
	return 0
}

// uninstall entfernt die Nutzerinstallation und nennt entfernte und
// behaltene Pfade (nur Textausgabe, keine Dialoge).
func (l *launcher) uninstall() int {
	layout, err := l.newLayout()
	if err != nil {
		l.say(l.stderr, lmUninstallFailed, err.Error())
		return 1
	}
	removed, err := linuxsetup.Uninstall(layout)
	for _, p := range removed {
		l.say(l.stdout, lmRemoved, p)
	}
	if err != nil {
		l.say(l.stderr, lmUninstallFailed, err.Error())
		return 1
	}
	title, body := linuxsetup.Text(l.lang(), linuxsetup.MsgUninstalled, strings.Join(layout.KeptPaths, "\n"))
	_, _ = fmt.Fprintf(l.stdout, "\n%s\n%s\n", title, body)
	return 0
}

// install installiert die laufende Datei ins Home-Verzeichnis (nur
// Textausgabe, keine Dialoge). Bei einer Paketinstallation nur mit force.
func (l *launcher) install(force bool) int {
	layout, err := l.newLayout()
	if err != nil {
		l.say(l.stderr, lmInstallFailed, err.Error())
		return 1
	}
	if linuxsetup.PackageInstalled(layout) && !force {
		title, body := linuxsetup.Text(l.lang(), linuxsetup.MsgPackageInstalled)
		_, _ = fmt.Fprintf(l.stderr, "%s\n%s\n", title, body)
		return 1
	}
	// Ein Starter ohne Nutzlast wäre installiert nutzlos (keine GUI, keine Icons).
	if !l.hasPayload() {
		_, _ = fmt.Fprintln(l.stderr, l.text(lmNoPayload))
		return 1
	}
	_, selfPath, err := resolveExePath(l.executable, layout.BinPath)
	if err != nil {
		l.say(l.stderr, lmInstallFailed, err.Error())
		return 1
	}
	if err := linuxsetup.Install(layout, selfPath, buildInt(l.build), l.icons); err != nil {
		l.say(l.stderr, lmInstallFailed, err.Error())
		return 1
	}
	title, body := linuxsetup.Text(l.lang(), linuxsetup.MsgInstalled)
	_, _ = fmt.Fprintf(l.stdout, "%s\n%s\n", title, body)
	return 0
}

// startGUI ist der normale Start: memfd, Bibliotheken prüfen,
// Installationsangebot, exec.
func (l *launcher) startGUI(rest []string) int {
	if len(l.gui) == 0 {
		_, _ = fmt.Fprintln(l.stderr, l.text(lmNoPayload))
		return 1
	}
	asker := &lazyAsker{factory: l.newAsker}

	fd, err := l.mem.create(l.gui)
	if err != nil {
		return l.memExecFailed(asker, err)
	}
	if code, ok := l.checkLibraries(fd, asker); !ok {
		return code
	}
	l.offerInstall(asker)

	// exec kehrt im Betrieb nur bei einem Fehler zurück; nil heißt „ersetzt“
	// (nur mit einem Fake möglich) und ergibt Exit 0.
	if err := l.mem.exec(fd, rest); err != nil {
		return l.memExecFailed(asker, err)
	}
	return 0
}

// checkLibraries prüft per Ladeprotokoll, ob Bibliotheken fehlen, und bietet
// das Nachinstallieren an (danach höchstens eine zweite Prüfung).
//
// @return (0, true), wenn der Start weitergehen soll; sonst (Exit-Code, false).
func (l *launcher) checkLibraries(fd int, asker linuxsetup.Asker) (int, bool) {
	missing, err := l.mem.trace(fd)
	if isStartError(err) {
		return l.memExecFailed(asker, err), false
	}
	if len(missing) == 0 {
		if err != nil {
			// Kein eindeutiges Ergebnis: weiterstarten, exec meldet echte Fehler.
			l.warn(err)
		}
		return 0, true
	}

	if !linuxsetup.HandleMissingLibraries(l.env(asker), missing) {
		return 1, false
	}
	// Höchstens eine zweite Prüfung – keine Endlosschleife.
	missing, err = l.mem.trace(fd)
	if isStartError(err) {
		return l.memExecFailed(asker, err), false
	}
	if len(missing) > 0 {
		l.say(l.stderr, lmStillMissing, strings.Join(missing, ", "))
		return 1, false
	}
	if err != nil {
		l.warn(err)
	}
	return 0, true
}

// offerInstall bietet abhängig von der Lage die Installation ins Menü an.
// Jeder Fehler wird nur protokolliert; der Start geht weiter.
func (l *launcher) offerInstall(asker linuxsetup.Asker) {
	layout, err := l.newLayout()
	if err != nil {
		l.warn(err)
		return
	}
	exePath, selfPath, err := resolveExePath(l.executable, layout.BinPath)
	if err != nil {
		l.warn(err)
		return
	}
	ctx := linuxsetup.StartContext{
		ExePath:      exePath,
		CurrentBuild: buildInt(l.build),
		Layout:       layout,
		Package:      linuxsetup.PackageInstalled(layout),
		Prefs:        linuxsetup.LoadPrefs(layout.PrefsPath),
	}
	ctx.InstalledBuild, ctx.Installed = linuxsetup.InstalledBuild(layout)
	if err := linuxsetup.HandleInstallOffer(l.env(asker), ctx, selfPath, l.icons); err != nil {
		l.warn(err)
	}
}

// memExecFailed meldet, dass die GUI nicht aus dem Speicher starten kann
// (stderr und Hinweis), und liefert Exit 1.
func (l *launcher) memExecFailed(asker linuxsetup.Asker, cause error) int {
	title := l.text(lmMemExecTitle)
	body := fmt.Sprintf(l.text(lmMemExecBody), cause.Error())
	_, _ = fmt.Fprintf(l.stderr, "%s\n%s\n", title, body)
	asker.Inform(title, body)
	return 1
}
