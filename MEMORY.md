# Projektspeicher MD Reader

## Überblick
Markdown-Betrachter mit GitHub-ähnlicher Darstellung. WebView-basiert für echtes HTML/CSS-Rendering.

## Technologie-Stack
- **Go 1.26** – Hauptprogrammiersprache
- **github.com/webview/webview_go** – WebView (WebKitGTK4.1 auf Linux, Edge WebView2 auf Windows)
- **github.com/yuin/goldmark** + **goldmark-highlighting/v2** – MD→HTML Konvertierung
- **PKG_CONFIG_PATH**: `src/pkgconfig/` enthält WebKit 4.0→4.1 Kompatibilitäts-Shim

## Wichtige Pfade
- Binaries: `build/md-reader` (Linux), `build/md-reader.exe` (Windows)
- Quellcode: `src/` (go.mod hier, nicht im Projektroot)
- Pkg-Config-Shim: `src/pkgconfig/webkit2gtk-4.0.pc`
- Tests: `src/renderer/*_test.go` + `src/ui/template_test.go` (alle grün)
- Assets: `src/ui/assets/styles.css`, `src/ui/assets/body.html`, `src/ui/assets/scripts.js` (go:embed)

## Build-Befehle
```bash
# Immer aus src/ ausführen
cd src

# Linux:
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o ../build/md-reader .

# Windows (Cross-Compilation):
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -o ../build/md-reader.exe .

# Tests:
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go test ./renderer/... -v
```

## Bekannte Probleme
- Windows Cross-Compilation: EventToken.h Stub wurde in webview_go Module-Cache erstellt
- WebKit-Version: Nur 4.1 verfügbar auf diesem System (nicht 4.0)

## Architektur-Entscheidungen
- **WebView statt Fyne**: Für echtes GitHub-ähnliches Rendering (HTML/CSS/JS im Browser)
- **Goldmark**: GFM-Support + Syntax-Highlighting
- **Inline HTML**: Gesamtes UI (Toolbar + Markdown-Bereich) als ein HTML-Template
- **Build Tags**: platform_linux.go / platform_windows.go / platform_darwin.go für native APIs
- **macOS-Build nur auf echtem Mac** (GitHub-Runner macos-15): Das macOS-SDK ist auf Linux nicht lizenzkonform nutzbar (osxcross wäre technisch möglich, verstößt aber gegen die Xcode-Lizenz)

## Linux-Starter und Container-Build (Build 47)
- `src/launcher/` (Starter, `CGO_ENABLED=0`, Tag `embedgui`, GUI per `go:embed` aus `launcher/payload/`, Ausführung aus `memfd`) und `src/linuxsetup/` (Fachlogik: Ladeprotokoll, Paketmanager, Desktop-Datei, Nutzerinstallation, Asker D-Bus/kdialog/zenity/stderr, Nachrichten de/en). Details `.claude/rules/platform.md`.
- Ablauf Linux-Build: GUI im Container `md-reader-build:22.04-<arch>` (Ubuntu 22.04, Images per `make build-images`, Container ohne Netz, Go-Module vorab per `make gomod-download`) → glibc-Wächter (Obergrenze 2.35, tatsächlich 2.34) → Nutzlast → statischer Starter; `.deb` aus der reinen GUI (`installer/linux/build-deb.sh`). Der Paketwechsel `:arm64`/`:armhf` auf dem Host entfällt.
- Zielsysteme seit Build 47: Ubuntu 22.04+, Mint 21+, Debian 12+, Raspberry Pi OS Bookworm (Pi 2 oder neuer, armhf = ARMv7), KDE neon; glibc ≥ 2.34. Gilt statt der früheren Grenze Ubuntu 24.04+ (Nutzerentscheidung 2026-11-01 „ja baue“, Tests auf 22.04/24.04/26.04).
- GUI-Identität `md-reader` (`g_set_prgname`, `gdk_set_program_class`, `prctl`); Desktop-ID `md-reader.desktop`.
- Sicherheit Build 47: F1 Kontextmenü ganz gesperrt (Nutzer 2026-11-01) + native Navigations-Erlaubnisliste nur unter Linux (RISK-013 Rest Windows/macOS), F2 goldmark 1.8.7 + Link-Erlaubnisliste, F3 `--install` kopiert die laufende Datei, RISK-016 zufällige private Temp-Ansichtsdatei, RISK-017 `//host`/UNC-Links abgelehnt, H1–H5.
- **Zwischenfall 2026-11-01 ~17:57:** Ein RED-Test (specialist-security, N1) löschte per `os.RemoveAll` das Elternverzeichnis einer Temp-Datei, also `/tmp` rekursiv: Scratchpad-Inhalte und der X11-Socket `/tmp/.X11-unix/X99` samt `.X99-lock` des fremden Dienstes `aladin2-plus500-browser.service` (Xvfb :99 und Chrome liefen weiter, neue Clients erreichten :99 nicht mehr). Dienst nicht neu gestartet, Nutzer informiert. Lehre und Regel: in Tests nie `os.RemoveAll` auf Verzeichnisse aus Produktivcode; Aufräumen nur innerhalb `t.TempDir()` mit Prüfung des Pfads (`.claude/rules/tests.md`, „Fallen“). Der Test hat jetzt Schutz (`requireStrictlyInside`).
- Umgebung: LXC auf Ubuntu 26.04.1 (Kernel 7.0) aktualisiert, `libwebkit2gtk-4.1-dev` 2.52.6 neu installiert (2026-11-01).
- Nutzerentscheidungen: 2026-10-24 Starter + Selbst-Installation + .deb (kein AppImage); 2026-10-25 Auto-Update wie ssh-easy mit Signatur für nach Build 47 (Build 48); 2026-10-30 „härte den Code anhand der Befunde“; 2026-10-31 effort medium für Modelle unter Opus; 2026-11-01 Rechtsklick im Fenster ganz sperren; 2026-11-01 Programm soll auf Ubuntu 22/24 laufen → Container-Build.
- Offen: Windows-`w == nil`-Prüfung ist toter Code (BUGS.md), RISK-013, Auto-Update (Build 48).

