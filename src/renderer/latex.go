// Package renderer – LaTeX-Quelldatei-Parsing und Rendering.
//
// Konvertiert LaTeX (.tex) Quelldateien in anzeigbares HTML.
// Unterstützt die gängigsten LaTeX-Befehle und Umgebungen.
// Mathematische Formeln werden unverändert ausgegeben,
// da KaTeX im Browser für die Darstellung zuständig ist.
//
// Unterstützte Elemente:
//   - Abschnitte: \section, \subsection, \subsubsection, \paragraph
//   - Textformatierung: \textbf, \textit, \emph, \underline, \texttt, \textsc
//   - Listen: \begin{itemize}, \begin{enumerate}, \item
//   - Umgebungen: verbatim, quote, abstract, center, figure
//   - Verlinkung: \href, \url
//   - Zeilenumbrüche: \\, \newline, \par
//   - Mathematik: $...$, $$...$$, \[...\], \(...\), equation, align (→ KaTeX)
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-11
package renderer

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

// ---- Reguläre Ausdrücke für LaTeX-Befehle ----

// reLatexComment erkennt LaTeX-Kommentare (% bis Zeilenende).
// Escaped Prozentzeichen (\%) werden nicht erfasst.
var reLatexComment = regexp.MustCompile(`(?m)(?:[^\\])(%[^\n]*)`)

// reLatexCommandStart erkennt Zeilenanfangs-Kommentare.
var reLatexCommentStart = regexp.MustCompile(`(?m)^%[^\n]*`)

// reMathInline erkennt einfache Inline-Mathematik $...$ (nicht $$...$$).
var reMathInline = regexp.MustCompile(`(?s)\$\$.*?\$\$|\$[^$\n]+?\$|\\\(.*?\\\)|\\\[.*?\\\]`)

// reBeginTag erkennt öffnende \begin{umgebung} Tags.
var reBeginTag = regexp.MustCompile(`\\begin\{([^}]+)\}`)

// reItem erkennt \item innerhalb von Listen-Umgebungen.
var reItem = regexp.MustCompile(`(?m)^[ \t]*\\item(?:\[([^\]]*)\])?[ \t]*`)

// reSectionCmd erkennt Abschnittsbefehle (inkl. Sternvarianten wie \section*).
var reSectionCmd = regexp.MustCompile(`\\((?:sub)*section|paragraph|subparagraph)\*?\{`)

// reHref erkennt \href{url}{text}.
var reHref = regexp.MustCompile(`\\href\{([^}]*)\}\{`)

// reUrl erkennt \url{url}.
var reUrl = regexp.MustCompile(`\\url\{([^}]*)\}`)

// reFootnote erkennt \footnote{text}.
var reFootnote = regexp.MustCompile(`\\footnote\{`)

// reLabel erkennt \label{id}.
var reLabel = regexp.MustCompile(`\\label\{([^}]*)\}`)

// reRef erkennt \ref{id}.
var reRef = regexp.MustCompile(`\\ref\{([^}]*)\}`)

// reCite erkennt \cite{schlüssel}.
var reCite = regexp.MustCompile(`\\cite(?:\[[^\]]*\])?\{([^}]*)\}`)

// reLinebreak erkennt \\ oder \newline als Zeilenumbruch.
var reLinebreak = regexp.MustCompile(`\\\\|\\newline\b`)

// reSpaces normalisiert mehrfache Leerzeichen/Zeilenumbrüche.
var reMultiNewlines = regexp.MustCompile(`\n{3,}`)

// ParseLaTeXContent konvertiert LaTeX-Quelltext in HTML.
//
// Extrahiert Titel und Dokumentkörper aus der .tex-Datei und wandelt
// gängige LaTeX-Befehle in HTML-Äquivalente um. Mathematische Ausdrücke
// werden unverändert ausgegeben, damit KaTeX sie im Browser rendern kann.
// Custom-Makros (\newcommand, \DeclareMathOperator) werden als window.__latexMacros
// eingebettet, damit KaTeX sie korrekt rendern kann.
//
// @param content  Der LaTeX-Quelltext der Datei.
// @param filename Dateiname (für den Titel-Fallback).
// @return Result mit gerendertem HTML und extrahiertem Titel, oder Fehler.
// @lastModified 2026-03-11
func ParseLaTeXContent(content, filename string) (*Result, error) {
	// Kommentare entfernen (aber \% behalten)
	content = removeLatexComments(content)

	// Präambel isolieren (vor \begin{document}) für Makro-Extraktion
	preamble := ""
	if idx := strings.Index(content, `\begin{document}`); idx >= 0 {
		preamble = content[:idx]
	}

	// Metadaten aus der Präambel extrahieren
	title := extractLatexCommand(content, "title")
	author := extractLatexCommand(content, "author")
	date := extractLatexCommand(content, "date")

	// Theorem-Umgebungen und Custom-Makros aus der Präambel extrahieren
	theorems := extractTheoremEnvs(preamble)
	macros := extractLatexMacros(preamble)

	// Dokumentkörper zwischen \begin{document} und \end{document} isolieren
	body := extractLatexBody(content)

	// LaTeX in HTML umwandeln (mit Theorem-Map für schöne Ausgabe)
	htmlContent := convertLatexToHTML(body, title, author, date, theorems, macros)

	// Fallback-Titel: Dateiname ohne Erweiterung
	if title == "" {
		title = fileBaseName(filename)
	}

	return &Result{
		HTML:       htmlContent,
		Title:      title,
		RawContent: content,
	}, nil
}

