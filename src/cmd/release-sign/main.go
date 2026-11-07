// Programm release-sign – signiert und prüft die Release-Dateien von MD Reader.
//
// Einmalig einen Schlüssel erzeugen (gibt den öffentlichen Schlüssel als Hex
// aus, der im Updater fest eingetragen wird; überschreibt nie):
//
//	release-sign -genkey -key ~/.config/md-reader/release-signing.key
//
// Öffentlichen Schlüssel eines vorhandenen Schlüssels ausgeben:
//
//	release-sign -pub -key <schlüssel>
//
// Release-Dateien signieren (schreibt md-reader-SHA256SUMS und .sig):
//
//	release-sign -key <schlüssel> -tag v1.0.<n> -out <ordner> <dateien…>
//
// Liste und Signatur prüfen (gültig, wenn einer der Schlüssel passt):
//
//	release-sign -verify -pubkeys <hex>[,<hex>] -tag v1.0.<n> <SUMS> <SIG>
//
// Die Modi schließen sich aus (sonst Exit 2): -verify nicht zusammen mit
// -key, -genkey, -pub oder -out; -pubkeys nur mit -verify; -pub nur mit -key.
// Wie beim flag-Paket üblich enden die Optionen beim ersten Dateiargument.
//
// Exit-Codes: 0 ok, 1 Fehler, 2 Aufruf falsch.
// Der private Schlüssel wird nie ausgegeben.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 1: -pubkeys statt überladenem -pub)
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"md-reader/releasesig"
)

// Exit-Codes des Werkzeugs.
const (
	exitOK    = 0 // alles in Ordnung
	exitError = 1 // Fehler beim Lesen, Schreiben oder Prüfen
	exitUsage = 2 // Aufruf falsch
)

// main startet das Werkzeug mit den Programmargumenten.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run wertet die Argumente aus und führt den gewählten Modus aus.
// Ein einziger Optionssatz; die Modi schließen sich gegenseitig aus:
// -verify nicht mit -key/-genkey/-pub/-out, -pubkeys nur mit -verify,
// -pub (Schalter) nur mit -key.
//
// @param args – Argumente ohne Programmnamen
// @param stdout – Ausgabe für Ergebnisse (öffentlicher Schlüssel, Meldung)
// @param stderr – Ausgabe für Fehlermeldungen
// @return int – Exit-Code (0 ok, 1 Fehler, 2 Aufruf falsch)
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 1: -pubkeys statt überladenem -pub, ein Optionssatz)
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("release-sign", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keyPath := fs.String("key", "", "Datei des privaten Signaturschlüssels")
	tag := fs.String("tag", "", "Release-Tag, z. B. v1.0.48")
	outDir := fs.String("out", ".", "Zielordner für Prüfsummenliste und Signatur")
	genKey := fs.Bool("genkey", false, "neuen Schlüssel erzeugen (überschreibt nie)")
	showPub := fs.Bool("pub", false, "öffentlichen Schlüssel eines vorhandenen Schlüssels ausgeben (mit -key)")
	verify := fs.Bool("verify", false, "Liste und Signatur prüfen (mit -pubkeys, -tag, <SUMS> <SIG>)")
	pubKeys := fs.String("pubkeys", "", "öffentliche Schlüssel als Hex, durch Komma getrennt (nur mit -verify)")
	if code, done := parseFlags(fs, args); done {
		return code
	}
	// Welche Optionen wurden ausdrücklich angegeben (auch mit Standardwert)?
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })

	if *verify {
		for _, name := range []string{"key", "genkey", "pub", "out"} {
			if given[name] {
				return usage(stderr, "-verify schließt -"+name+" aus")
			}
		}
		return runVerify(*pubKeys, *tag, fs.Args(), stdout, stderr)
	}
	if given["pubkeys"] {
		return usage(stderr, "-pubkeys gibt es nur mit -verify")
	}

	switch {
	case *genKey && *showPub:
		return usage(stderr, "-genkey und -pub schließen sich aus")
	case *genKey:
		if *keyPath == "" || *tag != "" || fs.NArg() != 0 {
			return usage(stderr, "-genkey braucht genau -key <pfad>")
		}
		pub, err := releasesig.GenerateKey(*keyPath)
		if err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, hex.EncodeToString(pub))
		return exitOK
	case *showPub:
		if *keyPath == "" || *tag != "" || fs.NArg() != 0 {
			return usage(stderr, "-pub braucht genau -key <pfad>")
		}
		priv, err := releasesig.LoadPrivateKey(*keyPath)
		if err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, hex.EncodeToString(priv.Public().(ed25519.PublicKey)))
		return exitOK
	}

	// Signieren: erst den Aufruf prüfen, dann den Schlüssel laden
	if *keyPath == "" {
		return usage(stderr, "-key fehlt")
	}
	if !releasesig.ValidTag(*tag) {
		return usage(stderr, fmt.Sprintf("-tag %q ist nicht v1.0.<zahl>", *tag))
	}
	if fs.NArg() == 0 {
		return usage(stderr, "mindestens eine Datei ist nötig")
	}
	priv, err := releasesig.LoadPrivateKey(*keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	if err := releasesig.SignFiles(priv, *tag, *outDir, fs.Args()); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "signiert: %d Dateien für %s\n", fs.NArg(), *tag)
	return exitOK
}

