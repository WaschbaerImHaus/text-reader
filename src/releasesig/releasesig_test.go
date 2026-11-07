// Tests für die signierte Prüfsummenliste der Releases (Paket releasesig).
//
// Geprüft werden:
//   - das genaue Format der Liste (Kopfzeile, Sortierung, Zeilenende),
//   - das strenge Einlesen (falscher Tag, doppelte Namen, Pfade, Hexwerte),
//   - die Signatur mit Kontext-Präfix und einer Liste öffentlicher Schlüssel
//     (gültig, wenn einer passt),
//   - das Erzeugen und Laden des privaten Schlüssels (Seed als Hex, 0600,
//     nie überschreiben, Rechteprüfung unter Unix),
//   - das Signieren echter Dateien (beide Dateien, neu signieren möglich).
//
// Alle Schlüssel sind Testschlüssel aus ed25519.GenerateKey bzw. GenerateKey
// in t.TempDir(); es entsteht nie ein echter Release-Schlüssel.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 2: führender Bindestrich im Namen)
package releasesig_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"md-reader/releasesig"
)

// sumOf liefert die SHA-256 eines Textes als kleines Hex (Testhilfe).
func sumOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// newKeyPair erzeugt ein Testschlüsselpaar und bricht den Test bei Fehler ab.
func newKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// TestManifest_RoundTrip prüft das genaue Format und das Zurücklesen.
func TestManifest_RoundTrip(t *testing.T) {
	sums := map[string]string{
		"md-reader-setup.exe":        sumOf("setup"),
		"md-reader":                  sumOf("linux"),
		"md-reader_1.0.48_amd64.deb": sumOf("deb"),
		"md-reader-linux-arm64":      sumOf("arm64"),
	}
	got := releasesig.BuildManifest("v1.0.48", sums)
	want := "md-reader release v1.0.48\n" +
		sumOf("linux") + "  md-reader\n" +
		sumOf("arm64") + "  md-reader-linux-arm64\n" +
		sumOf("setup") + "  md-reader-setup.exe\n" +
		sumOf("deb") + "  md-reader_1.0.48_amd64.deb\n"
	if string(got) != want {
		t.Fatalf("Format falsch:\n%s\nerwartet:\n%s", got, want)
	}
	parsed, err := releasesig.ParseManifest(got, "v1.0.48")
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if !reflect.DeepEqual(parsed, sums) {
		t.Fatalf("Zurückgelesen %v, erwartet %v", parsed, sums)
	}
}

// TestManifest_FileNames prüft die festen Dateinamen im Release.
func TestManifest_FileNames(t *testing.T) {
	if releasesig.ManifestName != "md-reader-SHA256SUMS" {
		t.Errorf("ManifestName = %q", releasesig.ManifestName)
	}
	if releasesig.SignatureName != "md-reader-SHA256SUMS.sig" {
		t.Errorf("SignatureName = %q", releasesig.SignatureName)
	}
	if releasesig.MaxManifestBytes != 1<<20 {
		t.Errorf("MaxManifestBytes = %d, erwartet 1 MiB", releasesig.MaxManifestBytes)
	}
}

// TestParseManifest_WrongTag: Eine echte Liste eines anderen Releases gilt nicht.
func TestParseManifest_WrongTag(t *testing.T) {
	data := releasesig.BuildManifest("v1.0.48", map[string]string{"md-reader": sumOf("x")})
	if _, err := releasesig.ParseManifest(data, "v1.0.49"); err == nil {
		t.Fatal("Liste für v1.0.48 wurde als v1.0.49 angenommen")
	}
	// Auch ein Tag, der nur ein Präfix ist, darf nicht passen
	data = releasesig.BuildManifest("v1.0.480", map[string]string{"md-reader": sumOf("x")})
	if _, err := releasesig.ParseManifest(data, "v1.0.48"); err == nil {
		t.Fatal("Liste für v1.0.480 wurde als v1.0.48 angenommen")
	}
}

// TestParseManifest_InvalidTagArgument: Der erwartete Tag selbst muss
// dem Schema v1.0.<zahl> folgen, sonst wird nichts angenommen.
func TestParseManifest_InvalidTagArgument(t *testing.T) {
	for _, tag := range []string{"", "v1.0.", "1.0.48", "v1.0.48a", "v1.1.48", "v1.0.048", "v1.0.0", "v1.0.-1", "v1.0.48 "} {
		data := []byte("md-reader release " + tag + "\n" + sumOf("x") + "  md-reader\n")
		if _, err := releasesig.ParseManifest(data, tag); err == nil {
			t.Errorf("Tag %q wurde angenommen", tag)
		}
	}
}

