// Package renderer – Suche nach externen Programmen (pdftoppm, Ghostscript)
// und plattformabhängige Entscheidungen beim PDF/PS-Rendering.
//
// macOS (Build 46): Apps aus dem Finder erben nur einen minimalen PATH
// (/usr/bin:/bin:/usr/sbin:/sbin). Homebrew legt Programme aber unter
// /opt/homebrew/bin (Apple Silicon) bzw. /usr/local/bin (Intel) ab,
// MacPorts unter /opt/local/bin. Diese Orte werden zusätzlich durchsucht.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24
package renderer

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// toolSearchDirs liefert zusätzliche Verzeichnisse, in denen nach externen
// Programmen gesucht wird, falls sie nicht im PATH stehen.
//
// @param goos Betriebssystem (wie runtime.GOOS).
// @return Feste Verzeichnisse (nur macOS), sonst nil.
func toolSearchDirs(goos string) []string {
	if goos == "darwin" {
		return []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin"}
	}
	return nil
}

// findToolInDirs sucht ein ausführbares Programm in den angegebenen Verzeichnissen.
//
// Der Name muss ein reiner Dateiname sein (kein Pfadtrenner, nicht "." oder ".."),
// damit die Suche die Verzeichnisse nicht verlassen kann.
//
// @param name Programmname, z. B. "pdftoppm".
// @param dirs Zu durchsuchende Verzeichnisse in Prioritätsreihenfolge.
// @return Vollständiger Pfad des ersten Treffers oder "".
func findToolInDirs(name string, dirs []string) string {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ""
	}
	for _, dir := range dirs {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		// Mindestens ein Ausführungsbit muss gesetzt sein
		if info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate
	}
	return ""
}

// lookupTool sucht ein externes Programm zuerst im PATH, danach in den
// zusätzlichen Orten des aktuellen Systems (siehe toolSearchDirs).
//
// @param name Programmname.
// @return Vollständiger Pfad oder "" wenn nicht gefunden.
func lookupTool(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return findToolInDirs(name, toolSearchDirs(runtime.GOOS))
}

// pdfUsesPageImages gibt an, ob PDFs per pdftoppm als Seitenbilder gerendert
// werden sollen. Nur Windows zeigt PDFs stattdessen per <embed> an.
//
// @param goos Betriebssystem (wie runtime.GOOS).
// @return true außer unter Windows.
func pdfUsesPageImages(goos string) bool {
	return goos != "windows"
}

// ghostscriptInstallHint liefert den Installationshinweis für Ghostscript
// passend zum Betriebssystem (HTML, ohne Nutzertext).
//
// @param goos Betriebssystem (wie runtime.GOOS).
// @return HTML-Hinweistext.
func ghostscriptInstallHint(goos string) string {
	switch goos {
	case "windows":
		return `Install Ghostscript via the MD Reader Setup (optional component "Ghostscript") ` +
			`or download from <b>ghostscript.com</b>.`
	case "darwin":
		return `Install Ghostscript with Homebrew: <code>brew install ghostscript</code>`
	default:
		return `Install Ghostscript: <code>sudo apt-get install ghostscript</code>`
	}
}
