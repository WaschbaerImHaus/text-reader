// Interne Tests für die Link-Erlaubnisliste des Markdown-Renderers (F2, Build 47).
//
// CVE-2026-5160 (goldmark < 1.7.17): Über Entities wie "javascript&colon;"
// entstanden Links mit href="javascript:…". Unabhängig von der goldmark-
// Version prüft ein AST-Transformer jedes Link- und Bildziel mit derselben
// Erlaubnisliste wie SanitizeHTML (isSafeLinkURL). Interner Test, weil
// isSafeLinkURL nicht exportiert ist.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F2)
package renderer

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// attrValuePattern findet alle href- und src-Attribute samt Wert.
var attrValuePattern = regexp.MustCompile(`\b(href|src)="([^"]*)"`)

// TestParseMarkdown_DangerousLinkSchemesRemoved prüft, dass Markdown keine
// Links/Bilder mit gefährlichen Schemata mehr erzeugt – auch nicht über
// Entities (CVE-2026-5160), Autolinks oder Referenz-Definitionen – und keine
// Schemata außerhalb der Erlaubnisliste (smb:, ms-settings:, ftp:, tel:).
func TestParseMarkdown_DangerousLinkSchemesRemoved(t *testing.T) {
	input := "[a](javascript&colon;alert(1)) [b](&#106;avascript:x) <javascript&colon;x> " +
		"![i](javascript&colon;x) [c](vbscript:x) [d](data:text/html,x)\n\n" +
		"<javascript:alert(2)> [e](JaVaScRiPt:alert(3)) [f][ref] ![g](file:///etc/passwd) " +
		"![h](data:text/html;base64,PHNjcmlwdD4=) [k](file://server/share/x.md)\n\n" +
		// goldmark filtert selbst nur per Sperrliste (javascript/vbscript/file/data):
		// smb: (NTLM-Leck unter Windows), ms-settings:, ftp:, tel: kämen durch
		"[s](smb://server/share/x.md) [t](ms-settings:privacy) [u](ftp://x/y) [v](tel:123) " +
		"![w](smb://server/share/x.png)\n\n" +
		"[ref]: javascript:alert(4)\n"
	res, err := ParseContent(input, "x.md")
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	matches := attrValuePattern.FindAllStringSubmatch(res.HTML, -1)
	for _, m := range matches {
		value := html.UnescapeString(m[2])
		if m[1] == "href" && !isSafeLinkURL(value) {
			t.Errorf("gefährlicher Link: %s=%q", m[1], value)
		}
		if m[1] == "src" && !isSafeMarkdownImageURL(value) {
			t.Errorf("gefährliches Bild: %s=%q", m[1], value)
		}
	}
	// Kein einziger Verweis darf übrig bleiben (alle Ziele sind verboten) …
	if strings.Contains(res.HTML, "<a ") || strings.Contains(res.HTML, "<img") {
		t.Errorf("verbotener Link/verbotenes Bild überlebt: %s", res.HTML)
	}
	// … aber Linktext und Alternativtext bleiben als normaler Text lesbar
	for _, text := range []string{"<p>a b ", " i c d", "javascript:alert(2) e f g h k", "s t u v w"} {
		if !strings.Contains(res.HTML, text) {
			t.Errorf("Text %q fehlt in %q", text, res.HTML)
		}
	}
}

// TestParseMarkdown_SafeLinksKept prüft die Gegenrichtung: erlaubte Ziele
// (http, https, mailto, Anker, relative und absolute lokale Bildpfade,
// eingebettete Bilder) bleiben unverändert erhalten.
func TestParseMarkdown_SafeLinksKept(t *testing.T) {
	input := "[w](https://example.com/a?b=1) [h](http://x.org) [m](mailto:a@b.c) " +
		"[anker](#kapitel) [rel](kapitel2.md) <https://auto.example> www.linkify.example\n\n" +
		"![lokal](bild.png) ![abs](/home/u/bild.png) ![win](C:\\Bilder\\x.png) " +
		"![daten](data:image/png;base64,iVBORw0KGgo=)\n"
	res, err := ParseContent(input, "x.md")
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	for _, want := range []string{
		`href="https://example.com/a?b=1"`, `href="http://x.org"`, `href="mailto:a@b.c"`,
		`href="#kapitel"`, `href="kapitel2.md"`, `href="https://auto.example"`,
		`href="http://www.linkify.example"`,
		`src="bild.png"`, `src="/home/u/bild.png"`, `src="C:%5CBilder%5Cx.png"`,
		`src="data:image/png;base64,iVBORw0KGgo="`,
	} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("erwartet %s in %s", want, res.HTML)
		}
	}
}

// TestIsSafeMarkdownImageURL prüft die Erlaubnisliste für Bildziele:
// relativ, absoluter lokaler Pfad (Unix/Windows-Laufwerk) oder data:image/.
func TestIsSafeMarkdownImageURL(t *testing.T) {
	cases := map[string]bool{
		"bild.png": true, "img/b.png": true, "/abs/b.png": true, `C:\x.png`: true, "d:/x.png": true,
		"data:image/png;base64,AA": true, "#x": true,
		"javascript:x": false, "data:text/html,x": false, "file:///etc/passwd": false,
		"vbscript:x": false, "java\tscript:x": false, "cc:/x.png": false, "C:x.png": false,
		"https://example.com/x.png": false,
	}
	for in, want := range cases {
		if got := isSafeMarkdownImageURL(in); got != want {
			t.Errorf("isSafeMarkdownImageURL(%q) = %v, erwartet %v", in, got, want)
		}
	}
}