// extractTheoremEnvs liest \newtheorem{envName}{Anzeigetext} Definitionen aus
// der Präambel und gibt eine Map envName→Anzeigetext zurück.
//
// Unterstützt beide Varianten:
//   - \newtheorem{theorem}{Satz}[section]
//   - \newtheorem{lemma}[theorem]{Lemma}
//
// @param preamble LaTeX-Präambel (vor \begin{document}).
// @return Map von Umgebungsname zu Anzeigetext (z.B. "theorem" → "Satz").
// @lastModified 2026-03-11
func extractTheoremEnvs(preamble string) map[string]string {
	result := make(map[string]string)
	marker := `\newtheorem{`
	rest := preamble
	for {
		idx := strings.Index(rest, marker)
		if idx < 0 {
			break
		}
		// Umgebungsname aus dem ersten {}-Block extrahieren
		envName, pos1 := extractBraceContent(rest, idx+len(marker)-1)
		envName = strings.TrimSpace(envName)
		if envName == "" {
			rest = rest[idx+len(marker):]
			continue
		}
		// Optionalen [shared-counter] oder [parent-counter] überspringen
		skip := pos1
		for skip < len(rest) && (rest[skip] == ' ' || rest[skip] == '\t' || rest[skip] == '\n') {
			skip++
		}
		if skip < len(rest) && rest[skip] == '[' {
			endBracket := strings.IndexByte(rest[skip:], ']')
			if endBracket >= 0 {
				skip += endBracket + 1
			}
		}
		// Anzeigetext aus dem nächsten {}-Block extrahieren
		displayName, _ := extractBraceContent(rest, skip)
		displayName = strings.TrimSpace(displayName)
		if displayName != "" {
			result[envName] = displayName
		}
		rest = rest[pos1:]
	}
	return result
}

