# Design: Linux-Starter mit WebKit-Prüfung, Selbst-Installation und .deb-Paket

Stand: 2026-10-24 · Autor: Kurt Ingwer · Ziel: Build 47

## 1. Anlass und Absicht

Der Nutzer hat unter **KDE neon (aktuell, Basis Ubuntu 24.04)** die Linux-Datei
gestartet und bekam:

```
./md-reader: error while loading shared libraries: libwebkit2gtk-4.1.so.0: cannot open shared object file
```

KDE baut auf Qt auf und bringt WebKitGTK nicht mit. Der dynamische Lader bricht
den Start ab, **bevor** Go-Code läuft, deshalb kann die App selbst nichts melden.

Der Nutzer will:
1. Beim Start prüfen, ob WebKit da ist. Fehlt es, soll eine **Desktop-Nachricht**
   erscheinen, die anbietet, es **per apt nachzuinstallieren**.
2. MD Reader **installierbar** machen, mit dem vorhandenen App-Icon im **Anwendungsmenü**,
   und zwar als **Selbst-Installation aus der einen Datei**.
3. **Zusätzlich ein `.deb`-Paket.**

Verworfen wurde ein AppImage mit eingebautem WebKit: zu groß (80–100 MB), und das
eingebaute WebKit bekäme keine Sicherheitsupdates über das System. Das wiegt schwer,
weil MD Reader fremde Dokumente anzeigt.

## 2. Ziele

- G1: Die Datei `md-reader` (sowie `-linux-arm64`, `-linux-armhf`) bleibt **eine einzige
  Datei**. Fehlen Bibliotheken, startet sie trotzdem und meldet das verständlich.
- G2: Fehlt WebKitGTK, erscheint eine Desktop-Nachricht „WebKitGTK ist nicht installiert“
  mit dem Knopf **„Jetzt installieren“**. Ein Klick führt über die grafische
  Passwortabfrage (polkit/`pkexec`) `apt-get install -y libwebkit2gtk-4.1-0` aus.
  Danach startet MD Reader von selbst.
- G3: Beim ersten Start fragt MD Reader einmal **„MD Reader ins Anwendungsmenü
  aufnehmen?“** mit den Knöpfen *Ja*, *Später* und *Nie*. Zusätzlich gibt es
  `md-reader --install` und `md-reader --uninstall`.
- G4: Die Installation erfolgt nur für den Benutzer, ohne Root:
  - Programm nach `~/.local/bin/md-reader`
  - Desktop-Eintrag nach `~/.local/share/applications/io.github.waschbaerimhaus.md-reader.desktop`
  - Icons nach `~/.local/share/icons/hicolor/{16x16,32x32,48x48,256x256}/apps/` und `scalable/apps/`
  - Kategorie **Office** (KDE: „Büro“)
  - `MimeType=` für md, markdown, txt, epub, fb2, tex, html, htm, pdf und ps, sodass
    „Öffnen mit“ funktioniert. MD Reader wird **nicht** ungefragt zur Standard-App.
- G5: Startet der Nutzer eine **neuere** heruntergeladene Datei, erkennt sie die
  installierte (über die Build-Nummer) und bietet an, sie zu ersetzen.
- G6: Zum Release kommt pro Linux-Architektur ein **`.deb`** hinzu (amd64, arm64, armhf):
  - `Depends:` WebKitGTK und libc in der passenden Mindestversion
  - `Recommends:` zenity, poppler-utils, ghostscript
  - Installiert systemweit nach `/usr/bin`, `/usr/share/applications` und `/usr/share/icons/hicolor`
- G7: Im Menü, in der Taskleiste und im Fensterwechsler erscheint das MD-Reader-Icon,
  auch unter Wayland. Dafür muss die App-ID beziehungsweise WM_CLASS des Fensters zum
  Namen der Desktop-Datei passen.

## 3. Nicht-Ziele

- Kein AppImage, kein eingebautes WebKit.
- Keine automatische Installation auf Distributionen ohne `apt`. Fedora (`dnf install
  webkit2gtk4.1`), Arch (`pacman -S webkit2gtk-4.1`) und openSUSE (`zypper install
  libwebkit2gtk-4_1-0`) bekommen den passenden Befehl in der Nachricht, mehr nicht.
