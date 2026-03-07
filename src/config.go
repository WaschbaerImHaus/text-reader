// Persistente Konfigurationsverwaltung für MD Reader.
//
// Speichert Anwendungseinstellungen (zuletzt geöffnete Datei, Schriftgröße,
// Theme, Layout) in einer JSON-Datei im systemüblichen Konfigurationsverzeichnis.
//
// Pfade:
//   - Linux:   ~/.config/md-reader/state.json
//   - Windows: %APPDATA%\md-reader\state.json
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// AppConfig speichert die persistenten Anwendungseinstellungen.
type AppConfig struct {
	// LastFile ist der vollständige Pfad der zuletzt geöffneten Datei.
	// Wird nur gesetzt wenn die Datei per CLI-Argument oder Startup geöffnet wurde.
	LastFile string `json:"lastFile"`
	// FontSize ist die gespeicherte Schriftgröße in Pixeln.
	FontSize int `json:"fontSize"`
	// Theme ist das aktive Farbschema: "light", "dark" oder "retro".
	Theme string `json:"theme"`
	// Layout ist die Ausrichtung: "landscape" oder "portrait".
	Layout string `json:"layout"`
}

// defaultConfig erzeugt eine AppConfig mit Standardwerten.
//
// @return Konfiguration mit Standardeinstellungen.
func defaultConfig() AppConfig {
	return AppConfig{
		FontSize: defaultFontSize,
		Theme:    "light",
		Layout:   "landscape",
	}
}

// configFilePath gibt den plattformspezifischen Pfad zur Konfigurationsdatei zurück.
//
// @return Absoluter Pfad zur state.json, oder Fehler.
func configFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "md-reader", "state.json"), nil
}

// loadConfig liest die gespeicherte Konfiguration von Disk.
//
// Gibt die Standardkonfiguration zurück wenn die Datei nicht existiert
// oder nicht gelesen werden kann.
//
// @return Geladene oder Standard-Konfiguration.
func loadConfig() AppConfig {
	cfg := defaultConfig()
	path, err := configFilePath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// Datei existiert nicht → Standardwerte verwenden (kein Fehler)
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Konfiguration konnte nicht geladen werden: %v", err)
		return defaultConfig()
	}
	// Sanity-Checks: ungültige Werte durch Defaults ersetzen
	if cfg.FontSize < 4 || cfg.FontSize > 128 {
		cfg.FontSize = defaultFontSize
	}
	if cfg.Theme != "light" && cfg.Theme != "dark" && cfg.Theme != "retro" {
		cfg.Theme = "light"
	}
	if cfg.Layout != "landscape" && cfg.Layout != "portrait" {
		cfg.Layout = "landscape"
	}
	return cfg
}

// saveConfig schreibt die aktuelle Konfiguration auf Disk.
//
// Legt das Konfigurationsverzeichnis an falls es nicht existiert.
//
// @param cfg Zu speichernde Konfiguration.
func saveConfig(cfg AppConfig) {
	path, err := configFilePath()
	if err != nil {
		log.Printf("Konfigurationspfad konnte nicht ermittelt werden: %v", err)
		return
	}
	// Verzeichnis anlegen falls nicht vorhanden
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("Konfigurationsverzeichnis konnte nicht erstellt werden: %v", err)
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("Konfiguration konnte nicht serialisiert werden: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Konfiguration konnte nicht gespeichert werden: %v", err)
	}
}
