//go:build darwin

package deck

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// macOS via AppleScript (System Events needs Accessibility permission) and
// `open`. Media keys are not supported. Untested on real hardware in CI.

var macKeyCode = map[string]int{
	"space": 49, "enter": 36, "tab": 48, "esc": 53, "backspace": 51, "delete": 117,
	"home": 115, "end": 119, "pageup": 116, "pagedown": 121, "left": 123,
	"right": 124, "down": 125, "up": 126,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118, "f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
}

var macMod = map[string]string{"ctrl": "control down", "shift": "shift down", "alt": "option down", "win": "command down"}

func pressMedia(string) error { return errors.New("media keys are not supported on macOS") }

func pressHotkey(hk Hotkey) error {
	var using []string
	for _, m := range hk.Mods {
		using = append(using, macMod[m])
	}
	var stroke string
	if code, ok := macKeyCode[hk.Key]; ok {
		stroke = fmt.Sprintf("key code %d", code)
	} else if len(hk.Key) == 1 {
		stroke = fmt.Sprintf("keystroke %q", hk.Key)
	} else {
		return fmt.Errorf("key %q is not supported on macOS", hk.Key)
	}
	if len(using) > 0 {
		stroke += " using {" + strings.Join(using, ", ") + "}"
	}
	out, err := exec.Command("osascript", "-e", `tell application "System Events" to `+stroke).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func lockScreen() error { return exec.Command("pmset", "displaysleepnow").Run() }

func openURL(u string) error { return exec.Command("open", u).Start() }

func launch(a LaunchAction) error {
	args := append([]string{a.Path}, strings.Fields(a.Args)...)
	if a.Args != "" {
		args = append([]string{a.Path, "--args"}, strings.Fields(a.Args)...)
	}
	return exec.Command("open", args...).Start()
}
