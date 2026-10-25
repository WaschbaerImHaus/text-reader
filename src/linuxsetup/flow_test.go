// Tests für flow.go – Ablaufsteuerung bei fehlender WebKitGTK-Bibliothek
// (Nachinstallieren per apt) und beim Installationsangebot.
//
// Alle Tests arbeiten mit einem Fake-Asker (vorgegebene Antworten) und einem
// Fake-Run (zeichnet die Argumentlisten auf, führt nichts aus). stderr geht
// in einen Puffer, damit die Testausgabe sauber bleibt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 5: Ablaufsteuerung)
package linuxsetup

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// askCall ist eine aufgezeichnete Frage des Fake-Askers.
type askCall struct {
	title, body string
	buttons     []Button
	timeout     time.Duration
}

// informCall ist ein aufgezeichneter Hinweis des Fake-Askers.
type informCall struct {
	title, body string
}

// scriptedAsker beantwortet Fragen der Reihe nach mit den vorgegebenen
// Schlüsseln und zeichnet alle Fragen und Hinweise auf. Sind die Antworten
// aufgebraucht, liefert Ask "" (wie ein Zeitablauf).
type scriptedAsker struct {
	answers []string
	askErr  error
	asks    []askCall
	informs []informCall
}

// Ask liefert die nächste vorgegebene Antwort.
func (a *scriptedAsker) Ask(title, body string, buttons []Button, timeout time.Duration) (string, error) {
	a.asks = append(a.asks, askCall{title, body, buttons, timeout})
	if a.askErr != nil {
		return "", a.askErr
	}
	if len(a.answers) == 0 {
		return "", nil
	}
	key := a.answers[0]
	a.answers = a.answers[1:]
	return key, nil
}

// Inform zeichnet den Hinweis auf.
func (a *scriptedAsker) Inform(title, body string) {
	a.informs = append(a.informs, informCall{title, body})
}

// runResult ist ein vorgegebenes Ergebnis des Fake-Run.
type runResult struct {
	code int
	err  error
}

// scriptedRun liefert eine Run-Funktion, die die Argumentlisten in *calls
// aufzeichnet und die Ergebnisse der Reihe nach zurückgibt. Sind sie
// aufgebraucht, scheitert der Test (unerwarteter Aufruf).
func scriptedRun(t *testing.T, calls *[][]string, results ...runResult) func([]string) (int, error) {
	t.Helper()
	return func(argv []string) (int, error) {
		*calls = append(*calls, append([]string(nil), argv...))
		if len(results) == 0 {
			t.Errorf("unerwarteter Aufruf von Run: %v", argv)
			return 1, nil
		}
		r := results[0]
		results = results[1:]
		return r.code, r.err
	}
}

// missingEnv baut eine Env für HandleMissingLibraries mit apt-get.
func missingEnv(t *testing.T, asker *scriptedAsker, calls *[][]string, results ...runResult) (Env, *bytes.Buffer) {
	t.Helper()
	var stderr bytes.Buffer
	return Env{
		Asker:    asker,
		Lang:     LangDE,
		Run:      scriptedRun(t, calls, results...),
		LookPath: fakeLookPath("apt-get", "pkexec"),
		Stderr:   &stderr,
	}, &stderr
}

// testMissing ist die Liste fehlender Bibliotheken in den Tests.
var testMissing = []string{"libjavascriptcoregtk-4.1.so.0", "libwebkit2gtk-4.1.so.0"}

// checkKeys prüft die Schlüssel der Knöpfe einer Frage.
func checkKeys(t *testing.T, got []Button, want ...string) {
	t.Helper()
	keys := make([]string, 0, len(got))
	for _, b := range got {
		keys = append(keys, b.Key)
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("Knöpfe = %v, erwartet %v", keys, want)
	}
}

