//go:build !windows

package service

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// Writes only inside a temporary HOME / XDG_CONFIG_HOME.
func TestAutostartRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir+"/config")
	t.Setenv("APPIMAGE", `/opt/apps/PC Monitor "beta".AppImage`)

	if AutostartEnabled() {
		t.Fatal("unexpectedly enabled")
	}
	if err := AutostartEnable(); err != nil {
		t.Fatal(err)
	}
	if !AutostartEnabled() {
		t.Fatal("not enabled after AutostartEnable")
	}
	data, err := os.ReadFile(autostartFile())
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if runtime.GOOS == "darwin" {
		if !strings.Contains(s, "&#34;beta&#34;") || !strings.Contains(s, "<string>--background</string>") {
			t.Fatalf("plist not escaped/complete:\n%s", s)
		}
	} else if !strings.Contains(s, `Exec="/opt/apps/PC Monitor \"beta\".AppImage" start`) {
		t.Fatalf("desktop entry Exec not quoted correctly:\n%s", s)
	}
	if err := AutostartDisable(); err != nil {
		t.Fatal(err)
	}
	if AutostartEnabled() {
		t.Fatal("still enabled after AutostartDisable")
	}
}