// TestValidTag prüft das Tag-Schema v1.0.<build> (Build ab 1, ohne führende Null).
func TestValidTag(t *testing.T) {
	valid := []string{"v1.0.1", "v1.0.48", "v1.0.123456789"}
	invalid := []string{"", "v1.0.", "v1.0.0", "v1.0.048", "1.0.48", "V1.0.48", "v1.0.48\n",
		"v1.0.48a", "v1.0.4 8", "v2.0.48", "v1.0.1234567890", "v1.0.+4", "v1.0.٤٨"}
	for _, tag := range valid {
		if !releasesig.ValidTag(tag) {
			t.Errorf("ValidTag(%q) = false, erwartet true", tag)
		}
	}
	for _, tag := range invalid {
		if releasesig.ValidTag(tag) {
			t.Errorf("ValidTag(%q) = true, erwartet false", tag)
		}
	}
}

// TestParseManifest_BadLines: Jede Formatabweichung ist ein Fehler.
func TestParseManifest_BadLines(t *testing.T) {
	header := "md-reader release v1.0.48\n"
	good := sumOf("a")
	cases := map[string]string{
		"doppelter Name":            header + good + "  md-reader\n" + sumOf("b") + "  md-reader\n",
		"Pfad nach oben":            header + good + "  ../x\n",
		"Schrägstrich":              header + good + "  a/b\n",
		"Backslash":                 header + good + "  a\\b\n",
		"zwei Punkte im Namen":      header + good + "  a..b\n",
		"nur zwei Punkte":           header + good + "  ..\n",
		"führender Punkt":           header + good + "  .hidden\n",
		"führender Bindestrich":     header + good + "  -verify\n",
		"Großbuchstaben-Hex":        header + strings.ToUpper(good) + "  md-reader\n",
		"Hex zu kurz":               header + good[:62] + "  md-reader\n",
		"Hex zu lang":               header + good + "00  md-reader\n",
		"Nicht-Hex":                 header + strings.Repeat("g", 64) + "  md-reader\n",
		"ein Leerzeichen":           header + good + " md-reader\n",
		"Tabulator":                 header + good + "\tmd-reader\n",
		"Leerzeichen im Namen":      header + good + "  md reader\n",
		"leerer Name":               header + good + "  \n",
		"Windows-Zeilenende":        "md-reader release v1.0.48\r\n" + good + "  md-reader\r\n",
		"CR am Namen":               header + good + "  md-reader\r\n",
		"Leerzeile":                 header + good + "  md-reader\n\n",
		"Zeilenende fehlt":          header + good + "  md-reader",
		"nur Kopfzeile":             header,
		"leer":                      "",
		"unsortiert":                header + good + "  md-reader-setup.exe\n" + sumOf("b") + "  md-reader\n",
		"fremde Kopfzeile":          "ssh-easy release v1.0.48\n" + good + "  md-reader\n",
		"Kopfzeile mit Leerzeichen": "md-reader release v1.0.48 \n" + good + "  md-reader\n",
	}
	for name, text := range cases {
		if _, err := releasesig.ParseManifest([]byte(text), "v1.0.48"); err == nil {
			t.Errorf("%s: wurde angenommen", name)
		}
	}
}

// TestParseManifest_TooLarge: Mehr als MaxManifestBytes wird abgelehnt.
func TestParseManifest_TooLarge(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("md-reader release v1.0.48\n")
	for i := 0; buf.Len() <= releasesig.MaxManifestBytes; i++ {
		buf.WriteString(sumOf("x") + "  f" + strings.Repeat("0", 8-len(itoa(i))) + itoa(i) + "\n")
	}
	if _, err := releasesig.ParseManifest(buf.Bytes(), "v1.0.48"); err == nil {
		t.Fatal("übergroße Liste wurde angenommen")
	}
}

// itoa wandelt eine nicht negative Zahl ohne strconv in Text (Testhilfe).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for ; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits)
}

