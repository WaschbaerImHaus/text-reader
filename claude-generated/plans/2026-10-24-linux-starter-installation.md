# Linux-Starter, Selbst-Installation und .deb – Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Ziel:** Die Linux-Dateien starten auch ohne WebKitGTK. Sie bieten dann per Desktop-Nachricht die Installation an, können sich selbst ins Anwendungsmenü installieren und erscheinen zusätzlich als `.deb`.

**Architektur:**
- Die heutige GUI-Binärdatei (cgo, WebKitGTK) wird zur **Nutzlast**. Ein neuer, statischer Starter (`CGO_ENABLED=0`) bettet sie per `go:embed` ein.
- Der Starter prüft mit dem Ladeprotokoll des dynamischen Laders (`LD_TRACE_LOADED_OBJECTS=1`), ob Bibliotheken fehlen. Er fragt den Nutzer über D-Bus-Benachrichtigungen (Rückfall kdialog → zenity → stderr) und startet die GUI aus einem `memfd` per `exec`.
- Die Fachlogik liegt testbar in `src/linuxsetup/`, die dünne Hülle in `src/launcher/`.
- Das `.deb` enthält die reine GUI ohne Starter.

**Tech Stack:**
- Go 1.27, `github.com/godbus/dbus/v5` (neu, reines Go), `syscall` (memfd über `SYS_MEMFD_CREATE`)
- `dpkg-deb` und `fakeroot`, Docker `ubuntu:24.04` für die Abnahme

**Spec:** `claude-generated/specs/2026-10-24-linux-starter-installation-design.md` (vor jeder Aufgabe lesen)

## Global Constraints

Diese Vorgaben gelten für jede Aufgabe:

- **Code-Stil:**
  - Bezeichner englisch.
  - Deutsche DocBlocks mit `Autor: Kurt Ingwer` und `Letzte Änderung: 2026-10-24 (…)`.
  - Einrückung nur mit Tabs.
  - Kommentare auf Deutsch, ausreichend dicht wie im übrigen Code.
- **TDD:** erst Test, RED sehen (aus dem richtigen Grund), dann Code.
- **Tests:** liegen neben dem Code (`src/linuxsetup/*_test.go`, Paket `linuxsetup`).
- **Paketname:** fest `libwebkit2gtk-4.1-0`.
- **Befehle:**
  - Installieren: exakt `pkexec apt-get install -y libwebkit2gtk-4.1-0`.
  - Paketlisten: exakt `pkexec apt-get update`.
  - Immer über `exec.Command` mit fester Argumentliste, nie über eine Shell, nie mit Inhalten aus Dokumenten, Dateinamen oder Umgebung.
- **Desktop-ID:** `md-reader` (`md-reader.desktop`), `StartupWMClass=md-reader`.
- **Installationsziele:**
  - Programm: `~/.local/bin/md-reader`.
  - Desktop-Datei: `$XDG_DATA_HOME/applications/md-reader.desktop`, Standard `~/.local/share`.
  - Icons: `$XDG_DATA_HOME/icons/hicolor/{16x16,32x32,48x48,256x256}/apps/md-reader.png` und `…/scalable/apps/md-reader.svg`.
- **Kategorie:** `Office;Viewer;`.
- **MimeType:** `text/markdown;text/x-markdown;text/plain;application/epub+zip;application/x-fictionbook+xml;text/x-tex;text/html;application/pdf;application/postscript;`.
- **Antwortspeicher:** `$XDG_CONFIG_HOME/md-reader/launcher.json`, Standard `~/.config`. Nie `state.json`.
- **Paketinstallation erkennen:** `/usr/share/applications/md-reader.desktop` existiert. Dann keine Installationsfrage, `--install` nur mit `--force`.
- **`--uninstall`:**
  - Entfernt Programm, Desktop-Datei und Icons.
  - Behält `~/.config/md-reader/`, `~/.cache/md-reader/` und `~/.local/share/md-reader/` und nennt diese Pfade in der Ausgabe.
- **glibc-Grenze:** keine Linux-Datei darf ein Symbol neuer als `GLIBC_2.39` verlangen.
- **`.deb`:**
  - `Package: md-reader`, `Version: 1.0.<build>`.
  - `Maintainer: Kurt Ingwer <WaschbaerImHaus@users.noreply.github.com>`.
  - `Depends: libc6 (>= 2.38), libwebkit2gtk-4.1-0, libgtk-3-0`.
  - `Recommends: zenity, poppler-utils, ghostscript`.
  - Abhängigkeiten nicht aus `dpkg-shlibdeps` übernehmen.
- **Texte:** Deutsch, wenn `LC_ALL`, `LC_MESSAGES` oder `LANG` (erste gesetzte) mit `de` beginnt, sonst Englisch.
- **Build und Release:** Build-Nummer 47. Release Notes, CHANGELOG und Commits englisch.
- **Fremde Dateien nie anfassen und nie stagen:** `SECURITY_HISTORY.md`, `SECURITY_REPORT.md`, `debugging/2026-10-14-offene-fragen.md`.
- **Subagents committen nicht und pushen nicht.** Commits macht die Hauptsitzung nach der Abnahme jeder Aufgabe.

## Review Focus

Diese fünf Fälle sind am wahrscheinlichsten nicht abgedeckt:

