// Tests für das LaTeX-Datei-Rendering.
//
// Prüft Parsing von .tex Quelldateien: Abschnitte, Formatierung,
// Listen, Umgebungen, Mathematik und Sonderfälle.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14 (RISK-007: Bereinigung, Makros als JSON-Datenblock)
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// TestIsLaTeXFile prüft die Erkennung von LaTeX-Dateien.
func TestIsLaTeXFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"document.tex", true},
		{"paper.TEX", true},
		{"thesis.tex", true},
		{"/pfad/zur/datei.tex", true},
		{"doc.md", false},
		{"text.txt", false},
		{"book.epub", false},
		{"style.css", false},
		{"data.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsLaTeXFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsLaTeXFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestParseLaTeXTitleExtraction prüft Extraktion des Titels aus \title{}.
func TestParseLaTeXTitleExtraction(t *testing.T) {
	content := `\documentclass{article}
\title{Mein LaTeX Dokument}
\author{Max Mustermann}
\begin{document}
\maketitle
Hallo Welt.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if result.Title != "Mein LaTeX Dokument" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "Mein LaTeX Dokument")
	}
}

// TestParseLaTeXTitleFallback prüft Fallback auf Dateinamen wenn kein Titel.
func TestParseLaTeXTitleFallback(t *testing.T) {
	content := `\begin{document}
Einfacher Inhalt ohne Titel.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "mein-dokument.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if result.Title != "mein-dokument" {
		t.Errorf("Title = %q, erwartet %q", result.Title, "mein-dokument")
	}
}

// TestParseLaTeXSections prüft Konvertierung von Abschnittsbefehlen.
func TestParseLaTeXSections(t *testing.T) {
	content := `\begin{document}
\section{Einleitung}
Text der Einleitung.

\subsection{Hintergrund}
Hintergrundtext.

\subsubsection{Details}
Detailtext.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<h2>") {
		t.Error("HTML enthält kein <h2> für \\section")
	}
	if !strings.Contains(result.HTML, "Einleitung") {
		t.Error("HTML enthält nicht den Abschnittstitel 'Einleitung'")
	}
	if !strings.Contains(result.HTML, "<h3>") {
		t.Error("HTML enthält kein <h3> für \\subsection")
	}
	if !strings.Contains(result.HTML, "<h4>") {
		t.Error("HTML enthält kein <h4> für \\subsubsection")
	}
}

// TestParseLaTeXFormatting prüft Textformatierungs-Befehle.
func TestParseLaTeXFormatting(t *testing.T) {
	content := `\begin{document}
Normaler Text mit \textbf{fettem Text} und \textit{kursivem Text}.
Auch \emph{betonter Text} und \underline{unterstrichener Text}.
\texttt{Monospace-Text} für Code.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<strong>") {
		t.Error("HTML enthält kein <strong> für \\textbf")
	}
	if !strings.Contains(result.HTML, "fettem Text") {
		t.Error("HTML enthält nicht 'fettem Text'")
	}
	if !strings.Contains(result.HTML, "<em>") {
		t.Error("HTML enthält kein <em> für \\textit oder \\emph")
	}
	if !strings.Contains(result.HTML, "<u>") {
		t.Error("HTML enthält kein <u> für \\underline")
	}
	if !strings.Contains(result.HTML, "<code>") {
		t.Error("HTML enthält kein <code> für \\texttt")
	}
}

// TestParseLaTeXItemize prüft Konvertierung von itemize-Listen.
func TestParseLaTeXItemize(t *testing.T) {
	content := `\begin{document}
\begin{itemize}
  \item Erster Punkt
  \item Zweiter Punkt
  \item Dritter Punkt
\end{itemize}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<ul") {
		t.Error("HTML enthält keine <ul> für itemize")
	}
	if !strings.Contains(result.HTML, "<li>") {
		t.Error("HTML enthält keine <li> Elemente")
	}
	if !strings.Contains(result.HTML, "Erster Punkt") {
		t.Error("HTML enthält nicht 'Erster Punkt'")
	}
	if !strings.Contains(result.HTML, "Zweiter Punkt") {
		t.Error("HTML enthält nicht 'Zweiter Punkt'")
	}
}

// TestParseLaTeXEnumerate prüft Konvertierung von enumerate-Listen.
func TestParseLaTeXEnumerate(t *testing.T) {
	content := `\begin{document}
\begin{enumerate}
  \item Erstens
  \item Zweitens
\end{enumerate}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<ol") {
		t.Error("HTML enthält keine <ol> für enumerate")
	}
	if !strings.Contains(result.HTML, "Erstens") {
		t.Error("HTML enthält nicht 'Erstens'")
	}
}

