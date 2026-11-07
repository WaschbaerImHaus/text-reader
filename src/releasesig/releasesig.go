// Paket releasesig – signierte Prüfsummenliste für die Releases von MD Reader.
//
// Jedes Release enthält zwei zusätzliche Dateien:
//   - md-reader-SHA256SUMS: Kopfzeile "md-reader release v1.0.<n>" und je
//     Datei eine Zeile "<sha256-hex>  <Dateiname>" (sortiert, Zeilenende \n)
//   - md-reader-SHA256SUMS.sig: genau eine Ed25519-Signatur (Base64) über
//     signatureContext + Liste
//
// Der Auto-Updater prüft die Signatur gegen die im Programm eingetragenen
// öffentlichen Schlüssel (Haupt- und Notfallschlüssel); gültig ist sie, wenn
// einer davon passt. Die privaten Schlüssel liegen nie im Repo. Wer nur das
// GitHub-Konto oder den Token kontrolliert, kann deshalb keine fremde
// Programmdatei unterschieben (CWE-494).
//
// Die Kopfzeile bindet die Liste an genau ein Release: Eine echte Liste
// eines alten Releases lässt sich nicht als neues Release ausgeben
// (kein Downgrade). Das Einlesen ist absichtlich streng (fail closed):
// lieber kein Update als eine falsch verstandene Liste.
//
// Vorlage: ssh-easy (src/releasesig), angepasst an Namen, Tag-Schema und
// die Liste mehrerer öffentlicher Schlüssel.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu – Auto-Update Build 48, Aufgabe 1)
package releasesig

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	// ManifestName ist der Dateiname der Prüfsummenliste im Release.
	ManifestName = "md-reader-SHA256SUMS"
	// SignatureName ist der Dateiname der Signatur im Release.
	SignatureName = ManifestName + ".sig"
	// MaxManifestBytes begrenzt Liste und Signatur (Download und Einlesen).
	MaxManifestBytes = 1 << 20
	// tagPrefix ist der feste Anfang jedes Release-Tags (v1.0.<build>).
	tagPrefix = "v1.0."
	// maxBuildDigits begrenzt die Build-Nummer auf 9 Ziffern (passt sicher in int).
	maxBuildDigits = 9
	// manifestHeader ist der Anfang der Kopfzeile, gefolgt vom Release-Tag.
	manifestHeader = "md-reader release "
	// signatureContext ist das Kontext-Präfix der Signatur (Domain Separation):
	// Eine Signatur für einen anderen Zweck oder ein anderes Projekt gilt hier nicht.
	signatureContext = "md-reader-release-manifest-v1\n"
	// maxKeyFileBytes begrenzt die Größe der Schlüsseldatei beim Lesen.
	maxKeyFileBytes = 4096
)

// ValidTag prüft das Tag-Schema "v1.0.<build>": Build-Nummer ab 1, nur
// ASCII-Ziffern, ohne führende Null, höchstens 9 Stellen.
//
// @param tag – Release-Tag
// @return bool – true, wenn der Tag dem Schema folgt
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func ValidTag(tag string) bool {
	build, ok := strings.CutPrefix(tag, tagPrefix)
	if !ok || build == "" || len(build) > maxBuildDigits || build[0] == '0' {
		return false
	}
	for i := 0; i < len(build); i++ {
		if build[i] < '0' || build[i] > '9' {
			return false
		}
	}
	return true
}

