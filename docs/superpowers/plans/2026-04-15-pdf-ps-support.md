# PDF und PostScript (.ps) Unterstützung – Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** MD Reader kann `.pdf`- und `.ps`-Dateien öffnen, anzeigen und per Drag & Drop laden – auf Linux und Windows, ohne Pflichtabhängigkeiten.

**Architecture:** PDF-Dateien werden als base64-Daten-URI in einem `<embed>`-Tag eingebettet und vom WebKit/Edge WebView2 nativ gerendert. PostScript-Dateien werden via Ghostscript (`gs`) zu PDF konvertiert und dann ebenfalls nativ angezeigt; fehlt `gs`, wird der PS-Quelltext als `<pre>`-Block dargestellt. Der Windows-Installer erhält einen optionalen Ghostscript-Abschnitt.

**Tech Stack:** Go 1.26, renderer-Package, webview_go, GTK-Dateidialog (Linux), Windows-Dateidialog (CGo), NSIS

---

## Dateiübersicht

| Aktion | Datei | Zweck |
|--------|-------|-------|
| Neu    | `src/renderer/pdf.go` | `IsPDFFile`, `ParsePDF` |
| Neu    | `src/renderer/pdf_test.go` | Tests für PDF-Renderer |
| Neu    | `src/renderer/ps.go` | `IsPSFile`, `ParsePS` (gs oder Text-Fallback) |
| Neu    | `src/renderer/ps_test.go` | Tests für PS-Renderer |
| Ändern | `src/renderer/markdown.go` | `LoadFile` + `IsSupportedFile` erweitern |
| Ändern | `src/bindings.go` | `processBinaryFile`-Binding hinzufügen |
| Ändern | `src/ui/assets/scripts.js` | `supportedExtensions` + Drop-Handler |
| Ändern | `src/platform_linux.go` | Dateidialog-Filter |
| Ändern | `src/platform_windows.go` | Dateidialog-Filter |
| Ändern | `installer/md-reader.nsi` | Ghostscript-Abschnitt |
| Ändern | `src/build.txt` | 32 → 33 |

---

## Task 1: PDF Renderer

**Files:**
- Create: `src/renderer/pdf.go`
- Create: `src/renderer/pdf_test.go`

- [ ] **Schritt 1: Failing Tests schreiben**

```go
// src/renderer/pdf_test.go
// Tests für den PDF-Renderer.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsPDFFile prüft die Erkennung von PDF-Dateien anhand der Endung.
func TestIsPDFFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.pdf", true},
		{"Doc.PDF", true},
		{"BERICHT.Pdf", true},
		{"doc.md", false},
		{"doc.ps", false},
		{"book.epub", false},
		{"/home/user/datei.pdf", true},
		{"C:/Users/user/datei.pdf", true},
	}
	for _, tt := range tests {
		got := renderer.IsPDFFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsPDFFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}

// TestParsePDF prüft dass eine base64-Einbettung erzeugt wird.
func TestParsePDF(t *testing.T) {
	data := []byte("%PDF-1.4 test")
	result, err := renderer.ParsePDF(data, "test.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() HTML enthält keine base64 PDF-Einbettung")
	}
	if !strings.Contains(result.HTML, `type="application/pdf"`) {
		t.Error("ParsePDF() HTML enthält keinen MIME-Typ application/pdf")
	}
	if !strings.Contains(result.HTML, "<embed") {
		t.Error("ParsePDF() HTML enthält kein <embed>-Element")
	}
	if result.Title != "test" {
		t.Errorf("ParsePDF() Title = %q, erwartet %q", result.Title, "test")
	}
}

// TestParsePDFEmpty prüft Verhalten bei leeren PDF-Daten.
func TestParsePDFEmpty(t *testing.T) {
	result, err := renderer.ParsePDF([]byte{}, "empty.pdf")
	if err != nil {
		t.Fatalf("ParsePDF() sollte keinen Fehler für leere Daten liefern: %v", err)
	}
	if !strings.Contains(result.HTML, "data:application/pdf;base64,") {
		t.Error("ParsePDF() HTML fehlt auch bei leeren Daten")
	}
}

// TestParsePDFTitleFromFilename prüft Titelextraktion aus Dateiname.
func TestParsePDFTitleFromFilename(t *testing.T) {
	result, _ := renderer.ParsePDF([]byte("%PDF"), "jahresbericht-2025.pdf")
	if result.Title != "jahresbericht-2025" {
		t.Errorf("ParsePDF() Title = %q, erwartet %q", result.Title, "jahresbericht-2025")
	}
}

// TestIsSupportedFileWithPDF prüft dass .pdf als unterstützt erkannt wird.
func TestIsSupportedFileWithPDF(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.pdf", true},
		{"README.md", true},
		{"book.epub", true},
		{"image.png", false},
		{"video.mp4", false},
	}
	for _, tt := range tests {
		got := renderer.IsSupportedFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsSupportedFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}
```