// TestVerify_EitherKey: Gültig ist die Signatur, wenn einer der Schlüssel passt.
func TestVerify_EitherKey(t *testing.T) {
	pubA, _ := newKeyPair(t)
	pubB, privB := newKeyPair(t)
	manifest := releasesig.BuildManifest("v1.0.48", map[string]string{"md-reader": sumOf("x")})
	sig := releasesig.Sign(privB, manifest)

	if !bytes.HasSuffix(sig, []byte("\n")) {
		t.Errorf("Signatur ohne Zeilenende: %q", sig)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(string(sig), "\n"))
	if err != nil || len(raw) != ed25519.SignatureSize {
		t.Fatalf("Signatur ist kein Base64 einer Ed25519-Signatur: %q (%v)", sig, err)
	}
	if err := releasesig.Verify([]ed25519.PublicKey{pubA, pubB}, manifest, sig); err != nil {
		t.Errorf("[A, B] mit Signatur von B: %v", err)
	}
	if err := releasesig.Verify([]ed25519.PublicKey{pubB, pubA}, manifest, sig); err != nil {
		t.Errorf("[B, A] mit Signatur von B: %v", err)
	}
	if err := releasesig.Verify([]ed25519.PublicKey{pubA}, manifest, sig); err == nil {
		t.Error("[A] mit Signatur von B wurde angenommen")
	}
	if err := releasesig.Verify(nil, manifest, sig); err == nil {
		t.Error("leere Schlüsselliste wurde angenommen")
	}
	if err := releasesig.Verify([]ed25519.PublicKey{pubB[:31]}, manifest, sig); err == nil {
		t.Error("zu kurzer Schlüssel wurde angenommen")
	}
	// Ein kaputter Schlüssel wird übersprungen (kein Absturz); gültig bleibt
	// die Signatur, wenn ein brauchbarer Schlüssel passt
	if err := releasesig.Verify([]ed25519.PublicKey{nil, pubB}, manifest, sig); err != nil {
		t.Errorf("[kaputt, B] mit Signatur von B: %v", err)
	}
	if err := releasesig.Verify([]ed25519.PublicKey{pubB[:31], pubB}, manifest, sig); err != nil {
		t.Errorf("[B gekürzt, B] mit Signatur von B: %v", err)
	}
	// Bleibt kein brauchbarer Schlüssel übrig, ist das ein Fehler
	if err := releasesig.Verify([]ed25519.PublicKey{nil}, manifest, sig); err == nil {
		t.Error("[nil] wurde angenommen")
	}
	if err := releasesig.Verify([]ed25519.PublicKey{nil, pubA}, manifest, sig); err == nil {
		t.Error("[kaputt, A] mit Signatur von B wurde angenommen")
	}
}

// TestVerify_ContextIsBound: Signiert wird genau über Kontext + Liste.
func TestVerify_ContextIsBound(t *testing.T) {
	pub, priv := newKeyPair(t)
	manifest := releasesig.BuildManifest("v1.0.48", map[string]string{"md-reader": sumOf("x")})

	// Die erzeugte Signatur passt zur Nachricht mit dem festen Kontext
	sig := releasesig.Sign(priv, manifest)
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if !ed25519.Verify(pub, append([]byte("md-reader-release-manifest-v1\n"), manifest...), raw) {
		t.Fatal("Signatur ist nicht über md-reader-release-manifest-v1\\n + Liste")
	}
	// Eine Signatur ohne Kontext (oder mit fremdem Kontext) gilt nicht
	for _, prefix := range []string{"", "ssh-easy release manifest v1\n", "md-reader-release-manifest-v2\n"} {
		other := ed25519.Sign(priv, append([]byte(prefix), manifest...))
		text := []byte(base64.StdEncoding.EncodeToString(other) + "\n")
		if err := releasesig.Verify([]ed25519.PublicKey{pub}, manifest, text); err == nil {
			t.Errorf("Signatur mit Kontext %q wurde angenommen", prefix)
		}
	}
}

