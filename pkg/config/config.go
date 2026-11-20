// Package config stores the server's saved settings in
// ~/.pc_monitor/config.json. Command-line flags override saved values.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"pc_monitor/pkg/control"
)

// DefaultWebURL is the hosted dashboard.
const DefaultWebURL = "https://garrettcorn.github.io/remote-monitor/"

// Config is the saved server configuration.
type Config struct {
	Name         string `json:"name"`
	Port         int    `json:"port"`          // direct HTTP/WebSocket port (0 disables)
	TailcatPort  int    `json:"tailcat_port"`  // port inside the Tailcat tunnel
	NoTailcat    bool   `json:"no_tailcat"`    // direct mode only
	WebURL       string `json:"web_url"`       // dashboard base URL for pairing links
	ReadOnly     bool   `json:"read_only"`     // disable remote control
	NoScreen     bool   `json:"no_screen"`     // disable screen previews
	SettingsPort int    `json:"settings_port"` // local settings page (127.0.0.1 only)
	UpdateCheck  bool   `json:"update_check"`  // check GitHub for new releases
	AutoUpdate   bool   `json:"auto_update"`   // install new releases automatically
	ExportHTML   string `json:"export_html"`   // pairing card file ("" to skip)
}

// Dir is the per-user state folder (~/.pc_monitor).
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".pc_monitor")
}

// Path is the config file location.
func Path() string { return filepath.Join(Dir(), "config.json") }

// Defaults returns the built-in settings.
func Defaults() Config {
	host, _ := os.Hostname()
	if host == "" {
		host = "pc-server"
	}
	return Config{
		Name:         host,
		Port:         9120,
		TailcatPort:  8080,
		WebURL:       DefaultWebURL,
		SettingsPort: 9121,
		UpdateCheck:  true,
		ExportHTML:   "",
	}
}

// Load reads the saved config; missing fields keep their defaults and a
// missing file means all defaults.
func Load() (Config, error) {
	c := Defaults()
	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	data = []byte(strings.TrimPrefix(string(data), "\xef\xbb\xbf")) // tolerate a BOM
	if err := json.Unmarshal(data, &c); err != nil {
		return Defaults(), fmt.Errorf("parse %s: %w", Path(), err)
	}
	return c, c.Validate()
}

// Save writes the config atomically (owner-only permissions).
func (c Config) Save() error {
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return control.WriteFileAtomic(Path(), append(data, '\n'))
}

// Validate checks value ranges.
func (c Config) Validate() error {
	for name, p := range map[string]int{"port": c.Port, "tailcat_port": c.TailcatPort, "settings_port": c.SettingsPort} {
		if p < 0 || p > 65535 || (p == 0 && name != "port") {
			return fmt.Errorf("%s must be between 1 and 65535", name)
		}
	}
	if c.SettingsPort == c.Port {
		return errors.New("settings_port must differ from port")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name can't be empty")
	}
	if !strings.HasPrefix(c.WebURL, "https://") && !strings.HasPrefix(c.WebURL, "http://") {
		return errors.New("web_url must start with http:// or https://")
	}
	return nil
}

// fields maps config keys to accessors for `pc_monitor config get/set`.
func (c *Config) fields() map[string]any {
	return map[string]any{
		"name": &c.Name, "port": &c.Port, "tailcat_port": &c.TailcatPort, "no_tailcat": &c.NoTailcat,
		"web_url": &c.WebURL, "read_only": &c.ReadOnly, "no_screen": &c.NoScreen,
		"settings_port": &c.SettingsPort, "update_check": &c.UpdateCheck, "auto_update": &c.AutoUpdate, "export_html": &c.ExportHTML,
	}
}

// Keys lists the setting names.
func Keys() []string {
	var c Config
	keys := make([]string, 0)
	for k := range c.fields() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Get returns one setting as text.
func (c *Config) Get(key string) (string, error) {
	switch v := c.fields()[key].(type) {
	case *string:
		return *v, nil
	case *int:
		return strconv.Itoa(*v), nil
	case *bool:
		return strconv.FormatBool(*v), nil
	}
	return "", fmt.Errorf("unknown setting %q (known: %s)", key, strings.Join(Keys(), ", "))
}

// Set changes one setting from text.
func (c *Config) Set(key, value string) error {
	switch v := c.fields()[key].(type) {
	case *string:
		*v = value
	case *int:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be a number", key)
		}
		*v = n
	case *bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%s must be true or false", key)
		}
		*v = b
	default:
		return fmt.Errorf("unknown setting %q (known: %s)", key, strings.Join(Keys(), ", "))
	}
	return c.Validate()
}
