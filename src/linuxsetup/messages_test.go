// Tests für messages.go – Spracherkennung, Nachrichtentexte und
// Button-Beschriftungen (Deutsch und Englisch).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Korrekturrunde: Standardpfade für MsgUninstalled)
package linuxsetup

import (
	"strings"
	"testing"
)

// TestDetectLang prüft die Reihenfolge LC_ALL, LC_MESSAGES, LANG: Die erste
// gesetzte Variable entscheidet, Deutsch nur bei Präfix "de".
func TestDetectLang(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want Lang
	}{
		{"LANG deutsch", map[string]string{"LANG": "de_DE.UTF-8"}, LangDE},
		{"LANG nur de", map[string]string{"LANG": "de"}, LangDE},
		{"LANG österreichisch", map[string]string{"LANG": "de_AT.UTF-8"}, LangDE},
		{"LC_ALL=C vor LANG", map[string]string{"LC_ALL": "C", "LANG": "de_DE"}, LangEN},
		{"LC_MESSAGES vor LANG", map[string]string{"LC_MESSAGES": "de_CH.UTF-8", "LANG": "en_US"}, LangDE},
		{"LC_ALL vor LC_MESSAGES", map[string]string{"LC_ALL": "en_GB", "LC_MESSAGES": "de_DE"}, LangEN},
		{"leeres LC_ALL zählt nicht", map[string]string{"LC_ALL": "", "LANG": "de_DE"}, LangDE},
		{"alles leer", map[string]string{}, LangEN},
		{"fremde Sprache", map[string]string{"LANG": "fr_FR.UTF-8"}, LangEN},
		{"nur Präfix zählt", map[string]string{"LANG": "en_DE.UTF-8"}, LangEN},
		{"Großschreibung ist kein de", map[string]string{"LANG": "DE_de"}, LangEN},
	}
	for _, c := range cases {
		if got := DetectLang(envFrom(c.env)); got != c.want {
			t.Errorf("%s: DetectLang = %q, erwartet %q", c.name, got, c.want)
		}
	}
}

// TestText_AllIDsBothLangs prüft, dass jede Nachricht in beiden Sprachen
// Titel und Text hat – auch ohne Argumente.
func TestText_AllIDsBothLangs(t *testing.T) {
	for id := MessageID(0); id < messageCount; id++ {
		for _, lang := range []Lang{LangDE, LangEN} {
			title, body := Text(lang, id, "x")
			if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
				t.Errorf("Text(%q, %d) mit Argument: Titel %q, Text %q – leer", lang, id, title, body)
			}
			title, body = Text(lang, id)
			if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
				t.Errorf("Text(%q, %d) ohne Argument: Titel %q, Text %q – leer", lang, id, title, body)
			}
		}
	}
	if messageCount != 8 {
		t.Errorf("messageCount = %d, erwartet 8 Nachrichten", messageCount)
	}

	_, body := Text(LangDE, MsgInstalled)
	if body != "MD Reader ist jetzt im Menü unter Büro." {
		t.Errorf("Text(de, MsgInstalled) = %q", body)
	}
	_, body = Text(LangEN, MsgInstalled)
	if body != "MD Reader is now in the menu under Office." {
		t.Errorf("Text(en, MsgInstalled) = %q", body)
	}
}

// TestText_UnknownLangFallsBackToEnglish prüft den Rückfall auf Englisch.
func TestText_UnknownLangFallsBackToEnglish(t *testing.T) {
	ft, fb := Text(Lang("fr"), MsgInstalled)
	et, eb := Text(LangEN, MsgInstalled)
	if ft != et || fb != eb {
		t.Errorf("Text(fr) = (%q, %q), erwartet Englisch (%q, %q)", ft, fb, et, eb)
	}
}

// TestText_UnknownID prüft, dass eine unbekannte ID nicht abstürzt und
// zumindest einen Titel liefert.
func TestText_UnknownID(t *testing.T) {
	title, _ := Text(LangDE, MessageID(999))
	if title == "" {
		t.Error("unbekannte ID: leerer Titel")
	}
	title, _ = Text(LangEN, MessageID(-1))
	if title == "" {
		t.Error("negative ID: leerer Titel")
	}
}

