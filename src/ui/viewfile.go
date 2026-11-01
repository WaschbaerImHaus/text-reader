// Package ui - Temp-Datei-Navigation für große HTML-Dokumente.
//
// Hintergrund (Bug #012): WebView2 (Windows) begrenzt NavigateToString auf
// 2 MB Inhalt. Große Dokumente (z.B. EPUBs mit vielen eingebetteten Bildern)
// überschreiten das Limit → die Navigation schlägt still fehl und das Fenster
// bleibt leer. Lösung: Das HTML wird in eine temporäre Datei geschrieben und
// per file://-URL geladen. Das funktioniert identisch auf WebKitGTK (Linux)
// und WebView2 (Windows) und hat kein Größenlimit.
//
// Sicherheit (Fund N1, Build 47): Die Datei ist der einzige file://-Pfad,
// zu dem das Fenster navigieren darf. Sie liegt deshalb in einem privaten
// Verzeichnis mit Zufallsnamen (os.MkdirTemp, 0700) und wird selbst mit
// Zufallsnamen exklusiv angelegt (os.CreateTemp: O_EXCL, 0600). Eine von
// einem anderen Nutzer vorab abgelegte Datei oder ein Symlink wird so nie
// benutzt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (privates Verzeichnis, Zufallsname, O_EXCL: N1)
package ui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MaxInlineHTMLSize ist die maximale HTML-Größe (in Bytes) für direktes
// SetHtml. WebView2 erlaubt maximal 2 MB (2*1024*1024 Bytes) bei
// NavigateToString; mit 1,5 MB bleibt ein Sicherheitsabstand.
const MaxInlineHTMLSize = 1_500_000

// tempViewStore verwaltet die Temp-Ansichtsdatei einer Programmsitzung
// (Fund N1, Build 47).
//
// Beim ersten Schreiben entstehen ein privates Verzeichnis mit Zufallsnamen
// (0700) und darin die Datei mit Zufallsnamen (0600, exklusiv angelegt).
// Weitere Schreibvorgänge überschreiben genau diese Datei, damit die URL
// gleich bleibt – aber nur, wenn sie noch dieselbe eigene Datei ist
// (os.SameFile über das offene Handle). Fehlt sie oder wurde sie ersetzt
// (z. B. durch einen Symlink), wird eine neue Datei angelegt.
//
// Der Mutex schützt die Felder, weil der Navigationsschutz die URL aus
// einem GTK-Rückruf liest.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
type tempViewStore struct {
	mu   sync.Mutex
	dir  string      // privates Verzeichnis (leer = noch keine Datei)
	path string      // Pfad der aktuellen Temp-Ansichtsdatei
	info os.FileInfo // Kennung der selbst angelegten Datei (für SameFile)
}

// tempView ist die Temp-Ansichtsdatei dieses Prozesses.
var tempView tempViewStore

// errTempViewReplaced meldet, dass die eigene Datei nicht mehr dieselbe ist.
var errTempViewReplaced = errors.New("Temp-Ansichtsdatei wurde ersetzt")

// write schreibt das HTML in die Temp-Ansichtsdatei und gibt deren Pfad
// zurück. Ist die bisherige Datei nicht mehr nutzbar, entsteht eine neue.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @param html Vollständiges Seiten-HTML.
// @return Pfad der geschriebenen Datei, oder Fehler.
func (s *tempViewStore) write(html string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path != "" {
		if err := s.overwriteLocked(html); err == nil {
			return s.path, nil
		}
		// Datei (oder Verzeichnis) fehlt oder wurde ersetzt → alles neu anlegen
		s.discardLocked()
	}
	if err := s.createLocked(html); err != nil {
		return "", err
	}
	return s.path, nil
}

// createLocked legt privates Verzeichnis und Datei neu an und schreibt das
// HTML hinein. Aufrufer hält den Mutex.
//
// os.MkdirTemp erzeugt das Verzeichnis mit Zufallsnamen und Rechten 0700,
// os.CreateTemp die Datei mit Zufallsnamen per O_CREATE|O_EXCL und 0600 –
// beide scheitern, statt einem vorhandenen Eintrag oder Symlink zu folgen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @param html Vollständiges Seiten-HTML.
// @return Fehler beim Anlegen oder Schreiben.
func (s *tempViewStore) createLocked(html string) error {
	dir, err := os.MkdirTemp("", "md-reader-view-")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "view-*.html")
	if err != nil {
		os.RemoveAll(dir)
		return err
	}
	info, err := f.Stat()
	if err == nil {
		_, err = f.WriteString(html)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.RemoveAll(dir)
		return err
	}
	s.dir, s.path, s.info = dir, f.Name(), info
	return nil
}

