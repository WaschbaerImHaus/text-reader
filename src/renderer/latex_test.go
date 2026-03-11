// Tests für das LaTeX-Datei-Rendering.
//
// Prüft Parsing von .tex Quelldateien: Abschnitte, Formatierung,
// Listen, Umgebungen, Mathematik und Sonderfälle.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-11
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
	if !strings.Contains(result.HTML, "<br>") {
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