// TestText_WebKitMissingListsLibs prüft Titel und die Nennung der fehlenden
// Bibliotheken.
func TestText_WebKitMissingListsLibs(t *testing.T) {
	libs := "libjavascriptcoregtk-4.1.so.0, libwebkit2gtk-4.1.so.0"
	title, body := Text(LangDE, MsgWebKitMissing, libs)
	if title != "WebKitGTK ist nicht installiert" {
		t.Errorf("Titel de = %q", title)
	}
	if !strings.Contains(body, "libwebkit2gtk-4.1.so.0") || !strings.Contains(body, "libjavascriptcoregtk-4.1.so.0") {
		t.Errorf("Text de nennt die Bibliotheken nicht: %q", body)
	}
	if !strings.Contains(body, WebKitPackage) {
		t.Errorf("Text de nennt das Paket %s nicht: %q", WebKitPackage, body)
	}
	title, body = Text(LangEN, MsgWebKitMissing, libs)
	if title != "WebKitGTK is not installed" {
		t.Errorf("Titel en = %q", title)
	}
	if !strings.Contains(body, "libwebkit2gtk-4.1.so.0") {
		t.Errorf("Text en nennt die Bibliothek nicht: %q", body)
	}
	// Ohne Argument nennt der Text die Hauptbibliothek.
	_, body = Text(LangDE, MsgWebKitMissing)
	if !strings.Contains(body, "libwebkit2gtk-4.1.so.0") {
		t.Errorf("Text ohne Argument nennt libwebkit2gtk-4.1.so.0 nicht: %q", body)
	}
}

// TestText_HintAndUninstallUseArgs prüft, dass Befehl und behaltene Pfade
// unverändert im Text stehen.
func TestText_HintAndUninstallUseArgs(t *testing.T) {
	hint := InstallHint(PMDnf)
	for _, lang := range []Lang{LangDE, LangEN} {
		_, body := Text(lang, MsgWebKitHint, hint)
		if !strings.Contains(body, hint) {
			t.Errorf("%s: MsgWebKitHint ohne Befehl: %q", lang, body)
		}
		kept := "/home/a b/.config/md-reader\n/home/a b/.cache/md-reader"
		_, body = Text(lang, MsgUninstalled, kept)
		if !strings.Contains(body, kept) {
			t.Errorf("%s: MsgUninstalled ohne Pfade: %q", lang, body)
		}
	}
	// Ohne Argument nennt MsgUninstalled die drei üblichen Pfade.
	for _, lang := range []Lang{LangDE, LangEN} {
		_, body := Text(lang, MsgUninstalled)
		for _, p := range []string{"~/.config/md-reader", "~/.cache/md-reader", "~/.local/share/md-reader"} {
			if !strings.Contains(body, p) {
				t.Errorf("%s: MsgUninstalled ohne Argument nennt %s nicht: %q", lang, p, body)
			}
		}
	}
	// Prozentzeichen im Argument dürfen nicht als Formatanweisung wirken.
	_, body := Text(LangEN, MsgUninstalled, "/home/100%s/.config")
	if !strings.Contains(body, "/home/100%s/.config") || strings.Contains(body, "%!") {
		t.Errorf("Prozent im Pfad verfälscht: %q", body)
	}
}

// TestButtonLabel prüft alle Button-Schlüssel in beiden Sprachen.
func TestButtonLabel(t *testing.T) {
	keys := []string{"install", "cancel", "update", "yes", "later", "never", "replace", "notnow"}
	for _, key := range keys {
		for _, lang := range []Lang{LangDE, LangEN} {
			if strings.TrimSpace(ButtonLabel(lang, key)) == "" {
				t.Errorf("ButtonLabel(%q, %q) ist leer", lang, key)
			}
		}
		if ButtonLabel(LangDE, key) == ButtonLabel(LangEN, key) {
			t.Errorf("ButtonLabel(%q): Deutsch gleich Englisch (%q)", key, ButtonLabel(LangDE, key))
		}
	}
	want := map[string]string{
		"install": "Jetzt installieren",
		"cancel":  "Abbrechen",
		"update":  "Paketlisten aktualisieren und erneut versuchen",
		"yes":     "Ja",
		"later":   "Später",
		"never":   "Nie",
		"replace": "Ersetzen",
		"notnow":  "Nicht jetzt",
	}
	for key, label := range want {
		if got := ButtonLabel(LangDE, key); got != label {
			t.Errorf("ButtonLabel(de, %q) = %q, erwartet %q", key, got, label)
		}
	}
	// Unbekannter Schlüssel: nie leer (Schlüssel selbst), unbekannte Sprache: Englisch.
	if got := ButtonLabel(LangDE, "unknown"); got != "unknown" {
		t.Errorf("unbekannter Schlüssel = %q", got)
	}
	if ButtonLabel(Lang("fr"), "later") != ButtonLabel(LangEN, "later") {
		t.Error("unbekannte Sprache fällt nicht auf Englisch zurück")
	}
}

// TestButtons prüft den Helfer, der Schlüssel zu Buttons mit Beschriftung macht.
func TestButtons(t *testing.T) {
	got := Buttons(LangDE, "yes", "later", "never")
	want := []Button{{"yes", "Ja"}, {"later", "Später"}, {"never", "Nie"}}
	if len(got) != len(want) {
		t.Fatalf("Buttons = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Buttons[%d] = %v, erwartet %v", i, got[i], want[i])
		}
	}
}
