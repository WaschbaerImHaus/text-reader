// Package extlink öffnet externe Links im System-Browser bzw. Mailprogramm.
//
// Hintergrund (RISK-008): Ein Klick auf einen http(s)-Link navigierte früher
// das App-Fenster selbst zur fremden Website. webview_go stellt die
// Go-Bindings aber JEDER geladenen Seite zur Verfügung – die fremde Seite
// hätte also App-Funktionen aufrufen können. Deshalb öffnet die App externe
// Links nur noch über das Betriebssystem und verlässt nie die eigene Seite.
//
// Sicherheitsregeln für URLs:
//   - nur http, https (mit Host) und mailto
//   - keine Steuer-, Leer- oder Anführungszeichen (Schutz der Kommandozeile)
//   - höchstens maxURLLength Zeichen
//
// Programme werden direkt ohne Shell gestartet (keine Shell-Interpretation).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14
package extlink

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// maxURLLength begrenzt die URL-Länge (übliche Browser-Grenze für Links).
const maxURLLength = 2048

// allowedSchemes enthält die einzigen erlaubten URL-Schemata.
var allowedSchemes = map[string]bool{"http": true, "https": true, "mailto": true}

// ErrInvalidURL wird für jede abgelehnte URL zurückgegeben.
var ErrInvalidURL = errors.New("ungültiger oder nicht erlaubter externer Link")

// Opener öffnet geprüfte externe Links mit dem Standardprogramm des Systems.
type Opener struct {
	// goos ist das Ziel-Betriebssystem ("linux", "windows", "darwin").
	goos string
	// start startet ein Programm ohne Shell und wartet NICHT auf dessen Ende.
	start func(name string, args ...string) error
}

// NewOpener erzeugt einen Öffner für das laufende Betriebssystem.
//
// @return Einsatzbereiter Opener.
func NewOpener() *Opener {
	return &Opener{goos: runtime.GOOS, start: startDetached}
}

// Open prüft die URL und öffnet sie im Standardprogramm.
//
// @param rawURL Link-Ziel aus dem Dokument.
// @return Fehler bei ungültiger URL (dann wird nichts gestartet) oder Startfehler.
func (o *Opener) Open(rawURL string) error {
	safeURL, err := Validate(rawURL)
	if err != nil {
		return err
	}
	name, args, err := commandFor(o.goos, safeURL)
	if err != nil {
		return err
	}
	return o.start(name, args...)
}

// Validate prüft eine externe URL und gibt sie normalisiert zurück.
//
// @param rawURL Zu prüfende URL (führende/folgende Leerzeichen werden entfernt).
// @return Normalisierte URL oder ErrInvalidURL.
func Validate(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" || len(trimmed) > maxURLLength {
		return "", ErrInvalidURL
	}
	// Steuerzeichen, Leerzeichen und Anführungszeichen könnten die
	// Kommandozeile (v. a. unter Windows) oder die URL-Auswertung verändern
	for _, r := range trimmed {
		if r <= 0x20 || r == 0x7f || r == '"' {
			return "", ErrInvalidURL
		}
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", ErrInvalidURL
	}
	scheme := strings.ToLower(parsed.Scheme)
	if !allowedSchemes[scheme] {
		return "", ErrInvalidURL
	}
	// http(s) braucht einen Host, mailto eine Adresse
	if scheme == "mailto" {
		if parsed.Opaque == "" {
			return "", ErrInvalidURL
		}
	} else if parsed.Host == "" {
		return "", ErrInvalidURL
	}
	parsed.Scheme = scheme
	normalized := parsed.String()
	// Sicherheitsnetz: auch die Neuserialisierung darf keine verbotenen Zeichen enthalten
	if strings.ContainsAny(normalized, "\" \t\r\n") {
		return "", ErrInvalidURL
	}
	return normalized, nil
}

// commandFor liefert Programm und Argumente zum Öffnen einer URL.
//
// @param goos    Betriebssystem.
// @param safeURL Bereits geprüfte URL.
// @return Programmname, Argumente oder Fehler bei unbekanntem System.
func commandFor(goos, safeURL string) (string, []string, error) {
	switch goos {
	case "linux":
		return "xdg-open", []string{safeURL}, nil
	case "windows":
		// Übergibt die URL an den registrierten Standard-Handler (Browser/Mail)
		return "rundll32", []string{"url.dll,FileProtocolHandler", safeURL}, nil
	case "darwin":
		return "open", []string{safeURL}, nil
	}
	return "", nil, fmt.Errorf("externe Links auf %s nicht unterstützt", goos)
}

// startDetached startet ein Programm ohne Shell und räumt den Prozess im
// Hintergrund auf, sobald er endet (keine Zombie-Prozesse).
//
// @param name Programmname.
// @param args Argumente.
// @return Startfehler (z. B. Programm nicht installiert).
func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
