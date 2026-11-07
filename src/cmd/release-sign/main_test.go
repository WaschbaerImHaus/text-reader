// Tests für das Signierwerkzeug release-sign (Paket main).
//
// Geprüft wird run() direkt (ohne Unterprozess):
//   - -genkey, Signieren und Prüfen mit releasesig.Verify,
//   - -pub gibt den öffentlichen Schlüssel eines vorhandenen Schlüssels aus,
//   - -verify mit -pubkeys (ein oder mehrere öffentliche Schlüssel),
//   - gegenseitige Ausschlüsse der Modi (-pub nur mit -key, -pubkeys nur mit -verify),
//   - Exit-Codes: 0 ok, 1 Fehler, 2 Aufruf falsch (u. a. Tag nicht v1.0.<zahl>).
//
// Alle Schlüssel entstehen in t.TempDir(); nie ein echter Release-Schlüssel.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 1: -pubkeys statt überladenem -pub, neue Fehlerfälle)
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"md-reader/releasesig"
)

// runCLI ruft run() auf und liefert Exit-Code, stdout und stderr.
func runCLI(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// genKey erzeugt per CLI einen Testschlüssel und liefert Pfad und Hex.
func genKey(t *testing.T, dir string) (string, string) {
	t.Helper()
	keyPath := filepath.Join(dir, "keys", "release-signing.key")
	code, out, errOut := runCLI("-genkey", "-key", keyPath)
	if code != 0 {
		t.Fatalf("-genkey: Exit %d, stderr %q", code, errOut)
	}
	pubHex := strings.TrimSpace(out)
	if len(pubHex) != ed25519.PublicKeySize*2 || out != pubHex+"\n" {
		t.Fatalf("-genkey gibt %q aus, erwartet genau 64 Hex-Zeichen und Zeilenende", out)
	}
	return keyPath, pubHex
}

// makeAssets legt zwei Testdateien an und liefert ihre Pfade.
func makeAssets(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	for name, content := range map[string]string{"md-reader": "linux", "md-reader.exe": "windows"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

// TestRun_GenkeySignVerify: Schlüssel erzeugen, signieren, Ergebnis prüfen.
func TestRun_GenkeySignVerify(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	out := filepath.Join(dir, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}

	args := append([]string{"-key", keyPath, "-tag", "v1.0.48", "-out", out}, assets...)
	if code, _, errOut := runCLI(args...); code != 0 {
		t.Fatalf("Signieren: Exit %d, stderr %q", code, errOut)
	}
	manifest, err := os.ReadFile(filepath.Join(out, releasesig.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	sig, err := os.ReadFile(filepath.Join(out, releasesig.SignatureName))
	if err != nil {
		t.Fatal(err)
	}
	rawPub, _ := hex.DecodeString(pubHex)
	if err := releasesig.Verify([]ed25519.PublicKey{rawPub}, manifest, sig); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	sums, err := releasesig.ParseManifest(manifest, "v1.0.48")
	if err != nil || len(sums) != 2 {
		t.Fatalf("ParseManifest: %v (%d Einträge)", err, len(sums))
	}

	// -pub gibt denselben öffentlichen Schlüssel aus
	code, pubOut, errOut := runCLI("-pub", "-key", keyPath)
	if code != 0 || pubOut != pubHex+"\n" {
		t.Fatalf("-pub: Exit %d, Ausgabe %q (erwartet %q), stderr %q", code, pubOut, pubHex, errOut)
	}

	// -verify: gültig mit dem richtigen Schlüssel, auch als zweiter in der Liste
	sumsPath := filepath.Join(out, releasesig.ManifestName)
	sigPath := filepath.Join(out, releasesig.SignatureName)
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	otherHex := hex.EncodeToString(otherPub)
	for _, pubs := range []string{pubHex, otherHex + "," + pubHex, pubHex + "," + otherHex} {
		if code, _, errOut := runCLI("-verify", "-pubkeys", pubs, "-tag", "v1.0.48", sumsPath, sigPath); code != 0 {
			t.Errorf("-verify mit %q: Exit %d, stderr %q", pubs, code, errOut)
		}
	}
	// -verify=1 ist ein gültiger Bool-Wert des flag-Pakets
	if code, _, errOut := runCLI("-verify=1", "-pubkeys", pubHex, "-tag", "v1.0.48", sumsPath, sigPath); code != 0 {
		t.Errorf("-verify=1: Exit %d, stderr %q", code, errOut)
	}
	// -verify: fremder Schlüssel oder anderer Tag → Exit 1
	if code, _, _ := runCLI("-verify", "-pubkeys", otherHex, "-tag", "v1.0.48", sumsPath, sigPath); code != 1 {
		t.Errorf("-verify mit fremdem Schlüssel: Exit %d, erwartet 1", code)
	}
	if code, _, _ := runCLI("-verify", "-pubkeys", pubHex, "-tag", "v1.0.49", sumsPath, sigPath); code != 1 {
		t.Errorf("-verify mit anderem Tag: Exit %d, erwartet 1", code)
	}
	// -verify: veränderte Liste → Exit 1
	tampered := filepath.Join(dir, "tampered")
	if err := os.WriteFile(tampered, bytes.Replace(manifest, []byte("md-reader.exe"), []byte("md-reader.exf"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI("-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", tampered, sigPath); code != 1 {
		t.Errorf("-verify mit veränderter Liste: Exit %d, erwartet 1", code)
	}

	// Zweites -genkey auf denselben Pfad: Fehler, Schlüssel unverändert
	before, _ := os.ReadFile(keyPath)
	if code, out, _ := runCLI("-genkey", "-key", keyPath); code != 1 || out != "" {
		t.Errorf("zweites -genkey: Exit %d, Ausgabe %q, erwartet Exit 1 ohne Ausgabe", code, out)
	}
	after, _ := os.ReadFile(keyPath)
	if !bytes.Equal(before, after) {
		t.Error("vorhandener Schlüssel wurde verändert")
	}
}

// TestRun_BadTagIsUsageError: Ein Tag, der nicht v1.0.<zahl> ist, ergibt Exit 2.
func TestRun_BadTagIsUsageError(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	for _, tag := range []string{"", "1.0.48", "v1.0.x", "v1.1.48", "v1.0.048", "v0.51"} {
		args := append([]string{"-key", keyPath, "-tag", tag, "-out", dir}, assets...)
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("Signieren mit Tag %q: Exit %d, erwartet 2", tag, code)
		}
		if code, _, _ := runCLI("-verify", "-pubkeys", pubHex, "-tag", tag, assets[0], assets[1]); code != 2 {
			t.Errorf("-verify mit Tag %q: Exit %d, erwartet 2", tag, code)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, releasesig.ManifestName)); err == nil {
		t.Error("trotz falschem Tag wurde eine Liste geschrieben")
	}
}

// TestRun_MissingKeyFails: Eine fehlende Schlüsseldatei ergibt Exit 1.
func TestRun_MissingKeyFails(t *testing.T) {
	dir := t.TempDir()
	assets := makeAssets(t, dir)
	missing := filepath.Join(dir, "fehlt.key")
	args := append([]string{"-key", missing, "-tag", "v1.0.48", "-out", dir}, assets...)
	code, out, errOut := runCLI(args...)
	if code != 1 {
		t.Errorf("Signieren ohne Schlüsseldatei: Exit %d, erwartet 1", code)
	}
	if out != "" || errOut == "" {
		t.Errorf("Ausgabe %q / Fehlermeldung %q, erwartet nur eine Fehlermeldung", out, errOut)
	}
	if code, _, _ := runCLI("-pub", "-key", missing); code != 1 {
		t.Errorf("-pub ohne Schlüsseldatei: Exit %d, erwartet 1", code)
	}
}

// TestRun_UsageErrors: falsche Aufrufe ergeben Exit 2.
func TestRun_UsageErrors(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	cases := map[string][]string{
		"keine Argumente":            {},
		"unbekannte Option":          {"-foo"},
		"-genkey ohne -key":          {"-genkey"},
		"-genkey mit Dateien":        {"-genkey", "-key", filepath.Join(dir, "neu.key"), assets[0]},
		"-genkey und -pub":           {"-genkey", "-pub", "-key", filepath.Join(dir, "neu2.key")},
		"-pub ohne -key":             {"-pub"},
		"Signieren ohne -key":        {"-tag", "v1.0.48", assets[0]},
		"Signieren ohne Dateien":     {"-key", keyPath, "-tag", "v1.0.48", "-out", dir},
		"-verify ohne -pubkeys":      {"-verify", "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit Kurz-Hex":       {"-verify", "-pubkeys", pubHex[:62], "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit Nicht-Hex":      {"-verify", "-pubkeys", strings.Repeat("zz", 32), "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit leerem Eintrag": {"-verify", "-pubkeys", pubHex + ",", "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit einer Datei":    {"-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", assets[0]},
		"-verify mit drei Dateien":   {"-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", assets[0], assets[1], assets[0]},
		"-verify mit -key":           {"-verify", "-pubkeys", pubHex, "-key", keyPath, "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit -genkey":        {"-verify", "-genkey", "-pubkeys", pubHex, "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit -pub <hex>":     {"-verify", "-pub", pubHex, "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit -pub":           {"-verify", "-pubkeys", pubHex, "-pub", "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit -out":           {"-verify", "-pubkeys", pubHex, "-out", dir, "-tag", "v1.0.48", assets[0], assets[1]},
		"-pubkeys beim Signieren":    {"-key", keyPath, "-pubkeys", pubHex, "-tag", "v1.0.48", "-out", dir, assets[0]},
		"-pubkeys mit -pub":          {"-pub", "-key", keyPath, "-pubkeys", pubHex},
		"-pubkeys mit -genkey":       {"-genkey", "-key", filepath.Join(dir, "neu3.key"), "-pubkeys", pubHex},
	}
	for name, args := range cases {
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("%s: Exit %d, erwartet 2", name, code)
		}
	}
	// Kein Aufruf darf einen neuen Schlüssel angelegt haben
	for _, name := range []string{"neu.key", "neu2.key", "neu3.key"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s wurde trotz Aufruffehler angelegt", name)
		}
	}
	// Kein Aufruf darf eine Liste geschrieben haben
	if _, err := os.Stat(filepath.Join(dir, releasesig.ManifestName)); err == nil {
		t.Error("trotz Aufruffehler wurde eine Liste geschrieben")
	}
}

// TestRun_PubkeysErrorDoesNotEchoValue: Ein ungültiger -pubkeys-Eintrag wird
// nur mit Position und Länge gemeldet, der Wert selbst erscheint nie
// (falls jemand versehentlich den privaten Seed übergibt).
func TestRun_PubkeysErrorDoesNotEchoValue(t *testing.T) {
	dir := t.TempDir()
	_, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	secretLike := strings.Repeat("ab", 31)
	cases := map[string]struct {
		list, value, position, length string
	}{
		"zu kurz":   {pubHex + "," + secretLike, secretLike, "Eintrag 2", "Länge 62"},
		"Nicht-Hex": {strings.Repeat("zq", 32), strings.Repeat("zq", 32), "Eintrag 1", "Länge 64"},
		"zu lang":   {pubHex + "00", pubHex + "00", "Eintrag 1", "Länge 66"},
	}
	for name, c := range cases {
		code, out, errOut := runCLI("-verify", "-pubkeys", c.list, "-tag", "v1.0.48", assets[0], assets[1])
		if code != 2 {
			t.Errorf("%s: Exit %d, erwartet 2", name, code)
		}
		if strings.Contains(out+errOut, c.value) {
			t.Errorf("%s: Ausgabe enthält den Wert: %q", name, errOut)
		}
		if !strings.Contains(errOut, c.position) || !strings.Contains(errOut, c.length) {
			t.Errorf("%s: Meldung %q nennt nicht %q und %q", name, errOut, c.position, c.length)
		}
	}
}

// TestRun_FileNamedVerifyDoesNotSwitchMode: Eine Datei namens "-verify"
// hinter den Dateiargumenten ist eine Datei, kein Schalter.
func TestRun_FileNamedVerifyDoesNotSwitchMode(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "-verify"), []byte("odd"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	code, _, errOut := runCLI("-key", keyPath, "-tag", "v1.0.48", "-out", out, assets[0], "-verify")
	if code != 0 {
		t.Fatalf("Signieren mit Datei -verify: Exit %d, stderr %q", code, errOut)
	}
	manifest, err := os.ReadFile(filepath.Join(out, releasesig.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	sums, err := releasesig.ParseManifest(manifest, "v1.0.48")
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if _, ok := sums["-verify"]; !ok || len(sums) != 2 {
		t.Errorf("Liste %v enthält nicht genau md-reader und -verify", sums)
	}
	sigPath := filepath.Join(out, releasesig.SignatureName)
	if code, _, errOut := runCLI("-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", filepath.Join(out, releasesig.ManifestName), sigPath); code != 0 {
		t.Errorf("Prüfen: Exit %d, stderr %q", code, errOut)
	}
}

// TestRun_VerifyMissingFiles: Fehlt Liste oder Signatur, ergibt -verify Exit 1.
func TestRun_VerifyMissingFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubHex := genKey(t, dir)
	assets := makeAssets(t, dir)
	args := append([]string{"-key", keyPath, "-tag", "v1.0.48", "-out", dir}, assets...)
	if code, _, errOut := runCLI(args...); code != 0 {
		t.Fatalf("Signieren: Exit %d, stderr %q", code, errOut)
	}
	sumsPath := filepath.Join(dir, releasesig.ManifestName)
	sigPath := filepath.Join(dir, releasesig.SignatureName)
	missing := filepath.Join(dir, "fehlt")
	for name, files := range map[string][2]string{
		"Liste fehlt":    {missing, sigPath},
		"Signatur fehlt": {sumsPath, missing},
	} {
		if code, _, _ := runCLI("-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", files[0], files[1]); code != 1 {
			t.Errorf("%s: Exit %d, erwartet 1", name, code)
		}
	}
	// Gegenprobe: beide vorhanden → 0
	if code, _, errOut := runCLI("-verify", "-pubkeys", pubHex, "-tag", "v1.0.48", sumsPath, sigPath); code != 0 {
		t.Errorf("Gegenprobe: Exit %d, stderr %q", code, errOut)
	}
}

// TestRun_SignIntoMissingOutDir: Ein fehlender Zielordner ergibt Exit 1.
func TestRun_SignIntoMissingOutDir(t *testing.T) {
	dir := t.TempDir()
	keyPath, _ := genKey(t, dir)
	assets := makeAssets(t, dir)
	out := filepath.Join(dir, "gibt-es-nicht")
	args := append([]string{"-key", keyPath, "-tag", "v1.0.48", "-out", out}, assets...)
	if code, _, _ := runCLI(args...); code != 1 {
		t.Errorf("Signieren in fehlenden Ordner: Exit %d, erwartet 1", code)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("Zielordner wurde angelegt")
	}
}

// TestRun_PubDoesNotLeakPrivateKey: Keine Ausgabe enthält den Seed.
func TestRun_PubDoesNotLeakPrivateKey(t *testing.T) {
	dir := t.TempDir()
	keyPath, _ := genKey(t, dir)
	seed, _ := os.ReadFile(keyPath)
	seedHex := strings.TrimSpace(string(seed))
	assets := makeAssets(t, dir)
	calls := [][]string{
		{"-pub", "-key", keyPath},
		append([]string{"-key", keyPath, "-tag", "v1.0.48", "-out", dir}, assets...),
		{"-genkey", "-key", keyPath},
	}
	for _, args := range calls {
		_, out, errOut := runCLI(args...)
		if strings.Contains(out+errOut, seedHex) {
			t.Errorf("Aufruf %v gibt den privaten Seed aus", args)
		}
	}
}
