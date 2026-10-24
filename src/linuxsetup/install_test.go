// Tests für install.go – Pfade (Layout), Installieren, Entfernen und
// Erkennen einer vorhandenen Installation.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 3: Installieren, Entfernen, Entscheiden)
package linuxsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// envFrom baut eine Env-Funktion aus einer Map (ersetzt os.Getenv in Tests).
func envFrom(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

// testIcons liefert einen vollständigen Satz Test-Icons (4 PNG + 1 SVG).
func testIcons() Icons {
	return Icons{
		"16x16":    []byte("png16"),
		"32x32":    []byte("png32"),
		"48x48":    []byte("png48"),
		"256x256":  []byte("png256"),
		"scalable": []byte("<svg/>"),
	}
}

// newTestLayout legt ein Layout in einem temporären HOME an.
func newTestLayout(t *testing.T) Layout {
	t.Helper()
	l, err := NewLayout(envFrom(map[string]string{"HOME": t.TempDir()}))
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}
	return l
}

// writeSource legt eine Quell-"Binärdatei" an und liefert ihren Pfad.
func writeSource(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "md-reader-download")
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// stubRunQuiet ersetzt runQuiet und protokolliert die Aufrufe ("name arg arg").
func stubRunQuiet(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	old := runQuiet
	runQuiet = func(name string, args ...string) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
	}
	t.Cleanup(func() { runQuiet = old })
	return &calls
}

// TestNewLayout_Defaults prüft die Standardpfade ohne XDG-Variablen.
func TestNewLayout_Defaults(t *testing.T) {
	l, err := NewLayout(envFrom(map[string]string{"HOME": "/h"}))
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string][2]string{
		"BinPath":           {l.BinPath, "/h/.local/bin/md-reader"},
		"DesktopPath":       {l.DesktopPath, "/h/.local/share/applications/md-reader.desktop"},
		"IconBase":          {l.IconBase, "/h/.local/share/icons/hicolor"},
		"PrefsPath":         {l.PrefsPath, "/h/.config/md-reader/launcher.json"},
		"SystemDesktopPath": {l.SystemDesktopPath, "/usr/share/applications/md-reader.desktop"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, erwartet %q", name, c[0], c[1])
		}
	}
	wantKept := []string{"/h/.config/md-reader", "/h/.cache/md-reader", "/h/.local/share/md-reader"}
	if strings.Join(l.KeptPaths, "|") != strings.Join(wantKept, "|") {
		t.Errorf("KeptPaths = %v, erwartet %v", l.KeptPaths, wantKept)
	}
}

// TestNewLayout_XDG prüft, dass XDG_DATA_HOME und XDG_CONFIG_HOME gelten.
func TestNewLayout_XDG(t *testing.T) {
	l, err := NewLayout(envFrom(map[string]string{"HOME": "/h", "XDG_DATA_HOME": "/d", "XDG_CONFIG_HOME": "/c"}))
	if err != nil {
		t.Fatal(err)
	}
	if l.DesktopPath != "/d/applications/md-reader.desktop" {
		t.Errorf("DesktopPath = %q", l.DesktopPath)
	}
	if l.IconBase != "/d/icons/hicolor" {
		t.Errorf("IconBase = %q", l.IconBase)
	}
	if l.PrefsPath != "/c/md-reader/launcher.json" {
		t.Errorf("PrefsPath = %q", l.PrefsPath)
	}
	// Das Programm liegt weiterhin unter HOME, nicht unter XDG_DATA_HOME.
	if l.BinPath != "/h/.local/bin/md-reader" {
		t.Errorf("BinPath = %q", l.BinPath)
	}
	if l.KeptPaths[0] != "/c/md-reader" || l.KeptPaths[2] != "/d/md-reader" {
		t.Errorf("KeptPaths = %v", l.KeptPaths)
	}
}

// TestNewLayout_NoHome: ohne HOME gibt es keine sinnvollen Pfade.
func TestNewLayout_NoHome(t *testing.T) {
	if _, err := NewLayout(envFrom(map[string]string{})); err == nil {
		t.Fatal("Fehler erwartet, wenn HOME leer ist")
	}
}

// TestNewLayout_RelativeXDGIgnored: relative XDG-Pfade sind laut Spezifikation ungültig.
func TestNewLayout_RelativeXDGIgnored(t *testing.T) {
	l, err := NewLayout(envFrom(map[string]string{"HOME": "/h", "XDG_DATA_HOME": "rel", "XDG_CONFIG_HOME": "rel"}))
	if err != nil {
		t.Fatal(err)
	}
	if l.DesktopPath != "/h/.local/share/applications/md-reader.desktop" || l.PrefsPath != "/h/.config/md-reader/launcher.json" {
		t.Errorf("relative XDG-Pfade müssen ignoriert werden: %q %q", l.DesktopPath, l.PrefsPath)
	}
}