// TestParseLaTeXVerbatim prüft Konvertierung von verbatim-Umgebungen.
func TestParseLaTeXVerbatim(t *testing.T) {
	content := `\begin{document}
\begin{verbatim}
def hello():
    print("Hallo Welt")
\end{verbatim}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<pre") {
		t.Error("HTML enthält kein <pre> für verbatim")
	}
	if !strings.Contains(result.HTML, "<code>") {
		t.Error("HTML enthält kein <code> für verbatim")
	}
	if !strings.Contains(result.HTML, "def hello") {
		t.Error("HTML enthält nicht den verbatim-Inhalt")
	}
}

// TestParseLaTeXMathProtection prüft dass Matheformeln unverändert bleiben.
func TestParseLaTeXMathProtection(t *testing.T) {
	content := `\begin{document}
Inline-Formel: $E = mc^2$ im Text.

Block-Formel:
$$\int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}$$

Display-Formel mit Klammern:
\[a^2 + b^2 = c^2\]
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	// Inline-Formel muss vollständig erhalten sein
	if !strings.Contains(result.HTML, "$E = mc^2$") {
		t.Error("Inline-Matheformel $E = mc^2$ wurde verändert oder entfernt")
	}
	// Block-Formel muss vollständig erhalten sein
	if !strings.Contains(result.HTML, "\\int_0^\\infty") {
		t.Error("Block-Matheformel wurde verändert oder entfernt")
	}
	// \[...\] muss erhalten sein
	if !strings.Contains(result.HTML, "a^2 + b^2 = c^2") {
		t.Error("\\[...\\] Formel wurde verändert oder entfernt")
	}
}

// TestParseLaTeXQuote prüft Konvertierung von quote-Umgebungen.
func TestParseLaTeXQuote(t *testing.T) {
	content := `\begin{document}
Text vor dem Zitat.
\begin{quote}
Dies ist ein Blockzitat.
\end{quote}
Text nach dem Zitat.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "<blockquote") {
		t.Error("HTML enthält kein <blockquote> für quote-Umgebung")
	}
	if !strings.Contains(result.HTML, "Blockzitat") {
		t.Error("HTML enthält nicht den Zitattext")
	}
}

// TestParseLaTeXCommentRemoval prüft Entfernung von LaTeX-Kommentaren.
func TestParseLaTeXCommentRemoval(t *testing.T) {
	content := `\begin{document}
Sichtbarer Text. % Dies ist ein Kommentar
Zweite Zeile. % Noch ein Kommentar
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if strings.Contains(result.HTML, "Dies ist ein Kommentar") {
		t.Error("HTML enthält Kommentar-Text der entfernt werden sollte")
	}
	if !strings.Contains(result.HTML, "Sichtbarer Text") {
		t.Error("HTML enthält nicht den sichtbaren Text")
	}
}

// TestParseLaTeXHref prüft Konvertierung von \href.
func TestParseLaTeXHref(t *testing.T) {
	content := `\begin{document}
Besuche \href{https://example.com}{diese Webseite}.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, `<a href="https://example.com">`) {
		t.Error("HTML enthält keinen Link mit korrekter URL")
	}
	if !strings.Contains(result.HTML, "diese Webseite") {
		t.Error("HTML enthält nicht den Linktext")
	}
}

// TestParseLaTeXUrl prüft Konvertierung von \url.
func TestParseLaTeXUrl(t *testing.T) {
	content := `\begin{document}
Webseite: \url{https://example.org}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "example.org") {
		t.Error("HTML enthält nicht die URL")
	}
	if !strings.Contains(result.HTML, "<a href=") {
		t.Error("HTML enthält keinen Link-Tag")
	}
}

// TestParseLaTeXLinebreak prüft Konvertierung von Zeilenumbrüchen.
func TestParseLaTeXLinebreak(t *testing.T) {
	content := `\begin{document}
Erste Zeile.\\
Zweite Zeile.\newline
Dritte Zeile.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	// \\ und \newline sollten in <br> umgewandelt werden
	// (seit Build 41 vom Bereiniger als <br/> ausgegeben – gleichwertig)
	if !strings.Contains(result.HTML, "<br") {
		t.Error("HTML enthält kein <br> für \\\\ oder \\newline")
	}
}

// TestParseLaTeXNoPreamble prüft Parsing ohne \begin{document}.
func TestParseLaTeXNoPreamble(t *testing.T) {
	// Einfaches .tex ohne document-Umgebung (z.B. Fragment)
	content := `\section{Einleitung}
Dies ist ein LaTeX-Fragment ohne Präambel.

\section{Hauptteil}
Mehr Inhalt hier.`

	result, err := renderer.ParseLaTeXContent(content, "fragment.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "Einleitung") {
		t.Error("HTML enthält nicht 'Einleitung'")
	}
	if !strings.Contains(result.HTML, "Hauptteil") {
		t.Error("HTML enthält nicht 'Hauptteil'")
	}
}

// TestParseLaTeXArticleWrapper prüft dass HTML in article-Tag eingebettet ist.
func TestParseLaTeXArticleWrapper(t *testing.T) {
	content := `\begin{document}
Inhalt.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, `class="latex-document"`) {
		t.Error("HTML ist nicht in latex-document Wrapper eingebettet")
	}
}

