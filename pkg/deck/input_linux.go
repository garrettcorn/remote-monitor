//go:build linux

package deck

import (
	"errors"
	"os/exec"
	"strings"
)

// Linux (X11) via xdotool, loginctl and xdg-open. Untested on real hardware
// in this project's CI; Wayland sessions generally block synthetic input.

var mediaKeysym = map[string]string{
	"play_pause": "XF86AudioPlay", "next": "XF86AudioNext", "previous": "XF86AudioPrev",
	"stop": "XF86AudioStop", "volume_mute": "XF86AudioMute",
	"volume_down": "XF86AudioLowerVolume", "volume_up": "XF86AudioRaiseVolume",
}

var modKeysym = map[string]string{"ctrl": "ctrl", "shift": "shift", "alt": "alt", "win": "super"}

var namedKeysym = map[string]string{
	"space": "space", "enter": "Return", "tab": "Tab", "esc": "Escape", "backspace": "BackSpace",
	"delete": "Delete", "insert": "Insert", "home": "Home", "end": "End", "pageup": "Prior",
	"pagedown": "Next", "left": "Left", "up": "Up", "right": "Right", "down": "Down",
	"printscreen": "Print",
}

func run(name string, args ...string) error {
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New(name + " is not installed")
		}
		return errors.New(strings.TrimSpace(string(out)) + " (" + err.Error() + ")")
	}
	return nil
}

func pressMedia(key string) error { return run("xdotool", "key", mediaKeysym[key]) }

func pressHotkey(hk Hotkey) error {
	key := hk.Key
	if k, ok := namedKeysym[key]; ok {
		key = k
	} else if key[0] == 'f' && len(key) > 1 {
		key = "F" + key[1:]
	}
	parts := make([]string, 0, len(hk.Mods)+1)
	for _, m := range hk.Mods {
		parts = append(parts, modKeysym[m])
	}
	parts = append(parts, key)
	return run("xdotool", "key", strings.Join(parts, "+"))
}

func lockScreen() error { return run("loginctl", "lock-session") }

func openURL(u string) error { return exec.Command("xdg-open", u).Start() }

func launch(a LaunchAction) error {
	cmd := exec.Command(a.Path, strings.Fields(a.Args)...)
	cmd.Dir = a.Dir
	return cmd.Start()
}
