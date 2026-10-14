// Tests für den Whitelist-HTML-Bereiniger (RISK-006/007).
//
// Prüft, dass fremdes HTML aus EPUB- und LaTeX-Dateien keine Skripte,
// Event-Handler, gefährlichen URLs oder Einbettungen mehr enthält, während
// normale Formatierung, Anker, eingebettete Bilder und SVG-Cover erhalten
// bleiben.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14
package renderer_test

import (
	"strings"
	"testing"

	"md-reader/renderer"
)

// mustNotContain prüft (case-insensitiv), dass keines der Muster vorkommt.
func mustNotContain(t *testing.T, got string, patterns ...string) {
	t.Helper()
	lower := strings.ToLower(got)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			t.Errorf("Ausgabe enthält verbotenes Muster %q:\n%s", p, got)
		}
	}
}

// mustContain prüft, dass alle Muster exakt vorkommen.
func mustContain(t *testing.T, got string, patterns ...string) {
	t.Helper()
	for _, p := range patterns {
		if !strings.Contains(got, p) {
			t.Errorf("Ausgabe enthält erwartetes Muster %q nicht:\n%s", p, got)
		}
	}
}

// TestSanitizeHTML_KeepsFormatting prüft, dass harmlose Formatierung erhalten bleibt.
func TestSanitizeHTML_KeepsFormatting(t *testing.T) {
	in := `<h2 id="k1" class="chapter">Titel</h2><p style="text-align:center">Ein <strong>fetter</strong>, <em>kursiver</em> und <code>Code</code>-Text.</p><ul><li>eins</li></ul><table><tr><td colspan="2">Zelle</td></tr></table><hr/><br/>`
	got := renderer.SanitizeHTML(in)
	mustContain(t, got,
		`<h2 id="k1" class="chapter">Titel</h2>`,
		`<p style="text-align:center">`,
		`<strong>fetter</strong>`, `<em>kursiver</em>`, `<code>Code</code>`,
		`<ul><li>eins</li></ul>`, `<td colspan="2">Zelle</td>`, `<hr/>`, `<br/>`)
}

// TestSanitizeHTML_RemovesScriptWithContent prüft, dass <script> samt Inhalt verschwindet.
func TestSanitizeHTML_RemovesScriptWithContent(t *testing.T) {
	got := renderer.SanitizeHTML(`<p>a</p><script>alert(1)</script><SCRIPT type="text/javascript">alert(2)</SCRIPT><p>b</p>`)
	mustNotContain(t, got, "script", "alert")
	mustContain(t, got, "<p>a</p>", "<p>b</p>")
}

// TestSanitizeHTML_RemovesEventHandlers prüft, dass alle on*-Attribute entfernt werden.
func TestSanitizeHTML_RemovesEventHandlers(t *testing.T) {
	cases := []string{
		`<img src="data:image/png;base64,AAAA" onerror="alert(1)">`,
		`<svg onload="alert(1)"></svg>`,
		`<p onclick='alert(1)'>x</p>`,
		`<div ONMOUSEOVER=alert(1)>x</div>`,
		`<body onload="alert(1)">x</body>`,
		`<a href="#a" onfocus="alert(1)" autofocus>x</a>`,
	}
	for _, c := range cases {
		got := renderer.SanitizeHTML(c)
		mustNotContain(t, got, "alert", " on")
	}
}

// TestSanitizeHTML_DangerousLinkSchemes prüft, dass javascript:/vbscript:/data:-Links
// entfernt werden – auch verschleiert durch Entities, Groß-/Kleinschreibung,
// Tabulatoren, Zeilenumbrüche und Steuerzeichen.
func TestSanitizeHTML_DangerousLinkSchemes(t *testing.T) {
	cases := []string{
		`<a href="javascript:alert(1)">x</a>`,
		`<a href="JaVaScRiPt:alert(1)">x</a>`,
		`<a href="  javascript:alert(1)">x</a>`,
		`<a href="java&#x09;script:alert(1)">x</a>`,
		`<a href="java&#10;script:alert(1)">x</a>`,
		`<a href="&#106;avascript:alert(1)">x</a>`,
		`<a href="&#x01;javascript:alert(1)">x</a>`,
		`<a href="vbscript:msgbox(1)">x</a>`,
		`<a href="data:text/html,<script>alert(1)</script>">x</a>`,
		`<a href="file:///etc/passwd">x</a>`,
	}
	for _, c := range cases {
		got := renderer.SanitizeHTML(c)
		mustNotContain(t, got, "href", "alert", "msgbox", "passwd")
		mustContain(t, got, ">x</a>")
	}
}

