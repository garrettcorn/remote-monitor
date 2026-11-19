// Package deck implements Stream Deck-style button actions on the host:
// media keys, keyboard shortcuts, lock screen, mic mute, opening URLs and
// launching apps from a server-side allowlist.
package deck

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"pc_monitor/pkg/audio"
)

// Action is one button press request from the dashboard.
type Action struct {
	Type string `json:"type"`           // "media", "hotkey", "lock", "mic_mute", "url", "launch"
	Key  string `json:"key,omitempty"`  // media key name
	Keys string `json:"keys,omitempty"` // hotkey, e.g. "ctrl+shift+m"
	URL  string `json:"url,omitempty"`  // http(s) URL to open
	ID   string `json:"id,omitempty"`   // launch action id from the actions file
}

// MediaKeys are the supported media/volume key names.
var MediaKeys = []string{"play_pause", "next", "previous", "stop", "volume_up", "volume_down", "volume_mute"}

// Modifiers are the supported hotkey modifiers.
var Modifiers = []string{"ctrl", "shift", "alt", "win"}

// LaunchAction is an app/file/URL the server owner allows dashboards to open.
type LaunchAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Args string `json:"args,omitempty"`
	Dir  string `json:"dir,omitempty"`
}

// LaunchInfo is the public view of a launch action (no path or args).
type LaunchInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type actionsFile struct {
	Launch []LaunchAction `json:"launch"`
}

// LoadLaunchActions reads the allowlist file. A missing file means none.
// It is re-read on every use so edits apply without restarting.
func LoadLaunchActions(path string) ([]LaunchAction, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Windows editors (and PowerShell's UTF8 encoding) often add a BOM.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var f actionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var out []LaunchAction
	for _, a := range f.Launch {
		if a.ID != "" && a.Path != "" {
			if a.Name == "" {
				a.Name = a.ID
			}
			out = append(out, a)
		}
	}
	return out, nil
}

// Hotkey is a parsed keyboard shortcut: modifiers plus exactly one key.
type Hotkey struct {
	Mods []string // subset of Modifiers, in canonical order
	Key  string   // lower-case key name (see keyNames)
}

// keyNames are the non-modifier keys a hotkey may use (besides a-z, 0-9, f1-f24).
var keyNames = map[string]bool{
	"space": true, "enter": true, "tab": true, "esc": true, "backspace": true,
	"delete": true, "insert": true, "home": true, "end": true, "pageup": true,
	"pagedown": true, "up": true, "down": true, "left": true, "right": true,
	"printscreen": true,
}

var keyAliases = map[string]string{
	"control": "ctrl", "cmd": "win", "command": "win", "super": "win", "meta": "win",
	"option": "alt", "escape": "esc", "return": "enter", "del": "delete", "ins": "insert",
	"pgup": "pageup", "pgdn": "pagedown", "arrowup": "up", "arrowdown": "down",
	"arrowleft": "left", "arrowright": "right", "prtsc": "printscreen", " ": "space",
}

// ParseHotkey parses "ctrl+shift+m" style shortcuts.
func ParseHotkey(s string) (Hotkey, error) {
	var hk Hotkey
	mods := map[string]bool{}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	if len(parts) == 0 || len(parts) > 5 {
		return hk, errors.New("invalid shortcut")
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if a, ok := keyAliases[p]; ok {
			p = a
		}
		switch {
		case p == "ctrl" || p == "shift" || p == "alt" || p == "win":
			mods[p] = true
		case hk.Key != "":
			return hk, errors.New("a shortcut can have only one non-modifier key")
		case isKey(p):
			hk.Key = p
		default:
			return hk, fmt.Errorf("unsupported key %q", p)
		}
	}
	if hk.Key == "" {
		return hk, errors.New("shortcut needs a non-modifier key")
	}
	for _, m := range Modifiers {
		if mods[m] {
			hk.Mods = append(hk.Mods, m)
		}
	}
	return hk, nil
}

func isKey(p string) bool {
	if len(p) == 1 && ((p[0] >= 'a' && p[0] <= 'z') || (p[0] >= '0' && p[0] <= '9')) {
		return true
	}
	if len(p) >= 2 && p[0] == 'f' {
		var n int
		if _, err := fmt.Sscanf(p[1:], "%d", &n); err == nil && n >= 1 && n <= 24 && fmt.Sprint(n) == p[1:] {
			return true
		}
	}
	return keyNames[p]
}

func isMediaKey(k string) bool {
	for _, m := range MediaKeys {
		if m == k {
			return true
		}
	}
	return false
}

// Press validates and performs an action. launchFile is the allowlist path.
func Press(a Action, launchFile string) error {
	switch a.Type {
	case "media":
		if !isMediaKey(a.Key) {
			return fmt.Errorf("unknown media key %q", a.Key)
		}
		return pressMedia(a.Key)
	case "hotkey":
		hk, err := ParseHotkey(a.Keys)
		if err != nil {
			return err
		}
		return pressHotkey(hk)
	case "lock":
		return lockScreen()
	case "mic_mute":
		return toggleMicMute()
	case "url":
		u, err := url.Parse(strings.TrimSpace(a.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("only http:// and https:// URLs can be opened")
		}
		return openURL(u.String())
	case "launch":
		actions, err := LoadLaunchActions(launchFile)
		if err != nil {
			return err
		}
		for _, la := range actions {
			if la.ID == a.ID {
				return launch(la)
			}
		}
		return fmt.Errorf("no launch action %q in %s", a.ID, launchFile)
	}
	return fmt.Errorf("unknown action type %q", a.Type)
}

// toggleMicMute flips mute on the default input device.
func toggleMicMute() error {
	st, err := audio.Get()
	if err != nil {
		return err
	}
	for _, d := range st.Inputs {
		if d.Default {
			return audio.SetMute(d.ID, !d.Muted)
		}
	}
	return errors.New("no default microphone")
}