1. **Datei liegt unter einem anderen Namen oder auf einem `noexec`-Datenträger (z. B. USB-Stick):** Der Start klappt trotzdem, weil die GUI aus `memfd` läuft und nicht aus dem Pfad. Scheitert memfd (`vm.memfd_noexec=2`), kommt eine klare Meldung statt eines kryptischen Fehlers. Test in Aufgabe 6.
2. **Benachrichtigungsdienst vorhanden, aber ohne `actions`-Fähigkeit** (minimale Dienste, z. B. dunst ohne Konfiguration), **oder der Nutzer ignoriert die Nachricht:** Rückfall auf kdialog/zenity. Zeitlimit 120 s, danach gilt die Antwort „Später“. Test in Aufgabe 4.
3. **Start aus `~/.local/bin` bei einem Namen mit Leerzeichen oder Sonderzeichen im HOME-Pfad:** `Exec=` in der Desktop-Datei muss nach der Desktop-Entry-Spezifikation gequotet sein. Test in Aufgabe 2.
4. **`apt-get` scheitert oder der Nutzer bricht die Passwortabfrage ab** (pkexec Exit 126/127): kein Endlos-Kreislauf. Bei 126 und 127 keine zweite Frage, nur der Hinweis mit dem Befehl. Test in Aufgabe 5.
5. **Die installierte Datei ist neuer als die gestartete** (alte Datei im Download-Ordner): Es wird nichts ersetzt und nichts gefragt (keine Herabstufung). Test in Aufgabe 3.

---

## Dateiübersicht

| Datei | Aufgabe |
|---|---|
| `src/linuxsetup/deps.go` | Ladeprotokoll auswerten, Paketmanager erkennen, Befehle und Hinweise |
| `src/linuxsetup/desktop.go` | Desktop-Datei erzeugen und Build-Nummer daraus lesen |
| `src/linuxsetup/install.go` | Pfade (`Layout`), Installieren, Entfernen, installierten Build lesen |
| `src/linuxsetup/prefs.go` | `launcher.json` lesen und schreiben |
| `src/linuxsetup/decide.go` | Reine Entscheidungslogik: Frage stellen, ersetzen oder nichts tun |
| `src/linuxsetup/messages.go` | Texte de/en |
| `src/linuxsetup/asker.go` | Schnittstelle `Asker` + Rückfallkette |
| `src/linuxsetup/asker_dbus.go` | D-Bus-Benachrichtigung mit Aktionen |
| `src/linuxsetup/asker_dialog.go` | kdialog, zenity, stderr |
| `src/linuxsetup/*_test.go` | Tests je Datei |
| `src/launcher/main.go` | Starter: Argumente, Ablauf, `exec` |
| `src/launcher/memexec_linux.go` | memfd anlegen, Nutzlast schreiben, Ladeprotokoll, `exec` |
| `src/launcher/payload_amd64.go`, `payload_arm64.go`, `payload_arm.go` | `//go:build embedgui`: `go:embed payload/<arch>/md-reader-gui` und Icons |
| `src/launcher/payload_none.go` | `//go:build !embedgui`: leere Nutzlast (für `go vet`/`go test`) |
| `src/platform_linux.go` / `_windows.go` / `_darwin.go`, `src/main.go` | `setupPlatformIdentity()` (Linux: `g_set_prgname("md-reader")`) |
| `installer/linux/check-glibc.sh` | glibc-Wächter |
| `installer/linux/build-deb.sh` | `.deb` bauen |
| `debugging/linux/launcher-docker-test.sh` | Abnahme im `ubuntu:24.04`-Container |
| `Makefile`, `.gitignore` | GUI → Nutzlast → Starter, `deb`, `check-glibc`, Release mit 8 Assets |
| Doku | README/README.de, CHANGELOG, FEATURES, BUGS, MEMORY, dev-log, `docs/index.html`, SECURITY_RISKS, `.claude/rules/*`, ZUSTAENDIGKEITEN |

---

### Aufgabe 1: Ladeprotokoll und Paketmanager (`deps.go`)

**Dateien:**
- Neu: `src/linuxsetup/deps.go`
- Test: `src/linuxsetup/deps_test.go`

**Schnittstellen:**
- Erzeugt:
  - `func ParseMissingLibraries(trace string) []string`, sortiert und ohne Duplikate.
  - `type PackageManager int` mit den Konstanten `PMNone, PMApt, PMDnf, PMPacman, PMZypper`.
  - `func DetectPackageManager(lookPath func(string) (string, error)) PackageManager`. Reihenfolge: `apt-get`, `dnf`, `pacman`, `zypper`.
  - `func InstallHint(pm PackageManager) string`.
  - `func AptInstallCommand() []string` und `func AptUpdateCommand() []string`.
  - `const WebKitPackage = "libwebkit2gtk-4.1-0"`.

- [ ] **Schritt 1: Tests schreiben**

```go
func TestParseMissingLibraries(t *testing.T) {
	trace := "\tlinux-vdso.so.1 (0x00007ffd)\n" +
		"\tlibwebkit2gtk-4.1.so.0 => not found\n" +
		"\tlibgtk-3.so.0 => /lib/x86_64-linux-gnu/libgtk-3.so.0 (0x7f)\n" +
		"\tlibjavascriptcoregtk-4.1.so.0 => not found\n" +
		"\tlibwebkit2gtk-4.1.so.0 => not found\n"
	got := ParseMissingLibraries(trace)
	want := []string{"libjavascriptcoregtk-4.1.so.0", "libwebkit2gtk-4.1.so.0"}
	// reflect.DeepEqual(got, want)
}
// Weitere Tests:
//   TestParseMissingLibraries_None: "" und vollständiges Protokoll → leer (nil oder len 0)
//   TestDetectPackageManager: Fake-lookPath, nur "dnf" vorhanden → PMDnf;
//     apt-get und dnf vorhanden → PMApt; nichts → PMNone
//   TestInstallHint: PMApt → "sudo apt install libwebkit2gtk-4.1-0";
//     PMDnf → "sudo dnf install webkit2gtk4.1";
//     PMPacman → "sudo pacman -S webkit2gtk-4.1";
//     PMZypper → "sudo zypper install libwebkit2gtk-4_1-0";
//     PMNone → enthält "WebKitGTK 4.1"
//   TestAptCommandsAreFixed: AptInstallCommand() == {"pkexec","apt-get","install","-y","libwebkit2gtk-4.1-0"},
//     AptUpdateCommand() == {"pkexec","apt-get","update"};
//     zweimal aufrufen und das erste Ergebnis verändern → das zweite bleibt unverändert (Kopie)
```

