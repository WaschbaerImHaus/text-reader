//go:build linux

// Ausführen der eingebetteten Oberfläche aus dem Arbeitsspeicher (Linux).
//
// Ablauf:
//  1. createMemFile legt per memfd_create eine anonyme Datei im Speicher an
//     ("/memfd:md-reader") und schreibt das GUI-Programm hinein. Es entsteht
//     keine Datei auf der Platte; deshalb startet die GUI auch dann, wenn der
//     Starter auf einem noexec-Datenträger liegt oder anders heißt.
//  2. traceMissing startet das Programm mit LD_TRACE_LOADED_OBJECTS=1: Der
//     dynamische Lader führt es dann nicht aus, sondern listet nur die
//     Bibliotheken (wie ldd) – fehlende stehen als "=> not found" darin.
//  3. execGUI ersetzt den Starter per execve durch das Programm.
//
// Geprüfte Einzelheiten (Kernel 6.8, siehe main_test.go):
//   - Der memfd trägt MFD_CLOEXEC (andere Kindprozesse wie pkexec erben ihn
//     nicht). execve("/proc/self/fd/N") funktioniert trotzdem, weil der Kern
//     die Programmdatei öffnet, bevor er die CLOEXEC-fds schließt. Das gilt
//     für ELF-Programme; Skripte mit "#!" würden scheitern (der Interpreter
//     fände den Pfad nicht mehr) – die GUI ist immer ein ELF-Programm.
//   - Für das Ladeprotokoll bekommt der Kindprozess eine Kopie der fd über
//     exec.Cmd.ExtraFiles als fd 3 (dup2 löscht dort CLOEXEC) und startet
//     "/proc/self/fd/3" – "self" ist dabei schon das Kind. Die Variante über
//     den Pfad des Elternprozesses (/proc/<pid>/fd/N) wird nicht gebraucht.
//   - MFD_EXEC (Kernel ab 6.3) wird ausdrücklich gesetzt: Bei
//     vm.memfd_noexec=1 wäre der memfd sonst nicht ausführbar, bei =2 scheitert
//     memfd_create ab Kernel 6.6 sofort mit EACCES – beides wird so zu einer
//     klaren Fehlermeldung statt eines rätselhaften exec-Fehlers. Ältere Kerne
//     kennen MFD_EXEC nicht (EINVAL); dann folgt ein zweiter Versuch ohne das
//     Flag. Kernel 6.3 bis 6.5 liefern bei vm.memfd_noexec=2 ebenfalls EINVAL
//     für MFD_EXEC; der zweite Versuch legt dann einen nicht ausführbaren
//     memfd an, und spätestens das Ladeprotokoll scheitert mit EACCES – das
//     endet in derselben klaren Meldung (Rat zu memfd/noexec).
//   - Nach execGUI ist argv[0] "md-reader" (GTK-WM_CLASS passt), aber der
//     Kern setzt den Prozessnamen (/proc/<pid>/comm) auf den letzten Teil des
//     exec-Pfads, also die fd-Nummer (z. B. "4"); /proc/<pid>/exe zeigt
//     "/memfd:md-reader (deleted)". "pgrep -f md-reader" findet die GUI,
//     "pgrep -x md-reader" nicht.
//   - Angebotsprozess: Der Starter startet sich über /proc/self/exe selbst
//     neu; dessen os.Executable ist wieder die Starter-Datei, nicht der memfd
//     (siehe spawnOfferProcess).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6, Korrekturrunde 1: Angebotsprozess, Lader-stderr)
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"md-reader/linuxsetup"
)

// Flags von memfd_create (linux/memfd.h).
const (
	// mfdCloexec: fd wird bei exec geschlossen.
	mfdCloexec = 0x1
	// mfdExec: Inhalt ausdrücklich ausführbar (Kernel ab 6.3).
	mfdExec = 0x10
)

// memfdName ist der Name des memfd (sichtbar als "/memfd:md-reader").
const memfdName = "md-reader"