- Keine Änderung an Windows (Setup bleibt NSIS) oder macOS.
- Keine systemweite Installation aus der Datei selbst, dafür gibt es das `.deb`.
- MD Reader wird nicht ungefragt Standard-App für Dateitypen.

## 4. Anti-Ziele (wäre ein Fehlschlag, selbst wenn es technisch läuft)

- Der Start wird spürbar langsamer, obwohl alles installiert ist.
- Die Installationsfrage erscheint bei jedem Start oder nach „Nie“ erneut.
- Inhalte aus Dokumenten, Dateinamen oder Umgebungsvariablen gelangen in den
  `pkexec`/`apt`-Befehl. Der Befehl ist fest, wird ohne Shell aufgerufen, Paketname konstant.
- Es bleiben Dateien liegen, die der Nutzer nicht findet. Ausnahme: ein klar benannter
  Cache, den `--uninstall` mit entfernt.
- Das `.deb` zeigt die Installationsfrage. Es ist schon installiert.
- Das Fenster zeigt in der Taskleiste ein generisches Icon statt des MD-Reader-Icons.

## 5. Randbedingungen

- Build-Maschine ist der LXC mit **Ubuntu 25.10** (glibc 2.42).
  - Die heutige Linux-Datei braucht höchstens `GLIBC_2.38` und läuft damit auf
    Ubuntu 24.04 / KDE neon (glibc 2.39).
  - Das darf durch den Starter **nicht** steigen. Der Starter ist reines Go
    (`CGO_ENABLED=0`), also statisch, ohne glibc-Bedarf.
- ARM-Builds brauchen den bekannten Paketwechsel `libwebkit2gtk-4.1-dev:<arch>`
  (siehe `.claude/rules/build-release.md`). Danach muss amd64 wieder hergestellt werden.
- Projektregeln gelten:
  - TDD, Tests neben dem Code
  - deutsche DocBlocks mit Datum, Autor Kurt Ingwer, Tabs
  - Build-Nummer +1
  - Release mit allen Assets (bisher 5 + macOS-Zip, künftig zusätzlich 3 `.deb`)
- Die App nimmt schon heute eine Datei als erstes Argument an (`src/main.go`). Der
  Starter reicht alle Argumente unverändert weiter.
- Icons liegen in `src/ui/assets/`: `favicon-16/32/48/256.png` und `favicon.svg`.
- Werkzeuge im LXC: `dpkg-deb` und `fakeroot` sind vorhanden.

## 6. Vom Nutzer entschieden

- Prüfung beim Start plus Desktop-Nachricht mit Installationsangebot per apt.
- Installation ins Menü mit dem vorhandenen Icon, als Selbst-Installation aus der Datei.
- Zusätzlich ein `.deb`.
- Systemaktuelles WebKit statt eingebautem (AppImage verworfen).

## 7. Von mir vorgeschlagen und vom Nutzer gutgeheißen („klingt gut“)

- Aufbau: kleiner **Starter** außen, die eigentliche GUI-App **eingebettet**.
- Fehlende Bibliotheken werden mit derselben Prüfung wie bei `ldd` ermittelt
  (Ladeprotokoll des dynamischen Laders), nicht über Dateisuche geraten.
- Ohne Benachrichtigungsdienst gibt es eine Rückfallkette:
  1. kdialog- oder zenity-Frage
  2. Text auf stderr
- Bei apt wird **ein** Paket installiert, `libwebkit2gtk-4.1-0`. Es zieht GTK,
  JavaScriptCore und libsoup selbst nach.
- Gilt für alle drei Linux-Architekturen.

## 8. Dem Bauer überlassen (mit Vorgaben)

1. **Wie der Starter die GUI ausführt:** per `memfd_create` + `exec /proc/self/fd/N`
   (keine Datei auf der Platte) oder entpackt in
   `$XDG_CACHE_HOME/md-reader/<build>-<hash>/`.
   - Bei Cache: Verzeichnis 0700, atomar schreiben, Hash vor jedem Start prüfen.
   - Rückfall von memfd auf Cache, falls memfd scheitert (z. B. `vm.memfd_noexec`).
   - `argv[0]` muss `md-reader` lauten.
2. **Benachrichtigung mit Aktionsknopf:** über D-Bus `org.freedesktop.Notifications`
   (z. B. `github.com/godbus/dbus/v5`, reines Go) oder `notify-send --action --wait`,
   falls vorhanden. Hartes Muss ist keins von beiden, die Rückfallkette aus Abschnitt 7 gilt.
