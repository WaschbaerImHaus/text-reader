// Selbst-Installation von MD Reader unter Linux: Pfade, Installieren,
// Entfernen und Erkennen einer vorhandenen Installation.
//
// Alles liegt im Home-Verzeichnis des Nutzers (keine Root-Rechte nötig):
// Programm in ~/.local/bin, Desktop-Datei und Icons unter $XDG_DATA_HOME.
// Reihenfolge beim Installieren: Programm zuerst, Icons, Desktop-Datei zuletzt –
// so zeigt ein Startmenü-Eintrag nie auf ein nicht vorhandenes Programm.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (Quelle ist die geöffnete laufende Datei F3, Pfadprüfung H1, atomares Schreiben H2)
package linuxsetup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// systemDesktopPath ist die Desktop-Datei, die das .deb-Paket mitbringt.
const systemDesktopPath = "/usr/share/applications/md-reader.desktop"

// iconKeys nennt alle erlaubten Icon-Schlüssel in fester Reihenfolge.
// "scalable" ist ein SVG, alle anderen sind PNG in der jeweiligen Größe.
var iconKeys = []string{"16x16", "32x32", "48x48", "256x256", "scalable"}

// Layout fasst alle Pfade der Selbst-Installation zusammen.
type Layout struct {
	// BinPath ist das installierte Programm (~/.local/bin/md-reader).
	BinPath string
	// DesktopPath ist die Desktop-Datei ($XDG_DATA_HOME/applications/md-reader.desktop).
	DesktopPath string
	// IconBase ist das Icon-Theme-Verzeichnis ($XDG_DATA_HOME/icons/hicolor).
	IconBase string
	// SystemDesktopPath ist die Desktop-Datei einer Paketinstallation.
	SystemDesktopPath string
	// PrefsPath ist der Antwortspeicher ($XDG_CONFIG_HOME/md-reader/launcher.json).
	PrefsPath string
	// KeptPaths sind Nutzerdaten, die beim Entfernen erhalten bleiben (Konfiguration, Cache, Daten).
	KeptPaths []string
}

// Icons ordnet einem Schlüssel ("16x16", "32x32", "48x48", "256x256" für PNG,
// "scalable" für SVG) den Dateiinhalt zu.
type Icons map[string][]byte

// runQuiet führt ein Hilfsprogramm aus, wenn es vorhanden ist, und ignoriert
// jeden Fehler. In Tests wird die Variable ersetzt.
//
// Sicherheit: feste Argumentliste, nie über eine Shell.
var runQuiet = func(name string, args ...string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, path, args...).Run()
}

// absEnv liefert den Wert einer Umgebungsvariable nur, wenn er ein absoluter
// Pfad ist (die XDG-Spezifikation verlangt, relative Werte zu ignorieren).
func absEnv(env func(string) string, key string) string {
	v := env(key)
	if filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	return ""
}

// NewLayout berechnet alle Pfade aus HOME, XDG_DATA_HOME und XDG_CONFIG_HOME.
//
// @param env Funktion zum Lesen von Umgebungsvariablen (os.Getenv oder Test-Map).
// @return Layout; Fehler, wenn HOME leer oder nicht absolut ist.
func NewLayout(env func(string) string) (Layout, error) {
	home := env("HOME")
	if !filepath.IsAbs(home) {
		return Layout{}, errors.New("HOME ist nicht gesetzt oder kein absoluter Pfad")
	}
	home = filepath.Clean(home)

	dataHome := absEnv(env, "XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	configHome := absEnv(env, "XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, AppID)

	return Layout{
		BinPath:           filepath.Join(home, ".local", "bin", AppID),
		DesktopPath:       filepath.Join(dataHome, "applications", AppID+".desktop"),
		IconBase:          filepath.Join(dataHome, "icons", "hicolor"),
		SystemDesktopPath: systemDesktopPath,
		PrefsPath:         filepath.Join(configDir, "launcher.json"),
		KeptPaths: []string{
			configDir,
			filepath.Join(home, ".cache", AppID),
			filepath.Join(dataHome, AppID),
		},
	}, nil
}

