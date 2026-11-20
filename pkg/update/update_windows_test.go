//go:build windows

package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// lock opens path the way a running image is held: readable, but it can't be
// deleted or renamed until the handle closes.
func lock(t *testing.T, path string) func() {
	t.Helper()
	p, _ := windows.UTF16PtrFromString(path)
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return func() { windows.CloseHandle(h) }
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The reported bug: pc_monitor-app.exe.old is still in use by a tray icon
// from an earlier version, so the next update can't reuse that name.
func TestReplaceFilesWithLockedOld(t *testing.T) {
	dir := t.TempDir()
	console := filepath.Join(dir, "pc_monitor.exe")
	app := filepath.Join(dir, "pc_monitor-app.exe")
	os.WriteFile(console, []byte("console-1"), 0o755)
	os.WriteFile(app, []byte("app-1"), 0o755)
	os.WriteFile(app+".old", []byte("app-0 (tray still running)"), 0o755)
	unlock := lock(t, app+".old")
	defer unlock()

	if err := replaceFiles([]newFile{{console, []byte("console-2")}, {app, []byte("app-2")}}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if read(t, console) != "console-2" || read(t, app) != "app-2" {
		t.Fatalf("not updated: %q %q", read(t, console), read(t, app))
	}
	asides, _ := filepath.Glob(app + ".old-*")
	if len(asides) != 1 || read(t, asides[0]) != "app-1" {
		t.Fatalf("expected the old app under a unique .old-* name, got %v", asides)
	}

	// Cleanup removes what it can and leaves the locked file alone.
	cleanupDir(dir)
	left, _ := filepath.Glob(filepath.Join(dir, "*.old*"))
	if len(left) != 1 || !strings.HasSuffix(left[0], "pc_monitor-app.exe.old") {
		t.Fatalf("after cleanup: %v", left)
	}
	unlock()
	cleanupDir(dir)
	if left, _ := filepath.Glob(filepath.Join(dir, "*.old*")); len(left) != 0 {
		t.Fatalf("after unlock: %v", left)
	}
}

// If the second file can't be swapped, the first is put back: never a
// half-updated install.
func TestReplaceFilesRollsBack(t *testing.T) {
	dir := t.TempDir()
	console := filepath.Join(dir, "pc_monitor.exe")
	app := filepath.Join(dir, "pc_monitor-app.exe")
	os.WriteFile(console, []byte("console-1"), 0o755)
	os.WriteFile(app, []byte("app-1"), 0o755)
	unlock := lock(t, app) // can't be renamed aside
	defer unlock()

	if err := replaceFiles([]newFile{{console, []byte("console-2")}, {app, []byte("app-2")}}); err == nil {
		t.Fatal("expected an error")
	}
	if read(t, console) != "console-1" || read(t, app) != "app-1" {
		t.Fatalf("not rolled back: %q %q", read(t, console), read(t, app))
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*.new-*")); len(left) != 0 {
		t.Fatalf("staged files left behind: %v", left)
	}
}