- [ ] **Schritt 2:** `cd src && go test ./linuxsetup/ -run 'Parse|Detect|Hint|Apt' -v`. Erwartet: FAIL (Paket oder Funktionen fehlen).
- [ ] **Schritt 3:** Umsetzen. `ParseMissingLibraries` wertet Zeilen der Form `<name> => not found` aus (Leerraum trimmen).
- [ ] **Schritt 4:** Gleicher Befehl → PASS.

### Aufgabe 2: Desktop-Datei (`desktop.go`)

**Dateien:**
- Neu: `src/linuxsetup/desktop.go`
- Test: `src/linuxsetup/desktop_test.go`

**Schnittstellen:**
- Erzeugt:
  - `const AppID = "md-reader"`.
  - `func DesktopEntry(execPath string, build int) string`.
  - `func BuildFromDesktopEntry(content string) (int, bool)`. Liest den Schlüssel `X-MD-Reader-Build`.
  - `func quoteExec(path string) string`, unexportiert.

- [ ] **Schritt 1: Tests schreiben**
  - `TestDesktopEntry_Fields`: Der Inhalt enthält genau diese Zeilen:
    - `[Desktop Entry]`
    - `Type=Application`
    - `Name=MD Reader`
    - `GenericName=Document Viewer`
    - `GenericName[de]=Dokumentbetrachter`
    - `Comment=View Markdown, EPUB, FB2, LaTeX, PDF and more`
    - `Comment[de]=Markdown, EPUB, FB2, LaTeX, PDF und mehr anzeigen`
    - `Exec="/home/u/.local/bin/md-reader" %f`
    - `Icon=md-reader`
    - `Terminal=false`
    - `Categories=Office;Viewer;`
    - `StartupWMClass=md-reader`
    - `X-MD-Reader-Build=47`
    - die `MimeType=`-Zeile aus den Global Constraints
  - `TestDesktopEntry_QuotesSpecialPaths`: `/home/a b/$x/"q"/back\slash/md-reader` → `Exec="/home/a b/\\$x/\\"q\\"/back\\\\slash/md-reader" %f`. Escape-Regeln nach der Desktop-Entry-Spec: in Anführungszeichen `"`, `` ` ``, `$` und `\` mit `\` escapen, danach für das String-Format jeden `\` verdoppeln.
  - `TestDesktopEntry_PercentInPath`: `%` im Pfad → `%%`.
  - `TestBuildFromDesktopEntry`:
    - Rundreise mit 47 → `(47, true)`.
    - Ohne Schlüssel → `(0, false)`.
    - `X-MD-Reader-Build=abc` → `(0, false)`.
- [ ] **Schritt 2:** RED. Befehl: `go test ./linuxsetup/ -run Desktop -v`.
- [ ] **Schritt 3:** Umsetzen. Zeilenende `\n`, Reihenfolge wie oben.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 3: Installieren, Entfernen, Entscheiden (`install.go`, `prefs.go`, `decide.go`)

**Dateien:**
- Neu: `src/linuxsetup/install.go`, `prefs.go`, `decide.go`
- Tests: `install_test.go`, `prefs_test.go`, `decide_test.go`

**Schnittstellen:**
- Verwendet: `DesktopEntry`, `BuildFromDesktopEntry`, `AppID` (Aufgabe 2).
- Erzeugt:
  - `type Layout struct { BinPath, DesktopPath, IconBase, SystemDesktopPath, PrefsPath string; KeptPaths []string }`
  - `func NewLayout(env func(string) string) (Layout, error)`
    - Liest `HOME`, `XDG_DATA_HOME` und `XDG_CONFIG_HOME`.
    - Fehler, wenn `HOME` leer ist.
    - `SystemDesktopPath = "/usr/share/applications/md-reader.desktop"`.
  - `type Icons map[string][]byte`. Schlüssel `"16x16"`, `"32x32"`, `"48x48"`, `"256x256"` für PNG, `"scalable"` für SVG.
  - `func Install(l Layout, selfPath string, build int, icons Icons) error`
    - Kopiert `selfPath` nach `BinPath`: erst `BinPath+".tmp"` mit Modus 0755 schreiben, dann `os.Rename`.
    - Legt fehlende Verzeichnisse mit 0755 an.
    - Schreibt die Desktop-Datei mit 0644 und die Icons.
    - Ruft danach, falls vorhanden, `update-desktop-database <dir>` und `gtk-update-icon-cache -f -t <IconBase>` auf. Fehler dieser Aufrufe werden ignoriert.
    - Für Tests ist der Aufruf austauschbar über das Paketfeld `var runQuiet = func(name string, args ...string) {}`.
  - `func Uninstall(l Layout) (removed []string, err error)`. Fehlende Dateien sind kein Fehler.
  - `func InstalledBuild(l Layout) (build int, installed bool)`. Liest die Desktop-Datei und prüft, dass `BinPath` existiert.
  - `func PackageInstalled(l Layout) bool`. `os.Stat(SystemDesktopPath)`.
  - `type Prefs struct { InstallPrompt string \`json:"installPrompt"\` }`. Werte `""`, `"later"` oder `"never"`.
  - `func LoadPrefs(path string) Prefs` (Fehler → leer) und `func SavePrefs(path string, p Prefs) error` (Verzeichnis 0700, Datei 0600).
  - `type StartContext struct { ExePath string; CurrentBuild int; Layout Layout; Installed bool; InstalledBuild int; Package bool; Prefs Prefs }`
  - `type Offer int` mit `OfferNothing, OfferInstall, OfferUpdate`.
  - `func Decide(c StartContext) Offer`.

- [ ] **Schritt 1: Tests schreiben**

```go
// decide_test.go – tabellengesteuert, je Zeile: Name, Kontext, erwartetes Offer
// 1. frisch, nichts installiert, Prefs leer                          → OfferInstall
// 2. Prefs "never"                                                   → OfferNothing
// 3. Prefs "later"                                                   → OfferInstall (nächster Start fragt wieder)
// 4. Package == true                                                 → OfferNothing
// 5. ExePath == Layout.BinPath (läuft aus ~/.local/bin)              → OfferNothing
// 6. ExePath == "/usr/bin/md-reader"                                 → OfferNothing
// 7. installiert Build 46, aktuell 47, ExePath woanders              → OfferUpdate (auch bei Prefs "never")
// 8. installiert Build 48, aktuell 47                                → OfferNothing  (Review Focus 5)
// 9. installiert Build 47, aktuell 47                                → OfferNothing
```

  - `install_test.go` (temporäres HOME über `t.TempDir()`, Env-Funktion aus einer Map):
    - `TestNewLayout_Defaults`: `HOME=/h` → `BinPath=/h/.local/bin/md-reader`, `DesktopPath=/h/.local/share/applications/md-reader.desktop`, `IconBase=/h/.local/share/icons/hicolor`, `PrefsPath=/h/.config/md-reader/launcher.json`.
    - `TestNewLayout_XDG`: `XDG_DATA_HOME=/d`, `XDG_CONFIG_HOME=/c` → `/d/applications/md-reader.desktop`, `/c/md-reader/launcher.json`.
    - `TestNewLayout_NoHome`: Fehler.
    - `TestInstall_CreatesAll`:
      - Danach existieren `BinPath` (Modus 0755, Inhalt gleich der Quelle), die Desktop-Datei mit `X-MD-Reader-Build=47` und alle 5 Icons unter `<IconBase>/<size>/apps/md-reader.png|svg`.
      - `InstalledBuild` liefert `(47, true)`.
      - `runQuiet` wurde mit `update-desktop-database` aufgerufen.
    - `TestInstall_ReplacesOlder`: zuerst Build 46, dann 47 → `InstalledBuild` liefert 47. Keine `.tmp`-Datei bleibt liegen.
    - `TestInstall_ReadOnlyTarget`: `.local/bin` mit 0500 → Fehler, keine halbe Desktop-Datei. Ablauf dafür: Binärdatei zuerst, Desktop-Datei zuletzt. Test überspringen, wenn er als root läuft.
    - `TestUninstall_RemovesOnlyOwnFiles`:
      - Nach Install eine fremde Datei `~/.local/bin/other` anlegen.
      - Nach Uninstall sind Binärdatei, Desktop-Datei und Icons weg, `other` existiert noch.
      - `removed` enthält die 7 Pfade.
      - Zweites Uninstall → kein Fehler, `removed` leer.
    - `TestPackageInstalled`: `SystemDesktopPath` auf eine Datei im TempDir zeigen lassen → true; nicht vorhanden → false.
  - `prefs_test.go`:
    - Rundreise `"never"`.
    - Kaputtes JSON → `Prefs{}`.
    - Dateimodus 0600.
- [ ] **Schritt 2:** RED. Befehl: `go test ./linuxsetup/ -v`.
- [ ] **Schritt 3:** Umsetzen. In `Decide` werden Pfade vor dem Vergleich mit `filepath.Clean` behandelt. `OfferUpdate` hat Vorrang vor `"never"`, weil der Nutzer schon installiert hat.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 4: Texte und Nachfragen (`messages.go`, `asker*.go`)

**Dateien:**
- Neu: `src/linuxsetup/messages.go`, `asker.go`, `asker_dbus.go`, `asker_dialog.go`
- Tests: `messages_test.go`, `asker_test.go`
- Ändern: `src/go.mod` und `src/go.sum`, für `github.com/godbus/dbus/v5`, neueste Version per `go get`.

**Schnittstellen:**
- Erzeugt:
  - `type Lang string`, `func DetectLang(env func(string) string) Lang`. Gibt `"de"` oder `"en"` zurück.
  - `type MessageID int` mit:
    - `MsgWebKitMissing` (Titel und Text: „WebKitGTK ist nicht installiert“, Text nennt die fehlenden Bibliotheken)
    - `MsgWebKitHint` (Text mit `InstallHint`)
    - `MsgInstallFailed`
    - `MsgOfferInstall`
    - `MsgOfferUpdate`
    - `MsgInstalled` („MD Reader ist jetzt im Menü unter Büro.“)
    - `MsgPackageInstalled` (Hinweis für `--install` neben dem `.deb`)
    - `MsgUninstalled` (nennt die behaltenen Pfade)
  - `func Text(lang Lang, id MessageID, args ...string) (title, body string)`.
  - `type Button struct { Key, Label string }`.
  - `type Asker interface { Ask(title, body string, buttons []Button, timeout time.Duration) (key string, err error); Inform(title, body string) }`.
    - Leerer Schlüssel bedeutet „keine Antwort“, bei Zeitablauf oder Schließen.
  - `func NewAsker(lookPath func(string) (string, error)) Asker`. Rückfallkette:
    1. D-Bus, nur wenn eine Session-Bus-Verbindung besteht und `GetCapabilities` den Wert `"actions"` enthält
    2. `kdialog`
    3. `zenity`
    4. stderr
  - `dbusAsker` arbeitet mit `org.freedesktop.Notifications.Notify`:
    - `app_name` `"MD Reader"`, `app_icon` `"md-reader"`.
    - Aktionen als Paare aus Schlüssel und Beschriftung.
    - Wartet auf `ActionInvoked` oder `NotificationClosed` mit passender ID. Bei Zeitablauf schließt er die Nachricht per `CloseNotification` und gibt `""` zurück.
  - `dialogAsker`:
    - kdialog mit `--title t --yesnocancel body --yes-label … --no-label … --cancel-label …`.
    - zenity mit `--question --title t --text body --ok-label … --cancel-label … --extra-button …`.
    - Exit-Code bzw. Ausgabe wird dem Button-Schlüssel zugeordnet. Höchstens 3 Buttons.
    - stderr gibt nur Titel und Text aus und liefert `""`.
- [ ] **Schritt 1: Tests schreiben**
  - `TestDetectLang`:
    - `LANG=de_DE.UTF-8` → `de`.
    - `LC_ALL=C` hat Vorrang vor `LANG=de_DE` → `en`.
    - Alles leer → `en`.
  - `TestText_AllIDsBothLangs`:
    - Für jede `MessageID` und jede Sprache sind Titel und Text nicht leer.
    - Für `de` gilt: `Text("de", MsgInstalled)` liefert genau „MD Reader ist jetzt im Menü unter Büro.“.
  - `TestText_WebKitMissingListsLibs`: Der Text enthält `libwebkit2gtk-4.1.so.0`.
  - `TestNewAsker_Fallback`:
    - Fake-lookPath ohne Programme und ohne Bus (Bus-Adresse über die Env-Variable `DBUS_SESSION_BUS_ADDRESS=""` sperren) → Typ `stderrAsker`.
    - Nur `zenity` → `dialogAsker{tool:"zenity"}`.
    - `kdialog` und `zenity` → kdialog.
  - `TestDialogAsker_ArgsAndMapping`:
    - `dialogAsker` mit austauschbarem `run func(name string, args []string) (exitCode int, out string, err error)`.
    - Für kdialog werden die Argumente exakt geprüft. Exit 0 → erster Key, 1 → zweiter, 2 → dritter.
    - Für zenity: Exit 0 → erster Key, Ausgabe gleich Extra-Label → dritter, Exit 1 → zweiter.
    - Body mit `$(rm -rf ~)` wird als ein einziges Argument übergeben, ohne Shell.
  - `TestDBusAsker` gegen einen privaten Session-Bus:
    - Läuft nur, wenn `dbus-daemon` vorhanden ist. Er wird unter `t.TempDir()` mit `--session --print-address --nofork` gestartet.
    - Ein Fake-Notification-Server wird mit `godbus` im Test exportiert. Er antwortet auf `GetCapabilities` mit `["actions"]`, gibt bei `Notify` ID 7 zurück und sendet sofort `ActionInvoked(7,"install")` → Ask liefert `"install"`.
    - Zweiter Fall: Der Server liefert keine `"actions"` → `NewAsker` wählt nicht D-Bus.
    - Dritter Fall: Ohne Signal und mit Timeout 200 ms → `""`, und `CloseNotification(7)` wurde gerufen. Das deckt Review Focus 2 ab.
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen. Die D-Bus-Verbindung ist eine eigene `dbus.SessionBusPrivate()`-Verbindung und wird nach `Ask` geschlossen.
- [ ] **Schritt 4:** GREEN. Zusätzlich `go vet ./...`.

### Aufgabe 5: Ablaufsteuerung (`flow.go`)

**Dateien:**
- Neu: `src/linuxsetup/flow.go`
- Test: `flow_test.go`

**Schnittstellen:**
- Verwendet: alles aus den Aufgaben 1–4.
- Erzeugt:
  - `type Env struct { Asker Asker; Lang Lang; Run func(argv []string) (exitCode int, err error); LookPath func(string) (string, error) }`
  - `func HandleMissingLibraries(e Env, missing []string) (retry bool)`
    - Hat das System `apt-get`, wird gefragt: „Jetzt installieren“ (`install`) oder „Abbrechen“ (`cancel`), Timeout 120 s.
    - Bei `install` läuft `AptInstallCommand`:
      - Exit 0 → `true`.
      - Exit 126 oder 127 (Abbruch oder kein pkexec) → `Inform(MsgWebKitHint)`, dann `false`, ohne weitere Frage.
      - Anderer Fehler → Frage `MsgInstallFailed` mit „Paketlisten aktualisieren und erneut versuchen“ (`update`) oder „Abbrechen“. Bei `update` laufen `AptUpdateCommand` und danach `AptInstallCommand`. Exit 0 → `true`, sonst `Inform(MsgWebKitHint)` und `false`. Höchstens ein Wiederholungsversuch.
    - Ohne `apt-get`: nur `Inform(MsgWebKitHint)`, dann `false`.
    - Immer und zusätzlich zum Asker: Titel, fehlende Bibliotheken und `InstallHint` auf stderr, für Terminal-Starts. Dafür hat `Env` ein Feld `Stderr io.Writer`; `TestHandleMissing_*` prüfen den Hinweis darin.
  - `func HandleInstallOffer(e Env, c StartContext, selfPath string, icons Icons) error`. Ruft `Decide` auf:
    - `OfferInstall`: Frage mit „Ja“ (`yes`), „Später“ (`later`), „Nie“ (`never`), Timeout 120 s.
      - `yes` → `Install` und `Inform(MsgInstalled)`.
      - `later` oder `""` → Prefs `"later"` speichern.
      - `never` → Prefs `"never"` speichern.
    - `OfferUpdate`: Frage mit „Ersetzen“ (`yes`) oder „Nicht jetzt“ (`later`). Bei `yes` → `Install`.
- [ ] **Schritt 1: Tests schreiben**. Fake-Asker mit vorgegebenen Antworten, Fake-Run zeichnet `argv` auf.
  - `TestHandleMissing_InstallOK`: Antwort `install`, Run liefert 0 → `true`. Aufgezeichnet ist genau `[AptInstallCommand()]`.
  - `TestHandleMissing_Cancelled`: Antwort `cancel` → `false`, kein Run.
  - `TestHandleMissing_AuthAborted`: Run liefert 126 → `false`. Nur ein Run, eine Frage, ein Inform. Das deckt Review Focus 4 ab.
  - `TestHandleMissing_RetryWithUpdate`: Run liefert zuerst 100, die zweite Antwort ist `update`, danach liefert Run 0 und 0 → `true`. Aufgezeichnet sind `[install, update, install]`.
  - `TestHandleMissing_RetryFails`: 100, `update`, 0, 100 → `false`, danach keine weitere Frage.
  - `TestHandleMissing_NoApt`: Ohne apt → `false`, keine Frage. Inform-Text enthält `dnf install webkit2gtk4.1`, wenn nur dnf vorhanden ist.
  - `TestHandleInstallOffer`: Je ein Fall für `yes`, `later`, `never` und Timeout `""`.
    - Prefs und Installation werden im TempDir geprüft.
    - Bei `OfferNothing` wird nicht gefragt.
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 6: Starter-Programm (`src/launcher/`)

**Dateien:**
- Neu: `src/launcher/main.go`, `memexec_linux.go`, `memexec_other.go` (`//go:build !linux`: liefert einen Fehler „nur Linux“), `payload_amd64.go`, `payload_arm64.go`, `payload_arm.go`, `payload_none.go`
- Test: `src/launcher/main_test.go`
- Ändern: `.gitignore` (Eintrag `src/launcher/payload/`)

**Schnittstellen:**
- Verwendet: `linuxsetup.*`
- Erzeugt:
  - `var buildNumber = "0"`. Gesetzt über `-ldflags "-X main.buildNumber=<n>"`.
  - `var guiBinary []byte` und `var iconFiles linuxsetup.Icons`. Gesetzt über die Payload-Dateien.
    - Mit Tag `embedgui` gilt `//go:embed payload/<goarch>/md-reader-gui`. Die Icons kommen als `payload/icons/*` per `embed.FS` und werden auf die `Icons`-Schlüssel abgebildet.
    - Ohne Tag sind beide leer.
  - `func parseCommand(args []string) (cmd string, force bool, rest []string)`.
    - `cmd` ist eines von `"install"`, `"uninstall"`, `"version"`, `"print-desktop-entry"` oder `""`.
    - Nur `args[0]` wird betrachtet: `--install`, `--uninstall`, `--launcher-version`, `--print-desktop-entry`. Danach optional `--force` (bei install) bzw. der Exec-Pfad (bei print-desktop-entry). Alles andere geht als `rest` unverändert an die GUI.
  - `func createMemFile(data []byte) (fd int, err error)`. Arbeitet mit `syscall.Syscall(SYS_MEMFD_CREATE, "md-reader", MFD_CLOEXEC)` und schreibt die Daten.
  - `func traceMissing(fd int) ([]string, error)`.
    - Startet `/proc/self/fd/<fd>` mit `Args[0]="md-reader"` und `LD_TRACE_LOADED_OBJECTS=1`.
    - Liest stdout und gibt das Ergebnis von `ParseMissingLibraries` zurück.
    - Für den Kindprozess wird die fd über `ExtraFiles` sichtbar gemacht. Dafür zuerst `os.NewFile(fd)` und im Kind `/proc/self/fd/3` verwenden; der Bauer prüft, welche Variante funktioniert.
  - `func execGUI(fd int, rest []string) error`. Ruft `syscall.Exec("/proc/self/fd/<fd>", append([]string{"md-reader"}, rest...), os.Environ())` auf.
- Ablauf in `main()`:
  1. `cmd` auswerten:
     - `version` → Build ausgeben.
     - `print-desktop-entry` → `DesktopEntry(path, build)` ausgeben, Ende.
     - `uninstall` → `Uninstall`, Pfade und `MsgUninstalled` auf stdout, Ende.
     - `install` → bei `PackageInstalled` ohne `--force` `MsgPackageInstalled` auf stderr und Exit 1, sonst `Install` und `MsgInstalled`, Ende.
  2. `guiBinary` leer → Fehlermeldung „Starter ohne eingebettete App gebaut“, Exit 1.
  3. `createMemFile` scheitert → stderr- und Inform-Meldung, Exit 1. Die Meldung nennt die Ursache und rät, die Datei aus einem Ordner mit Ausführungsrecht zu starten. Das deckt Review Focus 1 ab.
  4. `traceMissing`: Fehlen Bibliotheken → `HandleMissingLibraries`. Bei `true` erneut prüfen (höchstens einmal), sonst Exit 1.
  5. `HandleInstallOffer` mit `StartContext`. Darin stammt `ExePath` aus `os.Executable()` mit `filepath.EvalSymlinks`, `selfPath` ist derselbe Pfad. Fehler werden nur protokolliert, der Start geht weiter.
  6. `execGUI`.
- [ ] **Schritt 1: Tests schreiben**
  - `TestParseCommand`, Tabelle:
    - `[]` → `""`.
    - `["doc.md"]` → `rest=["doc.md"]`.
    - `["--install"]`.
    - `["--install","--force"]` → `force=true`.
    - `["--uninstall"]`.
    - `["--print-desktop-entry","/usr/bin/md-reader"]` → `rest=["/usr/bin/md-reader"]`.
    - `["doc.md","--install"]` → `cmd=""`, `rest=["doc.md","--install"]`. Nur das erste Argument zählt.
  - `TestMemExec_TraceAndExec` (nur Linux):
    - Ablauf: `createMemFile` mit dem Inhalt von `/bin/true`, dann `traceMissing` → leer, kein Fehler.
    - Dazu ein Unterprozess-Test: per `os.Args`-Schalter `GO_TEST_EXEC=1` ruft der Test `execGUI` mit dem Inhalt von `/bin/echo` und den Argumenten `["hallo"]` auf. Die Ausgabe ist `hallo`.
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN. Danach `go vet ./...` (ohne Tag) und `CGO_ENABLED=0 go build -tags embedgui ./launcher` (muss scheitern, weil die Nutzlast fehlt; Bestätigung, dass der Tag wirkt).

### Aufgabe 7: App-ID in der GUI (`setupPlatformIdentity`)

**Dateien:**
- Ändern:
  - `src/platform_linux.go`: C-Funktion `static void setProgramName(void) { g_set_prgname("md-reader"); }` und `func setupPlatformIdentity()`.
  - `src/platform_windows.go` und `src/platform_darwin.go`: No-Op `func setupPlatformIdentity() {}`.
  - `src/main.go`: Aufruf direkt vor `webview.New`.
- Die DocBlock-Daten der geänderten Dateien aktualisieren.
- [ ] **Schritt 1:** Kein Unit-Test möglich (cgo/GTK). Die Prüfung erfolgt in Aufgabe 9: `xprop WM_CLASS` des Fensters unter Xvfb liefert `"md-reader", "md-reader"`, auch wenn die Datei als `md-reader-test` umbenannt gestartet wird. Den Altzustand vorher messen (RED): umbenannt gestartet zeigt er `md-reader-test`.
- [ ] **Schritt 2:** Umsetzen, `go vet ./...`, `go test ./...`, Cross-Build Windows (`make` Probe ins Scratch-Verzeichnis).

### Aufgabe 8: Build, glibc-Wächter und `.deb` (Makefile, `installer/linux/`)

**Dateien:**
- Neu: `installer/linux/check-glibc.sh`, `installer/linux/build-deb.sh`
- Ändern: `Makefile`

**Vorgaben:**
- `check-glibc.sh <max> <datei>…`:
  - Prüft mit `objdump -T` die Symbolversionen und vergleicht sie per `sort -V` mit `<max>`.
  - Gibt die zu neuen Symbole aus und endet dann mit Exit 1.
  - Bei ARM-Dateien wird mit `objdump`, `aarch64-linux-gnu-objdump` bzw. `arm-linux-gnueabihf-objdump` geprüft. Die Variable `OBJDUMP` lässt sich überschreiben.
- Makefile:
  - Die GUI-Ziele bauen nach `build/gui/md-reader-<amd64|arm64|arm>`, mit denselben Flags wie heute.
  - Danach `check-glibc.sh 2.39`.
  - Dann wird die GUI nach `src/launcher/payload/<goarch>/md-reader-gui` kopiert. Die Icons kommen aus `src/ui/assets/favicon-{16,32,48,256}.png` und `favicon.svg` nach `src/launcher/payload/icons/`.
  - Dann wird der Starter gebaut, mit `CGO_ENABLED=0 GOOS=linux GOARCH=<…> [GOARM=7] go build -tags embedgui -trimpath -ldflags "-s -w -X main.buildNumber=$(BUILD_NUM)" -o ../build/md-reader[-linux-arm64|-linux-armhf] ./launcher`.
  - Die Zieldateinamen in `build/` bleiben gleich.
  - Ziel `deb`: `installer/linux/build-deb.sh <arch> build/gui/md-reader-<…> $(BUILD_NUM)` für amd64, arm64 und armhf → `build/md-reader_1.0.<n>_<arch>.deb`.
  - `release` und `release-notes` bekommen die 3 `.deb` als neue Zeilen, insgesamt 8 Assets plus macOS-Zip.
  - In den Requirements steht für Linux künftig „Ubuntu 24.04+, Debian 13+, Mint 22+, KDE neon; WebKitGTK is offered for installation on first start“.
- `build-deb.sh`:
  - Baut das Staging-Verzeichnis im Scratch (`mktemp -d`).
  - `usr/bin/md-reader` mit 0755 ist die GUI.
  - `usr/share/applications/md-reader.desktop` wird per `(cd src && go run ./launcher --print-desktop-entry /usr/bin/md-reader)` erzeugt. Damit gibt es keine doppelte Desktop-Vorlage.
  - Icons nach `usr/share/icons/hicolor/<size>/apps/md-reader.png` und `scalable/apps/md-reader.svg`.
  - `DEBIAN/control` mit den Feldern aus den Global Constraints, dazu `Architecture`, `Section: text`, `Priority: optional`, `Homepage: https://github.com/WaschbaerImHaus/text-reader` und `Description: Document viewer for Markdown, EPUB, FB2, LaTeX, PDF and PostScript` samt Langtext.
  - Aufbau mit `fakeroot dpkg-deb --build --root-owner-group`.
- [ ] **Schritt 1 (RED):**
  - `installer/linux/check-glibc.sh 2.30 build/md-reader` auf der heutigen Datei → muss mit Exit 1 scheitern und `GLIBC_2.3x` nennen.
  - `installer/linux/check-glibc.sh 2.39 build/md-reader` → Exit 0.
- [ ] **Schritt 2:** Makefile umbauen.
  - `rm -f build/md-reader* build/gui/*`
  - `make linux deb-amd64`
  - Danach prüfen:
    - `file build/md-reader` meldet „statically linked“.
    - `./build/md-reader --launcher-version` gibt 47 aus.
    - `dpkg-deb -I build/md-reader_1.0.47_amd64.deb` zeigt die richtigen Depends.
    - `dpkg-deb -c` listet die 7 Dateien.
- [ ] **Schritt 3:** ARM64 und ARMhf mit Paketwechsel nach `.claude/rules/build-release.md`, danach amd64 wiederherstellen.
  - Für jede Architektur `file` prüfen: Starter statisch, GUI dynamisch, glibc-Wächter grün.
  - Ergebnis sind 3 `.deb`.

### Aufgabe 9: Abnahme im Container und unter Xvfb (`debugging/linux/`)

**Dateien:**
- Neu: `debugging/linux/launcher-docker-test.sh`

**Ablauf des Skripts** (Docker, Image `ubuntu:24.04` = Basis von KDE neon):

1. **Ohne WebKit:**
   - `docker run --rm -v build:/b:ro ubuntu:24.04 /b/md-reader` muss Exit 1 liefern.
   - stderr enthält „WebKitGTK“, `libwebkit2gtk-4.1.so.0` und `sudo apt install libwebkit2gtk-4.1-0`.
   - Das zeigt den stderr-Weg ohne Bus und ohne Dialogwerkzeug.
2. **Ohne WebKit, mit `LANG=de_DE.UTF-8`:** Die deutsche Meldung erscheint.
3. **`--install` mit `HOME=/tmp/h`:**
   - Die 7 Dateien existieren.
   - `desktop-file-validate` (Paket `desktop-file-utils` im Container) meldet keine Fehler.
   - `--uninstall` entfernt sie und nennt die behaltenen Pfade.
4. **`.deb`-Installation:**
   - Ablauf: `apt-get update && apt-get install -y /b/md-reader_1.0.47_amd64.deb`. Das muss gelingen und WebKit mitziehen.
   - Danach existiert `/usr/share/applications/md-reader.desktop`.
   - Danach `xvfb-run -a /usr/bin/md-reader /b/README.md &`, 5 s warten.
   - `xprop -root _NET_CLIENT_LIST` bzw. `xdotool search --class md-reader` findet das Fenster.
   - `xprop WM_CLASS` liefert `"md-reader", "md-reader"` (Aufgabe 7).
   - Ein Bildschirmfoto wird nach `/b/shots/deb.png` geschrieben.
5. **Starter mit installiertem WebKit (aus dem `.deb`):**
   - `/b/md-reader` startet die GUI (Fenster erscheint).
   - Eine Installationsfrage gibt es nicht, weil das Paket installiert ist; stderr enthält keinen Fragetext.

Ergänzend im LXC unter Xvfb:
- `cp build/md-reader <scratch>/md-reader-test` und starten → `xprop WM_CLASS` liefert `md-reader`.
- Mit leerem Test-HOME und `zenity` (für den Test installieren; im LXC fehlt es) erscheint die Installationsfrage. Bildschirmfoto machen, Klick auf „Ja“ per `xdotool key Return`.
- Danach existiert `<HOME>/.local/share/applications/md-reader.desktop`.

- [ ] **Schritt 1:** Skript schreiben und ausführen. Alle Punkte grün, die Bildschirmfotos ansehen.
- [ ] **Schritt 2:** Gefundene Fehler gehen zurück in die betroffene Aufgabe, mit Test zuerst.

### Aufgabe 10: Doku, Sicherheit, Build 47, Release

**Dateien:**
- Ändern:
  - `README.md` und `README.de.md`, Linux-Abschnitt:
    - Start, Installationsfrage, `--install` / `--uninstall`.
    - `.deb` per `sudo apt install ./md-reader_1.0.47_amd64.deb`.
    - Zielsysteme, glibc 2.38.
    - Wie man WebKit von Hand installiert.
  - `CHANGELOG.md` (englisch, „Build 47“), `FEATURES.md`, `BUGS.md` (Bug: Start ohne WebKit unter KDE neon zeigt nur die Loader-Meldung → behoben).
  - `MEMORY.md`, `dev-log/2026-10-24-dev.log`, `docs/index.html` (Download-Karten `.deb`).
  - `.claude/rules/build-release.md`:
    - Zielsysteme korrigieren.
    - Neuer Build-Ablauf.
    - 8 Assets.
    - Release-Befehl mit den `.deb`.
  - `.claude/rules/platform.md` (Starter, App-ID), `.claude/rules/tests.md` (neue Testzahl, Docker-Abnahme).
  - `.claude/agents-shared/ZUSTAENDIGKEITEN.md`:
    - `src/linuxsetup/**` und `src/launcher/**` → `specialist-platform`.
    - `installer/linux/**` → `specialist-release` (schon über `installer/**` abgedeckt).
    - `debugging/linux/**` → `analyst-debug`.
  - `src/build.txt` → 47.
- [ ] **Schritt 1:** Sicherheitsprüfung der Änderungen. Dazu den Skill `security-review` und das Plugin `security-guidance` heranziehen. Schwerpunkte:
  - `pkexec`-Befehle fest.
  - memfd/exec.
  - Installationspfade: kein Überschreiben fremder Dateien, Symlink-Falle bei `BinPath.tmp`; mit `O_EXCL` anlegen, eine vorhandene `.tmp` vorher entfernen.
  - Desktop-Datei-Quoting.
  - Neue Abhängigkeit `godbus`.
  - Funde beheben (mit Test) bzw. in `SECURITY_RISKS.md` / `SECURITY_FIXED.md` eintragen.
- [ ] **Schritt 2:** Alle Tests (`go vet ./... && go test ./...`), Testzahl zählen und eintragen.
- [ ] **Schritt 3:** Voller Build:
  - Linux x86_64, ARM64 und ARMhf jeweils mit Starter.
  - Windows, Setup.
  - 3 × `.deb`.
  - glibc-Wächter grün.
- [ ] **Schritt 4:** Commit (englisch, nur eigene Dateien), Push auf `github` und `origin`.
- [ ] **Schritt 5:** Release `v1.0.47` mit 8 Assets. Den Workflow `macos.yml` prüfen; das 9. Asset ist das macOS-Zip.
- [ ] **Schritt 6:** Auto-Memory: Build-Nummer 47, Starter-Architektur und Docker-Abnahme.
