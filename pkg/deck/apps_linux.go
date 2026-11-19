//go:build linux

package deck

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Installed apps from freedesktop .desktop entries; launched with gtk-launch
// when available, else by running the entry's Exec line (field codes
// removed). Untested on real hardware in this project's CI.

func desktopDirs() []string {
	dirs := []string{"/usr/share/applications", "/usr/local/share/applications", "/var/lib/flatpak/exports/share/applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local/share/applications"))
	}
	return dirs
}

type desktopEntry struct{ name, exec string }

func parseDesktop(path string) (desktopEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return desktopEntry{}, false
	}
	defer f.Close()
	var e desktopEntry
	inMain, hidden := false, false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "[") {
			inMain = line == "[Desktop Entry]"
			continue
		}
		if !inMain {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "Name":
			e.name = v
		case "Exec":
			e.exec = v
		case "NoDisplay", "Hidden":
			hidden = hidden || v == "true"
		case "Type":
			hidden = hidden || v != "Application"
		}
	}
	return e, !hidden && e.name != "" && e.exec != ""
}

func listApps() ([]App, error) {
	var apps []App
	for _, dir := range desktopDirs() {
		files, _ := filepath.Glob(filepath.Join(dir, "*.desktop"))
		for _, f := range files {
			if e, ok := parseDesktop(f); ok {
				apps = append(apps, App{ID: strings.TrimSuffix(filepath.Base(f), ".desktop"), Name: e.name})
			}
		}
	}
	return apps, nil
}

func launchApp(a App) error {
	if _, err := exec.LookPath("gtk-launch"); err == nil {
		return exec.Command("gtk-launch", a.ID).Start()
	}
	for _, dir := range desktopDirs() {
		if e, ok := parseDesktop(filepath.Join(dir, a.ID+".desktop")); ok {
			var args []string
			for _, f := range strings.Fields(e.exec) {
				if !strings.HasPrefix(f, "%") {
					args = append(args, strings.Trim(f, `"`))
				}
			}
			if len(args) > 0 {
				return exec.Command(args[0], args[1:]...).Start()
			}
		}
	}
	return os.ErrNotExist
}

func typeText(s string) error { return run("xdotool", "type", "--delay", "8", "--", s) }