// TestSanitizeHTML_SafeLinksKept prüft, dass erlaubte Links unverändert bleiben.
func TestSanitizeHTML_SafeLinksKept(t *testing.T) {
	got := renderer.SanitizeHTML(`<a href="#kap2">a</a><a href="https://example.org/?a=1&amp;b=2">b</a><a href="http://example.org">c</a><a href="mailto:x@example.org">d</a><a href="kapitel3.xhtml">e</a>`)
	mustContain(t, got,
		`<a href="#kap2">a</a>`,
		`<a href="https://example.org/?a=1&amp;b=2">b</a>`,
		`<a href="http://example.org">c</a>`,
		`<a href="mailto:x@example.org">d</a>`,
		`<a href="kapitel3.xhtml">e</a>`)
}

// TestSanitizeHTML_RemovesEmbeddingElements prüft, dass Einbettungen samt Inhalt
// entfernt werden (iframe srcdoc, object, embed, meta refresh, base, form-Ziele).
func TestSanitizeHTML_RemovesEmbeddingElements(t *testing.T) {
	in := `<iframe srcdoc="&lt;script&gt;alert(1)&lt;/script&gt;">innen</iframe>` +
		`<object data="data:text/html,x">obj</object><embed src="x.swf">` +
		`<meta http-equiv="refresh" content="0;url=https://evil.example">` +
		`<base href="https://evil.example/"><link rel="stylesheet" href="https://evil.example/x.css">` +
		`<style>body{background:url(https://evil.example)}</style>` +
		`<noscript><p>ns</p></noscript><template><p>tpl</p></template>` +
		`<p>bleibt</p>`
	got := renderer.SanitizeHTML(in)
	mustNotContain(t, got, "iframe", "srcdoc", "object", "embed", "meta", "refresh",
		"base", "link", "style", "evil", "innen", "obj", "noscript", "ns<", "tpl")
	mustContain(t, got, "<p>bleibt</p>")
}

// TestSanitizeHTML_FormsNeutralised prüft, dass Formular-Elemente verschwinden,
// ihr Text aber erhalten bleibt.
func TestSanitizeHTML_FormsNeutralised(t *testing.T) {
	got := renderer.SanitizeHTML(`<form action="https://evil.example"><button formaction="https://evil.example">Senden</button><input value="x"></form>`)
	mustNotContain(t, got, "form", "button", "input", "evil")
	mustContain(t, got, "Senden")
}

// TestSanitizeHTML_UnknownTagsKeepText prüft, dass unbekannte Tags entfernt,
// ihr Textinhalt aber behalten wird.
func TestSanitizeHTML_UnknownTagsKeepText(t *testing.T) {
	got := renderer.SanitizeHTML(`<custom-el data-x="1">Hallo <blink>Welt</blink></custom-el>`)
	mustNotContain(t, got, "custom-el", "blink", "data-x")
	mustContain(t, got, "Hallo Welt")
}

// TestSanitizeHTML_TextEscaped prüft, dass Text korrekt escaped wieder ausgegeben
// wird und Entities aus der Quelle keine Tags erzeugen.
func TestSanitizeHTML_TextEscaped(t *testing.T) {
	got := renderer.SanitizeHTML(`<p>a &lt;img src=x onerror=alert(1)&gt; &amp; b &quot;c&quot; ä</p>`)
	mustContain(t, got, `<p>a &lt;img src=x onerror=alert(1)&gt; &amp; b &#34;c&#34; ä</p>`)
}

// TestSanitizeHTML_AttributeValuesEscaped prüft, dass Attributwerte nicht aus
// den Anführungszeichen ausbrechen können.
func TestSanitizeHTML_AttributeValuesEscaped(t *testing.T) {
	got := renderer.SanitizeHTML(`<span id='a"><img src=x onerror=alert(1)>'>x</span>`)
	mustNotContain(t, got, "<img")
	mustContain(t, got, `<span id="a&#34;&gt;&lt;img src=x onerror=alert(1)&gt;">x</span>`)
}