// TestParseLaTeXSectionStarVariant prüft Abschnitte mit Sternvariante (\section*).
func TestParseLaTeXSectionStarVariant(t *testing.T) {
	content := `\begin{document}
\section*{Nicht nummerierter Abschnitt}
Text.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "Nicht nummerierter Abschnitt") {
		t.Error("HTML enthält nicht den Abschnittstitel der Sternvariante")
	}
}

// TestParseLaTeXTheoremEnv prüft dass \newtheorem-Umgebungen mit Anzeigename gerendert werden.
func TestParseLaTeXTheoremEnv(t *testing.T) {
	content := `\documentclass{article}
\newtheorem{theorem}{Satz}
\newtheorem{lemma}[theorem]{Lemma}
\begin{document}
\begin{theorem}
Dies ist ein Satz.
\end{theorem}

\begin{lemma}
Dies ist ein Lemma.
\end{lemma}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	// Theorem-Label muss im HTML erscheinen
	if !strings.Contains(result.HTML, "Satz") {
		t.Error("HTML enthält nicht den Theorem-Anzeigenamen 'Satz'")
	}
	if !strings.Contains(result.HTML, "latex-theorem-label") {
		t.Error("HTML enthält kein latex-theorem-label Span")
	}
	if !strings.Contains(result.HTML, "Lemma") {
		t.Error("HTML enthält nicht den Lemma-Anzeigenamen")
	}
	// Umgebungsklasse muss gesetzt sein
	if !strings.Contains(result.HTML, "latex-env-theorem") {
		t.Error("HTML enthält keine CSS-Klasse latex-env-theorem")
	}
}

// TestParseLaTeXProofEnv prüft dass proof-Umgebungen korrekt gerendert werden.
func TestParseLaTeXProofEnv(t *testing.T) {
	content := `\begin{document}
\begin{proof}
Dies ist ein Beweis.
\end{proof}
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if !strings.Contains(result.HTML, "Beweis") {
		t.Error("HTML enthält nicht das Beweis-Label")
	}
	if !strings.Contains(result.HTML, "∎") {
		t.Error("HTML enthält kein QED-Symbol ∎")
	}
}

// TestParseLaTeXNewcommandMacros prüft dass Custom-Makros als nicht ausführbarer
// JSON-Datenblock (<script type="application/json" id="latex-macros">) eingebettet
// werden. Ein ausführbares Inline-Skript würde von der strikten CSP blockiert.
func TestParseLaTeXNewcommandMacros(t *testing.T) {
	content := `\documentclass{article}
\newcommand{\N}{\mathbb{N}}
\newcommand{\Z}{\mathbb{Z}}
\DeclareMathOperator{\ord}{ord}
\begin{document}
Es gilt $n \in \N$.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	// Makros müssen als JSON-Datenblock im HTML erscheinen
	if !strings.Contains(result.HTML, `<script type="application/json" id="latex-macros">`) {
		t.Error("HTML enthält keinen JSON-Datenblock für Custom-Makros")
	}
	if strings.Contains(result.HTML, `window.__latexMacros`) {
		t.Error("HTML enthält noch das ausführbare window.__latexMacros-Skript")
	}
	if !strings.Contains(result.HTML, `\\N`) {
		t.Error("HTML enthält nicht das Makro \\N")
	}
	if !strings.Contains(result.HTML, `\\Z`) {
		t.Error("HTML enthält nicht das Makro \\Z")
	}
	if !strings.Contains(result.HTML, `\\ord`) {
		t.Error("HTML enthält nicht das DeclareMathOperator \\ord")
	}
}

// TestParseLaTeXNoMacrosScript prüft dass bei fehlendem \newcommand kein leerer script-Tag erscheint.
func TestParseLaTeXNoMacrosScript(t *testing.T) {
	content := `\begin{document}
Einfacher Text ohne Makros.
\end{document}`

	result, err := renderer.ParseLaTeXContent(content, "test.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	// Ohne Custom-Makros kein Makro-Datenblock
	if strings.Contains(result.HTML, `latex-macros`) || strings.Contains(result.HTML, `<script`) {
		t.Error("HTML enthält einen Makro-Block obwohl keine Makros definiert wurden")
	}
}

// TestIsSupportedFileWithTex prüft dass .tex als unterstützt erkannt wird.
func TestIsSupportedFileWithTex(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"document.tex", true},
		{"paper.TEX", true},
		{"README.md", true},
		{"book.epub", true},
		{"style.css", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := renderer.IsSupportedFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsSupportedFile(%q) = %v, erwartet %v", tt.path, got, tt.expected)
			}
		})
	}
}