- [ ] **Schritt 2: Tests fehlschlagen lassen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v -run "TestIsPDFFile|TestParsePDF|TestIsSupportedFileWithPDF"
```

Erwartet: FAIL – `renderer.IsPDFFile undefined`

- [ ] **Schritt 3: PDF-Renderer implementieren**

```go
// src/renderer/pdf.go
// Package renderer – PDF-Datei-Einbettung für den WebView.
//
// Bettet PDF-Dateien als base64-Daten-URI in ein <embed>-Element ein.
// WebKitGTK (Linux) und Edge WebView2 (Windows) rendern PDFs nativ.
// Kein externes Tool benötigt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

// IsPDFFile prüft ob ein Dateipfad auf eine PDF-Datei (.pdf) zeigt.
//
// @param path Dateipfad (mit oder ohne Verzeichnis).
// @return true wenn die Datei .pdf Endung hat (Groß-/Kleinschreibung ignoriert).
func IsPDFFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".pdf"
}

// ParsePDF bettet PDF-Binärdaten als base64-Daten-URI in einen <embed>-Tag ein.
//
// Das erzeugte HTML nutzt position:fixed um unterhalb der Toolbar (44px)
// die gesamte Bildschirmfläche zu belegen – unabhängig vom max-width des
// #content-Containers.
//
// @param data     Rohe PDF-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Einbettung und Metadaten, oder Fehler.
func ParsePDF(data []byte, filename string) (*Result, error) {
	// PDF als base64 kodieren
	b64 := base64.StdEncoding.EncodeToString(data)

	// Embed-Element: position:fixed bricht aus dem #content max-width heraus
	// und belegt die gesamte Fläche unterhalb der Toolbar (--toolbar-height = 44px).
	html := fmt.Sprintf(
		`<div class="pdf-viewer" style="position:fixed;top:44px;left:0;right:0;bottom:0;overflow:hidden;">`+
			`<embed src="data:application/pdf;base64,%s" type="application/pdf" width="100%%" height="100%%">`+
			`</div>`,
		b64,
	)

	return &Result{
		HTML:  html,
		Title: fileBaseName(filename),
	}, nil
}
```

- [ ] **Schritt 4: Tests bestehen lassen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v -run "TestIsPDFFile|TestParsePDF"
```

Erwartet: alle PASS (TestIsSupportedFileWithPDF noch FAIL – kommt in Task 3)

- [ ] **Schritt 5: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/renderer/pdf.go src/renderer/pdf_test.go
git commit -m "feat(renderer): add PDF renderer with base64 embed"
```

---

## Task 2: PostScript Renderer

**Files:**
- Create: `src/renderer/ps.go`
- Create: `src/renderer/ps_test.go`

- [ ] **Schritt 1: Failing Tests schreiben**

```go
// src/renderer/ps_test.go
// Tests für den PostScript-Renderer.
//
// ParsePS produziert entweder eine PDF-Einbettung (wenn gs vorhanden)
// oder einen Text-Fallback. Beide Pfade werden getestet.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsPSFile prüft die Erkennung von PostScript-Dateien anhand der Endung.
func TestIsPSFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.ps", true},
		{"Doc.PS", true},
		{"Bericht.Ps", true},
		{"doc.pdf", false},
		{"doc.md", false},
		{"/home/user/datei.ps", true},
	}
	for _, tt := range tests {
		got := renderer.IsPSFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsPSFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}

