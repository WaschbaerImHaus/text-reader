# Features

## Implementiert ✅

- **Drag & Drop**: Dateien per Drag & Drop öffnen (.md, .markdown, .txt, .epub, .fb2)
- **Multi-Format**: Markdown, Plaintext, EPUB (ZIP+XHTML), FB2 (XML)
- **EPUB-Rendering**: Kapitel in Lesereihenfolge (Spine), Metadaten-Titel, Kapitel-Trennlinien
- **FB2-Rendering**: Kapitel, Absätze, Kursiv/Fett/Durchgestrichen, Gedichte, Epigraphen
- **TXT-Rendering**: Monospace-Darstellung mit erhaltenem Whitespace
- **Fehlerprüfung**: Klare Fehlermeldung bei nicht unterstützten Dateitypen
- **GitHub-Rendering**: HTML/CSS Rendering über WebView (echtes Browser-Engine)
- **Syntax-Highlighting**: Code-Blöcke mit GitHub-Stil (chroma)
- **GFM-Support**: Tabellen, Aufgabenlisten, Durchstreichung, Autolinks
- **Zoom In/Out**: Schriftgröße verdoppeln/halbieren (max. 400%, min. 25%)
- **Zoom Mausrad**: Strg+Scroll zum Zoomen
- **Zoom Tastatur**: Strg++, Strg+-, Strg+0 (zurücksetzen)
- **Zoom-Anzeige**: Prozentwert sichtbar in der Toolbar
- **Hochformat**: Inhalt zentriert, Breite skaliert mit Zoom (750px × Zoom-Faktor, max. Fensterbreite − 32 px)
- **Querformat**: Inhalt in voller Fensterbreite
- **Vollbild**: Nativer Vollbild-Modus (GTK/WinAPI) via F11 oder Toolbar-Button
- **Schließen**: Button rechts oben (✕)
- **Toolbar**: Keine Menüleiste, alle Steuerflächen oben
- **Scroll**: Pfeiltasten, PageUp/Down, Home/End
- **Datei-Argument**: Datei beim Start per CLI-Argument übergeben
- **Letzte Datei**: Beim Start wird die zuletzt geöffnete Datei automatisch wiedergeladen
- **Persistente Einstellungen**: Zoom, Theme und Layout werden zwischen Sitzungen gespeichert
- **Lokale Bilder**: Relative Bildpfade in Startup-Dateien werden als base64-Data-URI eingebettet
- **TOC-Seitenleiste**: Inhaltsverzeichnis (280px, links, Toggle ☰, nur wenn Überschriften vorhanden)
- **Suchfunktion**: Strg+F öffnet Suchleiste mit Treffernavigation, Counter, Highlighting
- **Dark Mode**: GitHub Dark (☽ → ▓)
- **Retro Mode**: CRT-Phosphor Grün (▓ → ☀)
- **Cross-Platform**: Linux (WebKitGTK), Windows (Edge WebView2) und macOS (WKWebView, ab Build 46)
- **ARM64-Build**: Linux-ARM64-Binary via Cross-Compilation (aarch64-linux-gnu-gcc)
- **Tastaturkürzel**: F11 Vollbild, Strg+F Suche, Strg+/- Zoom (unter macOS ⌘ statt Strg)
- **Mermaid.js**: Flowcharts, Sequenzdiagramme und andere Diagramme in Markdown (via CDN, fallback auf Code-Block bei fehlender Verbindung)
- **KaTeX LaTeX** (Build 19): Inline-Formeln `$...$`, Block-Formeln `$$...$$`, `\(...\)`, `\[...\]` – vollständig offline eingebettet (keine CDN-Abhängigkeit), alle Schriften als base64-Data-URIs
- **Scroll-History**: Scroll-Position wird pro Datei (FNV-64a Hash) gespeichert; beim Wiedereröffnen wird automatisch die letzte Leseposition wiederhergestellt (max. 200 Einträge)
- **Nativer Dateidialog**: 📂-Button / Strg+O öffnet OS-nativen Datei-Öffnen-Dialog (Lösung für WebView2-Einschränkung)
- [x] PDF-Anzeige (.pdf) – Linux: pdftoppm → PNG-Bilder (Build 36); Windows: Edge WebView2 embed (Build 34)
- [x] PostScript-Anzeige (.ps) – gs rendert direkt zu PNG-Bildern (Build 37); Fallback: Hinweis + Quelltext
- [x] **Text-Modus** (Build 45): C64-ähnliche Ansicht für Textformate inkl. EPUB – Pixelschrift „Press Start 2P“ (OFL, eingebettet, Latein/Griechisch/Kyrillisch), alles gleich groß, Überschriften fett, 50 Spalten × 32 Zeilen (1,5-facher Zeilenabstand) nach der kürzeren Seite des Inhaltsbereichs, Bilder 1-Bit-gerastert (Bayer 8×8, Bildpunkt = Schrift-Pixel), Tabellen mit Linien, Formeln/Diagramme als 1-Bit-Grafik, Schrift- und Hintergrundfarbe per Farbfeld in der Toolbar, Einstellung gespeichert