3. **Wo die Antwort „Später/Nie“ gespeichert wird:** eigene Datei, z. B.
   `~/.config/md-reader/launcher.json`, nicht `state.json` der GUI, damit sich
   Starter und GUI nicht gegenseitig überschreiben.
4. **Code-Aufteilung:** Vorschlag
   - Fachlogik (Abhängigkeitsprüfung, Paketmanager-Erkennung, Desktop-Datei erzeugen,
     Installieren/Entfernen, Versionsvergleich) als testbares Paket, z. B. `src/linuxsetup/`
   - Starter als eigenes `main`-Paket, z. B. `src/launcher/`
   - Die GUI-Binärdatei wird beim Build in ein gitignoriertes Verzeichnis kopiert und
     per `go:embed` eingebettet.
5. **Woran die Installation erkennt, dass sie schon läuft:** die App läuft schon aus
   `~/.local/bin` oder aus `/usr/bin` (Paketinstallation). In beiden Fällen keine Frage.
6. **App-ID für G7:** `g_set_prgname("io.github.waschbaerimhaus.md-reader")` vor dem
   Fensteraufbau in `platform_linux.go` (oder Gleichwertiges) und
   `StartupWMClass=` in der Desktop-Datei passend dazu.
7. **`.deb`-Inhalt:** die reine GUI-Binärdatei ohne Starter. apt garantiert die
   Abhängigkeiten, die Frage nach Installation entfällt. Abhängigkeiten siehe 8a.5.
   - Paketname `md-reader`, Version `1.0.<build>`
   - Maintainer siehe 8a.6
8. **Nach der Installation:** Die gerade laufende App weiterlaufen lassen. Der Nutzer
   bekommt eine kurze Bestätigung „MD Reader ist jetzt im Menü unter Büro“.
9. **Texte:** Die Nachrichten folgen der Systemsprache. Deutsch und Englisch reichen,
   alles andere fällt auf Englisch zurück.

## 8a. Nach dem Bauer-Check von mir entschieden (Claude, zur Prüfung durch den Nutzer)

1. **Selbst-Installation und `.deb` nebeneinander:**
   - Beide verwenden dieselbe Desktop-ID `io.github.waschbaerimhaus.md-reader`.
   - Ist das `.deb` installiert (`/usr/share/applications/io.github.waschbaerimhaus.md-reader.desktop`
     existiert), stellt die heruntergeladene Datei **keine** Installationsfrage.
   - `--install` bricht dann mit einem Hinweis ab, die Paketinstallation zu nutzen.
     Ein `--install --force` setzt sich trotzdem darüber.
   - G5 (Ersetzen durch eine neuere Datei) gilt nur für die Selbst-Installation in
     `~/.local`, nie für `/usr/bin`.
2. **Taskleisten-Icon:**
   - G7 und das zugehörige Anti-Ziel gelten nur im **installierten** Zustand.
   - Unter Wayland kommt das Icon nur über die Desktop-Datei. Ohne Installation zeigt
     X11 weiter das Icon aus `gtk_window_set_icon`, Wayland ein generisches.
   - Nach „Nie“ werden dafür keine Dateien abgelegt.
3. **Paketlisten (`apt-get update`):**
   - Zuerst wird nur `pkexec apt-get install -y libwebkit2gtk-4.1-0` ausgeführt
     (eine Passwortabfrage).
   - Scheitert das, bietet die Nachricht „Paketlisten aktualisieren und erneut versuchen“ an.
     Das sind zwei feste Aufrufe nacheinander, `pkexec apt-get update` und dann noch
     einmal `install`, jeder ohne Shell.
   - PackageKit wird nicht verwendet, weil `pkcon` nicht überall installiert ist.
