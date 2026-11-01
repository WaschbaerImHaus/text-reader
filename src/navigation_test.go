// Tests für den nativen Navigationsschutz des App-Fensters (Fund F1, Build 47).
//
// handleNavigationRequest ist die plattformunabhängige Entscheidung hinter dem
// WebKitGTK-decide-policy-Handler: eigene Seite erlauben, alles andere
// ablehnen; abgelehnte http/https/mailto-Links bei Nutzeraktion stattdessen
// im Standardprogramm öffnen (extlink), nie im App-Fenster.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F1)
package main

import (
	"md-reader/extlink"
	"testing"
)

// TestHandleNavigationRequest prüft Erlauben/Ablehnen und das externe Öffnen.
func TestHandleNavigationRequest(t *testing.T) {
	tempURL := "file:///tmp/md-reader-view-1.html"
	cases := []struct {
		uri        string
		userAction bool
		wantAllow  bool
		wantOpened string
	}{
		// Eigene Seiten: erlauben, nichts extern öffnen
		{"about:blank", false, true, ""},
		{tempURL, false, true, ""},
		// Kontextmenü „Verweis öffnen“ auf fremden Link: ablehnen, extern öffnen
		{"http://127.0.0.1:8765/index.html", true, false, "http://127.0.0.1:8765/index.html"},
		{"mailto:a@b.c", true, false, "mailto:a@b.c"},
		// Ohne Nutzeraktion (z. B. automatische Weiterleitung): nur ablehnen
		{"https://example.com/", false, false, ""},
		// Nicht erlaubte Schemata: ablehnen, extlink weist sie selbst zurück
		{"file:///etc/passwd", true, false, ""},
		{"javascript:alert(1)", true, false, ""},
		{"data:text/html,x", true, false, ""},
	}
	for _, c := range cases {
		opened := ""
		open := func(raw string) error {
			// Wie der echte Opener: erst prüfen, dann „starten“
			safe, err := extlink.Validate(raw)
			if err != nil {
				return err
			}
			opened = safe
			return nil
		}
		got := handleNavigationRequest(c.uri, c.userAction, tempURL, open)
		if got != c.wantAllow {
			t.Errorf("handleNavigationRequest(%q, %v) = %v, erwartet %v", c.uri, c.userAction, got, c.wantAllow)
		}
		if opened != c.wantOpened {
			t.Errorf("handleNavigationRequest(%q, %v) öffnete extern %q, erwartet %q", c.uri, c.userAction, opened, c.wantOpened)
		}
	}
}