// TestInstall_CreatesAll prüft Programm, Desktop-Datei und alle fünf Icons.
func TestInstall_CreatesAll(t *testing.T) {
	calls := stubRunQuiet(t)
	l := newTestLayout(t)
	src := writeSource(t, "BINARY-47")

	if err := Install(l, src, 47, testIcons()); err != nil {
		t.Fatalf("Install: %v", err)
	}

	info, err := os.Stat(l.BinPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("Modus = %o, erwartet 755", info.Mode().Perm())
	}
	if got, _ := os.ReadFile(l.BinPath); string(got) != "BINARY-47" {
		t.Errorf("Inhalt = %q", got)
	}

	desktop, err := os.ReadFile(l.DesktopPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desktop), "X-MD-Reader-Build=47") {
		t.Errorf("Desktop-Datei ohne Build 47:\n%s", desktop)
	}
	if !strings.Contains(string(desktop), `Exec="`+l.BinPath+`" %f`) {
		t.Errorf("Exec zeigt nicht auf BinPath:\n%s", desktop)
	}
	if dinfo, _ := os.Stat(l.DesktopPath); dinfo.Mode().Perm() != 0o644 {
		t.Errorf("Desktop-Modus = %o, erwartet 644", dinfo.Mode().Perm())
	}

	for key, want := range map[string]string{
		"16x16": "png16", "32x32": "png32", "48x48": "png48", "256x256": "png256", "scalable": "<svg/>",
	} {
		ext := ".png"
		if key == "scalable" {
			ext = ".svg"
		}
		got, err := os.ReadFile(filepath.Join(l.IconBase, key, "apps", "md-reader"+ext))
		if err != nil || string(got) != want {
			t.Errorf("Icon %s: %q, %v", key, got, err)
		}
	}

	if b, ok := InstalledBuild(l); !ok || b != 47 {
		t.Errorf("InstalledBuild = %d, %v", b, ok)
	}

	wantDesktopDB := "update-desktop-database " + filepath.Dir(l.DesktopPath)
	wantIconCache := "gtk-update-icon-cache -f -t " + l.IconBase
	joined := strings.Join(*calls, "\n")
	if !strings.Contains(joined, wantDesktopDB) || !strings.Contains(joined, wantIconCache) {
		t.Errorf("runQuiet-Aufrufe = %v", *calls)
	}
}

// TestInstall_ReplacesOlder: ein zweiter Install ersetzt die ältere Version.
func TestInstall_ReplacesOlder(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, writeSource(t, "OLD"), 46, testIcons()); err != nil {
		t.Fatal(err)
	}
	if err := Install(l, writeSource(t, "NEW"), 47, testIcons()); err != nil {
		t.Fatal(err)
	}
	if b, ok := InstalledBuild(l); !ok || b != 47 {
		t.Errorf("InstalledBuild = %d, %v", b, ok)
	}
	if got, _ := os.ReadFile(l.BinPath); string(got) != "NEW" {
		t.Errorf("Inhalt = %q", got)
	}
	if _, err := os.Stat(l.BinPath + ".tmp"); !os.IsNotExist(err) {
		t.Errorf(".tmp-Datei darf nicht liegen bleiben: %v", err)
	}
}

// TestInstall_StaleTmpAndSymlink: eine vorhandene .tmp-Datei (auch ein
// untergeschobener Symlink) wird entfernt, das Ziel des Symlinks bleibt unberührt.
func TestInstall_StaleTmpAndSymlink(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := os.MkdirAll(filepath.Dir(l.BinPath), 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, l.BinPath+".tmp"); err != nil {
		t.Fatal(err)
	}
	if err := Install(l, writeSource(t, "BIN"), 47, testIcons()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got, _ := os.ReadFile(victim); string(got) != "KEEP" {
		t.Errorf("Symlink-Ziel wurde überschrieben: %q", got)
	}
	if got, _ := os.ReadFile(l.BinPath); string(got) != "BIN" {
		t.Errorf("Inhalt = %q", got)
	}
}

// TestInstall_ReadOnlyTarget: scheitert das Schreiben des Programms, darf
// keine Desktop-Datei entstehen (Reihenfolge: Programm zuerst, Desktop zuletzt).
func TestInstall_ReadOnlyTarget(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("als root wirkt 0500 nicht")
	}
	stubRunQuiet(t)
	l := newTestLayout(t)
	binDir := filepath.Dir(l.BinPath)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(binDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(binDir, 0o755) })

	if err := Install(l, writeSource(t, "BIN"), 47, testIcons()); err == nil {
		t.Fatal("Fehler erwartet")
	}
	if _, err := os.Stat(l.DesktopPath); !os.IsNotExist(err) {
		t.Errorf("Desktop-Datei darf nicht existieren: %v", err)
	}
	if _, ok := InstalledBuild(l); ok {
		t.Error("InstalledBuild darf nach Fehlschlag nicht installiert melden")
	}
}

