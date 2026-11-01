// Package ui - Temp-Datei-Navigation für große HTML-Dokumente.
//
// Hintergrund (Bug #012): WebView2 (Windows) begrenzt NavigateToString auf
// 2 MB Inhalt. Große Dokumente (z.B. EPUBs mit vielen eingebetteten Bildern)
// überschreiten das Limit → die Navigation schlägt still fehl und das Fenster
// bleibt leer. Lösung: Das HTML wird in eine temporäre Datei geschrieben und
// per file://-URL geladen. Das funktioniert identisch auf WebKitGTK (Linux)
// und WebView2 (Windows) und hat kein Größenlimit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (TempViewFileURL, AllowNavigation: Navigationsschutz F1)
package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// MaxInlineHTMLSize ist die maximale HTML-Größe (in Bytes) für direktes
// SetHtml. WebView2 erlaubt maximal 2 MB (2*1024*1024 Bytes) bei
// NavigateToString; mit 1,5 MB bleibt ein Sicherheitsabstand.
const MaxInlineHTMLSize = 1_500_000

// tempViewFilePath merkt sich den Pfad der aktuell verwendeten Temp-Datei,
// damit wiederholtes Schreiben dieselbe Datei überschreibt (kein Müll im
// Temp-Verzeichnis) und Cleanup sie zuverlässig löschen kann.
var tempViewFilePath string

// NeedsFileNavigation entscheidet, ob ein HTML-Dokument zu groß für
// direktes SetHtml ist und stattdessen als Datei geladen werden muss.
//
// @param html Vollständiges Seiten-HTML.
// @return true wenn das Dokument per file://-URL geladen werden muss.
func NeedsFileNavigation(html string) bool {
	return len(html) > MaxInlineHTMLSize
}

// NeedsFileNavigationFor entscheidet wie NeedsFileNavigation, berücksichtigt
// aber das Betriebssystem: Unter macOS hat WKWebView kein 2-MB-Limit bei
// loadHTMLString, und file://-URLs per loadRequest sind dort unzuverlässig.
// Deshalb lädt macOS jede Seite direkt.
//
// @param goos Betriebssystem (wie runtime.GOOS).
// @param html Vollständiges Seiten-HTML.
// @return true wenn das Dokument per file://-URL geladen werden muss.
func NeedsFileNavigationFor(goos, html string) bool {
	if goos == "darwin" {
		return false
	}
	return NeedsFileNavigation(html)
}

// TempViewFilePath gibt den Pfad der Temp-Ansichtsdatei zurück.
//
// Der Dateiname enthält die Prozess-ID, damit mehrere gleichzeitig laufende
// md-reader-Instanzen sich nicht gegenseitig die Datei überschreiben.
//
// @return Absoluter Pfad der Temp-Ansichtsdatei.
func TempViewFilePath() string {
	if tempViewFilePath == "" {
		tempViewFilePath = filepath.Join(os.TempDir(),
			fmt.Sprintf("md-reader-view-%d.html", os.Getpid()))
	}
	return tempViewFilePath
}

// WriteTempViewFile schreibt das HTML in die Temp-Ansichtsdatei und gibt
// die zugehörige file://-URL zurück.
//
// Die Datei wird mit 0600 angelegt (nur der Besitzer darf lesen), da sie
// den kompletten Dokumentinhalt enthält.
//
// @param html Vollständiges Seiten-HTML.
// @return file://-URL der geschriebenen Datei, oder Fehler.
func WriteTempViewFile(html string) (string, error) {
	p := TempViewFilePath()
	if err := os.WriteFile(p, []byte(html), 0600); err != nil {
		return "", fmt.Errorf("Temp-Ansichtsdatei konnte nicht geschrieben werden: %w", err)
	}
	return TempViewFileURL(), nil
}