// checkStderrHint prüft, dass Titel, fehlende Bibliotheken und der
// Installationshinweis auf stderr stehen (für Starts im Terminal).
func checkStderrHint(t *testing.T, stderr *bytes.Buffer, hint string) {
	t.Helper()
	out := stderr.String()
	title, _ := Text(LangDE, MsgWebKitMissing)
	for _, want := range []string{title, testMissing[0], testMissing[1], hint} {
		if !strings.Contains(out, want) {
			t.Errorf("stderr enthält %q nicht:\n%s", want, out)
		}
	}
}

// TestHandleMissing_InstallOK: "install", Run liefert 0 → true, genau ein Aufruf.
func TestHandleMissing_InstallOK(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install"}}
	var calls [][]string
	e, stderr := missingEnv(t, asker, &calls, runResult{0, nil})

	if !HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet true nach erfolgreicher Installation")
	}
	if want := [][]string{AptInstallCommand()}; !reflect.DeepEqual(calls, want) {
		t.Errorf("Run-Aufrufe = %v, erwartet %v", calls, want)
	}
	if len(asker.asks) != 1 {
		t.Fatalf("Fragen = %d, erwartet 1", len(asker.asks))
	}
	ask := asker.asks[0]
	checkKeys(t, ask.buttons, "install", "cancel")
	if ask.buttons[0].Label != ButtonLabel(LangDE, "install") || ask.buttons[1].Label != ButtonLabel(LangDE, "cancel") {
		t.Errorf("Beschriftungen = %v", ask.buttons)
	}
	if ask.timeout != 120*time.Second {
		t.Errorf("Timeout = %v, erwartet 120s", ask.timeout)
	}
	wantTitle, wantBody := Text(LangDE, MsgWebKitMissing, strings.Join(testMissing, ", "))
	if ask.title != wantTitle || ask.body != wantBody {
		t.Errorf("Frage = %q / %q, erwartet %q / %q", ask.title, ask.body, wantTitle, wantBody)
	}
	if len(asker.informs) != 0 {
		t.Errorf("unerwartete Hinweise: %v", asker.informs)
	}
	checkStderrHint(t, stderr, InstallHint(PMApt))
}

// TestHandleMissing_Cancelled: "cancel" → false, kein Run.
func TestHandleMissing_Cancelled(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"cancel"}}
	var calls [][]string
	e, stderr := missingEnv(t, asker, &calls)

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false nach Abbrechen")
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
	if len(asker.asks) != 1 {
		t.Errorf("Fragen = %d, erwartet 1", len(asker.asks))
	}
	checkStderrHint(t, stderr, InstallHint(PMApt))
}

// TestHandleMissing_NoAnswer: Zeitablauf ("") gilt wie "cancel" (harmloser Knopf).
func TestHandleMissing_NoAnswer(t *testing.T) {
	asker := &scriptedAsker{}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls)

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false ohne Antwort")
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
}

// TestHandleMissing_UnknownKey: ein unbekannter Schlüssel startet nie apt.
func TestHandleMissing_UnknownKey(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"INSTALL"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls)

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false bei unbekanntem Schlüssel")
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
}

// TestHandleMissing_AskError: technischer Fehler der Frage → kein Run, aber Hinweis.
func TestHandleMissing_AskError(t *testing.T) {
	asker := &scriptedAsker{askErr: errors.New("kaputt")}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls)

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false bei Fehler der Frage")
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
	if len(asker.informs) != 1 {
		t.Errorf("Hinweise = %d, erwartet 1", len(asker.informs))
	}
}

