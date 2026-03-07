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
- **Hochformat**: Inhalt auf 750px Breite begrenzt, zentriert
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
- **Cross-Platform**: Linux (WebKitGTK) und Windows (Edge WebView2)
- **Tastaturkürzel**: F11 Vollbild, Strg+F Suche, Strg+/- Zoom

## Geplant 📋

- [ ] Mehrere Dateien nacheinander öffnen (Tab-Leiste)
- [ ] Druckfunktion
- [ ] Datei-Watcher (Auto-Reload bei Änderungen)
- [ ] Exportfunktion (HTML speichern)
- [ ] Verlauf mehrerer zuletzt geöffneter Dateien