// TestVerify_TamperedManifest: Jede Änderung an Liste oder Signatur fällt auf.
func TestVerify_TamperedManifest(t *testing.T) {
	pub, priv := newKeyPair(t)
	manifest := releasesig.BuildManifest("v1.0.48", map[string]string{
		"md-reader": sumOf("x"), "md-reader.exe": sumOf("y"),
	})
	sig := releasesig.Sign(priv, manifest)
	keys := []ed25519.PublicKey{pub}

	tampered := bytes.Clone(manifest)
	tampered[len(tampered)-2] ^= 0x01
	if err := releasesig.Verify(keys, tampered, sig); err == nil {
		t.Error("veränderte Liste wurde angenommen")
	}
	if err := releasesig.Verify(keys, append(bytes.Clone(manifest), "x\n"...), sig); err == nil {
		t.Error("verlängerte Liste wurde angenommen")
	}
	rawSig, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	rawSig[0] ^= 0x80
	if err := releasesig.Verify(keys, manifest, []byte(base64.StdEncoding.EncodeToString(rawSig))); err == nil {
		t.Error("veränderte Signatur wurde angenommen")
	}
	for name, bad := range map[string]string{
		"leer":            "",
		"kein Base64":     "!!!",
		"zu kurz":         base64.StdEncoding.EncodeToString(make([]byte, 63)),
		"zu lang":         base64.StdEncoding.EncodeToString(make([]byte, 65)),
		"zwei Signaturen": strings.TrimSpace(string(sig)) + "\n" + strings.TrimSpace(string(sig)) + "\n",
	} {
		if err := releasesig.Verify(keys, manifest, []byte(bad)); err == nil {
			t.Errorf("Signatur %s wurde angenommen", name)
		}
	}
	// Die unveränderte Signatur bleibt gültig (Gegenprobe)
	if err := releasesig.Verify(keys, manifest, sig); err != nil {
		t.Errorf("unveränderte Signatur abgelehnt: %v", err)
	}
}

// TestGenerateKey_NoOverwrite: Seed als Hex, Rechte 0600, nie überschreiben.
func TestGenerateKey_NoOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "release-signing.key")
	pub, err := releasesig.GenerateKey(path)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("öffentlicher Schlüssel hat %d Bytes", len(pub))
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Schlüsseldatei fehlt: %v", err)
	}
	text := string(first)
	if !strings.HasSuffix(text, "\n") || len(text) != ed25519.SeedSize*2+1 {
		t.Fatalf("Schlüsseldatei hat nicht das Format <64 hex>\\n (Länge %d)", len(text))
	}
	if _, err := hex.DecodeString(strings.TrimSuffix(text, "\n")); err != nil {
		t.Fatalf("Seed ist kein Hex: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("Rechte %o, erwartet 600", perm)
		}
	}

	// Zweiter Aufruf: Fehler, Datei unverändert
	if _, err := releasesig.GenerateKey(path); err == nil {
		t.Fatal("vorhandener Schlüssel wurde überschrieben (kein Fehler)")
	}
	second, _ := os.ReadFile(path)
	if !bytes.Equal(first, second) {
		t.Fatal("vorhandener Schlüssel wurde verändert")
	}

	// Der geladene Schlüssel gehört zum ausgegebenen öffentlichen Schlüssel
	priv, err := releasesig.LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}
	if !pub.Equal(priv.Public()) {
		t.Fatal("geladener Schlüssel passt nicht zum öffentlichen Schlüssel")
	}
}

// TestLoadPrivateKey_RejectsGroupReadable: Unter Unix nur 0600 (oder enger).
func TestLoadPrivateKey_RejectsGroupReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-Rechte gibt es unter Windows nicht")
	}
	path := filepath.Join(t.TempDir(), "key")
	if _, err := releasesig.GenerateKey(path); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	for _, perm := range []os.FileMode{0o640, 0o604, 0o660, 0o644, 0o610} {
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
		if _, err := releasesig.LoadPrivateKey(path); err == nil {
			t.Errorf("Rechte %o wurden angenommen", perm)
		}
	}
	for _, perm := range []os.FileMode{0o600, 0o400} {
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
		if _, err := releasesig.LoadPrivateKey(path); err != nil {
			t.Errorf("Rechte %o abgelehnt: %v", perm, err)
		}
	}
}

// TestLoadPrivateKey_BadContent: Nur genau ein Seed als Hex wird angenommen.
func TestLoadPrivateKey_BadContent(t *testing.T) {
	dir := t.TempDir()
	seed := make([]byte, ed25519.SeedSize)
	cases := map[string]string{
		"Base64 statt Hex": base64.StdEncoding.EncodeToString(seed) + "\n",
		"zu kurz":          hex.EncodeToString(seed[:31]) + "\n",
		"zu lang":          hex.EncodeToString(append(seed, 0)) + "\n",
		"leer":             "",
		"kein Hex":         strings.Repeat("z", 64) + "\n",
		"ganzer Schlüssel": hex.EncodeToString(ed25519.NewKeyFromSeed(seed)) + "\n",
	}
	for name, content := range cases {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_"))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := releasesig.LoadPrivateKey(path); err == nil {
			t.Errorf("%s: wurde angenommen", name)
		}
	}
	// Ein Verzeichnis ist keine Schlüsseldatei
	sub := filepath.Join(dir, "verzeichnis")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := releasesig.LoadPrivateKey(sub); err == nil {
		t.Error("Verzeichnis wurde als Schlüssel angenommen")
	}
	// Fehlende Datei
	if _, err := releasesig.LoadPrivateKey(filepath.Join(dir, "fehlt")); err == nil {
		t.Error("fehlende Datei ergab keinen Fehler")
	}
}