// iconPath liefert den Zielpfad eines Icons ("scalable" → SVG, sonst PNG).
func (l Layout) iconPath(key string) string {
	ext := ".png"
	if key == "scalable" {
		ext = ".svg"
	}
	return filepath.Join(l.IconBase, key, "apps", AppID+ext)
}

// validIconKey prüft, ob der Schlüssel zu den erlaubten gehört (verhindert
// Pfade wie "../../x" aus einem fehlerhaften Aufrufer).
func validIconKey(key string) bool {
	for _, k := range iconKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Install kopiert das Programm nach BinPath und legt Desktop-Datei und Icons an.
//
// Ablauf: Programm (erst BinPath+".tmp", dann Umbenennen), Icons, Desktop-Datei
// zuletzt. Danach werden – falls vorhanden – update-desktop-database und
// gtk-update-icon-cache aufgerufen; deren Fehler werden ignoriert.
//
// Sicherheit (F3, TOCTOU): Quelle ist eine bereits geöffnete Datei – im
// Betrieb /proc/self/exe, geöffnet beim Start des Angebots. Damit wird immer
// die laufende Datei kopiert, auch wenn unter ihrem Pfad inzwischen (während
// der bis zu 120 s langen Frage) eine andere Datei liegt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (self statt selfPath, F3; Pfadprüfung H1)
//
// @param l Zielpfade.
// @param self Geöffnete laufende Datei (Quelle der Kopie; wird nicht geschlossen).
// @param build Build-Nummer für die Desktop-Datei.
// @param icons Icon-Inhalte; unbekannte Schlüssel ergeben einen Fehler.
// @return Fehler beim ersten gescheiterten Schritt.
func Install(l Layout, self *os.File, build int, icons Icons) error {
	// Zuerst prüfen, dann schreiben: ein falscher Schlüssel darf nichts hinterlassen.
	if self == nil {
		return errors.New("laufende Programmdatei nicht geöffnet")
	}
	// H1: keine Steuerzeichen in den Zielpfaden (Zeilenumbruch → eingeschleuste
	// Schlüssel in der Desktop-Datei); geprüft, bevor etwas angelegt wird.
	for _, p := range []string{l.BinPath, l.DesktopPath, l.IconBase} {
		if err := ValidatePath(p); err != nil {
			return err
		}
	}
	for key := range icons {
		if !validIconKey(key) {
			return fmt.Errorf("unbekannter Icon-Schlüssel %q", key)
		}
	}

	if err := installBinary(self, l.BinPath); err != nil {
		return err
	}
	for _, key := range iconKeys {
		data, ok := icons[key]
		if !ok {
			continue
		}
		if err := writeFileMkdir(l.iconPath(key), data, 0o644); err != nil {
			return err
		}
	}
	// Die Desktop-Datei kommt zuletzt, damit sie nie auf ein fehlendes Programm zeigt.
	if err := writeFileMkdir(l.DesktopPath, []byte(DesktopEntry(l.BinPath, build)), 0o644); err != nil {
		return err
	}

	runQuiet("update-desktop-database", filepath.Dir(l.DesktopPath))
	runQuiet("gtk-update-icon-cache", "-f", "-t", l.IconBase)
	return nil
}

// installBinary kopiert die geöffnete Datei src nach dst (Modus 0755, über
// writeFileAtomic). Ist src schon dst (Start aus ~/.local/bin, gleiche Datei
// laut fstat), geschieht nichts. Gelesen wird ab Position 0 (ReadAt),
// unabhängig von der aktuellen Leseposition von src.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (geöffnete Quelle statt Pfad, F3; writeFileAtomic, H2)
func installBinary(src *os.File, dst string) error {
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}
	if dstInfo, err := os.Stat(dst); err == nil && os.SameFile(srcInfo, dstInfo) {
		return nil
	}
	return writeFileAtomic(dst, io.NewSectionReader(src, 0, srcInfo.Size()), 0o755)
}

