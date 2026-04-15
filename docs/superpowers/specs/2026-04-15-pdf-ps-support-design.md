# Design: PDF und PostScript (.ps) Unterstützung

**Datum:** 2026-04-15  
**Autor:** Kurt Ingwer  
**Status:** Genehmigt

---

## Ziel

MD Reader soll `.pdf`- und `.ps`-Dateien öffnen, anzeigen und per Drag & Drop laden können – auf Linux und Windows, ohne Pflicht-Abhängigkeiten für Endnutzer.

---

## Architektur

### 1. `renderer/pdf.go` (neu)

**Zweck:** PDF-Dateien nativ im WebView einbetten.

**API:**
- `IsPDFFile(path string) bool` – erkennt `.pdf` (Groß-/Kleinschreibung ignoriert)
- `ParsePDF(data []byte, filename string) (*Result, error)` – gibt HTML zurück

**HTML-Ausgabe:**
```html
<div class="pdf-viewer">
  <embed src="data:application/pdf;base64,<BASE64>" 
         type="application/pdf" 
         width="100%" height="100%">
</div>
```

WebKitGTK (Linux) und Edge WebView2 (Windows) rendern PDFs nativ über diesen Mechanismus. Kein externes Tool nötig.

---

### 2. `renderer/ps.go` (neu)

**Zweck:** PostScript-Dateien anzeigen – mit nativem PDF-Rendering wenn `gs` verfügbar, sonst als Text.

**API:**
- `IsPSFile(path string) bool` – erkennt `.ps`
- `ParsePS(data []byte, filename string) (*Result, error)`

**Ablauf in `ParsePS`:**
1. Ghostscript aufrufen:
   ```
   gs -sDEVICE=pdfwrite -sOutputFile=- -dBATCH -dNOPAUSE -q -
   ```
   PS-Daten über stdin, PDF-Bytes aus stdout lesen.
2. Erfolg → `ParsePDF(pdfBytes, filename)` → natives PDF-Rendering
3. Fehler (gs nicht vorhanden oder Konvertierung fehlgeschlagen) → `ParseTextContent(string(data), filename)` → Text-Fallback

**Fehlerbehandlung:** Timeout nach 30 Sekunden für gs-Prozess.

---

### 3. Änderungen `renderer/markdown.go`

- `IsSupportedFile`: `.pdf` und `.ps` hinzufügen
- `LoadFile`: PDF und PS als Binärdaten lesen (wie EPUB), an `ParsePDF`/`ParsePS` weiterleiten

```go
// LoadFile – erweiterte Logik:
if IsPDFFile(filePath) {
    data, _ := os.ReadFile(filePath)
    return ParsePDF(data, filepath.Base(filePath))
}
if IsPSFile(filePath) {
    data, _ := os.ReadFile(filePath)
    return ParsePS(data, filepath.Base(filePath))
}
```

---

### 4. Änderungen `bindings.go`

Neues Go-Binding **`processBinaryFile(base64Data, filename)`**:
- Dekodiert Base64
- Leitet an `renderer.LoadFile`-Logik weiter (erkennt Format anhand Dateiname)
- Ersetzt den EPUB-spezifischen Fallback für alle Binärformate

`processEpub` bleibt für Rückwärtskompatibilität erhalten, delegiert intern an `processBinaryFile`.

---

### 5. Änderungen `ui/assets/scripts.js`

- `supportedExtensions`: `.pdf`, `.ps` hinzufügen
- Drop-Handler:
  - `.epub`, `.pdf`, `.ps` → `FileReader` als `ArrayBuffer` → base64 → `window.processBinaryFile()`
  - Alle anderen → `FileReader` als Text → `window.processMarkdown()`

---

### 6. Änderungen `platform_linux.go`

GTK-Dateidialog-Filter:
```c
const char *patterns[] = {"*.md","*.markdown","*.txt","*.epub","*.fb2",
                           "*.html","*.htm","*.tex","*.pdf","*.ps"};
```

zenity-Filter:
```go
patterns := "*.md *.markdown *.txt *.epub *.fb2 *.html *.htm *.tex *.pdf *.ps"
```

---

### 7. Änderungen `platform_windows.go`

Windows-Dateidialog-Filter:
```c
L"Unterstützte Dateien\0*.md;*.markdown;*.txt;*.epub;*.fb2;*.html;*.htm;*.tex;*.pdf;*.ps\0..."
```

---

### 8. NSIS Installer (`installer/md-reader.nsi`)

Neuer optionaler Abschnitt **"Ghostscript (PostScript support)"**:

1. In `.onInit`: prüfen ob `gs.exe` in PATH vorhanden (via `where gs.exe`)
2. Neuer `Section "Ghostscript" SecGhostscript`:
   - Wenn `gs` bereits vorhanden: Meldung + Abschnitt überspringen
   - Wenn nicht: PowerShell lädt GPL Ghostscript 10.x Installer (`gs10xxxw64.exe`) von `github.com/ArtifexSoftware/ghostpdl-downloads/releases` herunter
   - Installer wird silent ausgeführt (`/S`)
   - Temp-Datei wird gelöscht
3. Abschnitt kann in Komponenten-Auswahl abgewählt werden

---

### 9. Tests

- `renderer/pdf_test.go`: `IsPDFFile`, `ParsePDF` (Mindestgröße, base64-Einbettung, ungültige Daten)
- `renderer/ps_test.go`: `IsPSFile`, `ParsePS` (Fallback auf Text wenn gs nicht verfügbar, Titelextraktion)

---

## Nicht im Scope

- PDF-Textextraktion / Durchsuchbarkeit
- PS → PNG via draw2d (llgcode/ps) – zu komplex, kein Mehrwert
- Bundling von Ghostscript in den Linux-Build

---

## Abhängigkeiten

| Format | Linux | Windows |
|--------|-------|---------|
| PDF    | WebKitGTK (bereits vorhanden) | Edge WebView2 (bereits vorhanden) |
| PS → PDF | `gs` (optional, Fallback: Text) | Ghostscript via Installer (optional) |
| PS → Text | Fallback ohne externe Deps | Fallback ohne externe Deps |
