// Whitelist-Bereiniger für fremdes HTML (EPUB-Kapitel, LaTeX-Ausgabe).
//
// Hintergrund (RISK-006/007): EPUB- und LaTeX-Dateien können beliebiges HTML
// enthalten bzw. erzeugen. Ohne Bereinigung laufen Event-Handler wie
// onerror="…" oder javascript:-Links im Viewer und haben Zugriff auf die
// Go-Bindings. Der Bereiniger arbeitet nach dem Whitelist-Prinzip: Nur
// bekannte, harmlose Elemente und Attribute werden übernommen, alles andere
// fällt weg. Text und Attributwerte werden neu escaped ausgegeben, damit
// keine Zeichenfolge aus der Quelle die Struktur verändern kann.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-14 (style-Filter gegen Ressourcen-Nachladen)
package renderer

import (
	"html"
	"strings"

	nethtml "golang.org/x/net/html"
)

// allowedElements enthält alle Elemente, die (ohne gefährliche Attribute)
// übernommen werden: HTML-Textauszeichnung, Tabellen, Listen, einfache SVG-
// Grafik (für Calibre-Cover) und MathML-Präsentationselemente.
var allowedElements = toSet(
	// HTML: Struktur und Text
	"a", "abbr", "acronym", "address", "article", "aside", "b", "bdi", "bdo",
	"big", "blockquote", "br", "caption", "center", "cite", "code", "col",
	"colgroup", "dd", "del", "details", "dfn", "div", "dl", "dt", "em",
	"figcaption", "figure", "footer", "h1", "h2", "h3", "h4", "h5", "h6",
	"header", "hgroup", "hr", "i", "img", "ins", "kbd", "li", "main", "mark",
	"nav", "ol", "p", "pre", "q", "rp", "rt", "ruby", "s", "samp", "section",
	"small", "span", "strike", "strong", "sub", "summary", "sup", "table",
	"tbody", "td", "tfoot", "th", "thead", "time", "tr", "tt", "u", "ul",
	"var", "wbr",
	// SVG: nur statische Formen und eingebettete Bilder
	"svg", "g", "image", "rect", "circle", "ellipse", "line", "polyline",
	"polygon", "path", "text", "tspan",
	// MathML: Präsentationselemente
	"math", "mrow", "mi", "mn", "mo", "ms", "mtext", "mspace", "msup", "msub",
	"msubsup", "mfrac", "msqrt", "mroot", "mtable", "mtr", "mtd", "mover",
	"munder", "munderover", "mstyle", "mpadded", "mphantom", "menclose",
	"semantics", "annotation",
)

// droppedWithContent enthält Elemente, die MITSAMT Inhalt entfernt werden:
// ausführbarer Code, Einbettungen, Formularfelder und Rohtext-Container,
// deren Inhalt nicht als sichtbarer Text gedacht ist.
var droppedWithContent = toSet(
	"script", "style", "iframe", "object", "applet", "frame", "frameset",
	"noscript", "noembed", "noframes", "template", "title", "textarea",
	"select", "xmp", "plaintext", "foreignobject", "annotation-xml",
)

// voidElements sind HTML-Elemente ohne Inhalt und ohne End-Tag.
var voidElements = toSet(
	"area", "base", "br", "col", "embed", "hr", "img", "input", "keygen",
	"link", "meta", "param", "source", "track", "wbr",
)

// allowedAttributes sind auf allen erlaubten Elementen zulässig.
// Event-Handler (on*), URL-Attribute und srcset fehlen bewusst.
var allowedAttributes = toSet(
	// HTML allgemein
	"id", "class", "title", "lang", "dir", "style", "align", "valign",
	"width", "height", "colspan", "rowspan", "alt", "start", "reversed",
	"value", "type", "datetime", "abbr", "scope", "span", "border",
	"cellpadding", "cellspacing", "nowrap",
	// SVG
	"xmlns", "xmlns:xlink", "version", "viewbox", "preserveaspectratio",
	"x", "y", "x1", "y1", "x2", "y2", "cx", "cy", "r", "rx", "ry", "d",
	"points", "fill", "stroke", "stroke-width", "transform", "opacity",
	"fill-opacity", "text-anchor", "font-size", "font-family",
	// MathML
	"display", "mathvariant", "stretchy", "fence", "separator", "accent",
	"accentunder", "columnalign", "rowalign", "lspace", "rspace", "encoding",
	"linethickness", "displaystyle", "scriptlevel", "notation", "mathsize",
)

