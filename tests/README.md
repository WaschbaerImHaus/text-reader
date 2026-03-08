# Tests

Unit-Tests befinden sich bei den jeweiligen Packages:

- `../src/renderer/markdown_test.go` – Tests für Markdown/TXT/FB2/HTML-Parsing
- `../src/renderer/epub_test.go` – Tests für EPUB-Parsing
- `../src/renderer/images_test.go` – Tests für lokale Bildpfad-Auflösung
- `../src/ui/template_test.go` – Tests für HTML/CSS/JS-Template

**Gesamt: 76 Tests** (alle grün, Stand Build 10)

## Tests ausführen

```bash
cd ../src
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... ./ui/... -v
```

Oder via Makefile (aus Projektroot):

```bash
make test
```

## Test-Abdeckung

| Package          | Tests | Bereich |
|-----------------|-------|---------|
| renderer/        | ~60   | MD, TXT, FB2, EPUB, HTML, Bilder, Datei-Erkennung |
| ui/              | 15    | Template-Ausgabe, Themes, Layouts, CSS, JS |
