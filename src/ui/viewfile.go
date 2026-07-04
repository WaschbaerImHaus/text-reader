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
// Letzte Änderung: 2026-07-04
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

	// Pfad in eine korrekte file://-URL umwandeln:
	// Windows-Pfade (C:\...) brauchen einen führenden Slash und Vorwärts-Slashes,
	// Sonderzeichen (Leerzeichen, Umlaute) werden von url.URL korrekt kodiert.
	slashPath := filepath.ToSlash(p)
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	u := url.URL{Scheme: "file", Path: slashPath}
	return u.String(), nil
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