## Sitzungsverlauf
- 2026-03-05: Erstversion. Go 1.26 installiert. App implementiert mit WebView.
- 2026-03-07: Multi-Format (EPUB, FB2, TXT, HTML), Config-Persistenz, TOC, Suche, Dark/Retro-Mode.
- 2026-03-08 (Build 8): Refactoring: ui/template.go aufgeteilt in styles.go, html_body.go, scripts.go, template.go. 15 UI-Tests hinzugefügt (31 gesamt).
- 2026-03-08 (Build 9): Bug #003 (letzte Drag-&-Drop-Datei nicht gespeichert) + Bug #004 (Scroll-Position nicht persistent) behoben. persistLastFile-Binding + text/uri-list-Extraktion.
- 2026-03-08 (Build 10): Linux + Windows Builds erstellt. CLAUDE.md-Pflichten erledigt. RISK-004 identifiziert (path traversal via persistLastFile). README.md vervollständigt (76 Tests).
- 2026-03-08 (Build 11): Security (RISK-001/002/004 behoben, RISK-003 mitigiert), go:embed (Assets als externe Dateien), bindings.go (aus main.go extrahiert), ARM64-Build, Mermaid.js-Integration. Build-Größe reduziert (-s -w). CSP-Header hinzugefügt.
- 2026-07-03 (Build 38): Bug #011 behoben – EPUB-Kapitellinks führten zu leerem Fenster. Interne Links werden zu Ankern umgeschrieben, Calibre-Body-IDs erhalten, JS-Sicherheitsnetz gegen relative Links. Release v1.0.39 enthält auch diesen Build.
- 2026-07-04 (Build 39): Bug #012 behoben – große EPUBs (17.epub, 4,3 MB HTML) ließen sich nicht öffnen. Ursachen: WebView2-2-MB-Limit bei NavigateToString (Fix: file://-Temp-Datei via ui/viewfile.go ab 1,5 MB) + Calibre-SVG-Cover nicht eingebettet (Fix: SVG <image xlink:href/href> in embedEpubImages). Release v1.0.39 mit allen Binaries veröffentlicht.
- 2026-10-14 (Build 40–43): Bug #013 (Inhalt im HTML-Kommentar → unformatiert, Start mitten im Text), Bug #014 (KaTeX lud nach dem Hauptskript), RISK-005 bis RISK-010: Seitentitel escaped, Whitelist-Bereiniger für EPUB/LaTeX, strikte CSP mit Nonce, externe Links im System-Browser (extlink), img-src nur data:, Bildpfade als Erlaubnisliste. GitHub-Pages-Workflow aktiv.
- 2026-10-22 (Build 44): RISK-011 (Gerätenamen-Umgehung bei Bildpfaden). Agent-Struktur eingeführt: 20 Agents in .claude/agents/ (lead-vorarbeiter, 6 specialists, 5 analysts, 3 worker, 5 helper), gemeinsame Regeln und überschneidungsfreie Zuständigkeiten in .claude/agents-shared/, Projektwissen aufgeteilt in .claude/rules/ (7 Dateien), Stop-/SessionStart-Hook check-project-structure.py prüft Eigentümer je Datei, Modell je Ebene, CLAUDE.md-Länge und Testzahl. Start als Vorarbeiter: `claude --agent lead-vorarbeiter`.
- 2026-10-24 (Build 45): Text-Modus (C64-Stil) für Textformate inkl. EPUB: Pixelschrift Press Start 2P (OFL) eingebettet, 50 Spalten/32 Zeilen ×1,5 nach kürzerer Seite, Bilder 1-Bit-gerastert (Bayer 8×8, auch SVG-Cover), Tabellen mit Linien, Formeln per SVG-Filter 1-Bit (Alpha-Schwelle 25 %), Farben frei wählbar (Toolbar-Farbfelder, Binding persistTextMode, Farben dreifach geprüft). Neue Dateien: ui/textmode.go, assets/textmode.css/.js, assets/fonts/, ui/testdata/textmode_test.js (JS-Tests per Node). 184 Tests.
- 2026-10-24 (Build 46): macOS-Port auf Nutzerwunsch (MacBook Pro 14: Linux-ARM-Binaries liefen dort natürlich nicht). platform_darwin.go (Cocoa: Menüleiste, NSOpenPanel, Vollbild, Dock-Icon), Tastenkürzel mit ⌘ (hasCommandModifier), renderer/tools.go (Homebrew-Suchorte, pdftoppm auch unter darwin), große Seiten unter macOS direkt per SetHtml. installer/macos/build-app.sh baut Universal-.app (ad hoc signiert, macOS 13+), Workflow .github/workflows/macos.yml baut/testet/macht Bildschirmfotos und hängt md-reader-macos.zip ans Release. 195 Tests.
- 2026-11-01 (Build 47): Linux-Starter mit eingebetteter GUI, WebKit-Prüfung und Installationsangebot, Selbst-Installation ins Menü (`--install`/`--uninstall`), drei .deb-Pakete, GUI-Identität `md-reader`, Container-Build auf Ubuntu 22.04 (glibc ≥ 2.34), Sicherheitsrunden (F1–F3, H1–H5, RISK-016/017). Entwurf und Plan unter `claude-generated/specs|plans/2026-10-24-linux-starter-installation*`. 354 Tests in 26 Dateien.