// TestHandleMissing_AuthAborted: pkexec 126 (Passwortabfrage abgebrochen) und
// 127 (kein pkexec) → false; genau ein Run, eine Frage, ein Hinweis (Review Focus 4).
func TestHandleMissing_AuthAborted(t *testing.T) {
	for _, code := range []int{126, 127} {
		asker := &scriptedAsker{answers: []string{"install", "update"}}
		var calls [][]string
		e, _ := missingEnv(t, asker, &calls, runResult{code, nil})

		if HandleMissingLibraries(e, testMissing) {
			t.Fatalf("Exit %d: erwartet false", code)
		}
		if len(calls) != 1 {
			t.Errorf("Exit %d: Run-Aufrufe = %d, erwartet 1", code, len(calls))
		}
		if len(asker.asks) != 1 {
			t.Errorf("Exit %d: Fragen = %d, erwartet 1", code, len(asker.asks))
		}
		if len(asker.informs) != 1 {
			t.Fatalf("Exit %d: Hinweise = %d, erwartet 1", code, len(asker.informs))
		}
		wantTitle, wantBody := Text(LangDE, MsgWebKitHint, InstallHint(PMApt))
		if got := asker.informs[0]; got.title != wantTitle || got.body != wantBody {
			t.Errorf("Exit %d: Hinweis = %q / %q", code, got.title, got.body)
		}
	}
}

// TestHandleMissing_StartError: pkexec lässt sich nicht starten (Fehler ohne
// Exit-Code) → wie 127: keine weitere Frage, nur der Hinweis.
func TestHandleMissing_StartError(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "update"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{-1, errors.New("exec: not found")})

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false")
	}
	if len(calls) != 1 || len(asker.asks) != 1 || len(asker.informs) != 1 {
		t.Errorf("Run=%d Fragen=%d Hinweise=%d, erwartet 1/1/1", len(calls), len(asker.asks), len(asker.informs))
	}
}

// TestHandleMissing_RetryWithUpdate: 100, "update", 0, 0 → true; install, update, install.
func TestHandleMissing_RetryWithUpdate(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "update"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{100, nil}, runResult{0, nil}, runResult{0, nil})

	if !HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet true nach erfolgreichem zweiten Versuch")
	}
	want := [][]string{AptInstallCommand(), AptUpdateCommand(), AptInstallCommand()}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("Run-Aufrufe = %v, erwartet %v", calls, want)
	}
	if len(asker.asks) != 2 {
		t.Fatalf("Fragen = %d, erwartet 2", len(asker.asks))
	}
	second := asker.asks[1]
	checkKeys(t, second.buttons, "update", "cancel")
	if second.timeout != 120*time.Second {
		t.Errorf("Timeout = %v, erwartet 120s", second.timeout)
	}
	wantTitle, wantBody := Text(LangDE, MsgInstallFailed)
	if second.title != wantTitle || second.body != wantBody {
		t.Errorf("zweite Frage = %q / %q", second.title, second.body)
	}
	if len(asker.informs) != 0 {
		t.Errorf("unerwartete Hinweise: %v", asker.informs)
	}
}

// TestHandleMissing_RetryFails: 100, "update", 0, 100 → false, keine dritte Frage.
func TestHandleMissing_RetryFails(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "update", "update"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{100, nil}, runResult{0, nil}, runResult{100, nil})

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false nach gescheitertem zweiten Versuch")
	}
	if len(calls) != 3 {
		t.Errorf("Run-Aufrufe = %d, erwartet 3", len(calls))
	}
	if len(asker.asks) != 2 {
		t.Errorf("Fragen = %d, erwartet 2 (höchstens ein Wiederholungsversuch)", len(asker.asks))
	}
	if len(asker.informs) != 1 {
		t.Errorf("Hinweise = %d, erwartet 1", len(asker.informs))
	}
}

// TestHandleMissing_RetryCancelled: 100, "cancel" → false, kein update.
func TestHandleMissing_RetryCancelled(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "cancel"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{100, nil})

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false")
	}
	if len(calls) != 1 {
		t.Errorf("Run-Aufrufe = %v, erwartet nur install", calls)
	}
}