// writeFileMkdir schreibt Desktop-Datei bzw. Icon atomar über writeFileAtomic
// (Härtung H2: früher os.WriteFile – folgte Symlinks und hinterließ bei einem
// Abbruch eine halbe Datei).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (atomar, ohne Symlink-Verfolgung, H2)
func writeFileMkdir(path string, data []byte, mode os.FileMode) error {
	return writeFileAtomic(path, bytes.NewReader(data), mode)
}

// writeFileAtomic schreibt den Inhalt von r nach dst, ohne einem Symlink zu
// folgen und ohne je eine halbe Datei unter dst zu hinterlassen.
//
// Ablauf: fehlende Verzeichnisse (0755) anlegen → Altlast dst+".tmp" entfernen
// (auch ein untergeschobener Symlink wird entfernt, nicht verfolgt) →
// dst+".tmp" mit O_CREATE|O_EXCL neu anlegen → schreiben, Modus setzen
// (gleicht die umask aus), schließen → Rename auf dst. Rename ersetzt einen
// Symlink an der Stelle von dst, statt ihm zu folgen.
//
// O_EXCL folgt laut POSIX nie einem Symlink (auch keinem verwaisten) und
// scheitert, falls zwischen Remove und Open etwas angelegt wurde – das deckt
// O_NOFOLLOW mit ab. O_NOFOLLOW selbst gibt es im Paket syscall unter Windows
// nicht; linuxsetup muss aber auch für GOOS=windows übersetzbar bleiben.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, aus installBinary herausgelöst, H2)
//
// @param dst  Zieldatei.
// @param r    Inhalt.
// @param mode Dateimodus der Zieldatei.
// @return Fehler beim ersten gescheiterten Schritt (.tmp wird dann entfernt).
func writeFileAtomic(dst string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Chmod(mode); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Uninstall entfernt Programm, Desktop-Datei und alle Icons.
// Nutzerdaten (Layout.KeptPaths) bleiben unberührt; fehlende Dateien sind kein Fehler.
//
// @param l Zielpfade.
// @return removed Alle tatsächlich gelöschten Pfade; err beim ersten echten Fehler.
func Uninstall(l Layout) (removed []string, err error) {
	paths := []string{l.BinPath, l.DesktopPath}
	for _, key := range iconKeys {
		paths = append(paths, l.iconPath(key))
	}
	for _, p := range paths {
		switch rmErr := os.Remove(p); {
		case rmErr == nil:
			removed = append(removed, p)
		case errors.Is(rmErr, fs.ErrNotExist):
			// Schon weg – nichts zu tun.
		default:
			return removed, rmErr
		}
	}
	runQuiet("update-desktop-database", filepath.Dir(l.DesktopPath))
	runQuiet("gtk-update-icon-cache", "-f", "-t", l.IconBase)
	return removed, nil
}

// InstalledBuild liest die Build-Nummer der Nutzerinstallation aus der Desktop-Datei.
//
// @return (build, true), wenn die Desktop-Datei einen Build-Schlüssel hat und
// BinPath existiert; sonst (0, false).
func InstalledBuild(l Layout) (build int, installed bool) {
	content, err := os.ReadFile(l.DesktopPath)
	if err != nil {
		return 0, false
	}
	build, ok := BuildFromDesktopEntry(string(content))
	if !ok {
		return 0, false
	}
	if info, err := os.Stat(l.BinPath); err != nil || info.IsDir() {
		return 0, false
	}
	return build, true
}

// PackageInstalled meldet, ob MD Reader als .deb-Paket installiert ist
// (die Desktop-Datei unter /usr/share/applications existiert).
func PackageInstalled(l Layout) bool {
	_, err := os.Stat(l.SystemDesktopPath)
	return err == nil
}
