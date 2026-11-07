# Auto-Update mit Signatur (Build 48) – Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Ziel:** MD Reader sucht beim Start ein neueres, signiertes GitHub-Release, lädt die passende Datei im Hintergrund und installiert sie beim Beenden. Das gilt für Linux-Starter, Windows (Setup bzw. `.exe`) und macOS (`.app`). Beim `.deb` und als Rückfall kommt nur ein Hinweis.

**Architektur:**
- `src/releasesig` (Manifest + Ed25519, zwei öffentliche Schlüssel) und `src/cmd/release-sign` (Werkzeug) nach dem Vorbild ssh-easy.
- `src/updater` mit der Fachlogik: Abfrage, Prüfung, Download, Auswahl des Installationswegs. Die Austauschschritte je Plattform liegen in Dateien mit Build-Tags hinter einer Schnittstelle.
- Die GUI (`src/main.go`) startet die Suche nach dem Fensteraufbau und wendet das Update nach `w.Run()` an.
- Hinweise erscheinen in der App (`scripts.js`).
- Das Signieren beim Release übernimmt ein Skript mit Gegenprüfung des macOS-ZIP aus dem Workflow-Artefakt.

**Tech Stack:** Go 1.27 (`crypto/ed25519`, `net/http`, `archive/zip`), NSIS, Bash + `gh`, GitHub Actions (macos-15).

**Spec:** `claude-generated/specs/2026-11-07-auto-update-design.md`. Die Abschnitte 7a und 7b gehen den älteren Abschnitten vor.

## Global Constraints

- **Repo:** `WaschbaerImHaus/text-reader`.
  - API: `https://api.github.com/repos/WaschbaerImHaus/text-reader/releases/latest`.
  - Download nur von Adressen mit dem Präfix `https://github.com/WaschbaerImHaus/text-reader/releases/download/`. Weiterleitungen sind nur auf `https://objects.githubusercontent.com/` und `https://release-assets.githubusercontent.com/` erlaubt, alles andere wird abgelehnt.
- **Tags und Versionen:** Tag `v1.0.<build>`. Ein Update gibt es nur bei strikt größerer Build-Nummer. Entwicklerbuilds (`buildNumber` = "0") suchen nicht.
- **Manifest:**
  - Dateien `md-reader-SHA256SUMS` und `md-reader-SHA256SUMS.sig`.
  - Kopfzeile genau `md-reader release v1.0.<n>`, danach je Asset `<sha256-hex>  <name>`, sortiert, mit `\n` als Zeilenende.
  - Signiert wird über `md-reader-release-manifest-v1\n` + Manifest. Die `.sig` enthält eine Ed25519-Signatur in Base64.
- **Schlüssel:**
  - Hauptschlüssel `~/.config/md-reader/release-signing.key` (0600, Seed als Hex).
  - Notfallschlüssel wird einmalig erzeugt, dem Nutzer übergeben und dann im LXC gelöscht.
  - Beide öffentlichen Schlüssel sind fest im Code eingetragen. Gültig ist eine Signatur, wenn einer der beiden passt.
  - Kein privater Schlüssel kommt je in Repo, Commit, Log oder Bericht.
- **Asset je Plattform:**

  | Plattform | Asset |
  |---|---|
  | linux/amd64 | `md-reader` |
  | linux/arm64 | `md-reader-linux-arm64` |
  | linux/arm | `md-reader-linux-armhf` |
  | windows/amd64, Setup-Installation | `md-reader-setup.exe` |
  | windows/amd64, sonst | `md-reader.exe` |
  | darwin | `md-reader-macos.zip` |

- **Grenzen:**
  - API-Antwort, SUMS und `.sig` höchstens 1 MiB, Asset höchstens 128 MiB.
  - Fristen: 15 s für die API-Abfrage, 5 min für alles zusammen.
  - User-Agent `md-reader/<build>`, keine weiteren Daten.
- **Konfiguration:**
  - `state.json` bekommt `"autoUpdate"`. Fehlt der Wert, gilt `true`, deshalb ist das Feld ein `*bool`.
  - Neu ist `"updatedToBuild": <n>` für den einmaligen Hinweis „Aktualisiert auf Build <n>“.
  - Neu ist `"updateNoticeBuild": <n>`, damit „Neue Version verfügbar“ je Version nur einmal erscheint.