// TestParsePSFallback prüft den Text-Fallback wenn gs nicht verfügbar ist
// oder die Konvertierung fehlschlägt.
func TestParsePSFallback(t *testing.T) {
	// Einfaches PS-Dokument (kein echtes PostScript, gs schlägt fehl → Fallback)
	psContent := `%!PS-Adobe-3.0
%%Title: Testdokument
%%Pages: 1
%%EndComments
/Helvetica findfont 12 scalefont setfont
100 700 moveto (Hallo Welt) show
showpage`

	result, err := renderer.ParsePS([]byte(psContent), "test.ps")
	if err != nil {
		t.Fatalf("ParsePS() sollte keinen Fehler zurückgeben: %v", err)
	}
	// Entweder PDF-Einbettung oder Text-Fallback – beides ist korrekt
	hasPDF := strings.Contains(result.HTML, "data:application/pdf;base64,")
	hasText := strings.Contains(result.HTML, "PS-Adobe") || strings.Contains(result.HTML, "Hallo Welt")
	if !hasPDF && !hasText {
		t.Error("ParsePS() HTML enthält weder PDF-Einbettung noch PS-Quelltext")
	}
}

// TestParsePSTitle prüft Titelextraktion aus dem Dateinamen.
func TestParsePSTitle(t *testing.T) {
	result, err := renderer.ParsePS([]byte("%!PS"), "mein-dokument.ps")
	if err != nil {
		t.Fatalf("ParsePS() Fehler: %v", err)
	}
	if result.Title != "mein-dokument" {
		t.Errorf("ParsePS() Title = %q, erwartet %q", result.Title, "mein-dokument")
	}
}

// TestParsePSEmpty prüft Verhalten bei leeren Daten.
func TestParsePSEmpty(t *testing.T) {
	result, err := renderer.ParsePS([]byte{}, "empty.ps")
	if err != nil {
		t.Fatalf("ParsePS() sollte keinen Fehler für leere Daten liefern: %v", err)
	}
	if result == nil {
		t.Fatal("ParsePS() sollte kein nil Result zurückgeben")
	}
}

// TestIsSupportedFileWithPS prüft dass .ps als unterstützt erkannt wird.
func TestIsSupportedFileWithPS(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"doc.ps", true},
		{"doc.pdf", true},
		{"README.md", true},
		{"image.png", false},
	}
	for _, tt := range tests {
		got := renderer.IsSupportedFile(tt.path)
		if got != tt.expected {
			t.Errorf("IsSupportedFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
		}
	}
}
```

- [ ] **Schritt 2: Tests fehlschlagen lassen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v -run "TestIsPSFile|TestParsePS"
```

Erwartet: FAIL – `renderer.IsPSFile undefined`

- [ ] **Schritt 3: PS-Renderer implementieren**

