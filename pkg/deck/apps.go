package deck

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// App is an application installed on the host that buttons can launch.
// Dashboards refer to apps only by ID; the server re-checks the ID against
// its own discovered list before launching, so no paths come from clients.
type App struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const appsCacheTTL = 5 * time.Minute

var (
	appsMu    sync.Mutex
	appsCache []App
	appsAt    time.Time
)

// Entries that aren't useful as buttons.
var appNoise = regexp.MustCompile(`(?i)\b(uninstall|uninstaller|read ?me|release notes|documentation|license|help|website|manual)\b`)

// ListApps returns installed apps sorted by name (cached for 5 minutes).
func ListApps() ([]App, error) {
	appsMu.Lock()
	defer appsMu.Unlock()
	if appsCache != nil && time.Since(appsAt) < appsCacheTTL {
		return appsCache, nil
	}
	raw, err := listApps()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	apps := make([]App, 0, len(raw))
	for _, a := range raw {
		a.Name = strings.TrimSpace(a.Name)
		if a.ID == "" || a.Name == "" || seen[a.ID] || appNoise.MatchString(a.Name) {
			continue
		}
		seen[a.ID] = true
		apps = append(apps, a)
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	appsCache, appsAt = apps, time.Now()
	return apps, nil
}

// launchInstalledApp launches an app only if its ID is in the discovered list.
func launchInstalledApp(id string) error {
	apps, err := ListApps()
	if err != nil {
		return err
	}
	for _, a := range apps {
		if a.ID == id {
			return launchApp(a)
		}
	}
	// The list may be stale (app just installed): refresh once.
	appsMu.Lock()
	appsCache = nil
	appsMu.Unlock()
	if apps, err = ListApps(); err == nil {
		for _, a := range apps {
			if a.ID == id {
				return launchApp(a)
			}
		}
	}
	return errors.New("that app isn't installed on this computer")
}

// MaxTextLen bounds text-snippet buttons.
const MaxTextLen = 1000

// validateText allows printable text plus newlines and tabs.
func validateText(s string) error {
	if s == "" {
		return errors.New("text is empty")
	}
	if !utf8.ValidString(s) {
		return errors.New("text is not valid UTF-8")
	}
	if utf8.RuneCountInString(s) > MaxTextLen {
		return errors.New("text is too long (max 1000 characters)")
	}
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' || r == 0x7f {
			return errors.New("text contains control characters")
		}
	}
	return nil
}
