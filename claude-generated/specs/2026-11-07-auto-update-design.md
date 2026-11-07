# Design: Auto-Update mit Signaturprüfung (Build 48)

Stand: 2026-11-07 · Autor: Kurt Ingwer · Ziel: Build 48

## 1. Anlass und Absicht

Der Nutzer (2026-10-25):
- „ich möchte ein auto update in diesem projekt haben das beim start versucht eine neue
  version zu laden und beim beenden das update ausführt.“
- „im projekt ssh-easy ist es auch schon mit signatur gelöst. das kann so gebaut werden.“

Vorbild ist `ssh-easy` (`src/updater*.go`, `src/releasesig/`, `src/cmd/release-sign/`):
- Ablauf: neuestes GitHub-Release abfragen, dann eine signierte Prüfsummenliste
  (`…-SHA256SUMS` + `.sig`, Ed25519) laden und prüfen. Danach die passende Datei herunterladen
  und prüfen (SHA-256, Größe, Dateiformat). Beim Beenden wird sie ausgetauscht.
- Der private Schlüssel liegt nur im LXC. Wer nur das GitHub-Konto oder den Token
  kontrolliert, kann kein Update unterschieben.

Von Claude angekündigt und vom Nutzer nicht widersprochen: Bei MD Reader ist das Auto-Update
**standardmäßig eingeschaltet** (bei ssh-easy ist es aus).

## 2. Ziele

- U1 **Beim Start** (im Hintergrund, ohne den Start zu verzögern):
  1. `https://api.github.com/repos/WaschbaerImHaus/text-reader/releases/latest` abfragen.
  2. Ist das Release neuer als der eigene Build (Tag `v1.0.<n>`), die signierte Liste
     `md-reader-SHA256SUMS` + `md-reader-SHA256SUMS.sig` laden und mit dem eingebetteten
     Ed25519-Schlüssel prüfen.
  3. Die Datei für die eigene Plattform herunterladen und prüfen: SHA-256 aus der Liste,
     Größengrenze, Dateiformat (ELF, PE bzw. ZIP mit `MD Reader.app`).
- U2 **Beim Beenden** die heruntergeladene Datei vor dem Austausch erneut prüfen und dann installieren:
  - **Linux, Starter-Datei** (Download-Ordner oder `~/.local/bin/md-reader`): die Starter-Datei
    atomar ersetzen (Rename im selben Verzeichnis). Voraussetzung: Das Verzeichnis ist beschreibbar.
  - **Linux, `.deb`** (`/usr/bin/md-reader`): nicht selbst ersetzen. Stattdessen einmal je
    neuer Version eine Desktop-Nachricht „Neue Version verfügbar“ mit Link zur Release-Seite.
    Es gibt kein apt-Repository.
  - **Windows, Setup-Installation** (`Program Files`): das neue Setup still (`/S`) mit
    Administratorrechten starten (UAC). Die Datei bleibt bis zum Start gesperrt geöffnet, wie bei ssh-easy.
  - **Windows, einzelne `.exe`** (beschreibbares Verzeichnis): die laufende Datei in `.old`
    umbenennen und die neue an ihre Stelle legen. `.old` wird beim nächsten Start gelöscht.
  - **macOS, `MD Reader.app`**: Das ZIP entpacken (nur Einträge unter `MD Reader.app/`, keine
    absoluten Pfade, kein `..`, keine Symlinks nach außen) in ein Nachbarverzeichnis des Bundles.
    Beim Beenden tauschen: altes Bundle nach `.old`, neues an seinen Platz, `.old` löschen.
    Ohne Schreibrecht wird nicht aktualisiert, nur gemeldet.
- U3 **Nach einem Update** zeigt der nächste Start in der App kurz „Aktualisiert auf Build <n>“.
- U4 **Abschaltbar** über die Konfiguration `"autoUpdate": false` in `state.json` (Standard: true).
- U5 **Signieren beim Release:**
  - Neues Werkzeug `src/cmd/release-sign` (wie ssh-easy) und Paket `src/releasesig`.
  - Schlüssel: `~/.config/md-reader/release-signing.key` (0600, nie im Repo, nie in Commits oder Berichten).
  - Der öffentliche Schlüssel ist im Programm eingebettet.
  - Die Liste enthält alle 9 Assets einschließlich `md-reader-macos.zip`. Die Kopfzeile bindet
    an den Tag: `md-reader release v1.0.<n>`.
