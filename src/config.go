// Persistente Konfigurationsverwaltung für MD Reader.
//
// Speichert Anwendungseinstellungen (zuletzt geöffnete Datei, Schriftgröße,
// Theme, Layout, Text-Modus) in einer JSON-Datei im systemüblichen Konfigurationsverzeichnis.
//
// Pfade:
//   - Linux:   ~/.config/md-reader/state.json
//   - Windows: %APPDATA%\md-reader\state.json
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24 (Text-Modus mit Farben)
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"md-reader/ui"
)

// AppConfig speichert die persistenten Anwendungseinstellungen.
type AppConfig struct {
	// LastFile ist der vollständige Pfad der zuletzt geöffneten Datei.
	LastFile string `json:"lastFile"`
	// FontSize ist die gespeicherte Schriftgröße in Pixeln.
	FontSize int `json:"fontSize"`
	// Theme ist das aktive Farbschema: "light", "dark" oder "retro".
	Theme string `json:"theme"`
	// Layout ist die Ausrichtung: "landscape" oder "portrait".
	Layout string `json:"layout"`
	// ScrollHistory speichert die Scroll-Position je Dateiinhalt-Hash.
	// Schlüssel: FNV-64a-Hash des Dateiinhalts, Wert: scrollTop in px.
	// Begrenzt auf 200 Einträge um die Config-Datei klein zu halten.
	ScrollHistory map[string]int `json:"scrollHistory,omitempty"`
	// TextMode gibt an, ob der Text-Modus (C64-Pixelschrift) eingeschaltet ist.
	TextMode bool `json:"textMode"`
	// TextModeForeground ist die Schriftfarbe des Text-Modus ("#rrggbb").
	TextModeForeground string `json:"textModeForeground,omitempty"`
	// TextModeBackground ist die Hintergrundfarbe des Text-Modus ("#rrggbb").
	TextModeBackground string `json:"textModeBackground,omitempty"`
}

// textModeSettings liefert die geprüften Text-Modus-Einstellungen.
//
// @return Text-Modus mit garantiert gültigen Farben.
// @lastModified 2026-10-24
func (c AppConfig) textModeSettings() ui.TextMode {
	return ui.NewTextMode(c.TextMode, c.TextModeForeground, c.TextModeBackground)
}

// setTextMode übernimmt Text-Modus-Einstellungen (Farben werden geprüft,
// ungültige durch die Standardfarben ersetzt).
//
// @param enabled    Modus eingeschaltet?
// @param foreground Schriftfarbe.
// @param background Hintergrundfarbe.
// @lastModified 2026-10-24
func (c *AppConfig) setTextMode(enabled bool, foreground, background string) {
	m := ui.NewTextMode(enabled, foreground, background)
	c.TextMode = m.Enabled
	c.TextModeForeground = m.Foreground
	c.TextModeBackground = m.Background
}

// defaultConfig erzeugt eine AppConfig mit Standardwerten.
//
// @return Konfiguration mit Standardeinstellungen.
func defaultConfig() AppConfig {
	return AppConfig{
		FontSize:           defaultFontSize,
		Theme:              "light",
		Layout:             "landscape",
		TextModeForeground: ui.DefaultTextModeForeground,
		TextModeBackground: ui.DefaultTextModeBackground,
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
	// Text-Modus-Farben prüfen (landen in CSS/JS der Seite)
	cfg.setTextMode(cfg.TextMode, cfg.TextModeForeground, cfg.TextModeBackground)
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
