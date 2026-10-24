// Antwortspeicher des Starters: merkt sich, ob der Nutzer die
// Installationsfrage mit "Später" oder "Nie" beantwortet hat.
//
// Die Datei liegt in $XDG_CONFIG_HOME/md-reader/launcher.json und ist bewusst
// getrennt von state.json der Anwendung.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Aufgabe 3: Installieren, Entfernen, Entscheiden)
package linuxsetup

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Antworten auf die Installationsfrage.
const (
	// PromptLater: beim nächsten Start wieder fragen.
	PromptLater = "later"
	// PromptNever: nie wieder fragen.
	PromptNever = "never"
)

// Prefs sind die gespeicherten Antworten des Starters.
type Prefs struct {
	// InstallPrompt ist "", "later" oder "never".
	InstallPrompt string `json:"installPrompt"`
}

// LoadPrefs liest die Datei. Fehlt sie, ist sie kaputt oder enthält sie einen
// unbekannten Wert, kommt Prefs{} zurück (gilt als "noch nicht beantwortet").
func LoadPrefs(path string) Prefs {
	data, err := os.ReadFile(path)
	if err != nil {
		return Prefs{}
	}
	var p Prefs
	if err := json.Unmarshal(data, &p); err != nil {
		return Prefs{}
	}
	if p.InstallPrompt != PromptLater && p.InstallPrompt != PromptNever {
		return Prefs{}
	}
	return p
}

// SavePrefs schreibt die Datei. Das Verzeichnis wird mit 0700, die Datei mit
// 0600 angelegt; der Dateimodus wird auch bei einer bestehenden Datei gesetzt.
func SavePrefs(path string, p Prefs) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
