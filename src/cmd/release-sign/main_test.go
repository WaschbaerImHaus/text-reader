// Tests für das Signierwerkzeug release-sign (Paket main).
//
// Geprüft wird run() direkt (ohne Unterprozess):
//   - -genkey, Signieren und Prüfen mit releasesig.Verify,
//   - -pub gibt den öffentlichen Schlüssel eines vorhandenen Schlüssels aus,
//   - -verify mit einem oder mehreren öffentlichen Schlüsseln,
//   - Exit-Codes: 0 ok, 1 Fehler, 2 Aufruf falsch (u. a. Tag nicht v1.0.<zahl>).
//
// Alle Schlüssel entstehen in t.TempDir(); nie ein echter Release-Schlüssel.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu – Auto-Update Build 48, Aufgabe 2)
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
		if code, _, errOut := runCLI("-verify", "-pub", pubs, "-tag", "v1.0.48", sumsPath, sigPath); code != 0 {
			t.Errorf("-verify mit %q: Exit %d, stderr %q", pubs, code, errOut)
		}
	}
	// -verify: fremder Schlüssel oder anderer Tag → Exit 1
	if code, _, _ := runCLI("-verify", "-pub", otherHex, "-tag", "v1.0.48", sumsPath, sigPath); code != 1 {
		t.Errorf("-verify mit fremdem Schlüssel: Exit %d, erwartet 1", code)
	}
	if code, _, _ := runCLI("-verify", "-pub", pubHex, "-tag", "v1.0.49", sumsPath, sigPath); code != 1 {
		t.Errorf("-verify mit anderem Tag: Exit %d, erwartet 1", code)
	}
	// -verify: veränderte Liste → Exit 1
	tampered := filepath.Join(dir, "tampered")
	if err := os.WriteFile(tampered, bytes.Replace(manifest, []byte("md-reader.exe"), []byte("md-reader.exf"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI("-verify", "-pub", pubHex, "-tag", "v1.0.48", tampered, sigPath); code != 1 {
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
		if code, _, _ := runCLI("-verify", "-pub", pubHex, "-tag", tag, assets[0], assets[1]); code != 2 {
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
		"-verify ohne -pub":          {"-verify", "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit Kurz-Hex":       {"-verify", "-pub", pubHex[:62], "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit Nicht-Hex":      {"-verify", "-pub", strings.Repeat("zz", 32), "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit leerem Eintrag": {"-verify", "-pub", pubHex + ",", "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit einer Datei":    {"-verify", "-pub", pubHex, "-tag", "v1.0.48", assets[0]},
		"-verify mit drei Dateien":   {"-verify", "-pub", pubHex, "-tag", "v1.0.48", assets[0], assets[1], assets[0]},
		"-verify mit -key":           {"-verify", "-pub", pubHex, "-key", keyPath, "-tag", "v1.0.48", assets[0], assets[1]},
		"-verify mit -genkey":        {"-verify", "-genkey", "-pub", pubHex, "-tag", "v1.0.48", assets[0], assets[1]},
	}
	for name, args := range cases {
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("%s: Exit %d, erwartet 2", name, code)
		}
	}
	// Kein Aufruf darf einen neuen Schlüssel angelegt haben
	for _, name := range []string{"neu.key", "neu2.key"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s wurde trotz Aufruffehler angelegt", name)
		}
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