- **Starter → GUI:** Umgebungsvariable `MD_READER_LAUNCHER=<aufgelöster Starter-Pfad>`, gesetzt nur vom Starter direkt vor `execGUI`.
- **Windows-Setup:**
  - Im Update-Modus läuft `md-reader-setup.exe /S /UPDATE`.
  - Dabei bleiben Desktop-Icon (nur wenn schon vorhanden) und alle übrigen Einstellungen erhalten.
  - Es gibt keinen Ghostscript-Download und keine MessageBox.
- **Build-Nummer der GUI:** `-X main.buildNumber=<n>`, gesetzt im Makefile (Linux-Container und Windows) und in `installer/macos/build-app.sh`.
- **Stil:**
  - Englische Bezeichner, deutsche DocBlocks mit `Autor: Kurt Ingwer` und `Letzte Änderung: <date +%F> (…)`, Tabs.
  - TDD: zuerst RED aus dem richtigen Grund.
  - Tests nie außerhalb von `t.TempDir()` aufräumen (`.claude/rules/tests.md`).
- **Prozess:**
  - Subagents committen und pushen nicht; das übernimmt die Hauptsitzung nach der Prüfung.
  - Nie anfassen: `SECURITY_HISTORY.md`, `SECURITY_REPORT.md`, `debugging/2026-10-14-offene-fragen.md`.
  - Neue Dateien trägt die Hauptsitzung in `ZUSTAENDIGKEITEN.md` ein.
- **Bestehende Härtungen bleiben erhalten:** Navigations-Erlaubnisliste, CSP, Starter-Regeln (`/proc/self/exe`, `--`, absolute pkexec/apt-get).

## Review Focus

1. **Austausch scheitert mittendrin** (Platte voll, Datei gesperrt, Stromausfall): Die alte Version muss lauffähig bleiben. Es gibt keine halbe Datei unter dem Zielnamen. Tests: Aufgabe 4 (`TestReplaceFile_CrashLeavesOld`) und Aufgabe 6 (`TestSwapBundle_FailureRestoresOld`).
2. **Ein Release ohne SUMS/`.sig`, mit SUMS ohne eigenes Asset oder mit neu signierter Liste** (Zwei-Schritt-Signatur): kein Update, kein Fehlerdialog, nur ein Logeintrag. Test: Aufgabe 3 (`TestCheck_UnsignedOrIncompleteRelease`).
3. **Kein Netz, Captive Portal (HTML statt JSON/Binärdatei) oder GitHub-Ratenlimit (403):** still, kein Absturz, kein Hinweis. Tests: Aufgabe 3 (`TestCheck_NetworkErrorsAreSilent`) und Aufgabe 3 (`TestDownload_RejectsHTML`).
4. **Zwei MD-Reader-Fenster gleichzeitig beendet:** nur eines installiert, das zweite erkennt die schon ersetzte Datei. Ein Sperrdatei-Lock (`O_EXCL`) liegt im Konfigurationsordner. Test: Aufgabe 4 (`TestApply_SecondInstanceSkips`).
5. **Rechte fehlen** (Starter in `/opt`, `.app` in fremdem Ordner, `.exe` in `Program Files` ohne Setup): Es gibt nur den Hinweis mit Link, keinen Fehlversuch bei jedem Start. Test: Aufgabe 3 (`TestDecideMode_NoWriteAccessGivesNotice`).

---

### Aufgabe 1: Manifest und Signatur (`src/releasesig`)

**Dateien:** neu `src/releasesig/releasesig.go`, `src/releasesig/releasesig_test.go`