// TestHandleMissing_UpdateAuthAborted: Abbruch der Passwortabfrage beim
// update (126) → kein zweites install, nur der Hinweis.
func TestHandleMissing_UpdateAuthAborted(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "update"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{100, nil}, runResult{126, nil})

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false")
	}
	want := [][]string{AptInstallCommand(), AptUpdateCommand()}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("Run-Aufrufe = %v, erwartet %v", calls, want)
	}
	if len(asker.informs) != 1 {
		t.Errorf("Hinweise = %d, erwartet 1", len(asker.informs))
	}
}

// TestHandleMissing_UpdatePartlyFailed: update meldet einen anderen Fehler
// (z. B. eine Paketquelle nicht erreichbar) → install wird trotzdem versucht.
func TestHandleMissing_UpdatePartlyFailed(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install", "update"}}
	var calls [][]string
	e, _ := missingEnv(t, asker, &calls, runResult{100, nil}, runResult{100, nil}, runResult{0, nil})

	if !HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet true, install nach teilweise gescheitertem update klappt")
	}
	if len(calls) != 3 {
		t.Errorf("Run-Aufrufe = %d, erwartet 3", len(calls))
	}
}

// TestHandleMissing_NoApt: ohne apt-get → false, keine Frage, kein Run;
// der Hinweis nennt den dnf-Befehl, wenn nur dnf vorhanden ist.
func TestHandleMissing_NoApt(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install"}}
	var calls [][]string
	var stderr bytes.Buffer
	e := Env{
		Asker:    asker,
		Lang:     LangDE,
		Run:      scriptedRun(t, &calls),
		LookPath: fakeLookPath("dnf", "pkexec"),
		Stderr:   &stderr,
	}

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false ohne apt-get")
	}
	if len(asker.asks) != 0 {
		t.Errorf("Fragen = %d, erwartet 0", len(asker.asks))
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
	if len(asker.informs) != 1 {
		t.Fatalf("Hinweise = %d, erwartet 1", len(asker.informs))
	}
	if !strings.Contains(asker.informs[0].body, "dnf install webkit2gtk4.1") {
		t.Errorf("Hinweis nennt dnf nicht: %q", asker.informs[0].body)
	}
	checkStderrHint(t, &stderr, "dnf install webkit2gtk4.1")
}

// TestHandleMissing_NoPkexec: apt-get vorhanden, aber kein pkexec → keine
// Frage, kein Run; nur der Hinweis mit dem apt-Befehl (wie ohne apt).
func TestHandleMissing_NoPkexec(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"install"}}
	var calls [][]string
	var stderr bytes.Buffer
	e := Env{
		Asker:    asker,
		Lang:     LangDE,
		Run:      scriptedRun(t, &calls),
		LookPath: fakeLookPath("apt-get"),
		Stderr:   &stderr,
	}

	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false ohne pkexec")
	}
	if len(asker.asks) != 0 {
		t.Errorf("Fragen = %d, erwartet 0", len(asker.asks))
	}
	if len(calls) != 0 {
		t.Errorf("Run aufgerufen: %v", calls)
	}
	if len(asker.informs) != 1 {
		t.Fatalf("Hinweise = %d, erwartet 1", len(asker.informs))
	}
	if !strings.Contains(asker.informs[0].body, "sudo apt install libwebkit2gtk-4.1-0") {
		t.Errorf("Hinweis nennt den apt-Befehl nicht: %q", asker.informs[0].body)
	}
	checkStderrHint(t, &stderr, "sudo apt install libwebkit2gtk-4.1-0")
}

// TestHandleMissing_EnglishStderr: englische Texte auch auf stderr.
func TestHandleMissing_EnglishStderr(t *testing.T) {
	asker := &scriptedAsker{answers: []string{"cancel"}}
	var calls [][]string
	e, stderr := missingEnv(t, asker, &calls)
	e.Lang = LangEN

	HandleMissingLibraries(e, testMissing)
	title, _ := Text(LangEN, MsgWebKitMissing)
	if !strings.Contains(stderr.String(), title) {
		t.Errorf("stderr ohne englischen Titel:\n%s", stderr.String())
	}
	if len(asker.asks) != 1 || asker.asks[0].buttons[0].Label != "Install now" {
		t.Errorf("englische Beschriftung fehlt: %v", asker.asks)
	}
}

