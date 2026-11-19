//go:build darwin

package deck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Installed apps are the .app bundles in the standard Applications folders.
// Untested on real hardware in this project's CI.

func listApps() ([]App, error) {
	dirs := []string{"/Applications", "/Applications/Utilities", "/System/Applications", "/System/Applications/Utilities"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}
	var apps []App
	for _, dir := range dirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.app"))
		for _, m := range matches {
			apps = append(apps, App{ID: m, Name: strings.TrimSuffix(filepath.Base(m), ".app")})
		}
	}
	return apps, nil
}

func launchApp(a App) error { return exec.Command("open", "-a", a.ID).Start() }

// typeText passes the text as an argument (not inside the script), so it
// can't break out of the AppleScript.
func typeText(s string) error {
	out, err := exec.Command("osascript",
		"-e", "on run argv",
		"-e", `tell application "System Events" to keystroke (item 1 of argv)`,
		"-e", "end run", "--", s).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}