// writeFile legt eine Testdatei an und liefert ihren Pfad.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSignFiles_WritesBoth: Liste und Signatur entstehen und passen zusammen;
// ein zweiter Lauf (Zwei-Schritt-Signatur mit macOS-ZIP) ersetzt beide.
func TestSignFiles_WritesBoth(t *testing.T) {
	pub, priv := newKeyPair(t)
	src := t.TempDir()
	out := t.TempDir()
	files := []string{
		writeFile(t, src, "md-reader", "linux-binary"),
		writeFile(t, src, "md-reader.exe", "windows-binary"),
	}
	if err := releasesig.SignFiles(priv, "v1.0.48", out, files); err != nil {
		t.Fatalf("SignFiles: %v", err)
	}
	check := func(want map[string]string) {
		t.Helper()
		manifest, err := os.ReadFile(filepath.Join(out, "md-reader-SHA256SUMS"))
		if err != nil {
			t.Fatalf("Liste fehlt: %v", err)
		}
		sig, err := os.ReadFile(filepath.Join(out, "md-reader-SHA256SUMS.sig"))
		if err != nil {
			t.Fatalf("Signatur fehlt: %v", err)
		}
		if err := releasesig.Verify([]ed25519.PublicKey{pub}, manifest, sig); err != nil {
			t.Fatalf("Verify: %v", err)
		}
		sums, err := releasesig.ParseManifest(manifest, "v1.0.48")
		if err != nil {
			t.Fatalf("ParseManifest: %v", err)
		}
		if !reflect.DeepEqual(sums, want) {
			t.Fatalf("Prüfsummen %v, erwartet %v", sums, want)
		}
	}
	check(map[string]string{"md-reader": sumOf("linux-binary"), "md-reader.exe": sumOf("windows-binary")})

	// Zweiter Schritt: mit macOS-ZIP neu signieren, beide Dateien werden ersetzt
	files = append(files, writeFile(t, src, "md-reader-macos.zip", "zip"))
	if err := releasesig.SignFiles(priv, "v1.0.48", out, files); err != nil {
		t.Fatalf("SignFiles (zweiter Schritt): %v", err)
	}
	check(map[string]string{
		"md-reader": sumOf("linux-binary"), "md-reader.exe": sumOf("windows-binary"),
		"md-reader-macos.zip": sumOf("zip"),
	})
	// Keine Temp-Reste im Zielordner
	entries, _ := os.ReadDir(out)
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("Zielordner enthält %v, erwartet nur Liste und Signatur", names)
	}
}

// TestSignFiles_Rejects: ungültige Aufrufe schreiben nichts.
func TestSignFiles_Rejects(t *testing.T) {
	_, priv := newKeyPair(t)
	src := t.TempDir()
	other := filepath.Join(src, "other")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	a := writeFile(t, src, "md-reader", "a")
	dupe := writeFile(t, other, "md-reader", "b")
	space := writeFile(t, src, "md reader", "c")
	sums := writeFile(t, src, "md-reader-SHA256SUMS", "alt")
	sig := writeFile(t, src, "md-reader-SHA256SUMS.sig", "alt")

	cases := map[string]struct {
		tag   string
		files []string
	}{
		"doppelter Name":      {"v1.0.48", []string{a, dupe}},
		"Leerzeichen im Name": {"v1.0.48", []string{a, space}},
		"Liste selbst":        {"v1.0.48", []string{a, sums}},
		"Signatur selbst":     {"v1.0.48", []string{a, sig}},
		"fehlende Datei":      {"v1.0.48", []string{a, filepath.Join(src, "fehlt")}},
		"Verzeichnis":         {"v1.0.48", []string{a, other}},
		"keine Dateien":       {"v1.0.48", nil},
		"falscher Tag":        {"1.0.48", []string{a}},
	}
	for name, c := range cases {
		out := t.TempDir()
		if err := releasesig.SignFiles(priv, c.tag, out, c.files); err == nil {
			t.Errorf("%s: kein Fehler", name)
		}
		if entries, _ := os.ReadDir(out); len(entries) != 0 {
			t.Errorf("%s: Zielordner nicht leer (%d Einträge)", name, len(entries))
		}
	}
}
