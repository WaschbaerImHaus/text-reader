//go:build !linux

// Platzhalter für alle Systeme außer Linux: Den Starter gibt es nur unter
// Linux (memfd_create und /proc/self/fd). Diese Datei sorgt nur dafür, dass
// das Paket überall übersetzbar bleibt (z. B. "GOOS=windows go vet").
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Aufgabe 6, Korrekturrunde 1: Angebotsprozess)
package main

import "errors"

// errOnlyLinux meldet, dass der Starter nur unter Linux arbeitet.
var errOnlyLinux = errors.New("der Starter von MD Reader läuft nur unter Linux")

// createMemFile gibt es nur unter Linux.
func createMemFile(data []byte) (fd int, err error) {
	return -1, errOnlyLinux
}

// traceMissing gibt es nur unter Linux.
func traceMissing(fd int) ([]string, error) {
	return nil, &execStartError{err: errOnlyLinux}
}

// execGUI gibt es nur unter Linux.
func execGUI(fd int, rest []string) error {
	return errOnlyLinux
}

// spawnOfferProcess gibt es nur unter Linux.
func spawnOfferProcess() error {
	return errOnlyLinux
}

// startDetachedOffer gibt es nur unter Linux.
func startDetachedOffer() error {
	return errOnlyLinux
}
