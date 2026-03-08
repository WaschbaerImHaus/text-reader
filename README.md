# MD Reader

Ein einfacher, schneller Betrachter für Markdown, EPUB, FB2, TXT und HTML mit GitHub-ähnlicher Darstellung.

## Features

- **Drag & Drop**: Dateien einfach auf das Fenster ziehen
- **Multi-Format**: Markdown (.md), EPUB (.epub), FictionBook (.fb2), Plaintext (.txt), HTML (.html, .htm)
- **GitHub-Rendering**: Identische Darstellung wie auf GitHub (Tabellen, Code-Highlighting, GFM)
- **Syntax-Highlighting**: Code-Blöcke mit Sprachhervorhebung (GitHub-Stil)
- **Zoom**: Schriftgröße verdoppeln/halbieren (25% bis 400%)
- **Layouts**: Hochformat (750px Breite) oder Querformat (volle Breite)
- **Vollbild**: Nativer Vollbild-Modus (F11 oder Button)
- **Dark Mode**: GitHub-Dark-Theme (☽ → ▓)
- **Retro Mode**: CRT-Phosphor-Grün (▓ → ☀)
- **TOC-Seitenleiste**: Automatisches Inhaltsverzeichnis (☰), nur wenn Überschriften vorhanden
- **Suche**: Strg+F öffnet Suchleiste mit Treffernavigation und Highlighting
- **Persistente Einstellungen**: Zoom, Theme und Layout bleiben über Starts erhalten
- **Letzte Datei**: Zuletzt geöffnete Datei wird beim Start automatisch wiedergeladen
- **Scroll-Position**: Scroll-Position wird gespeichert und beim nächsten Start wiederhergestellt
- **Lokale Bilder**: Relative Bildpfade werden als Base64 Data-URI eingebettet

## Steuerung

### Toolbar-Buttons

| Button | Funktion |
|--------|----------|
| `☰`    | Inhaltsverzeichnis (TOC) ein/aus |
| `▯` / `▭` | Hochformat / Querformat umschalten |
| `⊖`   | Schriftgröße halbieren (min. 25%) |
| `100%` | Aktuelle Zoomstufe |
| `⊕`   | Schriftgröße verdoppeln (max. 400%) |
| `☽` / `▓` / `☀` | Theme wechseln (Hell → Dark → Retro) |
| `☐`   | Vollbild ein/aus |
| `✕`   | Fenster schließen |

### Tastaturkürzel

| Taste | Funktion |
|-------|----------|
| `F11`       | Vollbild ein/aus |
| `Strg+F`    | Suche öffnen/schließen |
| `Strg++`    | Zoom vergrößern |
| `Strg+-`    | Zoom verkleinern |
| `Strg+0`    | Zoom zurücksetzen |
| `Strg+Scroll` | Zoom per Mausrad |
| `↑↓`        | Scrollen |
| `PageUp/Down` | Seitenweise scrollen |
| `Home/End`  | Anfang/Ende der Seite |

## Installation

### Linux

```bash
./build/md-reader [optionale-datei.md]
```

**Systemvoraussetzungen:**
- WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`)
- GTK 3 (`libgtk-3-0`)

### Windows

```
build\md-reader.exe [optionale-datei.md]
```

**Systemvoraussetzungen:**
- Windows 10 oder neuer (WebView2 ist vorinstalliert)
- Ältere Systeme: [WebView2 Runtime installieren](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)

## Verwendung

1. Programm starten
2. Datei per Drag & Drop auf das Fenster ziehen
3. **oder**: Programm mit Dateipfad als Argument starten

```bash
./build/md-reader /pfad/zu/datei.md
./build/md-reader /pfad/zu/buch.epub
./build/md-reader /pfad/zu/roman.fb2
./build/md-reader /pfad/zu/notizen.txt
```

Die zuletzt geöffnete Datei wird beim nächsten Start automatisch wieder geöffnet (inklusive Scroll-Position).

## Unterstützte Formate

### Markdown (.md, .markdown)

Unterstützte Elemente:
- Überschriften (H1–H6)
- Fett, Kursiv, Durchgestrichen
- Listen (geordnet/ungeordnet)
- Aufgabenlisten `- [x]`
- Code (inline und Blöcke mit Syntax-Highlighting)
- Blockquotes
- Tabellen (GFM)
- Links und Bilder
- Horizontale Linien
- Autolinks

### EPUB (.epub)
EPUB 2 und 3 – Kapitel werden in Spine-Reihenfolge gelesen. Metadaten-Titel wird übernommen.

### FictionBook 2 (.fb2)
XML-basiertes Format. Unterstützt: Kapitel, Absätze, Kursiv/Fett/Durchgestrichen, Gedichte, Epigraphen, Zitate.

### Plaintext (.txt)
Monospace-Darstellung mit erhaltenem Whitespace und Zeilenumbrüchen.

### HTML (.html, .htm)
Body-Inhalt wird extrahiert. style/script-Tags werden entfernt um Konflikte zu vermeiden.

## Build

```bash
# Alles kompilieren (Linux + Windows)
make all

# Nur Linux
make linux

# Nur Windows (benötigt mingw-w64)
make windows

# Tests ausführen
make test
```

## Technologie

- **Go 1.26** – Programmiersprache
- **webview_go** – Plattformübergreifende WebView-Integration
- **goldmark** – Markdown-Parser mit GFM-Unterstützung
- **goldmark-highlighting** – Syntax-Highlighting (GitHub-Stil)
- **WebKitGTK 4.1** (Linux) / **Edge WebView2** (Windows) – Browser-Engine