// latexSecurityHTML rendert ein LaTeX-Dokument und gibt das HTML zurück.
func latexSecurityHTML(t *testing.T, body string) string {
	t.Helper()
	result, err := renderer.ParseLaTeXContent("\\documentclass{article}\n\\begin{document}\n"+body+"\n\\end{document}", "x.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	return result.HTML
}

// TestParseLaTeXRawHTMLSanitized prüft RISK-007: HTML im LaTeX-Fließtext, in
// Abschnitten und Formatierungsbefehlen darf keine Event-Handler einschleusen.
func TestParseLaTeXRawHTMLSanitized(t *testing.T) {
	got := latexSecurityHTML(t, `Text <img src=x onerror=alert(1)> weiter.

\section{<b onmouseover=alert(2)>Abschnitt</b>}
\textbf{<i onclick=alert(3)>fett</i>}
<iframe srcdoc="x"></iframe><script>alert(4)</script>`)
	lower := strings.ToLower(got)
	// Ein leeres <img/> ohne Attribute ist harmlos – entscheidend ist, dass
	// keine Handler und kein Skript übrig bleiben.
	for _, bad := range []string{"onerror", "onmouseover", "onclick", "iframe", "alert(", "src="} {
		if strings.Contains(lower, bad) {
			t.Errorf("Gefährliches Muster %q im LaTeX-HTML:\n%s", bad, got)
		}
	}
	for _, want := range []string{"Text", "weiter.", "Abschnitt", "fett"} {
		if !strings.Contains(got, want) {
			t.Errorf("Erwarteter Text %q fehlt:\n%s", want, got)
		}
	}
}

// TestParseLaTeXJavascriptLinksRemoved prüft, dass \href und \url keine
// javascript:-Links erzeugen, normale Links aber erhalten bleiben.
func TestParseLaTeXJavascriptLinksRemoved(t *testing.T) {
	got := latexSecurityHTML(t, `\href{javascript:alert(1)}{Klick} \url{javascript:alert(2)} \href{JaVa&#115;cript:alert(3)}{K3} \href{https://example.org}{Beispiel}`)
	// K3: LaTeX entschlüsselt keine HTML-Entities – der Browser sieht wörtlich
	// "JaVa&#115;cript:…", das ist kein gültiges Schema (& ist unzulässig)
	// und wird zum harmlosen relativen Pfad.
	if strings.Contains(strings.ToLower(got), `href="javascript`) {
		t.Errorf("javascript:-Link im LaTeX-HTML:\n%s", got)
	}
	if !strings.Contains(got, `<a href="https://example.org">Beispiel</a>`) {
		t.Errorf("Normaler Link fehlt:\n%s", got)
	}
	if !strings.Contains(got, "Klick") {
		t.Errorf("Linktext fehlt:\n%s", got)
	}
}

// TestParseLaTeXMathEscaped prüft, dass Formeln HTML-escaped ausgegeben werden:
// "$a<b$" darf kein Tag öffnen, und HTML in Formeln darf nichts einschleusen.
// KaTeX liest den Textinhalt, sieht also weiterhin "a<b".
func TestParseLaTeXMathEscaped(t *testing.T) {
	got := latexSecurityHTML(t, `Es gilt $a<b$ und $x<img src=y onerror=alert(1)>$.`)
	if !strings.Contains(got, `$a&lt;b$`) {
		t.Errorf("Formel a<b nicht escaped:\n%s", got)
	}
	if strings.Contains(strings.ToLower(got), "<img") {
		t.Errorf("HTML aus Formel eingeschleust:\n%s", got)
	}
}

// TestParseLaTeXMacrosJSONEscaped prüft, dass Makro-Definitionen den JSON-Block
// nicht mit "</script>" verlassen können.
func TestParseLaTeXMacrosJSONEscaped(t *testing.T) {
	content := "\\documentclass{article}\n\\newcommand{\\X}{</script><img src=x onerror=alert(1)>}\n\\begin{document}\n$\\X$\n\\end{document}"
	result, err := renderer.ParseLaTeXContent(content, "x.tex")
	if err != nil {
		t.Fatalf("ParseLaTeXContent() Fehler: %v", err)
	}
	if strings.Count(result.HTML, "</script>") != 1 {
		t.Errorf("Makro-Block wurde vorzeitig beendet:\n%s", result.HTML)
	}
	if strings.Contains(strings.ToLower(result.HTML), "<img") {
		t.Errorf("HTML aus Makro eingeschleust:\n%s", result.HTML)
	}
}
