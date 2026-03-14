# Sicherheitsreport - md-reader
**Scan-Zeitpunkt:** 2026-03-14 18:07:33
**Scanner-Version:** 2.0.0
**Scan-Modus:** vollstaendig

## Projektinformationen

- **Sprachen:** C/C++, Go, JavaScript, Shell
- **Frameworks:** Keine erkannt
- **Paketmanager:** go modules
- **Docker:** Nein
- **CI/CD:** Nein
- **Tests vorhanden:** Ja

---

## Zusammenfassung

| Schweregrad | Anzahl |
|-------------|--------|
| KRITISCH    | 0 |
| HOCH        | 3 |
| MITTEL      | 2 |
| NIEDRIG     | 13 |
| INFO        | 9 |
| **GESAMT**  | **27** |

---

## Statische Code-Analyse (SAST)

## [2026-03-14 18:08:58] - Scan-Ergebnis für md-reader

**Gefundene Schwachstellen:** 27

- **HOCH:** 3
- **MITTEL:** 2
- **NIEDRIG:** 13
- **INFO:** 9

---

### 1. [2026-03-14 18:08:58] - HOCH - Import des unsafe-Packages

**Schweregrad:** HOCH
**CVSS Score:** 7.5
**Kategorie:** Unsicherer Code
**CWE:** CWE-676
**Betroffene Datei:** `src/platform_linux.go`
**Zeile:** 107

#### Betroffener Code
```
"unsafe"
```

#### Beschreibung
Das unsafe-Package umgeht Gos Sicherheitsmechanismen (Typensicherheit, Speichersicherheit). Manuelle Prüfung erforderlich.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um unsicherer code-basierte Angriffe durchzuführen.

#### Lösung
unsafe vermeiden. Sichere Alternativen wie encoding/binary verwenden.

#### Referenzen
- https://cwe.mitre.org/data/definitions/676.html

---

### 2. [2026-03-14 18:08:58] - HOCH - Import des unsafe-Packages

**Schweregrad:** HOCH
**CVSS Score:** 7.5
**Kategorie:** Unsicherer Code
**CWE:** CWE-676
**Betroffene Datei:** `src/platform_windows.go`
**Zeile:** 137

#### Betroffener Code
```
"unsafe"
```

#### Beschreibung
Das unsafe-Package umgeht Gos Sicherheitsmechanismen (Typensicherheit, Speichersicherheit). Manuelle Prüfung erforderlich.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um unsicherer code-basierte Angriffe durchzuführen.

#### Lösung
unsafe vermeiden. Sichere Alternativen wie encoding/binary verwenden.

#### Referenzen
- https://cwe.mitre.org/data/definitions/676.html

---

### 3. [2026-03-14 18:08:58] - HOCH - Sensible Daten in Log-Ausgabe (A09:2021)

**Schweregrad:** HOCH
**CVSS Score:** 7.5
**Kategorie:** Logging / Information Disclosure
**CWE:** CWE-532
**Betroffene Datei:** `src/ui/assets/katex/katex.min.js`
**Zeile:** 1

#### Betroffener Code
```
!function(e,t){"object"==typeof exports&&"object"==typeof module?module.exports=t():"function"==typeof define&&define.amd?define([],t):"object"==typeof exports?exports.katex=t():e.katex=t()}("undefine
```

#### Beschreibung
Log-Statements enthalten moeglicherweise sensible Daten wie Passwoerter, API-Keys, Tokens oder private Schluessel. Dies kann zu Informations-Offenlegung fuehren (OWASP A09:2021).

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um logging / information disclosure-basierte Angriffe durchzuführen.

#### Lösung
Sensible Daten vor dem Logging maskieren oder entfernen. Structured Logging verwenden und PII/Credentials ausfiltern.

#### Referenzen
- https://cwe.mitre.org/data/definitions/532.html

---

### 4. [2026-03-14 18:08:58] - MITTEL - Möglicher Integer-Overflow bei Typ-Konvertierung