// allowedLinkSchemes sind die einzigen erlaubten Schemata für <a href>.
// Relative Verweise (ohne Schema) und Anker (#…) sind zusätzlich erlaubt.
var allowedLinkSchemes = toSet("http", "https", "mailto")

// toSet baut aus einer Liste eine Menge für schnelle Nachschläge.
//
// @param items Elemente der Menge.
// @return Menge als map[string]bool.
func toSet(items ...string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// SanitizeHTML bereinigt fremdes HTML nach dem Whitelist-Prinzip.
//
// Erlaubte Elemente bleiben mit erlaubten Attributen erhalten, unbekannte
// Elemente verlieren ihr Tag (Text bleibt), gefährliche Container werden
// samt Inhalt entfernt, Kommentare verworfen. Links dürfen nur auf http(s),
// mailto, Anker oder relative Ziele zeigen; Bilder nur data:image-URIs sein.
//
// @param input Unsicheres HTML-Fragment.
// @return Bereinigtes HTML-Fragment.
func SanitizeHTML(input string) string {
	tokenizer := nethtml.NewTokenizer(strings.NewReader(input))
	var out strings.Builder
	out.Grow(len(input))

	// Wird gerade ein Element samt Inhalt übersprungen, steht hier sein Name;
	// skipDepth zählt gleichnamige Verschachtelungen (z. B. <object><object>).
	skipTag := ""
	skipDepth := 0

	for {
		tokenType := tokenizer.Next()
		if tokenType == nethtml.ErrorToken {
			// io.EOF = reguläres Ende; bei anderen Fehlern wird der Rest verworfen
			// (sicherer als unvollständig geprüftes HTML durchzulassen)
			break
		}
		token := tokenizer.Token()

		// Innerhalb eines verworfenen Elements nur dessen Ende suchen
		if skipTag != "" {
			switch {
			case tokenType == nethtml.StartTagToken && token.Data == skipTag:
				skipDepth++
			case tokenType == nethtml.EndTagToken && token.Data == skipTag:
				skipDepth--
				if skipDepth == 0 {
					skipTag = ""
				}
			}
			continue
		}

		switch tokenType {
		case nethtml.TextToken:
			out.WriteString(html.EscapeString(token.Data))

		case nethtml.StartTagToken, nethtml.SelfClosingTagToken:
			name := token.Data
			if droppedWithContent[name] {
				// Selbstschließend (z. B. <foreignObject/>) hat keinen Inhalt
				if tokenType == nethtml.StartTagToken && !voidElements[name] {
					skipTag = name
					skipDepth = 1
				}
				continue
			}
			if !allowedElements[name] {
				continue
			}
			writeStartTag(&out, name, token.Attr, tokenType == nethtml.SelfClosingTagToken || voidElements[name])

		case nethtml.EndTagToken:
			if allowedElements[token.Data] && !voidElements[token.Data] {
				out.WriteString("</" + token.Data + ">")
			}

			// CommentToken und DoctypeToken werden bewusst verworfen
		}
	}
	return out.String()
}

// writeStartTag schreibt ein Start-Tag mit allen zulässigen Attributen.
//
// @param out         Ziel-Puffer.
// @param name        Elementname (kleingeschrieben).
// @param attrs       Attribute aus dem Tokenizer (Werte bereits entschlüsselt).
// @param selfClosing true → Tag wird als "<name …/>" geschrieben.
func writeStartTag(out *strings.Builder, name string, attrs []nethtml.Attribute, selfClosing bool) {
	out.WriteString("<" + name)
	for _, attr := range attrs {
		key := attr.Key
		if attr.Namespace != "" {
			key = attr.Namespace + ":" + key
		}
		if !isAllowedAttribute(name, key, attr.Val) {
			continue
		}
		out.WriteString(" " + key + `="` + html.EscapeString(attr.Val) + `"`)
	}
	if selfClosing {
		out.WriteString("/>")
	} else {
		out.WriteString(">")
	}
}

// isAllowedAttribute entscheidet, ob ein Attribut übernommen wird.
//
// URL-Attribute sind nur an bestimmten Elementen und nur mit sicheren Werten
// erlaubt: <a href>/<a name>, <img src> und <image href|xlink:href>.
//
// @param element Elementname.
// @param key     Attributname (kleingeschrieben, ggf. mit Präfix).
// @param value   Entschlüsselter Attributwert.
// @return true wenn das Attribut sicher ist.
func isAllowedAttribute(element, key, value string) bool {
	switch {
	case element == "a" && key == "href":
		return isSafeLinkURL(value)
	case element == "a" && key == "name":
		return true
	case element == "img" && key == "src":
		return isDataImageURI(value)
	case element == "image" && (key == "href" || key == "xlink:href"):
		return isDataImageURI(value)
	case key == "style":
		return isSafeStyle(value)
	}
	return allowedAttributes[key]
}

// forbiddenStyleParts sind Bestandteile, die ein style-Attribut unbrauchbar machen:
// Nachladen von Ressourcen (url, image-set, import), alte Skript-Mechanismen
// (expression, behavior, -moz-binding) und Überlagerung der Oberfläche (position).
var forbiddenStyleParts = []string{"url", "image", "import", "expression", "behavior", "binding", "javascript", "position"}

// isSafeStyle prüft, ob ein style-Attribut keine Ressourcen nachlädt.
//
// Hintergrund (Security-Review Build 41): url(file://server/…) löst unter
// Windows einen SMB-Zugriff aus und kann den NTLM-Hash preisgeben.
// CSS-Escapes ("\72") und Kommentare ("/**/") könnten verbotene Wörter
// verschleiern – Werte mit Backslash, "/*" oder "@" werden daher komplett
// verworfen. Harmlose Angaben wie text-align oder font-variant bleiben.
//
// @param value Entschlüsselter style-Wert.
// @return true wenn der Wert unbedenklich ist.
func isSafeStyle(value string) bool {
	lower := strings.ToLower(value)
	if strings.ContainsAny(lower, "\\@<>") || strings.Contains(lower, "/*") {
		return false
	}
	for _, part := range forbiddenStyleParts {
		if strings.Contains(lower, part) {
			return false
		}
	}
	return true
}

// normalizeURLForCheck bereitet eine URL für die Schema-Prüfung vor.
//
// Browser ignorieren Tabulatoren und Zeilenumbrüche überall in URLs sowie
// führende Steuerzeichen/Leerzeichen ("java\tscript:" ist ausführbar).
// Daher werden für die Prüfung alle Zeichen <= 0x20 entfernt.
//
// @param value Entschlüsselter Attributwert.
// @return Kleingeschriebene URL ohne Steuer- und Leerzeichen.
func normalizeURLForCheck(value string) string {
	var sb strings.Builder
	for _, r := range value {
		if r > 0x20 {
			sb.WriteRune(r)
		}
	}
	return strings.ToLower(sb.String())
}

// isSafeLinkURL prüft, ob ein Link-Ziel ungefährlich ist.
//
// Erlaubt: Anker (#…), relative Ziele ohne Schema, http, https, mailto.
//
// @param value Entschlüsselter href-Wert.
// @return true wenn der Link sicher ist.
func isSafeLinkURL(value string) bool {
	normalized := normalizeURLForCheck(value)
	colon := strings.Index(normalized, ":")
	if colon < 0 {
		return true // kein Schema → relativ oder Anker
	}
	// Steht "/", "?" oder "#" vor dem Doppelpunkt, ist es kein Schema
	if strings.ContainsAny(normalized[:colon], "/?#") {
		return true
	}
	return allowedLinkSchemes[normalized[:colon]]
}

// isDataImageURI prüft, ob ein Wert eine eingebettete Bild-Daten-URI ist.
//
// @param value Entschlüsselter src/href-Wert.
// @return true bei "data:image/…".
func isDataImageURI(value string) bool {
	return strings.HasPrefix(normalizeURLForCheck(value), "data:image/")
}