- [x] **macOS** (Build 46): Universal-App (Apple Silicon + Intel, ab macOS 13) als `md-reader-macos.zip`, gebaut auf einem GitHub-Mac-Runner (Workflow `macos.yml`, Tests + Rauchtest mit Bildschirmfotos) und automatisch ans Release gehängt; Menüleiste (⌘C/⌘V/⌘A/⌘Q), NSOpenPanel, natives Vollbild, Dock-Icon; pdftoppm/gs auch aus Homebrew; ad hoc signiert, nicht notarisiert

- [x] **Linux-Starter** (Build 47, Nutzerwunsch 2026-10-24): `md-reader`, `-linux-arm64`, `-linux-armhf` bleiben eine Datei, sind aber ein statischer Starter mit eingebetteter GUI (läuft aus `memfd`); prüft vor dem Start per Ladeprotokoll fehlende Bibliotheken, meldet per Desktop-Nachricht (D-Bus, kdialog, zenity, Terminal) und bietet `pkexec apt-get install libwebkit2gtk-4.1-0` an; zu alte Systeme werden klar abgelehnt
- [x] **Selbst-Installation ins Anwendungsmenü** (Build 47): Frage beim Erststart (Ja/Später/Nie), `--install [--force]`, `--uninstall`, `--` beendet die Optionen; pro Benutzer (`~/.local/bin`, Desktop-Datei `md-reader.desktop`, hicolor-Icons, „Öffnen mit“-MIME-Typen), Einstellungen bleiben beim Entfernen; Ersetzen-Angebot bei neuerer Datei
- [x] **.deb-Pakete** (Build 47): `md-reader_1.0.<build>_amd64|arm64|armhf.deb` (reine GUI, `Depends` WebKitGTK/GTK/libc6, `Recommends` zenity/poppler-utils/ghostscript)
- [x] **Container-Build der Linux-GUI** (Build 47): Ubuntu 22.04 (amd64/arm64/armhf), glibc-Wächter; läuft auf Ubuntu 22.04+, Mint 21+, Debian 12+, Raspberry Pi OS Bookworm, KDE neon (glibc ≥ 2.34)
- [x] **GUI-Identität `md-reader`** (Build 47): Programmname, WM-Klasse und Prozessname, damit das Menü-Icon auch in Taskleiste und unter Wayland erscheint
- [x] **Härtung** (Build 47): Kontextmenü gesperrt, native Navigations-Erlaubnisliste unter Linux, Markdown-Link-Erlaubnisliste (externe Bilder zeigen den Alternativtext), private Temp-Ansichtsdatei, UNC-/protokollrelative Links abgelehnt

## Geplant 📋

- [ ] **Auto-Update mit Signatur** (geplant für Build 48, Nutzerwunsch 2026-10-25): beim Start wird nach einer neuen Version gesucht und geladen, beim Beenden ausgeführt; Umsetzung wie in ssh-easy (`src/updater*.go`, `src/releasesig`, `cmd/release-sign`; Ed25519, `SHA256SUMS` + `.sig`, Kopfzeile an das Release-Tag gebunden, privater Schlüssel nur im LXC)
- [ ] Nativer Navigationsschutz für Windows (WebView2) und macOS (WKWebView) – dort wirkt nur die JS-Schicht (RISK-013)

- [ ] Mehrere Dateien nacheinander öffnen (Tab-Leiste)
- [ ] Druckfunktion
- [ ] Datei-Watcher (Auto-Reload bei Änderungen)
- [ ] Exportfunktion (HTML speichern)
- [ ] Verlauf mehrerer zuletzt geöffneter Dateien