// guiArgv0 ist argv[0] der Oberfläche (GTK leitet daraus WM_CLASS ab,
// passend zu StartupWMClass=md-reader).
const guiArgv0 = "md-reader"

// traceTimeout begrenzt das Ladeprotokoll (der Lader beendet sich sofort;
// das Zeitlimit schützt nur vor einem hängenden Programm).
const traceTimeout = 30 * time.Second

// memfdCreateNr ist die Systemaufrufnummer von memfd_create je Architektur
// (das Paket syscall kennt sie nur für arm64).
var memfdCreateNr = map[string]uintptr{
	"amd64":   319,
	"arm64":   279,
	"arm":     385,
	"386":     356,
	"riscv64": 279,
	"loong64": 279,
}

// memfdCreate ruft memfd_create(name, flags) auf.
func memfdCreate(name string, flags uintptr) (int, error) {
	nr, ok := memfdCreateNr[runtime.GOARCH]
	if !ok {
		return -1, fmt.Errorf("memfd_create ist für %s nicht bekannt", runtime.GOARCH)
	}
	p, err := syscall.BytePtrFromString(name)
	if err != nil {
		return -1, err
	}
	fd, _, errno := syscall.Syscall(nr, uintptr(unsafe.Pointer(p)), flags, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

// createMemFile legt einen memfd an und schreibt data vollständig hinein.
//
// @param data Inhalt (das GUI-Programm).
// @return fd des memfd (mit CLOEXEC); Fehler, wenn memfd_create oder das
// Schreiben scheitert (die fd ist dann geschlossen).
func createMemFile(data []byte) (fd int, err error) {
	fd, err = memfdCreate(memfdName, mfdCloexec|mfdExec)
	if errors.Is(err, syscall.EINVAL) {
		// Kernel vor 6.3: MFD_EXEC unbekannt, ausführbar ist der memfd dort immer.
		fd, err = memfdCreate(memfdName, mfdCloexec)
	}
	if err != nil {
		return -1, fmt.Errorf("memfd_create: %w", err)
	}
	for rest := data; len(rest) > 0; {
		n, werr := syscall.Write(fd, rest)
		if werr == syscall.EINTR {
			continue
		}
		if werr == nil && n <= 0 {
			werr = errors.New("write schrieb nichts")
		}
		if werr != nil {
			syscall.Close(fd)
			return -1, fmt.Errorf("memfd schreiben: %w", werr)
		}
		rest = rest[n:]
	}
	return fd, nil
}

// dupCloexec liefert eine Kopie der fd mit CLOEXEC (F_DUPFD_CLOEXEC).
func dupCloexec(fd int) (int, error) {
	r, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_DUPFD_CLOEXEC, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(r), nil
}

// traceMissing startet das Programm im memfd mit LD_TRACE_LOADED_OBJECTS=1
// und wertet die Liste der Bibliotheken aus.
//
// Die fd wird für das Kind kopiert (die Kopie gehört dem os.File und wird
// danach geschlossen); die fd des Aufrufers bleibt offen für execGUI.
//
// @param fd memfd aus createMemFile.
// @return Fehlende Bibliotheken (leer, wenn alles da ist). *execStartError,
// wenn das Programm gar nicht startet (z. B. memfd nicht ausführbar);
// anderer Fehler bei Exit-Code ungleich 0 ohne fehlende Bibliothek.
func traceMissing(fd int) ([]string, error) {
	dup, err := dupCloexec(fd)
	if err != nil {
		return nil, &execStartError{err: fmt.Errorf("fd kopieren: %w", err)}
	}
	file := os.NewFile(uintptr(dup), "/memfd:"+memfdName)
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), traceTimeout)
	defer cancel()
	// ExtraFiles[0] wird im Kind zu fd 3.
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3")
	cmd.Args = []string{guiArgv0}
	// Bei doppelten Schlüsseln gilt in os/exec der letzte Eintrag.
	cmd.Env = append(os.Environ(), "LD_TRACE_LOADED_OBJECTS=1")
	cmd.ExtraFiles = []*os.File{file}
	var out bytes.Buffer
	cmd.Stdout = &out
	// Meldungen des Laders (z. B. "version `GLIBC_2.40' not found") kommen
	// im Ladeprotokoll auf stderr, oft sogar mit Exit 0; begrenzt sammeln.
	loaderErr := &limitedBuffer{max: loaderStderrLimit}
	cmd.Stderr = loaderErr
	if err := cmd.Start(); err != nil {
		return nil, &execStartError{err: err}
	}
	waitErr := cmd.Wait()

	if missing := linuxsetup.ParseMissingLibraries(out.String()); len(missing) > 0 {
		return missing, nil
	}
	loader := strings.TrimSpace(loaderErr.String())
	switch {
	case waitErr != nil && loader != "":
		return nil, fmt.Errorf("Ladeprotokoll: %w; Lader: %s", waitErr, loader)
	case waitErr != nil:
		return nil, fmt.Errorf("Ladeprotokoll: %w", waitErr)
	case loader != "":
		return nil, fmt.Errorf("Ladeprotokoll meldet: %s", loader)
	}
	return nil, nil
}

