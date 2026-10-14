// Tests für das Öffnen externer Links im System-Browser (RISK-008).
//
// Prüft die URL-Validierung (nur http/https/mailto, keine Steuerzeichen,
// keine Zeichen, die eine Kommandozeile verändern könnten) und die
// plattformspezifischen Befehle – ohne echte Programme zu starten.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14
package extlink

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestValidate_AcceptsSafeURLs prüft, dass normale Links akzeptiert werden.
func TestValidate_AcceptsSafeURLs(t *testing.T) {
	cases := map[string]string{
		"https://example.org":                      "https://example.org",
		"http://example.org/pfad?a=1&b=2#anker":    "http://example.org/pfad?a=1&b=2#anker",
		"HTTPS://Example.org/x":                    "https://Example.org/x",
		"  https://example.org/  ":                 "https://example.org/",
		"mailto:info@example.org":                  "mailto:info@example.org",
		"mailto:info@example.org?subject=Hallo":    "mailto:info@example.org?subject=Hallo",
		"https://example.org/%C3%A4rger?q=%22x%22": "https://example.org/%C3%A4rger?q=%22x%22",
	}
	for in, want := range cases {
		got, err := Validate(in)
		if err != nil {
			t.Errorf("Validate(%q) unerwarteter Fehler: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Validate(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

// TestValidate_RejectsDangerousSchemes prüft, dass nur http/https/mailto erlaubt sind.
func TestValidate_RejectsDangerousSchemes(t *testing.T) {
	for _, in := range []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"file:///etc/passwd",
		"file://C:/Windows/System32/calc.exe",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"ftp://example.org/x",
		"ms-settings:privacy",
		"search-ms:query=x",
		"\\\\server\\share\\x.exe",
		"C:\\Windows\\System32\\calc.exe",
		"/etc/passwd",
		"relativ/pfad.html",
		"#anker",
	} {
		if _, err := Validate(in); err == nil {
			t.Errorf("Validate(%q) hätte abgelehnt werden müssen", in)
		}
	}
}

// TestValidate_RejectsMalformed prüft Sonderfälle: leer, ohne Host, Steuerzeichen,
// Anführungszeichen/Leerzeichen (Kommandozeilen-Schutz) und Überlänge.
func TestValidate_RejectsMalformed(t *testing.T) {
	for _, in := range []string{
		"",
		"   ",
		"https://",
		"http:///pfad",
		"mailto:",
		"https://example.org/\x00x",
		"https://example.org/\nx",
		"https://example.org/\tx",
		"https://example.org/a b",
		"https://example.org/\"x",
		"https://example.org/?q=\" & calc",
		"https://example.org/\x7f",
		"https://example.org/" + strings.Repeat("a", 2100),
	} {
		if _, err := Validate(in); err == nil {
			t.Errorf("Validate(%q) hätte abgelehnt werden müssen", in)
		}
	}
}

// TestCommandFor prüft die plattformspezifischen Befehle (ohne Shell).
func TestCommandFor(t *testing.T) {
	u := "https://example.org/x?a=1&b=2"
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"linux", "xdg-open", []string{u}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", u}},
		{"darwin", "open", []string{u}},
	}
	for _, c := range cases {
		name, args, err := commandFor(c.goos, u)
		if err != nil {
			t.Errorf("%s: unerwarteter Fehler: %v", c.goos, err)
			continue
		}
		if name != c.wantName || !reflect.DeepEqual(args, c.wantArgs) {
			t.Errorf("%s: %s %v, erwartet %s %v", c.goos, name, args, c.wantName, c.wantArgs)
		}
	}
	if _, _, err := commandFor("plan9", u); err == nil {
		t.Error("Unbekanntes Betriebssystem hätte einen Fehler liefern müssen")
	}
}

// TestOpener_RunsValidatedURL prüft, dass Open den Befehl mit der geprüften URL startet.
func TestOpener_RunsValidatedURL(t *testing.T) {
	var gotName string
	var gotArgs []string
	o := &Opener{goos: "linux", start: func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	}}
	if err := o.Open("  https://example.org/  "); err != nil {
		t.Fatalf("Open: unerwarteter Fehler: %v", err)
	}
	if gotName != "xdg-open" || !reflect.DeepEqual(gotArgs, []string{"https://example.org/"}) {
		t.Errorf("Gestartet: %s %v", gotName, gotArgs)
	}
}

// TestOpener_RejectsWithoutStarting prüft, dass bei ungültiger URL NICHTS gestartet wird.
func TestOpener_RejectsWithoutStarting(t *testing.T) {
	started := false
	o := &Opener{goos: "windows", start: func(string, ...string) error {
		started = true
		return nil
	}}
	if err := o.Open("file:///C:/Windows/System32/calc.exe"); err == nil {
		t.Error("Open hätte einen Fehler liefern müssen")
	}
	if started {
		t.Error("Bei ungültiger URL wurde trotzdem ein Programm gestartet")
	}
}

// TestOpener_PropagatesStartError prüft, dass Startfehler (z. B. xdg-open fehlt)
// an den Aufrufer weitergegeben werden.
func TestOpener_PropagatesStartError(t *testing.T) {
	startErr := errors.New("xdg-open nicht gefunden")
	o := &Opener{goos: "linux", start: func(string, ...string) error { return startErr }}
	if err := o.Open("https://example.org"); !errors.Is(err, startErr) {
		t.Errorf("Fehler %v, erwartet %v", err, startErr)
	}
}

// TestNewOpener prüft, dass der Standard-Öffner vollständig initialisiert ist.
func TestNewOpener(t *testing.T) {
	o := NewOpener()
	if o.goos == "" || o.start == nil {
		t.Error("NewOpener liefert unvollständigen Öffner")
	}
}