**Schweregrad:** MITTEL
**CVSS Score:** 5.3
**Kategorie:** Integer Overflow
**CWE:** CWE-190
**Betroffene Datei:** `src/bindings.go`
**Zeile:** 124

#### Betroffener Code
```
app.config.FontSize = int(fontSize)
```

#### Beschreibung
Direkte Typ-Konvertierung (z.B. int(uint64Value)) kann bei großen Werten zu Integer-Overflow führen (Vorzeichen-Umkehr).

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um integer overflow-basierte Angriffe durchzuführen.

#### Lösung
Wertbereich vor der Konvertierung prüfen. math.MaxInt verwenden für Grenzwertprüfung.

#### Referenzen
- https://cwe.mitre.org/data/definitions/190.html

---

### 5. [2026-03-14 18:08:58] - MITTEL - Möglicher Integer-Overflow bei Typ-Konvertierung

**Schweregrad:** MITTEL
**CVSS Score:** 5.3
**Kategorie:** Integer Overflow
**CWE:** CWE-190
**Betroffene Datei:** `src/bindings.go`
**Zeile:** 162

#### Betroffener Code
```
app.config.ScrollHistory[hash] = int(scrollPos)
```

#### Beschreibung
Direkte Typ-Konvertierung (z.B. int(uint64Value)) kann bei großen Werten zu Integer-Overflow führen (Vorzeichen-Umkehr).

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um integer overflow-basierte Angriffe durchzuführen.

#### Lösung
Wertbereich vor der Konvertierung prüfen. math.MaxInt verwenden für Grenzwertprüfung.

#### Referenzen
- https://cwe.mitre.org/data/definitions/190.html

---

### 6. [2026-03-14 18:08:58] - NIEDRIG - cgo-Import - CVE-2025-61732 Code-Smuggling-Risiko

**Schweregrad:** NIEDRIG
**CVSS Score:** 9.4
**Kategorie:** Code Injection
**CWE:** CWE-94
**Betroffene Datei:** `src/platform_linux.go`
**Zeile:** 104

#### Betroffener Code
```
import "C"
```

#### Beschreibung
cgo ermoeglicht Einbettung von C-Code in Go. CVE-2025-61732 (CVSS 9.4): Diskrepanz zwischen Go- und C++-Kommentar-Parsing erlaubt Code-Smuggling in cgo-Binaries. Ein Angreifer kann ueber manipulierte C-Kommentare beliebigen Code einschleusen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um code injection-basierte Angriffe durchzuführen.

#### Lösung
Go-Version auf >= 1.26 aktualisieren. cgo-Nutzung minimieren. C-Code-Abschnitte manuell auf ungewoehnliche Kommentare pruefen.

#### Referenzen
- https://cwe.mitre.org/data/definitions/94.html

---

### 7. [2026-03-14 18:08:58] - NIEDRIG - cgo-Import - CVE-2025-61732 Code-Smuggling-Risiko

**Schweregrad:** NIEDRIG
**CVSS Score:** 9.4
**Kategorie:** Code Injection
**CWE:** CWE-94
**Betroffene Datei:** `src/platform_windows.go`
**Zeile:** 132

#### Betroffener Code
```
import "C"
```

#### Beschreibung
cgo ermoeglicht Einbettung von C-Code in Go. CVE-2025-61732 (CVSS 9.4): Diskrepanz zwischen Go- und C++-Kommentar-Parsing erlaubt Code-Smuggling in cgo-Binaries. Ein Angreifer kann ueber manipulierte C-Kommentare beliebigen Code einschleusen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um code injection-basierte Angriffe durchzuführen.

#### Lösung
Go-Version auf >= 1.26 aktualisieren. cgo-Nutzung minimieren. C-Code-Abschnitte manuell auf ungewoehnliche Kommentare pruefen.

#### Referenzen
- https://cwe.mitre.org/data/definitions/94.html

---