// execGUI ersetzt den laufenden Prozess durch das Programm im memfd.
//
// @param fd memfd aus createMemFile.
// @param rest Argumente für die Oberfläche (argv[0] ist immer "md-reader").
// @return Kehrt nur bei einem Fehler zurück.
func execGUI(fd int, rest []string) error {
	argv := append([]string{guiArgv0}, rest...)
	err := syscall.Exec("/proc/self/fd/"+strconv.Itoa(fd), argv, os.Environ())
	return fmt.Errorf("exec: %w", err)
}

// selfExe ist der Pfad, über den sich der Starter selbst erneut startet.
// Im neuen Prozess zeigt /proc/self/exe wieder auf die Starter-Datei.
const selfExe = "/proc/self/exe"

// detachStageTimeout begrenzt das Warten auf die Zwischenstufe (sie startet
// nur einen Prozess und endet; normal dauert das wenige Millisekunden).
const detachStageTimeout = 10 * time.Second

// terminalStderr liefert os.Stderr, wenn es ein Terminal ist, sonst nil
// (= /dev/null). So sieht ein Nutzer im Terminal die Frage des stderr-Askers;
// in Logdateien oder Pipes des Desktops landet nichts aus dem Hintergrund.
func terminalStderr() io.Writer {
	if fi, err := os.Stderr.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		return os.Stderr
	}
	return nil
}

// spawnOfferProcess startet den Angebotsprozess, ohne auf die Antwort des
// Nutzers zu warten.
//
// Doppelter Start: Der Starter wartet nur auf die Zwischenstufe
// ("--offer-install --detach"). Sie startet den eigentlichen Angebotsprozess
// in einer eigenen Sitzung (Setsid) und endet sofort. Grund: Der Starter
// ersetzt sich gleich per exec durch die GUI. Ein direktes Kind bliebe nach
// seinem Ende bis zum Schließen der GUI als Zombie stehen, weil die GUI es
// nicht abholt. Das Kind der Zwischenstufe übernimmt dagegen init bzw. der
// Subreaper der Sitzung. stdin und stdout sind /dev/null, stderr wird nur bei
// einem Terminal geerbt.
//
// @return Fehler, wenn die Zwischenstufe nicht startet oder scheitert.
func spawnOfferProcess() error {
	ctx, cancel := context.WithTimeout(context.Background(), detachStageTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, selfExe, offerInstallFlag, optionDetach)
	cmd.Args[0] = guiArgv0
	cmd.Stderr = terminalStderr()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Angebotsprozess: %w", err)
	}
	return nil
}

// startDetachedOffer startet "--offer-install" in einer eigenen Sitzung
// (Setsid, unabhängig vom Terminal und von der GUI) und wartet nicht.
func startDetachedOffer() error {
	cmd := exec.Command(selfExe, offerInstallFlag)
	cmd.Args[0] = guiArgv0
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = terminalStderr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Angebotsprozess: %w", err)
	}
	return cmd.Process.Release()
}
