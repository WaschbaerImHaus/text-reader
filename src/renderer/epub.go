// Package renderer - EPUB-Datei-Parsing und Rendering.
//
// Liest EPUB-Dateien (ZIP-Archive mit XHTML-Kapiteln gemäß EPUB 2/3 Standard)
// und wandelt den Inhalt in anzeigbares HTML um.
//
// EPUB-Struktur:
//   - META-INF/container.xml → Pfad zur OPF-Hauptdatei
//   - OPF (Open Packaging Format) → Manifest + Spine (Kapitelreihenfolge)
//   - XHTML-Dateien → eigentliche Kapitelinhalte
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package renderer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// epubContainer repräsentiert die META-INF/container.xml Datei.
// Sie enthält den Pfad zur OPF-Hauptdatei des EPUB.
type epubContainer struct {
	XMLName   xml.Name       `xml:"container"`
	Rootfiles []epubRootfile `xml:"rootfiles>rootfile"`
}

// epubRootfile enthält den Pfad zur OPF-Datei des EPUB.
type epubRootfile struct {
	// FullPath ist der Pfad zur OPF-Datei relativ zum EPUB-Root.
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// epubPackage repräsentiert das OPF-Dokument (Open Packaging Format).
// Enthält Metadaten, Manifest (alle Dateien) und Spine (Lesereihenfolge).
type epubPackage struct {
	XMLName  xml.Name     `xml:"package"`
	Metadata epubMetadata `xml:"metadata"`
	Manifest []epubItem   `xml:"manifest>item"`
	Spine    epubSpine    `xml:"spine"`
}

// epubMetadata enthält bibliografische Metadaten des EPUB-Buches.
type epubMetadata struct {
	Title  string `xml:"title"`
	Author string `xml:"creator"`
}

// epubItem repräsentiert eine Ressource im EPUB-Manifest (Kapitel, Bild, CSS...).
type epubItem struct {
	// ID ist der eindeutige Bezeichner innerhalb des Manifests.
	ID        string `xml:"id,attr"`
	// Href ist der relative Dateipfad innerhalb des EPUB-Archivs.
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// epubSpine definiert die Lesereihenfolge der Kapitel per Manifest-ID.
type epubSpine struct {
	Items []epubItemRef `xml:"itemref"`
}

// epubItemRef verweist per ID auf ein Element im Manifest.
type epubItemRef struct {
	// IDRef entspricht der ID eines epubItem im Manifest.
	IDRef string `xml:"idref,attr"`
	// Linear gibt an ob das Element Teil des Hauptlesewegs ist.
	// "no" bedeutet Hilfsinhalte (Fußnoten, Index) → überspringen.
	Linear string `xml:"linear,attr"`
}

// ParseEpub liest EPUB-Binärdaten und konvertiert den Inhalt zu HTML.
//
// Liest alle Kapitel in Spine-Reihenfolge und fügt sie zu einem
// einzigen HTML-Dokument zusammen.
//
// @param data     Rohe EPUB-Binärdaten (ZIP-Archiv).
// @param filename Dateiname für den Titel-Fallback.
// @return Result mit gesamtem HTML-Inhalt und Metadaten, oder Fehler.
func ParseEpub(data []byte, filename string) (*Result, error) {
	// EPUB als ZIP-Archiv öffnen
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("EPUB konnte nicht als ZIP geöffnet werden: %w", err)
	}

	// container.xml lesen um OPF-Dateipfad zu ermitteln
	containerData, err := readZipEntry(r, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("META-INF/container.xml nicht gefunden: %w", err)
	}

	var container epubContainer
	if err := xml.Unmarshal(containerData, &container); err != nil {
		return nil, fmt.Errorf("container.xml konnte nicht geparst werden: %w", err)
	}
	if len(container.Rootfiles) == 0 {
		return nil, fmt.Errorf("keine OPF-Datei in container.xml gefunden")
	}

	// OPF-Datei (Open Packaging Format) einlesen
	opfPath := container.Rootfiles[0].FullPath
	opfDir := path.Dir(opfPath)
	if opfDir == "." {
		opfDir = ""
	}

	opfData, err := readZipEntry(r, opfPath)
	if err != nil {
		return nil, fmt.Errorf("OPF-Datei nicht gefunden (%s): %w", opfPath, err)
	}

	var pkg epubPackage
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return nil, fmt.Errorf("OPF konnte nicht geparst werden: %w", err)
	}

	// Manifest als ID→Item-Map aufbauen für schnellen Zugriff
	manifest := make(map[string]epubItem, len(pkg.Manifest))
	for _, item := range pkg.Manifest {
		manifest[item.ID] = item
	}

	// Kapitel in Spine-Reihenfolge laden und zusammenfügen
	var htmlBuilder strings.Builder
	chapterNum := 0
	for _, itemRef := range pkg.Spine.Items {
		// Nicht-lineare Elemente überspringen (z.B. Fußnoten-Seiten, Index)
		if itemRef.Linear == "no" {
			continue
		}
		item, ok := manifest[itemRef.IDRef]
		if !ok {
			continue
		}
		// Nur XHTML/HTML-Inhalte verarbeiten, keine Bilder oder CSS-Dateien
		if !strings.Contains(item.MediaType, "html") && !strings.Contains(item.MediaType, "xml") {
			continue
		}

		// URL-Kodierung im Href auflösen (z.B. %20 → Leerzeichen)
		href, _ := url.PathUnescape(item.Href)
		// Vollständigen ZIP-Pfad relativ zum OPF-Verzeichnis berechnen
		var itemPath string
		if opfDir != "" {
			itemPath = path.Join(opfDir, href)
		} else {
			itemPath = href
		}

		chapterData, err := readZipEntry(r, itemPath)
		if err != nil {
			// Nicht gefundene Kapitel überspringen ohne Abbruch
			continue
		}

		// XHTML-Body-Inhalt extrahieren und von style/script befreien
		body := extractHTMLBody(string(chapterData))
		body = stripContentTags(body, []string{"style", "script", "link"})
		if strings.TrimSpace(body) == "" {
			continue
		}

		// Trenn-Linie zwischen Kapiteln einfügen (nicht vor dem ersten)
		if chapterNum > 0 {
			htmlBuilder.WriteString(`<hr class="epub-chapter-separator">`)
		}
		htmlBuilder.WriteString(body)
		chapterNum++
	}

	// Titel aus OPF-Metadaten oder Dateiname als Fallback
	title := strings.TrimSpace(pkg.Metadata.Title)
	if title == "" {
		title = fileBaseName(filename)
	}

	return &Result{
		HTML:       htmlBuilder.String(),
		Title:      title,
		RawContent: "", // Binärdaten werden nicht als RawContent gespeichert
	}, nil
}

// IsEpubFile prüft ob ein Dateipfad auf eine EPUB-Datei (.epub) zeigt.
//
// @param filePath Dateipfad.
// @return true wenn die Datei .epub Endung hat.
func IsEpubFile(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".epub"
}

// readZipEntry liest den Inhalt eines benannten Eintrags aus einem ZIP-Archiv.
//
// @param r    Geöffnetes ZIP-Archiv.
// @param name Pfad des gesuchten Eintrags im Archiv.
// @return Inhalt als Byte-Slice, oder Fehler wenn nicht gefunden.
func readZipEntry(r *zip.Reader, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(rc); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("Eintrag %q nicht im ZIP-Archiv gefunden", name)
}