```go
// src/renderer/ps.go
// Package renderer – PostScript-Datei-Rendering.
//
// Konvertiert PostScript-Dateien zu PDF via Ghostscript (gs).
// Ist gs nicht verfügbar oder schlägt die Konvertierung fehl,
// wird der PS-Quelltext als <pre>-Block dargestellt.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
package renderer

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IsPSFile prüft ob ein Dateipfad auf eine PostScript-Datei (.ps) zeigt.
//
// @param path Dateipfad (mit oder ohne Verzeichnis).
// @return true wenn die Datei .ps Endung hat (Groß-/Kleinschreibung ignoriert).
func IsPSFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".ps"
}

// ParsePS konvertiert PostScript-Daten zu HTML.
//
// Ablauf:
//  1. Ghostscript (gs) wird aufgerufen: PS-Daten über stdin, PDF-Bytes aus stdout.
//  2. Erfolg → ParsePDF(pdfBytes, filename) → natives PDF-Rendering im WebView.
//  3. Fehler (gs nicht vorhanden, Timeout, Konvertierungsfehler)
//     → ParseTextContent(string(data), filename) → <pre>-Block Fallback.
//
// @param data     Rohe PostScript-Binärdaten.
// @param filename Dateiname für den Titel (ohne Pfad).
// @return Result mit HTML-Inhalt und Metadaten, oder Fehler.
func ParsePS(data []byte, filename string) (*Result, error) {
	// Ghostscript-Konvertierung mit 30-Sekunden-Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// gs liest PS von stdin und schreibt PDF nach stdout
	cmd := exec.CommandContext(ctx, "gs",
		"-sDEVICE=pdfwrite",  // Ausgabe als PDF
		"-sOutputFile=-",     // - bedeutet stdout
		"-dBATCH",            // Kein interaktiver Modus
		"-dNOPAUSE",          // Nicht auf Tastendruck warten
		"-dQUIET",            // Keine Fortschrittsausgabe
		"-q",                 // Noch weniger Ausgabe
		"-",                  // Eingabe von stdin lesen
	)
	cmd.Stdin = bytes.NewReader(data)

	pdfBytes, err := cmd.Output()
	if err == nil && len(pdfBytes) > 4 {
		// Prüfen ob die Ausgabe wirklich eine PDF-Signatur hat (%PDF)
		if bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
			return ParsePDF(pdfBytes, filename)
		}
	}

	// Fallback: PS als Quelltext anzeigen
	if err != nil {
		log.Printf("ParsePS: gs nicht verfügbar oder Fehler: %v – zeige Text-Fallback", err)
	}
	return ParseTextContent(string(data), filename)
}
```

- [ ] **Schritt 4: Tests bestehen lassen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v -run "TestIsPSFile|TestParsePS"
```

Erwartet: alle PASS

- [ ] **Schritt 5: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/renderer/ps.go src/renderer/ps_test.go
git commit -m "feat(renderer): add PostScript renderer with gs conversion and text fallback"
```

---

## Task 3: LoadFile und IsSupportedFile erweitern

**Files:**
- Modify: `src/renderer/markdown.go`