- U6 **Release-Ablauf:**
  1. Release mit den 8 Assets anlegen. Der Workflow hängt das macOS-ZIP an.
  2. Im LXC das ZIP herunterladen, Prüfsummen bilden, signieren und
     `md-reader-SHA256SUMS` + `.sig` hochladen.
  3. Neues Make-Ziel `make sign-release BUILD_NUM=<n>`.
  4. Releases ohne die beiden Dateien ignoriert der Updater.

## 3. Nicht-Ziele

- Kein Update ohne gültige Signatur, keine Abschaltung der Prüfung.
- Kein eigenes apt-Repository (`.deb` nur Hinweis).
- Keine Delta-Updates, kein Rückweg auf ältere Versionen (Downgrade).
- Keine Telemetrie. Gesendet wird nur die GitHub-Abfrage, mit festem User-Agent `md-reader/<build>`,
  ohne Nutzerdaten.

## 4. Anti-Ziele

- Der Start wartet auf das Netz. Die Abfrage läuft im Hintergrund, Gesamtfrist 5 Minuten,
  die GUI ist sofort da.
- Ein abgebrochenes oder halbes Update macht die App unbrauchbar. Es gilt: atomarer Austausch,
  sonst alte Version behalten.
- Downloads von anderen Adressen als
  `https://github.com/WaschbaerImHaus/text-reader/releases/download/` bzw. GitHubs
  Weiterleitung auf `objects.githubusercontent.com`. Antworten ohne Größengrenze.
- Das Update ersetzt eine fremde Datei (Symlink-Falle) oder läuft mit Rechten, die der Nutzer
  nicht erwartet. Die UAC-Abfrage unter Windows ist ausdrücklich gewollt.
- Fehlermeldungen bei jedem Start, wenn kein Netz da ist. Netzfehler bleiben still und werden nur protokolliert.

## 5. Randbedingungen

- Plattformen: Linux amd64, arm64 und armhf (Starter), Windows amd64 (Setup bzw. `.exe`),
  macOS Universal (`.app`).
- Asset-Namen je Plattform:
  - `md-reader` (Linux amd64)
  - `md-reader-linux-arm64`
  - `md-reader-linux-armhf`
  - `md-reader-setup.exe`
  - `md-reader.exe`
  - `md-reader-macos.zip`
- Linux: Die GUI läuft aus einem memfd, der Starter ist nach `exec` nicht mehr da.
  - Der Starter gibt der GUI seinen aufgelösten Pfad über eine Umgebungsvariable mit
    (`MD_READER_LAUNCHER=<pfad>`, nur vom Starter gesetzt).
  - Die GUI ersetzt diese Datei beim Beenden.
  - Ohne diese Variable (`.deb`, direkter Aufruf) gibt es nur den Hinweis.
- Bestehende Härtungen bleiben erhalten:
  - Starter: `/proc/self/exe` als Installationsquelle, `--`, absolute Pfade.
  - Navigations-Erlaubnisliste, CSP.
- Build-Nummer 48. Tests per TDD. Release Notes, CHANGELOG und Commits englisch.
- Der LXC hat Netz zu GitHub. Für Tests gibt es einen lokalen HTTP-Server auf 127.0.0.1.
  Die Basis-URL ist im Testmodus über eine Paketvariable austauschbar, nicht über Umgebung oder Konfiguration.

## 6. Ebenfalls in Build 48 (offene Punkte aus Build 47, `BUGS.md`)

- M3: Meldung „Keine grafische Oberfläche“ zweisprachig (`linuxsetup.DetectLang`).
- M4: Den Angebotsprozess nur starten, wenn `DISPLAY` oder `WAYLAND_DISPLAY` gesetzt ist.
- M7: memfd in `missingVersions` schließen.
- zenity-Dialog „zu alt“: `--width=420`.

## 7. Dem Bauer überlassen

- Paketaufteilung:
  - Vorschlag: `src/releasesig/` (wie ssh-easy) und `src/updater/` mit Fachlogik
    (Release-Abfrage, Prüfung, Plattform-Austausch hinter Schnittstellen, testbar).
  - Anbindung in `main.go` beim Start und nach `w.Run()`.
  - Windows- und macOS-Teile per Build-Tags.
- Ob der Hinweis „Aktualisiert“ (U3) über die bestehende Seite (Toolbar-Toast) oder eine
  Desktop-Nachricht kommt. Bevorzugt in der App, ohne neue Bindings, wenn möglich.
- Protokoll: Fehler nach stderr bzw. in eine kleine Logdatei im Konfigurationsordner, höchstens 64 KB.

## 7a. Nach dem Bauer-Check von Claude entschieden

