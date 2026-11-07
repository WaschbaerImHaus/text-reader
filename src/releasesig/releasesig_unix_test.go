//go:build unix

// Unix-eigene Tests für releasesig: Eine benannte Pipe (FIFO) als
// Schlüsseldatei darf LoadPrivateKey nicht blockieren (os.Open auf eine
// FIFO wartet sonst endlos auf einen Schreiber).
//
// Eigene Datei, weil syscall.Mkfifo unter Windows fehlt (GOOS=windows vet).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-07 (Review-Runde 2: FIFO darf nicht hängen)
package releasesig_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"md-reader/releasesig"
)

// TestLoadPrivateKey_FifoDoesNotHang: Eine FIFO wird abgelehnt, ohne zu warten.
func TestLoadPrivateKey_FifoDoesNotHang(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "release-signing.key")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("Mkfifo nicht möglich: %v", err)
	}
	// Hängt LoadPrivateKey doch in os.Open, gibt ein nicht blockierender
	// Schreiber die Gegenseite frei, damit die Goroutine nicht hängen bleibt
	t.Cleanup(func() {
		if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			w.Close()
		}
	})

	done := make(chan error, 1)
	go func() {
		_, err := releasesig.LoadPrivateKey(fifo)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO wurde als Schlüsseldatei angenommen")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("LoadPrivateKey hängt an einer FIFO (länger als 2 s)")
	}
}