- [ ] **Schritt 1: Existierende Tests laufen lassen (Baseline)**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v -run "TestIsSupportedFile"
```

Erwartet: `TestIsSupportedFileWithPDF` und `TestIsSupportedFileWithPS` schlagen noch fehl.

- [ ] **Schritt 2: `IsSupportedFile` in `src/renderer/markdown.go` erweitern**

Zeile 135–142 ersetzen (aktuelle `IsSupportedFile`-Funktion):

```go
// IsSupportedFile prüft ob eine Datei ein unterstütztes Format hat.
//
// Unterstützte Endungen: .md, .markdown, .txt, .fb2, .epub, .tex, .pdf, .ps
//
// @param path Dateipfad der zu prüfenden Datei.
// @return true wenn das Format unterstützt wird.
func IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".fb2", ".epub", ".tex", ".pdf", ".ps":
		return true
	}
	return false
}
```

- [ ] **Schritt 3: `LoadFile` in `src/renderer/markdown.go` erweitern**

Die `LoadFile`-Funktion (Zeile 61–76) ersetzen:

```go
// LoadFile lädt eine unterstützte Datei und konvertiert sie zu HTML.
//
// Erkennt das Format anhand der Dateiendung und leitet an den
// entsprechenden Parser weiter. Binärdateien (EPUB, PDF, PS) werden
// als []byte gelesen; alle anderen Formate als UTF-8-Text.
//
// @param filePath Absoluter oder relativer Pfad zur Datei.
// @return Result mit gerendertem HTML und Metadaten, oder Fehler.
func LoadFile(filePath string) (*Result, error) {
	// EPUB benötigt Binärdaten (ZIP-Archiv)
	if IsEpubFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParseEpub(data, filepath.Base(filePath))
	}
	// PDF benötigt Binärdaten (base64-Einbettung)
	if IsPDFFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParsePDF(data, filepath.Base(filePath))
	}
	// PostScript benötigt Binärdaten (gs-Konvertierung oder Text-Fallback)
	if IsPSFile(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ParsePS(data, filepath.Base(filePath))
	}
	// Alle anderen Formate als Text lesen
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseContent(string(data), filepath.Base(filePath))
}
```

Den Dateikommentar am Anfang ebenfalls aktualisieren (Zeile 4–9):

```go
// Package renderer stellt Funktionen zum Parsen von Dokumentdateien bereit.
//
// Unterstützte Formate:
//   - Markdown (.md, .markdown) → goldmark mit GFM + Syntax-Highlighting
//   - Plaintext (.txt)          → <pre>-Block mit HTML-Escape
//   - HTML (.html, .htm)        → Body-Extraktion und Bereinigung
//   - FB2 (.fb2)                → XML-Token-Parser
//   - EPUB (.epub)              → ZIP + XHTML-Kapitel-Extraktion
//   - LaTeX (.tex)              → eigener LaTeX-Parser
//   - PDF (.pdf)                → base64-Daten-URI in <embed>, nativ im WebView
//   - PostScript (.ps)          → gs-Konvertierung zu PDF, Fallback: <pre>-Block
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-04-15
```

- [ ] **Schritt 4: Alle Renderer-Tests bestehen lassen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... -v
```

Erwartet: alle PASS

- [ ] **Schritt 5: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/renderer/markdown.go
git commit -m "feat(renderer): extend LoadFile and IsSupportedFile for PDF and PS"
```

---

## Task 4: processBinaryFile-Binding

**Files:**
- Modify: `src/bindings.go`

- [ ] **Schritt 1: Import `"strings"` in `src/bindings.go` hinzufügen**

Im Import-Block (nach `"strconv"`) einfügen:

```go
"strings"
```

- [ ] **Schritt 2: `processBinaryFile`-Binding in `registerBindings` einfügen**

Nach dem `processEpub`-Binding (nach Zeile ~228) einfügen:

```go
	// processBinaryFile: Konvertiert binäre Dateiinhalte (EPUB, PDF, PS) zu HTML.
	//
	// Universeller Fallback für Drag & Drop auf Windows, wenn kein nativer Pfad
	// aus der URI-Liste verfügbar ist. JS liest Dateien binär via FileReader,
	// kodiert als Base64 und sendet hierher. Go rendert und ruft w.SetHtml() auf.
	//
	// @param base64Data Dateiinhalt als Base64-kodierter String.
	// @param filename   Dateiname mit Erweiterung (für Format-Erkennung und Titel).
	w.Bind("processBinaryFile", func(base64Data string, filename string) {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			log.Printf("processBinaryFile Base64-Fehler: %v", err)
			return
		}

		// Format anhand der Dateiendung bestimmen
		var result *renderer.Result
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".epub":
			result, err = renderer.ParseEpub(data, filename)
		case ".pdf":
			result, err = renderer.ParsePDF(data, filename)
		case ".ps":
			result, err = renderer.ParsePS(data, filename)
		default:
			log.Printf("processBinaryFile: unbekanntes Binärformat %q", ext)
			return
		}
		if err != nil {
			log.Printf("processBinaryFile Render-Fehler: %v", err)
			return
		}
		hash := computeHash(data)
		scrollPos := scrollPosForHash(hash)
		renderAndDisplay(w, result.HTML, result.Title, hash, scrollPos)
	})
```

- [ ] **Schritt 3: Kompilierung prüfen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o /dev/null .
```