// overwriteLocked überschreibt die vorhandene eigene Datei. Aufrufer hält
// den Mutex.
//
// Die Datei wird ohne O_CREATE und ohne O_TRUNC geöffnet; erst wenn das
// offene Handle nachweislich dieselbe Datei wie beim Anlegen ist, wird sie
// gekürzt und beschrieben. Ein zwischenzeitlich eingeschobener Symlink
// verändert so nie ein fremdes Ziel.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @param html Vollständiges Seiten-HTML.
// @return Fehler, wenn die Datei fehlt, ersetzt wurde oder nicht schreibbar ist.
func (s *tempViewStore) overwriteLocked(html string) error {
	if st, err := os.Lstat(s.path); err != nil || !st.Mode().IsRegular() {
		return errTempViewReplaced
	}
	f, err := os.OpenFile(s.path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil || !os.SameFile(st, s.info) {
		f.Close()
		return errTempViewReplaced
	}
	err = f.Truncate(0)
	if err == nil {
		_, err = f.WriteString(html)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// discardLocked löscht das private Verzeichnis samt Inhalt und vergisst
// die Datei. Aufrufer hält den Mutex. RemoveAll folgt Symlinks nicht, ein
// eingeschobener Symlink wird nur selbst entfernt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
func (s *tempViewStore) discardLocked() {
	if s.dir != "" {
		os.RemoveAll(s.dir)
	}
	s.dir, s.path, s.info = "", "", nil
}

// currentPath gibt den Pfad der aktuellen Datei zurück (leer = keine).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, N1)
//
// @return Absoluter Pfad oder "".
func (s *tempViewStore) currentPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path
}

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

// TempViewFilePath gibt den Pfad der aktuellen Temp-Ansichtsdatei zurück.
//
// Der Pfad ist zufällig (privates Verzeichnis, Zufallsname; N1) und erst
// nach dem ersten WriteTempViewFile bekannt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (Zufallspfad statt Prozess-ID, N1)
//
// @return Absoluter Pfad, oder "" wenn (noch) keine Datei existiert.
func TempViewFilePath() string {
	return tempView.currentPath()
}

// WriteTempViewFile schreibt das HTML in die Temp-Ansichtsdatei und gibt
// die zugehörige file://-URL zurück.
//
// Die Datei liegt in einem privaten Verzeichnis (0700), ist selbst 0600
// (nur der Besitzer darf lesen, sie enthält den kompletten Dokumentinhalt)
// und wird exklusiv angelegt – siehe tempViewStore (N1). Wiederholtes
// Schreiben in derselben Sitzung behält die URL bei.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (privat und exklusiv, N1)
//
// @param html Vollständiges Seiten-HTML.
// @return file://-URL der geschriebenen Datei, oder Fehler.
func WriteTempViewFile(html string) (string, error) {
	p, err := tempView.write(html)
	if err != nil {
		return "", fmt.Errorf("Temp-Ansichtsdatei konnte nicht geschrieben werden: %w", err)
	}
	return fileURLFromPath(p), nil
}

// TempViewFileURL gibt die file://-URL der Temp-Ansichtsdatei zurück, ohne
// die Datei zu schreiben. Der Navigationsschutz (AllowNavigation) vergleicht
// damit, ob das Fenster die eigene große Seite lädt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (leer, solange keine Datei existiert, N1)
//
// @return file://-URL der Temp-Ansichtsdatei, oder "" (keine Datei, also keine file://-Navigation erlaubt).
func TempViewFileURL() string {
	p := TempViewFilePath()
	if p == "" {
		return ""
	}
	return fileURLFromPath(p)
}

// fileURLFromPath wandelt einen absoluten Pfad in eine file://-URL um.
//
// Windows-Pfade (C:\...) brauchen einen führenden Slash und Vorwärts-Slashes,
// Sonderzeichen (Leerzeichen, Umlaute) werden von url.URL korrekt kodiert.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (aus TempViewFileURL herausgelöst, N1)
//
// @param p Absoluter Dateipfad.
// @return file://-URL.
func fileURLFromPath(p string) string {
	slashPath := filepath.ToSlash(p)
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

// CleanupTempViewFile löscht die Temp-Ansichtsdatei samt privatem
// Verzeichnis (falls vorhanden).
//
// Wird beim Beenden der App aufgerufen, damit keine Buchinhalte im
// Temp-Verzeichnis zurückbleiben. Danach liefert TempViewFileURL "".
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (privates Verzeichnis löschen, N1)
func CleanupTempViewFile() {
	tempView.mu.Lock()
	defer tempView.mu.Unlock()
	tempView.discardLocked()
}