### 8. [2026-03-14 18:08:58] - NIEDRIG - Unsichere Verwendung von innerHTML mit variablem Inhalt

**Schweregrad:** NIEDRIG
**CVSS Score:** 6.1
**Kategorie:** Cross-Site Scripting (XSS)
**CWE:** CWE-79
**Betroffene Datei:** `src/ui/assets/scripts.js`
**Zeile:** 222

#### Betroffener Code
```
document.getElementById('content').innerHTML = html;
```

#### Beschreibung
innerHTML wird mit einer Variablen oder einem dynamischen Ausdruck befuellt. Benutzereingaben koennen so als HTML/JavaScript ausgefuehrt werden. Statische String-Literale werden nicht erkannt (z.B. innerHTML = '' ist sicher).

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um cross-site scripting (xss)-basierte Angriffe durchzuführen.

#### Lösung
textContent statt innerHTML verwenden. DOMPurify fuer HTML-Sanitisierung einsetzen.

#### Referenzen
- https://cwe.mitre.org/data/definitions/79.html

---

### 9. [2026-03-14 18:08:58] - NIEDRIG - Unsichere Verwendung von innerHTML mit variablem Inhalt

**Schweregrad:** NIEDRIG
**CVSS Score:** 6.1
**Kategorie:** Cross-Site Scripting (XSS)
**CWE:** CWE-79
**Betroffene Datei:** `src/ui/assets/scripts.js`
**Zeile:** 381

#### Betroffener Code
```
list.innerHTML = '';
```

#### Beschreibung
innerHTML wird mit einer Variablen oder einem dynamischen Ausdruck befuellt. Benutzereingaben koennen so als HTML/JavaScript ausgefuehrt werden. Statische String-Literale werden nicht erkannt (z.B. innerHTML = '' ist sicher).

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um cross-site scripting (xss)-basierte Angriffe durchzuführen.

#### Lösung
textContent statt innerHTML verwenden. DOMPurify fuer HTML-Sanitisierung einsetzen.

#### Referenzen
- https://cwe.mitre.org/data/definitions/79.html

---

### 10. [2026-03-14 18:08:58] - NIEDRIG - .then() ohne .catch()

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.0
**Kategorie:** Error Handling
**CWE:** CWE-755
**Betroffene Datei:** `src/ui/assets/scripts.js`
**Zeile:** 354

#### Betroffener Code
```
    window.nativeFullscreen().then(function(isNowFullscreen) {
```

#### Beschreibung
Promise-Chain mit .then() aber ohne .catch(). Unbehandelte Promise-Rejections koennen die Anwendung zum Absturz bringen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um error handling-basierte Angriffe durchzuführen.

#### Lösung
.catch() an die Promise-Chain anfuegen oder async/await mit try/catch verwenden.

#### Referenzen
- https://cwe.mitre.org/data/definitions/755.html

---

### 11. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 135

#### Betroffener Code
```
href, _ := url.PathUnescape(item.Href)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 12. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 164

#### Betroffener Code
```
href, _ := url.PathUnescape(item.Href)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 13. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 270

#### Betroffener Code
```
decodedSrc, _ := url.PathUnescape(src)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 14. [2026-03-14 18:08:58] - NIEDRIG - defer Close() ohne Fehlerprüfung

**Schweregrad:** NIEDRIG
**CVSS Score:** 2.0
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-754
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 323

#### Betroffener Code
```
defer rc.Close()
```

#### Beschreibung
defer file.Close() ignoriert den Fehler-Rückgabewert. Bei Schreiboperationen können Daten verloren gehen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
defer func() { if err := f.Close(); err != nil { log.Error(err) } }()

#### Referenzen
- https://cwe.mitre.org/data/definitions/754.html

---

### 15. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/latex.go`
**Zeile:** 160

#### Betroffener Code
```
displayName, _ := extractBraceContent(rest, skip)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 16. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/latex.go`
**Zeile:** 215