// TempViewFileURL gibt die file://-URL der Temp-Ansichtsdatei zurück, ohne
// die Datei zu schreiben. Der Navigationsschutz (AllowNavigation) vergleicht
// damit, ob das Fenster die eigene große Seite lädt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (aus WriteTempViewFile herausgelöst, F1)
//
// @return file://-URL der Temp-Ansichtsdatei.
func TempViewFileURL() string {
	// Pfad in eine korrekte file://-URL umwandeln:
	// Windows-Pfade (C:\...) brauchen einen führenden Slash und Vorwärts-Slashes,
	// Sonderzeichen (Leerzeichen, Umlaute) werden von url.URL korrekt kodiert.
	slashPath := filepath.ToSlash(TempViewFilePath())
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	u := url.URL{Scheme: "file", Path: slashPath}
	return u.String()
}

// ownPageURI ist die Adresse, unter der die per SetHtml geladene eigene Seite
// läuft. webview_go ruft unter WebKitGTK webkit_web_view_load_html mit
// Basis-URI NULL auf – WebKit meldet die Navigation dann als "about:blank"
// (unter Xvfb gemessen, Build 47). Eine data:-URL entsteht dabei nicht,
// data: bleibt deshalb verboten.
const ownPageURI = "about:blank"

// AllowNavigation entscheidet, ob das App-Fenster zu einer Adresse navigieren
// darf (Erlaubnisliste, fail closed; Fund F1, Build 47).
//
// Hintergrund: webview_go stellt die Go-Bindings JEDER Seite im Hauptfenster
// zur Verfügung. Navigiert das Fenster zu einer fremden Seite (z. B. über den
// Kontextmenüpunkt „Verweis öffnen“, der am JS-Klick-Handler vorbeigeht),
// könnte deren Code loadNativeFile, openExternal usw. aufrufen. Erlaubt sind
// deshalb nur:
//   - die eigene SetHtml-Seite ("about:blank"), auch mit Anker (#…),
//   - genau die eigene Temp-Ansichtsdatei (tempViewURL), auch mit Anker.
//
// Der Vergleich ist bewusst streng: exakte Schreibweise, keine Query, kein
// Host, kein abweichender Pfad. Fremde Links (http/https/mailto) öffnet der
// Aufrufer stattdessen über extlink im Standardprogramm.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F1)
//
// @param rawURI      Ziel der Navigation, wie die WebView es meldet.
// @param tempViewURL file://-URL der eigenen Temp-Ansichtsdatei (leer = keine).
// @return true, wenn die Navigation im App-Fenster erlaubt ist.
func AllowNavigation(rawURI, tempViewURL string) bool {
	// Anker abtrennen: Ein Sprung innerhalb derselben Seite verlässt sie nicht.
	base, _, _ := strings.Cut(rawURI, "#")
	if base == ownPageURI {
		return true
	}
	if tempViewURL == "" || base == "" {
		return false
	}
	return sameFileURL(base, tempViewURL)
}

// sameFileURL prüft, ob zwei file://-URLs exakt dieselbe lokale Datei
// bezeichnen. Kodierungsunterschiede (z. B. %20 gegen Leerzeichen) werden
// durch das Entschlüsseln des Pfads ausgeglichen; alles andere muss gleich
// sein (Schema file, kein Host, keine Query, keine Benutzerangabe).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F1)
//
// @param a Erste URL (ohne Anker).
// @param b Zweite URL (ohne Anker).
// @return true bei exakt gleicher lokaler Datei.
func sameFileURL(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}
	for _, u := range []*url.URL{ua, ub} {
		// Nur lokale file://-URLs in Normalform (kein Host, keine Query usw.)
		if u.Scheme != "file" || u.Host != "" || u.User != nil || u.Opaque != "" ||
			u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path == "" {
			return false
		}
	}
	return ua.Path == ub.Path
}

// CleanupTempViewFile löscht die Temp-Ansichtsdatei (falls vorhanden).
//
// Wird beim Beenden der App aufgerufen, damit keine Buchinhalte im
// Temp-Verzeichnis zurückbleiben.
func CleanupTempViewFile() {
	if tempViewFilePath != "" {
		os.Remove(tempViewFilePath)
	}
}