1. **Signieren in zwei Schritten.**
   - Direkt nach dem Release werden die 8 Assets signiert. Sobald das macOS-ZIP da ist, wird mit
     9 Assets neu signiert und die beiden Dateien werden ersetzt.
   - Der Updater kommt damit zurecht, dass sich die Liste eines Tags ändert: Der Tag ist gebunden,
     ein Downgrade ist nicht möglich.
   - Fehlt das eigene Asset in der Liste, gibt es kein Update. Sonst entsteht kein Fehler.
2. **macOS-Rückfall.** Lässt macOS den Austausch nicht zu (App-Verwaltung, kein Schreibrecht, ad hoc
   signiertes Bundle), gibt es nur den Hinweis „Neue Version verfügbar“ mit Link, wie beim `.deb`.
   Der Rauchtest auf dem Runner zeigt, welcher Weg gilt.
3. **Windows-Setup still.**
   - Das Setup übernimmt beim Update die bisherige Auswahl: kein neues Desktop-Icon, wenn keins da
     war, kein Ghostscript-Download.
   - Dafür gibt es einen Update-Schalter für das NSIS-Skript, z. B. `/S /UPDATE`.
4. **Linux-Nutzerinstallation.** Beim Austausch wird die Desktop-Datei mit der neuen Build-Nummer
   (`X-MD-Reader-Build`) neu geschrieben. Sonst bietet der Starter später fälschlich „Ersetzen“ an.
5. **Build-Nummer in der GUI.** Die GUI bekommt sie per `-ldflags -X` wie der Starter, auch im
   macOS-Skript und im Windows-Build.

## 7b. Vom Nutzer entschieden (2026-11-07: „ja, beide b“)

1. **macOS-ZIP wird vor dem Signieren gegengeprüft.** `make sign-release` lädt das ZIP nicht vom
   Release, sondern aus dem Artefakt des Workflow-Laufs.
   - Bedingung: Der Lauf gehört zum Commit des Tags (`head_sha` = Tag-Commit, Ereignis `release`
     bzw. `workflow_dispatch` auf diesem Commit).
   - Bedingung: `.github/workflows/macos.yml` ist in diesem Commit unverändert gegenüber der lokalen,
     committeten Fassung.
   - Dann wird es byte-genau mit dem Release-Asset verglichen. Bei jeder Abweichung wird nicht
     signiert (Fehler, klare Meldung).
   - Restrisiko „Vertrauen in GitHub/Runner“ kommt in `SECURITY_RISKS.md`.
2. **Zwei eingebettete öffentliche Schlüssel.**
   - Hauptschlüssel: `~/.config/md-reader/release-signing.key` im LXC.
   - Notfallschlüssel: wird einmalig im LXC erzeugt. Der private Teil wird dem Nutzer zur
     Offline-Aufbewahrung übergeben (Datei im Scratchpad + Hinweis) und danach im LXC gelöscht.
     Er steht nie im Repo, in Commits oder Berichten.
   - Ein Update gilt, wenn die Liste von **einem** der beiden Schlüssel gültig signiert ist.
   - `release-sign` kann mit jedem der beiden signieren (`-key`).
   - Die `.sig`-Datei enthält genau eine Signatur. Geprüft wird gegen beide öffentlichen Schlüssel.

## 8. Abnahme

- Unit-Tests:
  - Manifest bauen und prüfen (falscher Tag, falsche Signatur, fehlende Datei).
  - Versionsvergleich.
  - Asset-Auswahl je Plattform.
  - Nur erlaubte URL-Präfixe.
  - Größengrenzen.
  - Formatprüfung (ELF/PE/ZIP).
  - ZIP-Entpacken gegen Zip-Slip.
  - Atomarer Austausch.
  - `.old`-Aufräumen.
  - Abschaltbar über die Konfiguration.
  - Netzfehler bleibt still.
- Integration im LXC mit lokalem Release-Server (127.0.0.1), signiert mit einem
  Testschlüssel. Der Starter Build 48 tauscht sich beim Beenden gegen einen „Build 49“-Starter
  aus. Ein manipuliertes Asset wird abgelehnt.
- Windows: Cross-Build, Unit-Tests der Windows-Teile mit Wine, soweit möglich.
- macOS: Der Workflow läuft grün. Der Austausch wird im Rauchtest des Runners mit einem lokalen
  Release-Server geprüft.
- Echtes Release v1.0.48 mit `md-reader-SHA256SUMS` + `.sig`. Erst Build 49 kann sich auf
  Build 48 ersetzen, der erste echte Lauf ist also beim nächsten Release.