#### Betroffener Code
```
def, _ := extractBraceContent(rest, skip)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 17. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/latex.go`
**Zeile:** 239

#### Betroffener Code
```
opText, _ := extractBraceContent(rest, pos1)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 18. [2026-03-14 18:08:58] - NIEDRIG - Go Error-Wert mit _ ignoriert (OWASP A10:2025)

**Schweregrad:** NIEDRIG
**CVSS Score:** 3.7
**Kategorie:** Fehlerbehandlung
**CWE:** CWE-252
**Betroffene Datei:** `src/renderer/latex.go`
**Zeile:** 335

#### Betroffener Code
```
val, _ := extractBraceContent(content, start-1)
```

#### Beschreibung
Der Error-Rückgabewert einer Funktion wird explizit mit _ verworfen. Fehler können unbemerkt bleiben und zu unerwartetem Verhalten führen.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um fehlerbehandlung-basierte Angriffe durchzuführen.

#### Lösung
Error-Wert prüfen und behandeln: if err != nil { return err }

#### Referenzen
- https://cwe.mitre.org/data/definitions/252.html

---

### 19. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 217

#### Betroffener Code
```
return strings.ToLower(filepath.Ext(filePath)) == ".epub"
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 20. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/epub.go`
**Zeile:** 294

#### Betroffener Code
```
ext := strings.ToLower(path.Ext(imgZipPath))
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 21. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/latex.go`
**Zeile:** 276

#### Betroffener Code
```
return strings.ToLower(getFileExt(path)) == ".tex"
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 22. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/fb2.go`
**Zeile:** 73

#### Betroffener Code
```
return strings.ToLower(filepath.Ext(filePath)) == ".fb2"
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 23. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/images.go`
**Zeile:** 91

#### Betroffener Code
```
ext := strings.ToLower(filepath.Ext(absPath))
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 24. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/markdown.go`
**Zeile:** 87

#### Betroffener Code
```
ext := strings.ToLower(filepath.Ext(filename))
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 25. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/markdown.go`
**Zeile:** 125

#### Betroffener Code
```
ext := strings.ToLower(filepath.Ext(path))
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 26. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/markdown.go`
**Zeile:** 136

#### Betroffener Code
```
ext := strings.ToLower(filepath.Ext(path))
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---

### 27. [2026-03-14 18:08:58] - INFO - strings.ToLower/ToUpper fuer Pfadvergleiche - Path-Confusion-Risiko

**Schweregrad:** INFO
**CVSS Score:** 5.3
**Kategorie:** Path Traversal
**CWE:** CWE-178
**Betroffene Datei:** `src/renderer/text.go`
**Zeile:** 42

#### Betroffener Code
```
return strings.ToLower(filepath.Ext(filePath)) == ".txt"
```

#### Beschreibung
strings.ToLower()/ToUpper() kann bei bestimmten UTF-8-Zeichen die Byte-Laenge aendern. CVE-2026-24895 (FrankenPHP) zeigt, dass dies zu Path-Confusion und Sicherheitsumgehungen fuehrt.

#### Auswirkung
Ein Angreifer könnte diese Schwachstelle nutzen um path traversal-basierte Angriffe durchzuführen.

#### Lösung
strings.EqualFold() fuer case-insensitive Vergleiche verwenden. Pfade vor dem Vergleich normalisieren (filepath.Clean()).

#### Referenzen
- https://cwe.mitre.org/data/definitions/178.html

---


## Dependency-Analyse

## [2026-03-14 18:08:58] - Dependency-Analyse für md-reader

**Geprüfte Abhängigkeiten:** 7
**Gefundene Probleme:** 0


## Secrets-Scan

## [2026-03-14 18:08:58] - Secrets-Scan für md-reader

Keine hardcodierten Geheimnisse gefunden.


## Konfigurationsanalyse

## [2026-03-14 18:08:58] - Konfigurationsanalyse für md-reader

Keine Konfigurationsprobleme gefunden.