// TestInstall_MissingSource: eine nicht lesbare Quelle ergibt Fehler, ohne Reste.
func TestInstall_MissingSource(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, filepath.Join(t.TempDir(), "gibt-es-nicht"), 47, testIcons()); err == nil {
		t.Fatal("Fehler erwartet")
	}
	if _, err := os.Stat(l.DesktopPath); !os.IsNotExist(err) {
		t.Errorf("Desktop-Datei darf nicht existieren: %v", err)
	}
	if _, err := os.Stat(l.BinPath + ".tmp"); !os.IsNotExist(err) {
		t.Errorf(".tmp darf nicht existieren: %v", err)
	}
}

// TestInstall_UnknownIconKey: unbekannte Schlüssel (z. B. "../x") werden abgelehnt,
// bevor irgendetwas geschrieben wird.
func TestInstall_UnknownIconKey(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	icons := Icons{"../../evil": []byte("x")}
	if err := Install(l, writeSource(t, "BIN"), 47, icons); err == nil {
		t.Fatal("Fehler erwartet")
	}
	if _, err := os.Stat(l.BinPath); !os.IsNotExist(err) {
		t.Errorf("nichts darf geschrieben sein: %v", err)
	}
}

// TestInstall_SelfIsTarget: wird die installierte Datei selbst gestartet, bleibt
// sie unversehrt (kein Abschneiden durch Lesen und Schreiben derselben Datei).
func TestInstall_SelfIsTarget(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, writeSource(t, "BIN"), 46, testIcons()); err != nil {
		t.Fatal(err)
	}
	if err := Install(l, l.BinPath, 47, testIcons()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got, _ := os.ReadFile(l.BinPath); string(got) != "BIN" {
		t.Errorf("Inhalt = %q", got)
	}
}

// TestUninstall_RemovesOnlyOwnFiles: entfernt genau die sieben eigenen Dateien.
func TestUninstall_RemovesOnlyOwnFiles(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, writeSource(t, "BIN"), 47, testIcons()); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(filepath.Dir(l.BinPath), "other")
	if err := os.WriteFile(other, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	removed, err := Uninstall(l)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(removed) != 7 {
		t.Fatalf("removed = %v (%d), erwartet 7", removed, len(removed))
	}
	for _, p := range removed {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s existiert noch", p)
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("fremde Datei wurde entfernt: %v", err)
	}
	if _, ok := InstalledBuild(l); ok {
		t.Error("InstalledBuild muss nach Uninstall false liefern")
	}

	removed, err = Uninstall(l)
	if err != nil || len(removed) != 0 {
		t.Errorf("zweites Uninstall: %v, %v", removed, err)
	}
}

// TestUninstall_KeepsUserData: Konfiguration und Daten bleiben erhalten.
func TestUninstall_KeepsUserData(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, writeSource(t, "BIN"), 47, testIcons()); err != nil {
		t.Fatal(err)
	}
	if err := SavePrefs(l.PrefsPath, Prefs{InstallPrompt: "never"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(l); err != nil {
		t.Fatal(err)
	}
	if LoadPrefs(l.PrefsPath).InstallPrompt != "never" {
		t.Error("launcher.json wurde entfernt")
	}
}

// TestInstalledBuild_NeedsBinary: eine Desktop-Datei ohne Programm gilt nicht als installiert.
func TestInstalledBuild_NeedsBinary(t *testing.T) {
	stubRunQuiet(t)
	l := newTestLayout(t)
	if err := Install(l, writeSource(t, "BIN"), 47, testIcons()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(l.BinPath); err != nil {
		t.Fatal(err)
	}
	if _, ok := InstalledBuild(l); ok {
		t.Error("ohne Programm darf installed nicht true sein")
	}
}

// TestInstalledBuild_NoKey: eine Desktop-Datei ohne Build-Schlüssel ergibt (0, false).
func TestInstalledBuild_NoKey(t *testing.T) {
	l := newTestLayout(t)
	for _, p := range []string{l.BinPath, l.DesktopPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("[Desktop Entry]\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if b, ok := InstalledBuild(l); ok || b != 0 {
		t.Errorf("InstalledBuild = %d, %v", b, ok)
	}
}

// TestPackageInstalled prüft die Erkennung einer Paketinstallation.
func TestPackageInstalled(t *testing.T) {
	dir := t.TempDir()
	l := Layout{SystemDesktopPath: filepath.Join(dir, "md-reader.desktop")}
	if PackageInstalled(l) {
		t.Error("ohne Datei muss false herauskommen")
	}
	if err := os.WriteFile(l.SystemDesktopPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !PackageInstalled(l) {
		t.Error("mit Datei muss true herauskommen")
	}
}