// TestHandleMissing_NilFields: ohne Asker, Run und Stderr stürzt nichts ab;
// ohne Run wird nie etwas installiert.
func TestHandleMissing_NilFields(t *testing.T) {
	e := Env{Lang: LangEN, LookPath: fakeLookPath("apt-get")}
	// Ohne Asker gibt es keine Antwort, also auch keine Installation.
	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false ohne Asker")
	}
	e.Asker = &scriptedAsker{answers: []string{"install"}}
	if HandleMissingLibraries(e, testMissing) {
		t.Fatal("erwartet false ohne Run")
	}
}

// offerContext baut einen StartContext in einem temporären HOME und legt die
// gestartete Datei an.
func offerContext(t *testing.T) (StartContext, string) {
	t.Helper()
	l := newTestLayout(t)
	self := writeSource(t, "neues-programm")
	return StartContext{ExePath: self, CurrentBuild: 47, Layout: l}, self
}

// TestHandleInstallOffer prüft das Installationsangebot je Antwort.
func TestHandleInstallOffer(t *testing.T) {
	tests := []struct {
		name          string
		answer        string
		wantInstalled bool
		wantPrompt    string
	}{
		{"Ja installiert", "yes", true, ""},
		{"Später merkt later", "later", false, PromptLater},
		{"Nie merkt never", "never", false, PromptNever},
		{"Zeitablauf gilt als Später", "", false, PromptLater},
		{"unbekannter Schlüssel gilt als Später", "bogus", false, PromptLater},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubRunQuiet(t)
			c, self := offerContext(t)
			asker := &scriptedAsker{answers: []string{tc.answer}}
			e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

			if err := HandleInstallOffer(e, c, self, testIcons()); err != nil {
				t.Fatalf("HandleInstallOffer: %v", err)
			}
			if len(asker.asks) != 1 {
				t.Fatalf("Fragen = %d, erwartet 1", len(asker.asks))
			}
			ask := asker.asks[0]
			checkKeys(t, ask.buttons, "yes", "never", "later")
			if ask.timeout != 120*time.Second {
				t.Errorf("Timeout = %v, erwartet 120s", ask.timeout)
			}
			wantTitle, wantBody := Text(LangDE, MsgOfferInstall)
			if ask.title != wantTitle || ask.body != wantBody {
				t.Errorf("Frage = %q / %q", ask.title, ask.body)
			}

			build, installed := InstalledBuild(c.Layout)
			if installed != tc.wantInstalled {
				t.Errorf("installiert = %v, erwartet %v", installed, tc.wantInstalled)
			}
			if tc.wantInstalled {
				if build != 47 {
					t.Errorf("Build = %d, erwartet 47", build)
				}
				if data, err := os.ReadFile(c.Layout.BinPath); err != nil || string(data) != "neues-programm" {
					t.Errorf("Programm nicht kopiert: %q, %v", data, err)
				}
				if len(asker.informs) != 1 {
					t.Fatalf("Hinweise = %d, erwartet 1", len(asker.informs))
				}
				wantTitle, wantBody := Text(LangDE, MsgInstalled)
				if got := asker.informs[0]; got.title != wantTitle || got.body != wantBody {
					t.Errorf("Hinweis = %q / %q", got.title, got.body)
				}
			} else if len(asker.informs) != 0 {
				t.Errorf("unerwartete Hinweise: %v", asker.informs)
			}

			if got := LoadPrefs(c.Layout.PrefsPath).InstallPrompt; got != tc.wantPrompt {
				t.Errorf("Prefs = %q, erwartet %q", got, tc.wantPrompt)
			}
			if tc.wantPrompt == "" {
				if _, err := os.Stat(c.Layout.PrefsPath); err == nil {
					t.Errorf("Prefs-Datei sollte nicht existieren")
				}
			}
		})
	}
}