// BuildManifest erzeugt die Prüfsummenliste eines Releases.
//
// @param tag – Release-Tag (z. B. "v1.0.48")
// @param sums – Dateiname → SHA-256 (Hex, klein)
// @return []byte – Inhalt der Liste (deterministisch sortiert)
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func BuildManifest(tag string, sums map[string]string) []byte {
	names := make([]string, 0, len(sums))
	for name := range sums {
		names = append(names, name)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	buf.WriteString(manifestHeader + tag + "\n")
	for _, name := range names {
		buf.WriteString(sums[name] + "  " + name + "\n")
	}
	return buf.Bytes()
}

// ParseManifest liest eine Prüfsummenliste streng ein. Abgelehnt werden:
// zu große Liste, ungültiger oder fremder Tag, fehlendes Zeilenende,
// leere Liste, unsortierte oder doppelte Namen, ungültige Hexwerte und
// Dateinamen mit Pfadanteilen ("/", "\", "..") oder anderen Zeichen.
//
// @param data – Inhalt der Liste
// @param tag – erwarteter Release-Tag (Kopfzeile muss genau passen)
// @return map[string]string – Dateiname → SHA-256 (Hex)
// @return error – Formatfehler oder Liste eines anderen Releases
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func ParseManifest(data []byte, tag string) (map[string]string, error) {
	if len(data) > MaxManifestBytes {
		return nil, errors.New("Prüfsummenliste ist zu groß")
	}
	if !ValidTag(tag) {
		return nil, fmt.Errorf("ungültiger Release-Tag %q", tag)
	}
	text := string(data)
	if !strings.HasSuffix(text, "\n") {
		return nil, errors.New("Prüfsummenliste: Zeilenende fehlt")
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if lines[0] != manifestHeader+tag {
		return nil, fmt.Errorf("Prüfsummenliste gehört nicht zu Release %s", tag)
	}
	sums := make(map[string]string, len(lines)-1)
	previous := ""
	for _, line := range lines[1:] {
		sum, name, ok := strings.Cut(line, "  ")
		if !ok || !validHexSum(sum) || !validFileName(name) {
			return nil, fmt.Errorf("Prüfsummenliste: ungültige Zeile %q", line)
		}
		if _, dup := sums[name]; dup {
			return nil, fmt.Errorf("Prüfsummenliste: %s doppelt", name)
		}
		// Streng aufsteigend sortiert, wie BuildManifest schreibt
		if previous != "" && name < previous {
			return nil, fmt.Errorf("Prüfsummenliste: %s nicht sortiert", name)
		}
		previous = name
		sums[name] = sum
	}
	if len(sums) == 0 {
		return nil, errors.New("Prüfsummenliste ist leer")
	}
	return sums, nil
}

// validHexSum prüft eine SHA-256 als genau 64 kleine Hex-Zeichen.
//
// @param sum – Prüfsumme
// @return bool – true, wenn gültig
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func validHexSum(sum string) bool {
	if len(sum) != sha256.Size*2 {
		return false
	}
	for i := 0; i < len(sum); i++ {
		c := sum[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// validFileName erlaubt nur schlichte Dateinamen (Erlaubnisliste):
// Buchstaben, Ziffern, Punkt, Binde- und Unterstrich. Kein führender Punkt,
// kein "..", also keine Pfade, versteckten Dateien, Leer- oder Steuerzeichen.
//
// @param name – Dateiname
// @return bool – true, wenn gültig
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func validFileName(name string) bool {
	if name == "" || name[0] == '.' || strings.Contains(name, "..") {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '.' || c == '-' || c == '_'
		if !ok {
			return false
		}
	}
	return true
}

// Sign signiert eine Prüfsummenliste.
//
// @param priv – privater Schlüssel
// @param manifest – Inhalt der Liste
// @return []byte – Signatur als Base64-Text mit Zeilenende
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func Sign(priv ed25519.PrivateKey, manifest []byte) []byte {
	sig := ed25519.Sign(priv, signedMessage(manifest))
	return []byte(base64.StdEncoding.EncodeToString(sig) + "\n")
}

// Verify prüft die Signatur einer Prüfsummenliste gegen eine Liste
// öffentlicher Schlüssel. Gültig ist sie, wenn einer der Schlüssel passt.
// Eine leere Liste oder ein Schlüssel falscher Länge ist ein Fehler
// (fail closed – ein kaputter eingetragener Schlüssel soll auffallen).
//
// @param pubs – öffentliche Schlüssel (Haupt- und Notfallschlüssel)
// @param manifest – Inhalt der Liste
// @param sig – Inhalt der Signaturdatei (genau eine Base64-Zeile)
// @return error – nil nur bei gültiger Signatur
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func Verify(pubs []ed25519.PublicKey, manifest, sig []byte) error {
	if len(pubs) == 0 {
		return errors.New("keine Signaturschlüssel angegeben")
	}
	for _, pub := range pubs {
		if len(pub) != ed25519.PublicKeySize {
			return errors.New("Signaturschlüssel ungültig")
		}
	}
	if len(sig) > MaxManifestBytes {
		return errors.New("Signatur der Prüfsummenliste ist zu groß")
	}
	text := strings.TrimSpace(string(sig))
	// Genau eine Zeile: Der Base64-Decoder würde eingestreute Zeilenenden
	// sonst stillschweigend überspringen
	if strings.ContainsAny(text, "\r\n") {
		return errors.New("Signatur der Prüfsummenliste unlesbar")
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil || len(raw) != ed25519.SignatureSize {
		return errors.New("Signatur der Prüfsummenliste unlesbar")
	}
	message := signedMessage(manifest)
	for _, pub := range pubs {
		if ed25519.Verify(pub, message, raw) {
			return nil
		}
	}
	return errors.New("Signatur der Prüfsummenliste ungültig")
}

// signedMessage liefert die tatsächlich signierten Bytes (Kontext + Liste).
//
// @param manifest – Inhalt der Liste
// @return []byte – zu signierende Nachricht (neue Kopie)
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func signedMessage(manifest []byte) []byte {
	message := make([]byte, 0, len(signatureContext)+len(manifest))
	message = append(message, signatureContext...)
	return append(message, manifest...)
}

// GenerateKey erzeugt einen neuen Signaturschlüssel und speichert den
// 32-Byte-Seed als Hex (Rechte 0600). Eine vorhandene Datei wird nie
// überschrieben (O_EXCL): Der alte Schlüssel wäre sonst verloren und keine
// installierte Version könnte mehr Updates prüfen.
//
// @param path – Zieldatei des privaten Schlüssels
// @return ed25519.PublicKey – öffentlicher Schlüssel zum Eintragen im Code
// @return error – Fehler beim Anlegen (auch: Datei existiert schon)
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func GenerateKey(path string) (ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("Schlüsseldatei: %w", err)
	}
	// Die Datei wurde eben exklusiv angelegt – bei einem Fehler darf sie weg
	if _, err := f.WriteString(hex.EncodeToString(priv.Seed()) + "\n"); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return nil, err
	}
	return pub, nil
}

// LoadPrivateKey lädt den privaten Signaturschlüssel (Seed als Hex).
// Unter Unix wird ein Schlüssel mit Gruppen- oder Fremdrechten abgelehnt;
// nur eine reguläre Datei wird angenommen.
//
// @param path – Schlüsseldatei
// @return ed25519.PrivateKey – privater Schlüssel
// @return error – Lese-, Rechte- oder Formatfehler (ohne Inhalt der Datei)
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Schlüsseldatei %s ist keine reguläre Datei", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("Schlüsseldatei %s ist für andere zugänglich (chmod 600)", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxKeyFileBytes))
	if err != nil {
		return nil, err
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("Schlüsseldatei %s ist ungültig", path)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// SignFiles berechnet die Prüfsummen der Dateien und schreibt Liste und
// Signatur nach outDir. Vorhandene Dateien werden ersetzt (Zwei-Schritt-
// Signatur: erst 8 Assets, nach dem macOS-ZIP 9). Geschrieben wird jeweils
// über eine Temp-Datei und Umbenennen, damit nie eine halbe Datei entsteht.
// Bei ungültigen Angaben wird nichts geschrieben.
//
// @param priv – privater Schlüssel
// @param tag – Release-Tag (v1.0.<n>)
// @param outDir – Zielordner für Liste und Signatur
// @param files – zu signierende Dateien (Dateinamen müssen eindeutig sein)
// @return error – Prüf-, Lese- oder Schreibfehler
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func SignFiles(priv ed25519.PrivateKey, tag, outDir string, files []string) error {
	if !ValidTag(tag) {
		return fmt.Errorf("ungültiger Release-Tag %q", tag)
	}
	if len(files) == 0 {
		return errors.New("keine Dateien zum Signieren")
	}
	sums := make(map[string]string, len(files))
	for _, path := range files {
		name := filepath.Base(path)
		if !validFileName(name) {
			return fmt.Errorf("ungültiger Dateiname %q", name)
		}
		// Liste und Signatur selbst gehören nie in die Liste
		if name == ManifestName || name == SignatureName {
			return fmt.Errorf("%s darf nicht mitsigniert werden", name)
		}
		if _, dup := sums[name]; dup {
			return fmt.Errorf("Dateiname %s doppelt", name)
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		sums[name] = sum
	}
	manifest := BuildManifest(tag, sums)
	if err := writeFileAtomic(filepath.Join(outDir, ManifestName), manifest); err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(outDir, SignatureName), Sign(priv, manifest))
}

// fileSHA256 berechnet die SHA-256 einer regulären Datei.
//
// @param path – Datei
// @return string – Prüfsumme (Hex, klein)
// @return error – Lesefehler oder keine reguläre Datei
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s ist keine reguläre Datei", path)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// writeFileAtomic schreibt eine Datei über eine Temp-Datei im selben Ordner
// und benennt sie danach um (Rechte 0644, öffentliche Release-Dateien).
// Schlägt etwas fehl, wird nur die eigene Temp-Datei entfernt.
//
// @param path – Zieldatei
// @param data – Inhalt
// @return error – Schreibfehler
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, path)
	}
	if err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