// TestSanitizeHTML_CommentsRemoved prüft, dass HTML-Kommentare (auch bedingte
// IE-Kommentare) entfernt werden.
func TestSanitizeHTML_CommentsRemoved(t *testing.T) {
	got := renderer.SanitizeHTML(`<p>a</p><!-- geheim --><!--[if IE]><script>alert(1)</script><![endif]--><p>b</p>`)
	mustNotContain(t, got, "geheim", "<!--", "alert")
	mustContain(t, got, "<p>a</p><p>b</p>")
}

// TestSanitizeHTML_ImagesOnlyDataURIs prüft, dass Bilder nur als data:image-URIs
// erlaubt sind (EPUB bettet alle Bilder ein) und srcset entfernt wird.
func TestSanitizeHTML_ImagesOnlyDataURIs(t *testing.T) {
	got := renderer.SanitizeHTML(`<img src="data:image/png;base64,iVBORw0KGgo=" alt="ok" width="10">` +
		`<img src="https://tracker.example/pixel.gif" alt="t1">` +
		`<img src="data:text/html;base64,PHNjcmlwdD4=" alt="t2">` +
		`<img src="javascript:alert(1)" srcset="https://tracker.example/a.png 2x" alt="t3">`)
	mustContain(t, got, `<img src="data:image/png;base64,iVBORw0KGgo=" alt="ok" width="10"/>`)
	mustNotContain(t, got, "tracker", "text/html", "javascript", "srcset")
	mustContain(t, got, `alt="t1"`, `alt="t2"`, `alt="t3"`)
}

// TestSanitizeHTML_SVGCoverKept prüft, dass Calibre-SVG-Cover mit eingebettetem
// Bild erhalten bleiben (Bug #012), gefährliche SVG-Teile aber nicht.
func TestSanitizeHTML_SVGCoverKept(t *testing.T) {
	in := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="100%" height="100%" viewBox="0 0 600 800" preserveAspectRatio="none">` +
		`<image width="600" height="800" xlink:href="data:image/jpeg;base64,/9j/4AAQ"/></svg>`
	got := renderer.SanitizeHTML(in)
	mustContain(t, got, `<svg`, `viewbox="0 0 600 800"`, `<image width="600" height="800" xlink:href="data:image/jpeg;base64,/9j/4AAQ"/>`, `</svg>`)

	bad := renderer.SanitizeHTML(`<svg><a xlink:href="javascript:alert(1)"><text>x</text></a><use href="https://evil.example/x.svg#a"/><foreignObject><p>fo</p></foreignObject><animate attributeName="href" to="javascript:alert(1)"/><set attributeName="onmouseover" to="alert(1)"/><image href="javascript:alert(1)"/></svg>`)
	mustNotContain(t, bad, "javascript", "alert", "evil", "use", "foreignobject", "animate", "<set", "fo<")
}

// TestSanitizeHTML_MathMLKept prüft, dass einfache MathML-Formeln erhalten bleiben,
// ohne dass href-Attribute durchkommen.
func TestSanitizeHTML_MathMLKept(t *testing.T) {
	got := renderer.SanitizeHTML(`<math display="block"><mrow href="javascript:alert(1)"><mi>x</mi><mo>=</mo><mn>2</mn></mrow></math>`)
	mustContain(t, got, `<math display="block">`, `<mi>x</mi><mo>=</mo><mn>2</mn>`)
	mustNotContain(t, got, "href", "javascript")
}

// TestSanitizeHTML_UnclosedDropElement prüft den Sonderfall eines nicht
// geschlossenen <script>: Der Rest wird verworfen, aber es entsteht kein Skript.
func TestSanitizeHTML_UnclosedDropElement(t *testing.T) {
	got := renderer.SanitizeHTML(`<p>vorher</p><script>alert(1)`)
	mustNotContain(t, got, "script", "alert")
	mustContain(t, got, "<p>vorher</p>")
}

// TestSanitizeHTML_NestedDropElements prüft verschachtelte zu verwerfende Elemente.
func TestSanitizeHTML_NestedDropElements(t *testing.T) {
	got := renderer.SanitizeHTML(`<object><object>innen</object>noch innen</object><p>danach</p>`)
	mustNotContain(t, got, "innen", "object")
	mustContain(t, got, "<p>danach</p>")
}

// TestSanitizeHTML_EmptyInput prüft die leere Eingabe.
func TestSanitizeHTML_EmptyInput(t *testing.T) {
	if got := renderer.SanitizeHTML(""); got != "" {
		t.Errorf("leere Eingabe ergab %q", got)
	}
}