// extractLatexMacros liest \newcommand- und \DeclareMathOperator-Definitionen
// aus der Präambel und gibt eine KaTeX-kompatible Makro-Map zurück.
//
// Nur argumentlose Makros werden unterstützt (keine #1/#2 Substitution),
// da diese direkt als KaTeX-Makros übergeben werden können.
//
// Beispiele:
//   - \newcommand{\N}{\mathbb{N}} → {"\\N": "\\mathbb{N}"}
//   - \DeclareMathOperator{\ord}{ord} → {"\\ord": "\\operatorname{ord}"}
//
// @param preamble LaTeX-Präambel (vor \begin{document}).
// @return Map Befehlsname→KaTeX-Definition (beide mit führendem Backslash).
// @lastModified 2026-03-11
func extractLatexMacros(preamble string) map[string]string {
	macros := make(map[string]string)

	// \newcommand{\name}{definition} und \newcommand*{\name}{definition}
	for _, marker := range []string{`\newcommand{`, `\newcommand*{`, `\renewcommand{`, `\renewcommand*{`} {
		rest := preamble
		for {
			idx := strings.Index(rest, marker)
			if idx < 0 {
				break
			}
			// Befehlsname extrahieren
			cmdName, pos1 := extractBraceContent(rest, idx+len(marker)-1)
			cmdName = strings.TrimSpace(cmdName)
			if cmdName == "" || !strings.HasPrefix(cmdName, `\`) {
				rest = rest[idx+len(marker):]
				continue
			}
			// Optionalen [Anzahl Argumente] überspringen
			skip := pos1
			for skip < len(rest) && (rest[skip] == ' ' || rest[skip] == '\t') {
				skip++
			}
			hasArgs := false
			if skip < len(rest) && rest[skip] == '[' {
				endBracket := strings.IndexByte(rest[skip:], ']')
				if endBracket >= 0 {
					skip += endBracket + 1
					hasArgs = true
				}
			}
			// Definition extrahieren
			def, _ := extractBraceContent(rest, skip)
			def = strings.TrimSpace(def)
			// Nur argumentlose Makros ohne #1 etc. übernehmen
			if def != "" && !hasArgs && !strings.Contains(def, "#") {
				macros[cmdName] = def
			}
			rest = rest[pos1:]
		}
	}

	// \DeclareMathOperator{\name}{text}
	for _, marker := range []string{`\DeclareMathOperator{`, `\DeclareMathOperator*{`} {
		rest := preamble
		for {
			idx := strings.Index(rest, marker)
			if idx < 0 {
				break
			}
			cmdName, pos1 := extractBraceContent(rest, idx+len(marker)-1)
			cmdName = strings.TrimSpace(cmdName)
			if cmdName == "" || !strings.HasPrefix(cmdName, `\`) {
				rest = rest[idx+len(marker):]
				continue
			}
			opText, _ := extractBraceContent(rest, pos1)
			opText = strings.TrimSpace(opText)
			if opText != "" {
				macros[cmdName] = `\operatorname{` + opText + `}`
			}
			rest = rest[pos1:]
		}
	}

	return macros
}

// buildMacrosScript erzeugt einen <script>-Block, der window.__latexMacros
// für KaTeX mit den extrahierten Makros befüllt.
//
// Das Script wird dem HTML-Inhalt vorangestellt, damit initKaTeX() die
// Makros beim Rendern der Formeln verwenden kann.
//
// @param macros Map Befehlsname→KaTeX-Definition.
// @return HTML-Script-Tag oder leer wenn keine Makros vorhanden.
// @lastModified 2026-03-11
func buildMacrosScript(macros map[string]string) string {
	if len(macros) == 0 {
		return ""
	}
	jsonBytes, err := json.Marshal(macros)
	if err != nil {
		return ""
	}
	return `<script>window.__latexMacros=` + string(jsonBytes) + `;</script>`
}

// IsLaTeXFile prüft ob ein Dateipfad auf eine LaTeX-Datei (.tex) zeigt.
//
// @param path Dateipfad.
// @return true wenn die Datei .tex Endung hat.
func IsLaTeXFile(path string) bool {
	return strings.ToLower(getFileExt(path)) == ".tex"
}

// getFileExt gibt die Dateiendung eines Pfads zurück (inkl. Punkt).
func getFileExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

// removeLatexComments entfernt LaTeX-Kommentare (% bis Zeilenende).
// Escaped Prozentzeichen (\%) bleiben erhalten.
//
// @param content LaTeX-Quelltext.
// @return Text ohne Kommentare.
func removeLatexComments(content string) string {
	// Zeilenweise verarbeiten um \% korrekt zu behandeln
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		// Kommentarzeichen suchen, dabei \% überspringen
		i := 0
		for i < len(line) {
			if line[i] == '\\' {
				i += 2 // escaped Zeichen überspringen
				continue
			}
			if line[i] == '%' {
				// Kommentar gefunden – Rest der Zeile abschneiden
				line = line[:i]
				break
			}
			i++
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// extractLatexCommand extrahiert den Inhalt eines einfachen LaTeX-Befehls.
// Beispiel: \title{Mein Titel} → "Mein Titel"
//
// @param content LaTeX-Quelltext.
// @param cmd     Befehlsname ohne führenden Backslash (z.B. "title").
// @return Befehlsinhalt als String, leer wenn nicht gefunden.
func extractLatexCommand(content, cmd string) string {
	marker := `\` + cmd + `{`
	idx := strings.Index(content, marker)
	if idx < 0 {
		return ""
	}
	// Öffnende Klammer gefunden → Inhalt bis zur schließenden extrahieren
	start := idx + len(marker)
	val, _ := extractBraceContent(content, start-1)
	return strings.TrimSpace(val)
}

// extractLatexBody isoliert den Inhalt zwischen \begin{document} und \end{document}.
// Falls kein document-Environment gefunden wird, wird der gesamte Inhalt zurückgegeben.
//
// @param content LaTeX-Quelltext (ohne Kommentare).
// @return Dokumentkörper als String.
func extractLatexBody(content string) string {
	const beginDoc = `\begin{document}`
	const endDoc = `\end{document}`

	start := strings.Index(content, beginDoc)
	end := strings.LastIndex(content, endDoc)

	if start >= 0 && end > start {
		return content[start+len(beginDoc) : end]
	}
	// Kein document-Environment → gesamten Inhalt als Körper verwenden
	return content
}

// extractBraceContent extrahiert den Inhalt der {...} Gruppe beginnend bei pos.
// Berücksichtigt verschachtelte Klammern korrekt.
//
// @param s   Eingabestring.
// @param pos Startposition (muss auf '{' zeigen oder davor).
// @return (Inhalt, Position nach der schließenden Klammer).
func extractBraceContent(s string, pos int) (string, int) {
	// Öffnende Klammer suchen
	for pos < len(s) && s[pos] != '{' {
		pos++
	}
	if pos >= len(s) {
		return "", pos
	}
	pos++ // '{' überspringen
	start := pos
	depth := 1
	for pos < len(s) && depth > 0 {
		switch s[pos] {
		case '{':
			depth++
		case '}':
			depth--
		case '\\':
			pos++ // nächstes Zeichen ist escaped → überspringen
		}
		pos++
	}
	if depth != 0 {
		return s[start:], pos
	}
	return s[start : pos-1], pos
}

// convertLatexToHTML wandelt den LaTeX-Dokumentkörper in HTML um.
//
// Verarbeitet Umgebungen, Befehle und Textformatierung in der Reihenfolge:
//  1. Math-Ausdrücke durch Platzhalter schützen
//  2. Umgebungen (begin/end) konvertieren
//  3. Einfache Befehle konvertieren
//  4. Absätze erzeugen
//  5. Math-Platzhalter wiederherstellen
//
// @param body     LaTeX-Dokumentkörper.
// @param title    Titel (für \maketitle).
// @param author   Autor (für \maketitle).
// @param date     Datum (für \maketitle).
// @param theorems Map envName→Anzeigetext aus \newtheorem-Definitionen.
// @param macros   KaTeX-Makros aus \newcommand/\DeclareMathOperator.
// @return HTML-String.
// @lastModified 2026-03-11
func convertLatexToHTML(body, title, author, date string, theorems map[string]string, macros map[string]string) string {
	// ---- Schritt 1: Math-Ausdrücke schützen ----
	// Mathematik wird durch Platzhalter ersetzt, damit die weiteren
	// Ersetzungsschritte die Formeln nicht beschädigen.
	mathPlaceholders := make([]string, 0)
	body = protectMath(body, &mathPlaceholders)

	// ---- Schritt 2: \maketitle einfügen ----
	if title != "" {
		titleBlock := buildLatexTitleBlock(title, author, date)
		body = strings.ReplaceAll(body, `\maketitle`, titleBlock)
	} else {
		body = strings.ReplaceAll(body, `\maketitle`, "")
	}

	// ---- Schritt 3: Umgebungen konvertieren (innen vor außen) ----
	// Mehrere Durchläufe für verschachtelte Umgebungen
	for i := 0; i < 5; i++ {
		prev := body
		body = convertLatexEnvironments(body, theorems)
		if body == prev {
			break
		}
	}

	// ---- Schritt 4: Abschnittsbefehle konvertieren ----
	body = convertLatexSections(body)

	// ---- Schritt 5: Verlinkung konvertieren ----
	body = convertLatexLinks(body)

	// ---- Schritt 6: Textformatierung konvertieren ----
	body = convertLatexFormatting(body)

	// ---- Schritt 7: Sonstige Befehle ----
	body = convertLatexMisc(body)

	// ---- Schritt 8: Absätze aus Leerzeilen erzeugen ----
	body = convertLatexParagraphs(body)

	// ---- Schritt 9: Math-Platzhalter wiederherstellen ----
	body = restoreMath(body, mathPlaceholders)

	// Ergebnis in einen Artikel-Wrapper einbetten
	// Makros als <script> voranstellen, damit initKaTeX() sie verwenden kann
	macrosScript := buildMacrosScript(macros)
	return macrosScript + `<article class="latex-document">` + body + `</article>`
}

// mathEnvNames enthält die Namen mathematischer LaTeX-Umgebungen.
var mathEnvNames = map[string]bool{
	"equation": true, "equation*": true,
	"align": true, "align*": true,
	"eqnarray": true, "eqnarray*": true,
	"multline": true, "multline*": true,
	"gather": true, "gather*": true,
	"alignat": true, "alignat*": true,
	"math": true, "displaymath": true,
}

// protectMath ersetzt alle mathematischen Ausdrücke durch Platzhalter.
// Schützt sie vor Verfälschung durch die LaTeX-Befehlskonvertierung.
//
// @param content          LaTeX-Text.
// @param placeholders     Zeiger auf Slice für die gespeicherten Formeln.
// @return Text mit Platzhaltern statt Formeln.
func protectMath(content string, placeholders *[]string) string {
	// Verarbeitungsreihenfolge: längere/spezifischere Trennzeichen zuerst

	// \begin{math-env}...\end{math-env} → Platzhalter
	content = processLatexEnvironments(content, func(envName, envContent string) (string, bool) {
		if !mathEnvNames[envName] {
			return "", false
		}
		idx := len(*placeholders)
		formula := "$$" + `\begin{` + envName + `}` + envContent + `\end{` + envName + `}` + "$$"
		*placeholders = append(*placeholders, formula)
		return mathPlaceholderTag(idx), true
	})

	// $$...$$ (Display-Math, vor $...$ verarbeiten!)
	content = replaceMathDelimiter(content, "$$", "$$", placeholders)

	// \[...\] (Display-Math)
	content = replaceMathDelimiterPair(content, `\[`, `\]`, placeholders)

	// \(...\) (Inline-Math)
	content = replaceMathDelimiterPair(content, `\(`, `\)`, placeholders)

	// $...$ (Inline-Math, zuletzt)
	content = replaceMathDelimiter(content, "$", "$", placeholders)

	return content
}

// processLatexEnvironments iteriert durch alle \begin{X}...\end{X} Blöcke
// und ruft handler auf. Gibt der Handler (result, true) zurück, wird der Block
// durch result ersetzt. Bei (_, false) bleibt der Block unverändert.
//
// Diese Funktion unterstützt keine Backreferences (Go regexp-Einschränkung)
// und verarbeitet Umgebungen daher iterativ mit String-Suche.
//
// @param content LaTeX-Text.
// @param handler Callback (envName, envContent) → (replacement, matched).
// @return Text mit verarbeiteten Umgebungen.
func processLatexEnvironments(content string, handler func(envName, envContent string) (string, bool)) string {
	var sb strings.Builder
	rest := content
	for {
		// Nächstes \begin{ suchen
		loc := reBeginTag.FindStringIndex(rest)
		if loc == nil {
			sb.WriteString(rest)
			break
		}
		// Umgebungsname extrahieren
		match := reBeginTag.FindStringSubmatch(rest[loc[0]:loc[1]])
		if len(match) < 2 {
			sb.WriteString(rest[:loc[1]])
			rest = rest[loc[1]:]
			continue
		}
		envName := match[1]
		endTag := `\end{` + envName + `}`
		// Inhalt zwischen \begin{X} und passendem \end{X} suchen
		// (einfache Suche ohne echtes Verschachtelungs-Tracking für gleiche Umgebung)
		afterBegin := rest[loc[1]:]
		endIdx := strings.Index(afterBegin, endTag)
		if endIdx < 0 {
			// Kein passendes \end gefunden → unverändert übernehmen
			sb.WriteString(rest[:loc[1]])
			rest = afterBegin
			continue
		}
		envContent := afterBegin[:endIdx]
		// Handler aufrufen
		if replacement, ok := handler(envName, envContent); ok {
			sb.WriteString(rest[:loc[0]])
			sb.WriteString(replacement)
		} else {
			// Unverändert übernehmen
			sb.WriteString(rest[:loc[1]])
			sb.WriteString(envContent)
			sb.WriteString(endTag)
		}
		rest = afterBegin[endIdx+len(endTag):]
	}
	return sb.String()
}

// replaceMathDelimiter schützt symmetrische Mathe-Trennzeichen (z.B. $...$, $$...$$).
func replaceMathDelimiter(content, open, close string, placeholders *[]string) string {
	var sb strings.Builder
	rest := content
	for {
		start := strings.Index(rest, open)
		if start < 0 {
			sb.WriteString(rest)
			break
		}
		// Ende hinter dem öffnenden Trennzeichen suchen
		afterOpen := rest[start+len(open):]
		end := strings.Index(afterOpen, close)
		if end < 0 {
			sb.WriteString(rest)
			break
		}
		// Alles vor dem Match übernehmen
		sb.WriteString(rest[:start])
		// Formel als Platzhalter speichern
		formula := open + afterOpen[:end] + close
		idx := len(*placeholders)
		*placeholders = append(*placeholders, formula)
		sb.WriteString(mathPlaceholderTag(idx))
		// Rest nach dem Trennzeichen weiterverarbeiten
		rest = afterOpen[end+len(close):]
	}
	return sb.String()
}

// replaceMathDelimiterPair schützt asymmetrische Mathe-Trennzeichen (z.B. \[...\]).
func replaceMathDelimiterPair(content, open, close string, placeholders *[]string) string {
	var sb strings.Builder
	rest := content
	for {
		start := strings.Index(rest, open)
		if start < 0 {
			sb.WriteString(rest)
			break
		}
		afterOpen := rest[start+len(open):]
		end := strings.Index(afterOpen, close)
		if end < 0 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:start])
		formula := open + afterOpen[:end] + close
		idx := len(*placeholders)
		*placeholders = append(*placeholders, formula)
		sb.WriteString(mathPlaceholderTag(idx))
		rest = afterOpen[end+len(close):]
	}
	return sb.String()
}

// mathPlaceholderTag erzeugt einen eindeutigen Platzhalter für eine Math-Formel.
func mathPlaceholderTag(idx int) string {
	return strings.Repeat("X", 0) + "LATEX_MATH_" + strings.TrimSpace(strings.Repeat("0", 6-len(itoa(idx)))+itoa(idx)) + "_PLACEHOLDER"
}

// itoa wandelt int in String um (einfache Implementierung ohne fmt).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// restoreMath stellt die geschützten Math-Formeln aus den Platzhaltern wieder her.
func restoreMath(content string, placeholders []string) string {
	for i, formula := range placeholders {
		content = strings.ReplaceAll(content, mathPlaceholderTag(i), formula)
	}
	return content
}

// buildLatexTitleBlock erzeugt den HTML-Titelblock für \maketitle.
func buildLatexTitleBlock(title, author, date string) string {
	var sb strings.Builder
	sb.WriteString(`<header class="latex-title-block">`)
	sb.WriteString(`<h1 class="latex-title">`)
	sb.WriteString(html.EscapeString(title))
	sb.WriteString(`</h1>`)
	if author != "" {
		sb.WriteString(`<p class="latex-author">`)
		sb.WriteString(html.EscapeString(author))
		sb.WriteString(`</p>`)
	}
	if date != "" {
		sb.WriteString(`<p class="latex-date">`)
		sb.WriteString(html.EscapeString(date))
		sb.WriteString(`</p>`)
	}
	sb.WriteString(`</header>`)
	return sb.String()
}

// convertLatexEnvironments konvertiert \begin{X}...\end{X} Umgebungen.
// Mathematische Umgebungen wurden bereits durch protectMath behandelt.
// Theorem-Umgebungen aus \newtheorem-Definitionen werden mit ihrem Anzeigenamen
// und einer eigenen CSS-Klasse gerendert.
//
// @param content  LaTeX-Text.
// @param theorems Map envName→Anzeigetext (aus \newtheorem).
// @return HTML mit konvertierten Umgebungen.
// @lastModified 2026-03-11
func convertLatexEnvironments(content string, theorems map[string]string) string {
	return processLatexEnvironments(content, func(envName, envContent string) (string, bool) {
		envName = strings.TrimSpace(envName)
		switch envName {
		case "itemize":
			// Aufzählungsliste (Punkte)
			items := convertItems(envContent, false)
			return `<ul class="latex-itemize">` + items + `</ul>`, true

		case "enumerate":
			// Nummerierte Liste
			items := convertItems(envContent, true)
			return `<ol class="latex-enumerate">` + items + `</ol>`, true

		case "description":
			// Beschreibungsliste
			items := convertDescriptionItems(envContent)
			return `<dl class="latex-description">` + items + `</dl>`, true

		case "verbatim", "lstlisting", "Verbatim":
			// Code-Block – HTML-Escaping, keine weiteren Konvertierungen
			return `<pre class="latex-verbatim"><code>` + html.EscapeString(envContent) + `</code></pre>`, true

		case "quote", "quotation":
			// Blockzitat
			return `<blockquote class="latex-quote">` + envContent + `</blockquote>`, true

		case "abstract":
			// Zusammenfassung
			return `<div class="latex-abstract"><strong>Zusammenfassung</strong>` + envContent + `</div>`, true

		case "center":
			// Zentrierter Text
			return `<div class="latex-center">` + envContent + `</div>`, true

		case "flushleft":
			return `<div style="text-align:left">` + envContent + `</div>`, true

		case "flushright":
			return `<div style="text-align:right">` + envContent + `</div>`, true

		case "figure", "figure*":
			// Abbildung
			return `<figure class="latex-figure">` + envContent + `</figure>`, true

		case "table", "table*":
			// Tabelle
			return `<div class="latex-table-float">` + envContent + `</div>`, true

		case "tabular":
			// Tabelleninhalt → einfache HTML-Tabelle
			return convertTabular(envContent), true

		case "minipage":
			return `<div class="latex-minipage">` + envContent + `</div>`, true

		case "document":
			// document-Umgebung übrig (falls extractLatexBody versagt hat)
			return envContent, true

		default:
			// Prüfen ob es eine bekannte Theorem-Umgebung ist (\newtheorem)
			if displayName, ok := theorems[envName]; ok {
				// Theorem-Box mit Anzeigename-Label rendern
				label := `<span class="latex-theorem-label">` + html.EscapeString(displayName) + `.</span> `
				return `<div class="latex-env-` + html.EscapeString(envName) + `">` + label + envContent + `</div>`, true
			}
			// Proof-Umgebung speziell behandeln
			if envName == "proof" {
				return `<div class="latex-env-proof"><em>Beweis.</em> ` + envContent + ` ∎</div>`, true
			}
			// Unbekannte Umgebungen als generisches div
			return `<div class="latex-env-` + html.EscapeString(envName) + `">` + envContent + `</div>`, true
		}
	})
}

// convertItems wandelt \item-Befehle in HTML-Listenelemente um.
//
// @param content  Inhalt der Listen-Umgebung.
// @param ordered  true für <li>, false für <li> (beide identisch im HTML).
// @return HTML-Listenelemente ohne umschließende <ul>/<ol>.
func convertItems(content string, ordered bool) string {
	_ = ordered
	// \item [optionaler Beschriftungstext] Inhalt
	parts := reItem.Split(content, -1)
	labels := reItem.FindAllStringSubmatch(content, -1)

	var sb strings.Builder
	// Erstes Element ist Text vor dem ersten \item (überspringen, oft leer)
	for i, part := range parts {
		if i == 0 {
			continue // Inhalt vor dem ersten \item ignorieren
		}
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		// Optionales Label aus \item[label]
		label := ""
		if i-1 < len(labels) && labels[i-1][1] != "" {
			label = `<strong>` + html.EscapeString(labels[i-1][1]) + `</strong> `
		}
		sb.WriteString(`<li>`)
		sb.WriteString(label)
		sb.WriteString(text)
		sb.WriteString(`</li>`)
	}
	return sb.String()
}

// convertDescriptionItems wandelt \item[Begriff] in <dt>/<dd>-Paare um.
func convertDescriptionItems(content string) string {
	parts := reItem.Split(content, -1)
	labels := reItem.FindAllStringSubmatch(content, -1)

	var sb strings.Builder
	for i, part := range parts {
		if i == 0 {
			continue
		}
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		if i-1 < len(labels) && labels[i-1][1] != "" {
			sb.WriteString(`<dt>`)
			sb.WriteString(html.EscapeString(labels[i-1][1]))
			sb.WriteString(`</dt>`)
		}
		sb.WriteString(`<dd>`)
		sb.WriteString(text)
		sb.WriteString(`</dd>`)
	}
	return sb.String()
}

// convertTabular wandelt eine einfache LaTeX-Tabelle in HTML um.
// Unterstützt nur einfache Zeilen mit & als Spaltentrennzeichen.
func convertTabular(content string) string {
	// Erste Zeile enthält das Spaltenformat (z.B. {lcc}) → überspringen
	start := 0
	for start < len(content) && content[start] != '\n' {
		start++
	}
	if start < len(content) {
		content = content[start+1:]
	}

	var sb strings.Builder
	sb.WriteString(`<table class="latex-table"><tbody>`)

	lines := strings.Split(content, `\\`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// \hline → Trennlinie, wird ignoriert
		line = strings.ReplaceAll(line, `\hline`, "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Spalten durch & trennen
		cells := strings.Split(line, "&")
		sb.WriteString(`<tr>`)
		for _, cell := range cells {
			sb.WriteString(`<td>`)
			sb.WriteString(strings.TrimSpace(cell))
			sb.WriteString(`</td>`)
		}
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody></table>`)
	return sb.String()
}

// convertLatexSections wandelt Abschnittsbefehle in HTML-Überschriften um.
//
// Mapping:
//   - \section     → <h2>
//   - \subsection  → <h3>
//   - \subsubsection → <h4>
//   - \paragraph   → <h5>
//   - \subparagraph → <h6>
//
// @param content LaTeX-Text.
// @return HTML mit konvertierten Überschriften.
func convertLatexSections(content string) string {
	// Abschnittsbefehle iterativ konvertieren
	result := reSectionCmd.ReplaceAllStringFunc(content, func(match string) string {
		// match enthält z.B. `\section{` oder `\subsection*{`
		return match // Marker beibehalten, Inhalt separat extrahieren
	})

	// Zweiter Durchlauf: Befehl + Inhalt ersetzen
	type sectionDef struct {
		cmd  string
		tag  string
	}
	sections := []sectionDef{
		{`\chapter`, "h2"},
		{`\section`, "h2"},
		{`\subsection`, "h3"},
		{`\subsubsection`, "h4"},
		{`\paragraph`, "h5"},
		{`\subparagraph`, "h6"},
	}
	_ = result
	for _, sec := range sections {
		// Sternvariante zuerst (spezifischer)
		content = replaceLatexCommandHTML(content, sec.cmd+"*", sec.tag, "")
		content = replaceLatexCommandHTML(content, sec.cmd, sec.tag, "")
	}
	return content
}

// replaceLatexCommandHTML ersetzt \cmd{inhalt} durch <tag>inhalt</tag>.
// Optionale CSS-Klasse kann übergeben werden.
//
// @param content LaTeX-Text.
// @param cmd     Befehl inkl. Backslash (z.B. `\section`).
// @param tag     HTML-Tag (z.B. "h2").
// @param class   CSS-Klasse (leer = keine class-Attribute).
// @return Konvertierter Text.
func replaceLatexCommandHTML(content, cmd, tag, class string) string {
	marker := cmd + "{"
	var sb strings.Builder
	rest := content
	for {
		idx := strings.Index(rest, marker)
		if idx < 0 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:idx])
		// Inhalt der {...} Gruppe extrahieren
		innerContent, newPos := extractBraceContent(rest, idx+len(marker)-1)
		// HTML-Element erzeugen
		if class != "" {
			sb.WriteString(`<` + tag + ` class="` + class + `">`)
		} else {
			sb.WriteString(`<` + tag + `>`)
		}
		sb.WriteString(innerContent)
		sb.WriteString(`</` + tag + `>`)
		rest = rest[newPos:]
	}
	return sb.String()
}

// convertLatexLinks wandelt \href{url}{text} und \url{url} in HTML-Links um.
func convertLatexLinks(content string) string {
	// \href{url}{text} → <a href="url">text</a>
	var sb strings.Builder
	rest := content
	for {
		idx := strings.Index(rest, `\href{`)
		if idx < 0 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:idx])
		// URL extrahieren
		urlContent, pos1 := extractBraceContent(rest, idx+5)
		// Linktext extrahieren
		textContent, pos2 := extractBraceContent(rest, pos1-1)
		_ = reHref
		sb.WriteString(`<a href="`)
		sb.WriteString(html.EscapeString(urlContent))
		sb.WriteString(`">`)
		sb.WriteString(textContent)
		sb.WriteString(`</a>`)
		rest = rest[pos2:]
	}
	content = sb.String()

	// \url{url} → <a href="url">url</a>
	content = reUrl.ReplaceAllStringFunc(content, func(match string) string {
		sub := reUrl.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		u := html.EscapeString(sub[1])
		return `<a href="` + u + `">` + u + `</a>`
	})
	return content
}

// convertLatexFormatting wandelt Textformatierungs-Befehle in HTML um.
//
// Unterstützte Befehle:
//   - \textbf{text}   → <strong>text</strong>
//   - \textit{text}   → <em>text</em>
//   - \emph{text}     → <em>text</em>
//   - \underline{text}→ <u>text</u>
//   - \texttt{text}   → <code>text</code>
//   - \textsc{text}   → <span style="font-variant:small-caps">text</span>
//   - \textup{text}   → text (aufrecht, kein spezielles HTML)
//   - \textrm{text}   → text
func convertLatexFormatting(content string) string {
	// Formatierungsbefehle: Befehlsname → HTML-Tag oder Wrapper
	type fmtDef struct {
		cmd     string
		open    string
		close   string
	}
	formats := []fmtDef{
		{`\textbf`, `<strong>`, `</strong>`},
		{`\mathbf`, `<strong>`, `</strong>`},
		{`\textit`, `<em>`, `</em>`},
		{`\mathit`, `<em>`, `</em>`},
		{`\emph`, `<em>`, `</em>`},
		{`\underline`, `<u>`, `</u>`},
		{`\texttt`, `<code>`, `</code>`},
		{`\mathtt`, `<code>`, `</code>`},
		{`\textsc`, `<span style="font-variant:small-caps">`, `</span>`},
		{`\textrm`, ``, ``},
		{`\textup`, ``, ``},
		{`\textnormal`, ``, ``},
		{`\mathrm`, ``, ``},
		{`\text`, ``, ``},
		{`\mbox`, ``, ``},
	}
	for _, f := range formats {
		content = replaceLatexCommandWrap(content, f.cmd, f.open, f.close)
	}
	return content
}

// replaceLatexCommandWrap ersetzt \cmd{inhalt} durch open+inhalt+close.
func replaceLatexCommandWrap(content, cmd, open, close string) string {
	marker := cmd + "{"
	var sb strings.Builder
	rest := content
	for {
		idx := strings.Index(rest, marker)
		if idx < 0 {
			sb.WriteString(rest)
			break
		}
		// Sicherstellen dass es kein längerer Befehlsname ist
		// (z.B. \textbf soll nicht \textbfoo treffen)
		sb.WriteString(rest[:idx])
		inner, newPos := extractBraceContent(rest, idx+len(marker)-1)
		sb.WriteString(open)
		sb.WriteString(inner)
		sb.WriteString(close)
		rest = rest[newPos:]
	}
	return sb.String()
}

// convertLatexMisc wandelt sonstige LaTeX-Befehle um.
func convertLatexMisc(content string) string {
	// \footnote{text} → <sup title="text">[fn]</sup>
	var sb strings.Builder
	rest := content
	for {
		idx := strings.Index(rest, `\footnote{`)
		if idx < 0 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:idx])
		inner, newPos := extractBraceContent(rest, idx+9)
		sb.WriteString(`<sup class="latex-footnote" title="`)
		sb.WriteString(html.EscapeString(inner))
		sb.WriteString(`">[fn]</sup>`)
		rest = rest[newPos:]
	}
	content = sb.String()

	// \label{id} → <span id="id"></span>
	content = reLabel.ReplaceAllStringFunc(content, func(match string) string {
		sub := reLabel.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		return `<span id="` + html.EscapeString(sub[1]) + `"></span>`
	})

	// \ref{id} → <a href="#id">[ref]</a>
	content = reRef.ReplaceAllStringFunc(content, func(match string) string {
		sub := reRef.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		return `<a href="#` + html.EscapeString(sub[1]) + `">[ref]</a>`
	})

	// \cite{key} → <cite>key</cite>
	content = reCite.ReplaceAllStringFunc(content, func(match string) string {
		sub := reCite.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		return `<cite>[` + html.EscapeString(sub[1]) + `]</cite>`
	})

	// \\ und \newline → <br>
	content = reLinebreak.ReplaceAllString(content, `<br>`)

	// \par → Absatzwechsel
	content = strings.ReplaceAll(content, `\par`, "\n\n")

	// \noindent → entfernen
	content = strings.ReplaceAll(content, `\noindent`, "")

	// \clearpage, \newpage → <hr>
	content = strings.ReplaceAll(content, `\clearpage`, `<hr class="latex-pagebreak">`)
	content = strings.ReplaceAll(content, `\newpage`, `<hr class="latex-pagebreak">`)

	// \tableofcontents → Hinweis (TOC wird durch Sidebar gebaut)
	content = strings.ReplaceAll(content, `\tableofcontents`, `<p class="latex-toc-note"><em>[Inhaltsverzeichnis — verwende die TOC-Seitenleiste]</em></p>`)

	// \caption{text} → <figcaption>text</figcaption>
	content = replaceLatexCommandHTML(content, `\caption`, "figcaption", "")

	// Unbekannte Befehle ohne Argument (z.B. \LaTeX, \TeX, \today) vereinfachen
	content = strings.ReplaceAll(content, `\LaTeX`, "LaTeX")
	content = strings.ReplaceAll(content, `\TeX`, "TeX")
	content = strings.ReplaceAll(content, `\BibTeX`, "BibTeX")

	// \~ erzeugt nicht-brechendes Leerzeichen in LaTeX; hier als normales Leerzeichen
	content = strings.ReplaceAll(content, `~`, "\u00A0")

	// Verbleibende unbekannte \befehle{} entfernen (einfache Form ohne Verschachtelung)
	unknownCmd := regexp.MustCompile(`\\[a-zA-Z]+\*?\{([^{}]*)\}`)
	content = unknownCmd.ReplaceAllString(content, "$1")

	// Verbleibende \befehle ohne Argument entfernen
	unknownNoArg := regexp.MustCompile(`\\[a-zA-Z]+\*?\b`)
	content = unknownNoArg.ReplaceAllString(content, "")

	// Geschweifte Klammern die noch übrig sind, entfernen
	content = strings.ReplaceAll(content, "{", "")
	content = strings.ReplaceAll(content, "}", "")

	return content
}