// TestHandleInstallOffer_Nothing: bei OfferNothing (Paket, Nie, schon
// installiert) wird nicht gefragt und nichts geschrieben.
func TestHandleInstallOffer_Nothing(t *testing.T) {
	cases := map[string]func(*StartContext){
		"Paket installiert": func(c *StartContext) { c.Package = true },
		"Antwort Nie":       func(c *StartContext) { c.Prefs = Prefs{InstallPrompt: PromptNever} },
		"gleicher Build":    func(c *StartContext) { c.Installed, c.InstalledBuild = true, 47 },
		"neuere Installation (keine Herabstufung)": func(c *StartContext) {
			c.Installed, c.InstalledBuild = true, 48
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			calls := stubRunQuiet(t)
			c, self := offerContext(t)
			mutate(&c)
			asker := &scriptedAsker{answers: []string{"yes", "replace"}}
			e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

			if err := HandleInstallOffer(e, c, self, testIcons()); err != nil {
				t.Fatalf("HandleInstallOffer: %v", err)
			}
			if len(asker.asks) != 0 || len(asker.informs) != 0 {
				t.Errorf("Fragen=%d Hinweise=%d, erwartet 0/0", len(asker.asks), len(asker.informs))
			}
			if _, err := os.Stat(c.Layout.BinPath); err == nil {
				t.Errorf("Programm wurde installiert")
			}
			if _, err := os.Stat(c.Layout.PrefsPath); err == nil {
				t.Errorf("Prefs wurden geschrieben")
			}
			if len(*calls) != 0 {
				t.Errorf("Hilfsprogramme aufgerufen: %v", *calls)
			}
		})
	}
}

// installedContext legt eine ältere Nutzerinstallation (Build 46) an und
// liefert den passenden StartContext für Build 47.
func installedContext(t *testing.T) (StartContext, string) {
	t.Helper()
	c, self := offerContext(t)
	old := writeSource(t, "altes-programm")
	if err := Install(c.Layout, old, 46, testIcons()); err != nil {
		t.Fatalf("Install (alt): %v", err)
	}
	build, installed := InstalledBuild(c.Layout)
	c.Installed, c.InstalledBuild = installed, build
	return c, self
}

// TestHandleInstallOffer_Update prüft das Ersetzen einer älteren Installation.
func TestHandleInstallOffer_Update(t *testing.T) {
	tests := []struct {
		name        string
		answer      string
		wantReplace bool
	}{
		{"Ersetzen", "replace", true},
		{"Nicht jetzt", "later", false},
		{"Zeitablauf", "", false},
		{"yes ist kein Knopf dieser Frage", "yes", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubRunQuiet(t)
			c, self := installedContext(t)
			// Auch nach "Nie" wird das Update angeboten; die Antwort bleibt unberührt.
			if err := SavePrefs(c.Layout.PrefsPath, Prefs{InstallPrompt: PromptNever}); err != nil {
				t.Fatal(err)
			}
			c.Prefs = LoadPrefs(c.Layout.PrefsPath)
			asker := &scriptedAsker{answers: []string{tc.answer}}
			e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

			if err := HandleInstallOffer(e, c, self, testIcons()); err != nil {
				t.Fatalf("HandleInstallOffer: %v", err)
			}
			if len(asker.asks) != 1 {
				t.Fatalf("Fragen = %d, erwartet 1", len(asker.asks))
			}
			ask := asker.asks[0]
			checkKeys(t, ask.buttons, "replace", "later")
			if ask.buttons[0].Label != ButtonLabel(LangDE, "replace") || ask.buttons[1].Label != ButtonLabel(LangDE, "notnow") {
				t.Errorf("Beschriftungen = %v", ask.buttons)
			}
			if ask.timeout != 120*time.Second {
				t.Errorf("Timeout = %v, erwartet 120s", ask.timeout)
			}
			wantTitle, wantBody := Text(LangDE, MsgOfferUpdate)
			if ask.title != wantTitle || ask.body != wantBody {
				t.Errorf("Frage = %q / %q", ask.title, ask.body)
			}

			build, _ := InstalledBuild(c.Layout)
			data, _ := os.ReadFile(c.Layout.BinPath)
			if tc.wantReplace {
				if build != 47 || string(data) != "neues-programm" {
					t.Errorf("nicht ersetzt: Build %d, Inhalt %q", build, data)
				}
				if len(asker.informs) != 1 {
					t.Errorf("Hinweise = %d, erwartet 1 (Bestätigung)", len(asker.informs))
				}
			} else {
				if build != 46 || string(data) != "altes-programm" {
					t.Errorf("unerwartet ersetzt: Build %d, Inhalt %q", build, data)
				}
				if len(asker.informs) != 0 {
					t.Errorf("unerwartete Hinweise: %v", asker.informs)
				}
			}
			if got := LoadPrefs(c.Layout.PrefsPath).InstallPrompt; got != PromptNever {
				t.Errorf("Prefs = %q, Update darf sie nicht ändern", got)
			}
		})
	}
}