// runVerify prüft Liste und Signatur gegen die angegebenen öffentlichen
// Schlüssel und den erwarteten Tag (Aufruf aus dem Release-Skript).
//
// @param pubList – Wert von -pubkeys ("<hex>[,<hex>…]")
// @param tag – erwarteter Release-Tag
// @param files – genau zwei Dateien: <SUMS> <SIG>
// @param stdout – Ausgabe für die Erfolgsmeldung
// @param stderr – Ausgabe für Fehlermeldungen
// @return int – 0 gültig, 1 ungültig/Lesefehler, 2 Aufruf falsch
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 1: Werte aus dem gemeinsamen Optionssatz)
func runVerify(pubList, tag string, files []string, stdout, stderr io.Writer) int {
	pubs, err := parsePublicKeys(pubList)
	if err != nil {
		return usage(stderr, err.Error())
	}
	if !releasesig.ValidTag(tag) {
		return usage(stderr, fmt.Sprintf("-tag %q ist nicht v1.0.<zahl>", tag))
	}
	if len(files) != 2 {
		return usage(stderr, "-verify braucht genau zwei Dateien: <SUMS> <SIG>")
	}
	manifest, err := readLimited(files[0])
	if err != nil {
		return fail(stderr, err)
	}
	sig, err := readLimited(files[1])
	if err != nil {
		return fail(stderr, err)
	}
	if err := releasesig.Verify(pubs, manifest, sig); err != nil {
		return fail(stderr, err)
	}
	sums, err := releasesig.ParseManifest(manifest, tag)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "gültig: %d Dateien für %s\n", len(sums), tag)
	return exitOK
}

// parseFlags liest die Optionen. Bei Fehler oder Hilfe ist done = true.
//
// @param fs – Optionssatz
// @param args – Argumente
// @return int – Exit-Code, falls done (0 bei -h, sonst 2)
// @return bool – true, wenn run sofort enden soll
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func parseFlags(fs *flag.FlagSet, args []string) (int, bool) {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return exitOK, false
	case errors.Is(err, flag.ErrHelp):
		return exitOK, true
	default:
		// Die Meldung hat das flag-Paket schon nach stderr geschrieben
		return exitUsage, true
	}
}

// parsePublicKeys wandelt "<hex>[,<hex>…]" in öffentliche Schlüssel um.
// Ein ungültiger Eintrag wird nur mit Position und Länge gemeldet, nie mit
// seinem Wert (falls versehentlich ein privater Seed übergeben wurde).
//
// @param list – Schlüsselliste aus -pubkeys
// @return []ed25519.PublicKey – Schlüssel (mindestens einer)
// @return error – fehlende Liste, leerer Eintrag, kein Hex oder falsche Länge
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 1: Wert nie in der Meldung)
func parsePublicKeys(list string) ([]ed25519.PublicKey, error) {
	if list == "" {
		return nil, errors.New("-pubkeys <hex>[,<hex>] fehlt")
	}
	var pubs []ed25519.PublicKey
	for i, item := range strings.Split(list, ",") {
		raw, err := hex.DecodeString(item)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("-pubkeys: Eintrag %d ungültig (Länge %d, erwartet %d Hex-Zeichen)",
				i+1, len(item), ed25519.PublicKeySize*2)
		}
		pubs = append(pubs, ed25519.PublicKey(raw))
	}
	return pubs, nil
}

// readLimited liest eine Datei bis höchstens releasesig.MaxManifestBytes.
//
// @param path – Datei
// @return []byte – Inhalt
// @return error – Lesefehler oder Datei zu groß
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, releasesig.MaxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > releasesig.MaxManifestBytes {
		return nil, fmt.Errorf("%s ist zu groß", path)
	}
	return data, nil
}

// usage meldet einen falschen Aufruf.
//
// @param stderr – Ausgabe für Fehlermeldungen
// @param msg – Meldung
// @return int – exitUsage
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func usage(stderr io.Writer, msg string) int {
	fmt.Fprintln(stderr, "release-sign:", msg)
	return exitUsage
}

// fail meldet einen Fehler beim Ausführen.
//
// @param stderr – Ausgabe für Fehlermeldungen
// @param err – Fehler
// @return int – exitError
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (neu)
func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "release-sign:", err)
	return exitError
}