4. **`--uninstall`:**
   - Entfernt Programm, Desktop-Datei, Icons und den Starter-Cache.
   - Einstellungen bleiben: `~/.config/md-reader/` (Zoom, Theme, Scroll-Positionen,
     „Nie“-Antwort) und WebKit-Daten. So machen es übliche Linux-Programme.
     Die Ausgabe nennt diese Pfade, damit der Nutzer sie bei Bedarf selbst löscht.
   - Damit ist das Anti-Ziel „liegengebliebene Dateien“ so zu lesen: nichts, was **nicht**
     genannt wird.
   - WebKit-Datenordner sollen trotz App-ID-Wechsel beim Programmnamen `md-reader`
     bleiben. `g_set_prgname` nur so setzen, dass die Datenpfade (`~/.cache/md-reader`,
     `~/.local/share/md-reader`) unverändert bleiben. Das heißt: eigene
     `WebKitWebsiteDataManager`-Pfade setzen oder die Wayland-App-ID gezielt per
     `gdk_set_program_class` / `g_set_application_name` regeln. Die genaue Lösung
     bleibt dem Bauer überlassen, die Datenpfade dürfen sich nicht ändern.
5. **Zielsysteme (Nutzer, 2026-10-24: „es reicht wenn zwei jahre alte distributionen damit
   funktionieren“):**
   - Untergrenze sind Distributionen ab etwa Oktober 2024: Ubuntu 24.04 und KDE neon
     (glibc 2.39), Mint 22, Fedora 41, Debian 13.
   - Debian 12, Raspberry Pi OS Bookworm und Ubuntu 22.04 bleiben draußen. Die Dateien
     brauchen schon heute `GLIBC_2.38`.
   - **Neu: Wächter im Build.** Ein Make-Schritt prüft per `objdump -T`, dass keine
     Linux-Datei ein Symbol neuer als `GLIBC_2.39` verlangt, sonst bricht der Build ab.
     Das gilt auch für ARM64 und ARMhf. Grund: Die Build-Maschine hat glibc 2.42, eine
     unbedachte Funktion könnte die Untergrenze still anheben.
   - **`.deb`-Abhängigkeiten nicht von der Build-Maschine übernehmen.**
     `dpkg-shlibdeps` würde `libwebkit2gtk-4.1-0 (>= 2.52)` aus Ubuntu 25.10 eintragen,
     und dann wäre das Paket auf 24.04 nicht installierbar.
     - Fest eintragen: `libc6 (>= 2.38), libwebkit2gtk-4.1-0, libgtk-3-0`
     - Versionen nur, wo sie nötig sind
     - Prüfung: `dpkg-deb -I` und eine Probe-Installation in einem
       Ubuntu-24.04-Container oder per `apt-get install --simulate` gegen noble-Paketlisten
     - Wie das geprüft wird, entscheidet der Bauer.
   - Die Angabe „Debian 12+“ in `.claude/rules/build-release.md` und README wird
     korrigiert.
6. **Maintainer-Adresse im `.deb`:** `Kurt Ingwer <WaschbaerImHaus@users.noreply.github.com>`.
   Das ist mein Vorschlag, der Nutzer hat ihm nicht widersprochen. Nicht die private Adresse.

## 9. Abnahme

- Auf einem System ohne WebKit:
  - Start der Datei zeigt die Nachricht.
  - „Jetzt installieren“ installiert und startet MD Reader.
- Auf einem System mit WebKit kein spürbarer Unterschied beim Start.
- `--install` legt Menüeintrag und Icons an, KDE zeigt MD Reader unter „Büro“.
  „Öffnen mit“ bietet MD Reader für .md an. `--uninstall` entfernt alles wieder.
- Die Frage erscheint genau einmal; nach „Nie“ nie wieder, nach „Später“ beim nächsten Start.
- Das `.deb` installiert per `apt install ./md-reader_1.0.<n>_amd64.deb` inklusive WebKit,
  danach steht der Eintrag im Menü und die Installationsfrage erscheint nicht.
- Tests:
  - Fehlende Bibliotheken aus dem Ladeprotokoll lesen
  - Paketmanager-Erkennung
  - Inhalt der Desktop-Datei
  - Installieren/Entfernen in ein temporäres HOME
  - Versionsvergleich
  - Befehl ist fest, kein Shell-Aufruf
  - Randfälle: kein HOME, schreibgeschütztes Ziel, `~/.local/bin` fehlt, ältere oder
    neuere installierte Version
- Prüfung im LXC: Eine Probe-Binärdatei mit einer absichtlich fehlenden
  Bibliothek löst den Fehlerweg aus. Dazu Xvfb-Screenshots.
- Release enthält zusätzlich `md-reader_1.0.<n>_amd64.deb`, `_arm64.deb` und `_armhf.deb`.