Erwartet: keine Fehler

- [ ] **Schritt 4: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/bindings.go
git commit -m "feat(bindings): add processBinaryFile binding for PDF/PS/EPUB drag&drop"
```

---

## Task 5: JavaScript Drop-Handler aktualisieren

**Files:**
- Modify: `src/ui/assets/scripts.js`

- [ ] **Schritt 1: `supportedExtensions` auf Zeile 105 aktualisieren**

```javascript
/** Unterstützte Dateiendungen (muss mit IsSupportedFile in renderer/markdown.go übereinstimmen) */
var supportedExtensions = ['.md', '.markdown', '.txt', '.fb2', '.epub', '.pdf', '.ps'];
```

- [ ] **Schritt 2: Drop-Handler-Fallback (Zeilen 207–226) ersetzen**

Den Block von `var reader = new FileReader();` bis `}` (Ende des Drop-Handlers, aktuell Zeile 207–226) durch folgenden Code ersetzen:

```javascript
  var reader = new FileReader();
  // Binärformate als ArrayBuffer lesen und base64-kodiert an Go senden
  var binaryExts = ['.epub', '.pdf', '.ps'];
  var fileExt = '.' + file.name.toLowerCase().split('.').pop();
  if (binaryExts.indexOf(fileExt) !== -1) {
    reader.onload = function(ev) {
      if (typeof window.processBinaryFile === 'function') {
        window.processBinaryFile(arrayBufferToBase64(ev.target.result), file.name);
      }
    };
    reader.onerror = function() { showDropError('Datei konnte nicht gelesen werden.'); };
    reader.readAsArrayBuffer(file);
  } else {
    // Text-Formate: als UTF-8 lesen
    reader.onload = function(ev) {
      if (typeof window.processMarkdown === 'function') {
        window.processMarkdown(ev.target.result, file.name);
      }
    };
    reader.onerror = function() { showDropError('Datei konnte nicht gelesen werden.'); };
    reader.readAsText(file, 'utf-8');
  }
```

- [ ] **Schritt 3: Kompilierung prüfen** (scripts.js wird eingebettet)

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o /dev/null .
```

Erwartet: keine Fehler

- [ ] **Schritt 4: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/ui/assets/scripts.js
git commit -m "feat(ui): add PDF and PS to supported drop extensions"
```

---

## Task 6: Dateidialog-Filter erweitern

**Files:**
- Modify: `src/platform_linux.go`
- Modify: `src/platform_windows.go`

### Linux – GTK-Dateidialog

- [ ] **Schritt 1: GTK-Muster-Array in `platform_linux.go` aktualisieren**

Zeile 198–200 (GTK-Filter) ersetzen:

```c
    const char *patterns[] = {"*.md","*.markdown","*.txt","*.epub","*.fb2","*.html","*.htm","*.tex","*.pdf","*.ps"};
    for (int i = 0; i < 10; i++) {
        gtk_file_filter_add_pattern(filter, patterns[i]);
    }
```

### Linux – zenity/kdialog-Filter

- [ ] **Schritt 2: zenity-Muster in `showOpenFileDialogExternal` aktualisieren**

Zeile 299 ersetzen:

```go
patterns := "*.md *.markdown *.txt *.epub *.fb2 *.html *.htm *.tex *.pdf *.ps"
```

### Windows – Dateidialog-Filter

- [ ] **Schritt 3: Filter-String in `platform_windows.go` aktualisieren**

Zeile 118 ersetzen:

```c
    ofn.lpstrFilter  = L"Unterst\u00fctzte Dateien\0*.md;*.markdown;*.txt;*.epub;*.fb2;*.html;*.htm;*.tex;*.pdf;*.ps\0Alle Dateien (*.*)\0*.*\0";
```

- [ ] **Schritt 4: Kompilierung prüfen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o /dev/null .
```

Erwartet: keine Fehler

- [ ] **Schritt 5: Commit**