**Schnittstellen (erzeugt):**
- `func BuildManifest(tag string, sums map[string]string) []byte`
- `func ParseManifest(data []byte, tag string) (map[string]string, error)`: Falscher Tag, doppelter Name, ungültiger Hexwert oder Dateiname mit `/`, `\` oder `..` ergeben einen Fehler.
- `func Sign(priv ed25519.PrivateKey, manifest []byte) []byte` (Base64 mit `\n`)
- `func Verify(pubs []ed25519.PublicKey, manifest, sig []byte) error`: nil, wenn einer der Schlüssel passt.
- `func GenerateKey(path string) (ed25519.PublicKey, error)`: schreibt den Seed als Hex mit 0600 und `O_EXCL`, überschreibt also nie.
- `func LoadPrivateKey(path string) (ed25519.PrivateKey, error)`: Fehler, wenn die Datei Gruppen- oder Fremdrechte hat (nur unter Unix).
- `func SignFiles(priv ed25519.PrivateKey, tag, outDir string, files []string) error`: schreibt `md-reader-SHA256SUMS` und `.sig`.

Vorlage ist `/home/claude-code/project/ssh-easy/src/releasesig/releasesig.go`. Unterschiede: Namen nach den Global Constraints und `Verify` mit einer Liste von Schlüsseln.

- [ ] **Schritt 1: Tests**
  - `TestManifest_RoundTrip`
  - `TestParseManifest_WrongTag`: Liste für `v1.0.48`, geprüft gegen `v1.0.49` → Fehler.
  - `TestParseManifest_BadLines`: doppelte Namen, `../x`, Großbuchstaben-Hex, falsche Länge.
  - `TestVerify_EitherKey`: Signatur mit Schlüssel B, geprüft gegen `[A, B]` → ok, gegen `[A]` → Fehler.
  - `TestVerify_TamperedManifest`
  - `TestGenerateKey_NoOverwrite`
  - `TestLoadPrivateKey_RejectsGroupReadable`
  - `TestSignFiles_WritesBoth`
- [ ] **Schritt 2:** RED (`cd src && go test ./releasesig/ -v`).
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 2: Signierwerkzeug (`src/cmd/release-sign`)

**Dateien:** neu `src/cmd/release-sign/main.go`, `src/cmd/release-sign/main_test.go`

**Schnittstellen:**
- Verwendet: Aufgabe 1.
- Erzeugt die CLI:
  - `release-sign -genkey -key <pfad>`: gibt den öffentlichen Schlüssel als Hex auf stdout aus.
  - `release-sign -key <pfad> -tag v1.0.<n> -out <dir> <dateien…>`
  - `release-sign -pub -key <pfad>`: gibt den öffentlichen Schlüssel eines vorhandenen Schlüssels aus.
  - Exit-Codes: 0 ok, 1 Fehler, 2 Aufruf falsch.

- [ ] **Schritt 1: Tests**
  - Über `run(args []string, stdout, stderr io.Writer) int`: genkey, dann sign, dann mit `releasesig.Verify` prüfen.
  - Ein falscher Tag (nicht `v1.0.<zahl>`) ergibt Exit 2.
  - Ein fehlender Schlüssel ergibt Exit 1.
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 3: Updater-Kern (`src/updater`)

**Dateien:**
- Neu: `src/updater/updater.go` (Typen, Ablauf), `src/updater/fetch.go` (HTTP, Grenzen, URL-Prüfung), `src/updater/format.go` (ELF/PE/ZIP-Kopf), `src/updater/keys.go` (eingebettete öffentliche Schlüssel)
- Tests: `src/updater/*_test.go`

**Schnittstellen (erzeugt):**
- `type Mode int` mit den Werten `ModeNone` (kein Update, z. B. Entwicklerbuild), `ModeNoticeOnly` (`.deb`, kein Schreibrecht, macOS-Rückfall), `ModeReplaceFile` (Linux-Starter, Windows-`.exe`), `ModeRunSetup` (Windows-Setup) und `ModeSwapBundle` (macOS).
- `type Target struct { Mode Mode; Path string; Asset string }`. `Path` ist die zu ersetzende Datei bzw. das `.app`-Verzeichnis.
- `type Env struct { GOOS, GOARCH string; Executable string; LauncherPath string; Writable func(dir string) bool; SetupInstalled func() bool }`
- `func DecideTarget(e Env) Target`
  - Linux: Ist `LauncherPath` gesetzt und sein Verzeichnis beschreibbar, gilt `ModeReplaceFile` auf `LauncherPath`. Sonst gilt `ModeNoticeOnly`.
  - Windows: `SetupInstalled()` ergibt `ModeRunSetup`. Eine `.exe` in beschreibbarem Verzeichnis ergibt `ModeReplaceFile`. Sonst gilt `ModeNoticeOnly`.
  - darwin: Liegt `Executable` unter `<x>.app/Contents/MacOS/` und ist das Elternverzeichnis von `<x>.app` beschreibbar, gilt `ModeSwapBundle`. Sonst gilt `ModeNoticeOnly`.
- `type Release struct { Tag string; Build int; Assets map[string]string }` (Name auf Download-URL)
- `type Pending struct { Target Target; Build int; File string; SHA256 string }`
- `type Checker struct { CurrentBuild int; Keys []ed25519.PublicKey; Client *http.Client; APIURL string; DownloadDir string; Log io.Writer }`
- `func (c *Checker) Check(ctx context.Context, t Target) (*Pending, *Release, error)`:
  - Bei `ModeNoticeOnly` wird nur abgefragt, `Pending` ist dann nil und `Release` gesetzt, wenn neuer.
  - Sonst werden SUMS und `.sig` geprüft und die Datei geladen. Ihr SHA-256, ihre Größe und ihr Format werden geprüft. Danach ist `Pending` gesetzt.
- `func ParseBuild(tag string) (int, bool)`
- `var publicKeys = []string{"<haupt-hex>", "<notfall-hex>"}` in `keys.go`. Die Werte trägt Aufgabe 9 ein. Bis dahin stehen dort Platzhalter, und `Check` meldet mit leerer Liste einen Fehler.
- `func DefaultKeys() ([]ed25519.PublicKey, error)`
- `apiURL`, die Präfixliste und die Grenzen sind unexportierte Paketvariablen. Tests ersetzen sie per `httptest`. Für den macOS-Rauchtest lassen sich `apiURL` und ein zusätzlicher Testschlüssel `testPublicKeyHex` nur per `-ldflags -X` setzen (Aufgabe 7). Es gibt keinen Umgebungs- und keinen Konfigurationsweg.

- [ ] **Schritt 1: Tests**
  - Mit `httptest.NewServer` auf 127.0.0.1, Testschlüssel aus `ed25519.GenerateKey`. Die Präfixliste wird im Test auf die Server-URL gesetzt.
  - `TestParseBuild`: `v1.0.48` → 48, `v1.0.x`/`1.0.48`/`v2.0.1` → false.
  - `TestDecideTarget_Table` mit allen Zweigen oben, darunter `TestDecideMode_NoWriteAccessGivesNotice` (Review Focus 5).
  - `TestCheck_HappyPath`: Pending gesetzt, Datei im `DownloadDir` mit 0600, Hash stimmt.
  - `TestCheck_NotNewer`: gleiche oder ältere Build-Nummer → kein Download.
  - `TestCheck_UnsignedOrIncompleteRelease` (Review Focus 2): ohne SUMS, ohne `.sig`, mit falscher Signatur, mit SUMS ohne das eigene Asset → nil, nil, Fehler nur im Log.
  - `TestCheck_HashMismatch`: Datei wird gelöscht.
  - `TestCheck_SizeLimit`: Asset größer als die Grenze wird abgebrochen und gelöscht.
  - `TestFetch_RejectsForeignURL`: Asset-URL auf einem fremden Host, Weiterleitung auf einen fremden Host.
  - `TestDownload_RejectsHTML` (Review Focus 3): 200 mit `<!DOCTYPE html>` → Formatfehler.
  - `TestCheck_NetworkErrorsAreSilent`: geschlossener Port, 403-Ratenlimit, Zeitüberschreitung → Fehler zurück, kein Panic.
  - `TestFormat_ELF_PE_ZIP` mit den Magic-Bytes `\x7fELF`, `MZ` und `PK\x03\x04`.
  - `TestCheck_NoticeOnlyDoesNotDownload`
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen. Weiterleitungen per `CheckRedirect` auf die erlaubten Präfixe begrenzen. Vor dem Lesen `io.LimitReader(grenze+1)`. Die Datei wird mit `os.CreateTemp(DownloadDir, "md-reader-update-*")` angelegt.
- [ ] **Schritt 4:** GREEN, außerdem `GOOS=windows`/`darwin go vet ./updater/`.

### Aufgabe 4: Anwenden unter Linux und für Windows-`.exe` (`ModeReplaceFile`)

**Dateien:** neu `src/updater/apply.go` (gemeinsam: Lock, erneute Prüfung, Verteilung auf die Modi), `src/updater/replace_unix.go`, `src/updater/replace_windows.go`; Tests `src/updater/apply_test.go`

**Schnittstellen:**
- Verwendet: `Pending`, `Target` aus Aufgabe 3, `linuxsetup.DesktopEntry` und `linuxsetup.NewLayout` (Desktop-Datei der Nutzerinstallation).
- Erzeugt:
  - `func Apply(p *Pending, lockDir string) (applied bool, err error)`
    - Prüft den SHA-256 unmittelbar vor dem Austausch erneut.
    - Hält ein Lock `lockDir/update.lock` mit `O_CREATE|O_EXCL`, das danach entfernt wird.
    - Ist das Ziel schon die neue Datei (gleicher Hash), gibt es `applied=false` und keinen Fehler.
  - `func replaceFile(target, newFile string) error`
    - Unix: `newFile` auf Modus 0755 setzen und `os.Rename` im selben Verzeichnis. Liegt der Download in einem anderen Dateisystem, erst `target+".new"` mit `O_EXCL` im Zielverzeichnis anlegen und dann umbenennen.
    - Windows: Laufende `.exe` → `target+".old"`, neue Datei → `target`. Schlägt der zweite Schritt fehl, wird `.old` zurückbenannt.
  - `func CleanupOld(exe string)`: entfernt `<exe>.old` und verwaiste `md-reader-update-*` im Download-Ordner. Läuft beim Start.
  - Linux-Nutzerinstallation: Ist `target` gleich `Layout.BinPath`, wird die Desktop-Datei danach mit der neuen Build-Nummer neu geschrieben, mit `X-MD-Reader-Build=<neu>` (Spec 7a.4).
- [ ] **Schritt 1: Tests**
  - `TestApply_ReplacesAndKeepsMode`
  - `TestApply_HashChangedSinceDownload`: Datei wird zwischendurch verändert → Fehler, Ziel unverändert.
  - `TestReplaceFile_CrashLeavesOld`: Rename scheitert, z. B. weil das Ziel ein Verzeichnis ohne Schreibrecht ist → alte Datei unverändert, keine `.new`-Reste.
  - `TestApply_SecondInstanceSkips` (Review Focus 4)
  - `TestApply_UpdatesDesktopEntryBuild`: TempDir-HOME mit Nutzerinstallation.
  - `TestCleanupOld`
  - Windows-Teil: `replace_windows_test.go` (`//go:build windows`), läuft unter Wine oder wird nur übersetzt (`GOOS=windows go test -c`).
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 5: Windows-Setup-Weg (`ModeRunSetup`) und NSIS `/UPDATE`

**Dateien:** neu `src/updater/setup_windows.go`, `src/updater/setup_other.go` (Stub); Änderung `installer/md-reader.nsi`

**Schnittstellen:**
- `func runSetup(path, sha256 string) error`
  - Datei öffnen und sperren (Freigabe nur zum Lesen), Hash über den offenen Handle prüfen.
  - Prüfen, dass im Verzeichnis nur das Setup liegt (keine untergeschobenen DLLs).
  - `ShellExecuteW` mit `runas`, Parameter `/S /UPDATE`. Den Handle bis nach dem Start offen halten.
  - Vorlage ist ssh-easy `updater_windows.go` (`runVerifiedSetupInstaller`).
- `func SetupInstalled() bool`: Registrierungsschlüssel `HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\MD Reader` vorhanden und `InstallLocation` gleich dem Verzeichnis der laufenden `.exe`.
- NSIS:
  - `${GetOptions} $CMDLINE "/UPDATE"` setzt `$IsUpdate`.
  - Im Update-Modus:
    - Die Sektion Desktop-Icon nur auswählen, wenn `$DESKTOP\MD Reader.lnk` existiert.
    - Die Sektion Ghostscript abwählen und keine MessageBox zeigen.
    - Der Startmenüordner bleibt aus der Registry.
- [ ] **Schritt 1:** Tests für den Stub (`SetupInstalled()==false` außerhalb von Windows). Für Windows: `GOOS=windows go vet` und ein Wine-Test von `verifyLockedFile`, falls möglich. NSIS: `makensis` baut. Ein Probelauf `md-reader-setup.exe /S /UPDATE` unter Wine in einem Scratch-Präfix zeigt, dass ohne vorheriges Desktop-Icon keines angelegt wird; ist Wine ungeeignet, dokumentieren.
- [ ] **Schritt 2:** Umsetzen.
- [ ] **Schritt 3:** Prüfen.

### Aufgabe 6: macOS-Weg (`ModeSwapBundle`)

**Dateien:** neu `src/updater/unzip.go` (plattformneutral, sicher entpacken), `src/updater/bundle_darwin.go`, `src/updater/bundle_other.go`; Tests `src/updater/unzip_test.go`, `src/updater/bundle_test.go` (Austausch auf Linux mit Verzeichnissen testbar)

**Schnittstellen:**
- `func extractApp(zipPath, destDir string) (appDir string, err error)`
  - Nur Einträge unter `MD Reader.app/`.
  - Abgelehnt werden: absolute Pfade, `..`, Symlinks, die aus dem Bundle zeigen, mehr als 2000 Einträge, entpackt mehr als 256 MiB.
  - Modi aus dem ZIP; das Ausführbar-Bit bleibt erhalten.
- `func swapBundle(oldApp, newApp string) error`
  - `oldApp` → `oldApp+".old"`, dann `newApp` → `oldApp`, dann `.old` entfernen.
  - Bei einem Fehler im zweiten Schritt `.old` zurückbenennen.
- Beim Check (darwin) entpackt der Updater vorab nach `<elternverzeichnis>/.md-reader-update-<zufall>/`. Gelingt das nicht wegen fehlender Rechte, gilt `ModeNoticeOnly` (Spec 7a.2).
- [ ] **Schritt 1: Tests**
  - `TestExtractApp_ZipSlip`: `../x`, `/etc/x`, Symlink `MD Reader.app/Contents/x → /etc`.
  - `TestExtractApp_KeepsExecBit`
  - `TestExtractApp_Limits`
  - `TestSwapBundle_Success`
  - `TestSwapBundle_FailureRestoresOld`, z. B. wenn `newApp` fehlt (Review Focus 1).
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN.

### Aufgabe 7: Einbindung in GUI, Starter, Build und Hinweise

**Dateien:** Änderungen in
- `src/main.go`, `src/config.go`
- `src/bindings.go` (nur falls ein Binding nötig ist)
- `src/ui/assets/scripts.js`, `src/ui/template_test.go`
- `src/launcher/main.go`
- `Makefile` (GUI-ldflags `-X main.buildNumber=$(BUILD_NUM)` für Linux im Container und Windows)
- `installer/macos/build-app.sh` (`-X main.buildNumber`)
- `.github/workflows/macos.yml` und `debugging/macos/smoke-test.sh` (Rauchtest mit lokalem Release-Server)

**Schnittstellen:**
- `var buildNumber = "0"` im Paket main.
- `AppConfig` bekommt:
  - `AutoUpdate *bool \`json:"autoUpdate,omitempty"\``, `func (c AppConfig) autoUpdateEnabled() bool` (nil bedeutet true)
  - `UpdatedToBuild int \`json:"updatedToBuild,omitempty"\``
  - `UpdateNoticeBuild int \`json:"updateNoticeBuild,omitempty"\``
- **Ablauf:**
  1. Nach `webview.New`: `updater.CleanupOld`. Dann eine Goroutine: `DecideTarget` und `Check` mit einer Frist von 5 min, Ergebnis über einen Kanal.
  2. Vor `w.Run()`, falls `UpdatedToBuild == buildNumber`: Hinweis „Aktualisiert auf Build <n>“ per `w.Dispatch` / `w.Eval("showUpdateNotice(…)")` mit einem festen Text ohne Nutzerdaten. Danach `UpdatedToBuild = 0`.
  3. Ergebnis `ModeNoticeOnly` mit neuerem Release und `UpdateNoticeBuild < neu`: Hinweis „Neue Version verfügbar (Build <n>)“ mit Knopf „Release-Seite öffnen“. Das öffnet `https://github.com/WaschbaerImHaus/text-reader/releases/latest` über das vorhandene `openExternal`. Danach `UpdateNoticeBuild` setzen.
  4. Nach `w.Run()`: Liegt `Pending` vor, `Apply` ausführen. Bei Erfolg `UpdatedToBuild = p.Build` speichern. Fehler gehen ins Log.
- **Ruling (Abweichung von Spec U2):** Der Hinweis „Neue Version verfügbar“ (`.deb` und Rückfälle) erscheint in der App, nicht als Desktop-Nachricht. Gründe: einheitlich auf allen Plattformen, keine D-Bus-Abhängigkeit in der GUI.
- **Log:** `<UserConfigDir>/md-reader/update.log`, höchstens 64 KB. Beim Überschreiten wird auf die zweite Hälfte gekürzt.
- **scripts.js:**
  - `function showUpdateNotice(text, linkLabel)`: Leiste am oberen Rand, schließbar. Der Link ruft `openExternalLink` mit der festen Release-URL auf. Text als `textContent`, nie `innerHTML`.
  - Verdrahtet in `initUIListeners`, ohne Inline-Handler (CSP).
- **Starter:** setzt `MD_READER_LAUNCHER=<runningBinaryPath>` in der Umgebung von `execGUI` und nur dort. Test erweitert den argv/env-Exec-Test.
- **macOS-Rauchtest:**
  - Ein zweites Bundle wird mit `-X md-reader/updater.apiURL=http://127.0.0.1:<port>/api` und `-X md-reader/updater.testPublicKeyHex=<test-pub>` gebaut, nur im Workflow, nie für das Release-ZIP.
  - Ein lokaler Python-Server auf 127.0.0.1 liefert ein Fake-Release „Build 99999“ mit dem normalen ZIP und einer mit dem Testschlüssel signierten Liste.
  - Die App starten, sie per System Events beenden, dann prüfen, dass `Contents/Info.plist` die neue Version trägt oder dass der Rückfall-Hinweis kam.
  - Das Ergebnis geht ins Artefakt.
- [ ] **Schritt 1: Tests**
  - `TestConfig_AutoUpdateDefaultTrue`, wenn das Feld fehlt.
  - `TestConfig_AutoUpdateFalse`
  - `TestScripts_UpdateNoticeUsesTextContent`
  - `TestLauncher_SetsLauncherEnvOnlyForGUI`
  - Für die Reihenfolge in `main.go`: Ablauf in eine testbare Funktion `runUpdateFlow(cfg, check func…, apply func…)` auslagern; Test mit Fakes.
- [ ] **Schritt 2:** RED.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** GREEN. Dazu Windows-Cross-Build, `macos.yml`-Probelauf (`gh workflow run`) und Rauchtest-Artefakt ansehen.

### Aufgabe 8: Offene Punkte aus Build 47

**Dateien:** `src/platform_linux.go`, `src/launcher/main.go`, `src/linuxsetup/asker_dialog.go` + Tests

- M3: Die Meldung „Keine grafische Oberfläche“ wird zweisprachig (`linuxsetup.DetectLang`). Deutscher Text bleibt wörtlich; englisch: „MD Reader: No graphical display found (DISPLAY/WAYLAND_DISPLAY not set or not reachable).“
- M4: `maybeSpawnOffer` nur, wenn `DISPLAY` oder `WAYLAND_DISPLAY` nicht leer ist. Test `TestRun_NoDisplayNoOffer`.
- M7: Den memfd in `missingVersions` schließen.
- zenity: `--width=420` für Ask und Inform. Den Argumentlisten-Test anpassen.
- [ ] **Schritt 1:** Tests (RED).
- [ ] **Schritt 2:** Umsetzen.
- [ ] **Schritt 3:** GREEN.

### Aufgabe 9: Schlüssel, Signier-Ablauf und Release-Skript

**Dateien:** neu `installer/release/sign-release.sh`; Änderungen `Makefile` (Ziel `sign-release`), `src/updater/keys.go` (echte öffentliche Schlüssel), `.claude/rules/build-release.md`, `SECURITY_RISKS.md` (Restrisiko GitHub/Runner beim macOS-ZIP, Schlüsselverlust)

Die Schlüssel erzeugt die **Hauptsitzung** selbst, kein Subagent, wegen der Geheimnisse:
1. `go run ./cmd/release-sign -genkey -key ~/.config/md-reader/release-signing.key` → Hauptschlüssel.
2. `-genkey -key <scratchpad>/md-reader-emergency.key` → Notfallschlüssel.
3. Beide öffentlichen Hex-Werte in `keys.go` eintragen.
4. Die Notfall-Datei dem Nutzer übergeben (`SendUserFile`) und nach seiner Bestätigung im LXC löschen.

**`sign-release.sh <n>`:**
1. `gh release view v1.0.<n>` holen.
2. Die 8 lokalen Dateien aus `build/` mit den Release-Assets vergleichen (SHA-256). Bei Abweichung abbrechen.
3. macOS:
   - Den Workflow-Lauf `macos.yml` mit `head_sha` = Tag-Commit und Status `success` suchen.
   - Prüfen, dass `git show <tag-commit>:.github/workflows/macos.yml` byte-gleich mit `HEAD:.github/workflows/macos.yml` ist.
   - Das Artefakt `md-reader-macos` per `gh run download` in ein eigenes leeres Scratch-Verzeichnis laden.
   - Es byte-gleich mit dem Release-Asset `md-reader-macos.zip` vergleichen.
   - Fehlt der Lauf oder das ZIP noch: mit 8 Assets signieren und den Hinweis „später erneut ausführen“ ausgeben (Spec 7a.1).
4. `release-sign -key ~/.config/md-reader/release-signing.key -tag v1.0.<n> -out <scratch> <dateien>`.
5. `gh release upload v1.0.<n> <scratch>/md-reader-SHA256SUMS <scratch>/md-reader-SHA256SUMS.sig --clobber`.
6. Nach dem Hochladen die Signatur mit dem öffentlichen Schlüssel aus `keys.go` gegenprüfen (`go run` eines kleinen Prüfmodus von `release-sign`, z. B. `-verify`).

Weitere Vorgaben:
- Den Token nie ausgeben. Das Skript liest `GH_TOKEN` aus der Umgebung.
- Make-Ziel: `make sign-release BUILD_NUM=<n>`.
- [ ] **Schritt 1:** Skript mit `bash -n` prüfen. Probelauf gegen das echte Release v1.0.47, und zwar nur Prüfteil und Signieren ins Scratchpad, ohne Upload (`--dry-run`). Erwartet: alle 9 Dateien gleich, macOS-Lauf gefunden.
- [ ] **Schritt 2:** Regeln, `SECURITY_RISKS.md` und den Release-Abschnitt in `.claude/rules/build-release.md` anpassen (Ablauf: Release, Workflow abwarten, `make sign-release`).

### Aufgabe 10: Integrationstest, Doku, Build 48, Release

**Dateien:** neu `debugging/update/update-e2e.sh` (lokaler Release-Server auf 127.0.0.1, Testschlüssel, Starter-Austausch Build 48 → „Build 49“ unter Xvfb); Doku wie bei Build 47 (worker-docs); `src/build.txt` → 48

- [ ] **Schritt 1: E2E im LXC**
  - Einen Starter mit `-X …apiURL=http://127.0.0.1:<port>/api -X …testPublicKeyHex=<test>` bauen.
  - Unter Xvfb starten, Fenster schließen. Danach ist die Starter-Datei die „Build 49“-Datei, und `--launcher-version` gibt 49 aus.
  - Gegenproben:
    - manipuliertes Asset → nicht ersetzt,
    - unsignierte Liste → nicht ersetzt,
    - `autoUpdate:false` → keine Anfrage im Server-Log.
- [ ] **Schritt 2:** Sicherheitsprüfung (`analyst-security-audit`) über den ganzen Build-48-Diff, Funde durch `specialist-security`.
- [ ] **Schritt 3:** Doku:
  - README (Auto-Update, Abschalten, Datenschutz: nur GitHub-Abfrage), CHANGELOG Build 48.
  - FEATURES (Auto-Update erledigt), BUGS (M3/M4/M7/zenity erledigt).
  - MEMORY, dev-log, docs/index.html, `rules/platform.md`, `rules/tests.md`.
- [ ] **Schritt 4:** Schlussprüfung (`analyst-code-review`, Fable).
- [ ] **Schritt 5:** Endbuild aller 8 Assets aus dem committeten Stand, Endabnahme (`debugging/linux/launcher-docker-test.sh` und `verify-deb-resolve.sh 48`).
- [ ] **Schritt 6:** Commit und Push auf beide Remotes. Danach Release v1.0.48, macOS-Workflow abwarten und `make sign-release BUILD_NUM=48`. Prüfen, dass das Release 11 Assets hat (9 + SUMS + `.sig`).