// TestHandleInstallOffer_InstallError: scheitert die Installation, kommt der
// Fehler zurück, es gibt keine Bestätigung und keine gespeicherte Antwort.
func TestHandleInstallOffer_InstallError(t *testing.T) {
	stubRunQuiet(t)
	c, _ := offerContext(t)
	missing := filepath.Join(t.TempDir(), "gibt-es-nicht")
	asker := &scriptedAsker{answers: []string{"yes"}}
	e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

	if err := HandleInstallOffer(e, c, missing, testIcons()); err == nil {
		t.Fatal("erwartet Fehler bei fehlender Quelldatei")
	}
	if len(asker.informs) != 0 {
		t.Errorf("unerwartete Bestätigung: %v", asker.informs)
	}
	if _, err := os.Stat(c.Layout.PrefsPath); err == nil {
		t.Errorf("Prefs wurden geschrieben")
	}
}

// TestHandleInstallOffer_SavePrefsError: lässt sich die Antwort nicht
// speichern, kommt der Fehler zurück.
func TestHandleInstallOffer_SavePrefsError(t *testing.T) {
	stubRunQuiet(t)
	c, self := offerContext(t)
	// Eine Datei an Stelle des Verzeichnisses verhindert das Anlegen.
	blocker := filepath.Dir(c.Layout.PrefsPath)
	if err := os.MkdirAll(filepath.Dir(blocker), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	asker := &scriptedAsker{answers: []string{"never"}}
	e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

	if err := HandleInstallOffer(e, c, self, testIcons()); err == nil {
		t.Fatal("erwartet Fehler beim Speichern")
	}
}

// TestHandleInstallOffer_AskError: technischer Fehler der Frage → Fehler
// zurück, keine Installation; die Antwort gilt als "Später".
func TestHandleInstallOffer_AskError(t *testing.T) {
	stubRunQuiet(t)
	c, self := offerContext(t)
	asker := &scriptedAsker{askErr: errors.New("kaputt")}
	e := Env{Asker: asker, Lang: LangDE, Stderr: &bytes.Buffer{}}

	if err := HandleInstallOffer(e, c, self, testIcons()); err == nil {
		t.Fatal("erwartet Fehler der Frage")
	}
	if _, installed := InstalledBuild(c.Layout); installed {
		t.Errorf("trotz Fehler installiert")
	}
	if got := LoadPrefs(c.Layout.PrefsPath).InstallPrompt; got != PromptLater {
		t.Errorf("Prefs = %q, erwartet later", got)
	}
}