// convertLatexParagraphs wandelt doppelte Leerzeilen in HTML-Absätze um.
//
// @param content Verarbeiteter LaTeX-Text (ohne LaTeX-Befehle).
// @return HTML mit <p>-Tags.
func convertLatexParagraphs(content string) string {
	// Mehrfach-Leerzeilen auf maximal 2 reduzieren
	content = reMultiNewlines.ReplaceAllString(content, "\n\n")

	// Absätze durch doppelte Leerzeile trennen
	paragraphs := strings.Split(content, "\n\n")
	var sb strings.Builder
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		// Block-Level Elemente (<h2>, <ul>, <ol>, <pre>, <blockquote>, <table>, <hr>, <figure>, <header>, <div>)
		// werden nicht nochmals in <p> eingewickelt
		trimmed := strings.TrimSpace(para)
		if strings.HasPrefix(trimmed, "<h") ||
			strings.HasPrefix(trimmed, "<ul") ||
			strings.HasPrefix(trimmed, "<ol") ||
			strings.HasPrefix(trimmed, "<dl") ||
			strings.HasPrefix(trimmed, "<pre") ||
			strings.HasPrefix(trimmed, "<blockquote") ||
			strings.HasPrefix(trimmed, "<table") ||
			strings.HasPrefix(trimmed, "<hr") ||
			strings.HasPrefix(trimmed, "<figure") ||
			strings.HasPrefix(trimmed, "<header") ||
			strings.HasPrefix(trimmed, "<div") ||
			strings.HasPrefix(trimmed, "<p") ||
			strings.HasPrefix(trimmed, "LATEX_MATH_") {
			sb.WriteString(para)
			sb.WriteString("\n")
		} else {
			sb.WriteString(`<p>`)
			sb.WriteString(para)
			sb.WriteString(`</p>`)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