```bash
cd /home/claude-code/project/md-reader
git add src/platform_linux.go src/platform_windows.go
git commit -m "feat(platform): add PDF and PS to file picker filters"
```

---

## Task 7: NSIS-Installer – Ghostscript-Abschnitt

**Files:**
- Modify: `installer/md-reader.nsi`

- [ ] **Schritt 1: Ghostscript-Version als Konstante definieren**

Nach der Zeile `!define ICO_SRC` (ca. Zeile 46) einfügen:

```nsis
; Ghostscript – für PostScript-Unterstützung (optional)
!define GS_VERSION      "10.05.0"
!define GS_VERSION_FLAT "10050"
!define GS_INSTALLER    "gs${GS_VERSION_FLAT}w64.exe"
!define GS_URL          "https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs${GS_VERSION_FLAT}/${GS_INSTALLER}"
```

- [ ] **Schritt 2: Ghostscript-Section hinzufügen**

Nach der `Section "Desktop Shortcut"` (nach Zeile ~186) einfügen:

```nsis
; Optionale Komponente: Ghostscript für PostScript-Unterstützung
Section "Ghostscript (PostScript support)" SecGhostscript

    ; Prüfen ob gs.exe bereits im System-PATH vorhanden ist
    nsExec::ExecToStack 'cmd /C where gs.exe'
    Pop $0
    StrCmp $0 "0" gs_already_installed

    ; Ghostscript via PowerShell herunterladen
    DetailPrint "Downloading Ghostscript ${GS_VERSION}..."
    nsExec::ExecToLog 'powershell -NoProfile -Command "Invoke-WebRequest -Uri \"${GS_URL}\" -OutFile \"$TEMP\${GS_INSTALLER}\" -UseBasicParsing"'
    Pop $0
    StrCmp $0 "0" gs_download_ok

    ; Download fehlgeschlagen
    MessageBox MB_OK|MB_ICONINFORMATION \
        "Ghostscript download failed.$\n$\nPostScript files will be displayed as plain text.$\nYou can install Ghostscript manually from ghostscript.com."
    Goto gs_done

gs_download_ok:
    ; Ghostscript-Installer silent ausführen
    DetailPrint "Installing Ghostscript ${GS_VERSION}..."
    ExecWait '"$TEMP\${GS_INSTALLER}" /S' $0
    Delete "$TEMP\${GS_INSTALLER}"
    StrCmp $0 "0" gs_done

    ; Installation fehlgeschlagen
    MessageBox MB_OK|MB_ICONINFORMATION \
        "Ghostscript installation failed.$\n$\nPostScript files will be displayed as plain text."
    Goto gs_done

gs_already_installed:
    DetailPrint "Ghostscript already installed – skipping."

gs_done:
SectionEnd
```

- [ ] **Schritt 3: Ghostscript zur Komponenten-Beschreibung hinzufügen**

Im `MUI_FUNCTION_DESCRIPTION_BEGIN`-Block (nach `SecDesktop`-Beschreibung) einfügen:

```nsis
    !insertmacro MUI_DESCRIPTION_TEXT ${SecGhostscript} \
        "Downloads and installs GPL Ghostscript for PostScript (.ps) rendering. Optional – .ps files will display as plain text without it."
```

- [ ] **Schritt 4: Ghostscript aus Uninstall-Section entfernen** (gs verwaltet sich selbst)

Ghostscript hat eigenen Uninstaller – nichts in der Uninstall-Section nötig.

- [ ] **Schritt 5: Installer-Version aktualisieren** (Zeile 36)

```nsis
!define APP_VERSION     "1.0.33"
```

- [ ] **Schritt 6: Commit**

```bash
cd /home/claude-code/project/md-reader
git add installer/md-reader.nsi
git commit -m "feat(installer): add optional Ghostscript section for PostScript support"
```

---

## Task 8: Build, Tests, Release

**Files:**
- Modify: `src/build.txt`

