// Desktop-Datei von MD Reader (freedesktop.org Desktop Entry Specification).
//
// Die Datei wird bei der Selbst-Installation nach
// $XDG_DATA_HOME/applications/md-reader.desktop geschrieben. Der Schlüssel
// X-MD-Reader-Build speichert die Build-Nummer der installierten Version,
// damit der Starter erkennt, ob eine neuere Datei gestartet wurde.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 2: Desktop-Datei)
package linuxsetup

import (
	"strconv"
	"strings"
)

// AppID ist die Desktop-ID (md-reader.desktop), der Icon-Name und die
// StartupWMClass der Anwendung.
const AppID = "md-reader"

// buildKey ist der eigene Schlüssel der Desktop-Datei für die Build-Nummer.
const buildKey = "X-MD-Reader-Build"

// desktopMimeTypes sind alle Formate, die MD Reader öffnen kann.
const desktopMimeTypes = "text/markdown;text/x-markdown;text/plain;" +
	"application/epub+zip;application/x-fictionbook+xml;text/x-tex;" +
	"text/html;application/pdf;application/postscript;"

// DesktopEntry erzeugt den Inhalt der Desktop-Datei.
//
// @param execPath Absoluter Pfad des Programms (wird für Exec= gequotet).
// @param build Build-Nummer, wird als X-MD-Reader-Build gespeichert.
// @return Dateiinhalt mit "\n" als Zeilenende.
func DesktopEntry(execPath string, build int) string {
	lines := []string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=MD Reader",
		"GenericName=Document Viewer",
		"GenericName[de]=Dokumentbetrachter",
		"Comment=View Markdown, EPUB, FB2, LaTeX, PDF and more",
		"Comment[de]=Markdown, EPUB, FB2, LaTeX, PDF und mehr anzeigen",
		"Exec=" + quoteExec(execPath) + " %f",
		"Icon=" + AppID,
		"Terminal=false",
		"Categories=Office;Viewer;",
		"MimeType=" + desktopMimeTypes,
		"StartupWMClass=" + AppID,
		buildKey + "=" + strconv.Itoa(build),
	}
	return strings.Join(lines, "\n") + "\n"
}

// quoteExec setzt einen Programmpfad für die Exec-Zeile in Anführungszeichen.
//
// Regeln der Desktop-Entry-Spezifikation:
//  1. Innerhalb von Anführungszeichen werden ", `, $ und \ mit einem
//     Backslash maskiert.
//  2. Danach gilt das String-Format der Datei: jeder Backslash wird verdoppelt.
//  3. Ein % würde als Feldcode (%f, %u ...) gelesen und wird zu %% verdoppelt.
//
// @param path Programmpfad, darf Leerzeichen und Sonderzeichen enthalten.
// @return Gequoteter Pfad inklusive der umschließenden Anführungszeichen.
func quoteExec(path string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range path {
		switch r {
		case '"', '`', '$':
			// Schritt 1 (ein Backslash) und Schritt 2 (verdoppelt) in einem: zwei Backslashes.
			b.WriteString(`\\`)
			b.WriteRune(r)
		case '\\':
			// Der Backslash selbst: Schritt 1 ergibt zwei, Schritt 2 verdoppelt auf vier.
			b.WriteString(`\\\\`)
		case '%':
			b.WriteString("%%")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// BuildFromDesktopEntry liest die Build-Nummer (X-MD-Reader-Build) aus einer
// vorhandenen Desktop-Datei.
//
// @param content Dateiinhalt.
// @return Build-Nummer und true; (0, false), wenn der Schlüssel fehlt oder keine Zahl ist.
func BuildFromDesktopEntry(content string) (int, bool) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		value, found := strings.CutPrefix(line, buildKey+"=")
		if !found {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