- [ ] **Schritt 1: Build-Nummer erhöhen**

```bash
echo "33" > /home/claude-code/project/md-reader/src/build.txt
```

- [ ] **Schritt 2: Alle Tests ausführen**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH go test ./renderer/... ./ui/... -v
```

Erwartet: alle PASS – kein FAIL

- [ ] **Schritt 3: Linux x86_64 Build**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go build -o ../build/md-reader .
```

Erwartet: `../build/md-reader` wird erzeugt, keine Fehler

- [ ] **Schritt 4: Windows x86_64 Cross-Compile**

```bash
cd /home/claude-code/project/md-reader/src
PKG_CONFIG_PATH=$(pwd)/pkgconfig:$PKG_CONFIG_PATH \
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
  go build -ldflags="-H windowsgui" -o ../build/md-reader.exe .
```

Erwartet: `../build/md-reader.exe` wird erzeugt, keine Fehler

- [ ] **Schritt 5: Windows Installer bauen**

```bash
cd /home/claude-code/project/md-reader
makensis installer/md-reader.nsi
```

Erwartet: `build/md-reader-setup.exe` wird erzeugt

- [ ] **Schritt 6: CHANGELOG.md aktualisieren**

Oben in `CHANGELOG.md` einfügen:

```markdown
## Build 33 – 2026-04-15

### Added
- PDF support: native rendering via WebKit/Edge WebView2 (base64 embed)
- PostScript (.ps) support: converts to PDF via Ghostscript if available, falls back to plain text display
- Windows installer: optional Ghostscript component for PostScript rendering
- File picker and drag & drop now accept .pdf and .ps files
```

- [ ] **Schritt 7: FEATURES.md aktualisieren**

In der Feature-Liste markieren/ergänzen:

```markdown
- [x] PDF-Anzeige (.pdf) – nativ via WebView-Einbettung (Build 33)
- [x] PostScript-Anzeige (.ps) – via gs-Konvertierung oder Text-Fallback (Build 33)
```

- [ ] **Schritt 8: Release committen und pushen**

```bash
cd /home/claude-code/project/md-reader
git add src/build.txt CHANGELOG.md FEATURES.md
git commit -m "build(33): PDF and PostScript support"

GH_TOKEN=<TOKEN> gh release create v0.0.33 \
  build/md-reader \
  build/md-reader.exe \
  build/md-reader-setup.exe \
  --title "Build 33: PDF and PostScript support" \
  --notes "## What's new

- **PDF**: Native rendering via WebKit (Linux) and Edge WebView2 (Windows)
- **PostScript (.ps)**: Converts to PDF via Ghostscript if available, falls back to plain text
- **Windows installer**: Optional Ghostscript download for PostScript support
- File picker and drag & drop accept \`.pdf\` and \`.ps\` files"
```

---

## Selbst-Review

**Spec-Abdeckung:**
- ✅ PDF-Renderer mit base64-Einbettung → Task 1
- ✅ PS-Renderer mit gs-Fallback → Task 2
- ✅ LoadFile + IsSupportedFile → Task 3
- ✅ processBinaryFile-Binding → Task 4
- ✅ JS Drop-Handler → Task 5
- ✅ Platform-Filter Linux + Windows → Task 6
- ✅ NSIS Ghostscript-Abschnitt → Task 7
- ✅ Tests, Build, Release → Task 8

**Type-Konsistenz:**
- `ParsePDF(data []byte, filename string) (*Result, error)` – verwendet in Task 1 definiert, in Task 2+3 aufgerufen ✅
- `ParsePS(data []byte, filename string) (*Result, error)` – Task 2 definiert, Task 3+4 aufgerufen ✅
- `IsPDFFile` / `IsPSFile` – Task 1/2 definiert, Task 3 aufgerufen ✅
- `processBinaryFile` Go-Binding / JS-Aufruf – Task 4 definiert, Task 5 aufgerufen ✅
