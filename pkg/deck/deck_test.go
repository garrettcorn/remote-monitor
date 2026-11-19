package deck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestParseHotkey(t *testing.T) {
	ok := map[string]Hotkey{
		"ctrl+shift+m":    {Mods: []string{"ctrl", "shift"}, Key: "m"},
		"Shift+Ctrl+M":    {Mods: []string{"ctrl", "shift"}, Key: "m"}, // canonical order
		"alt+f4":          {Mods: []string{"alt"}, Key: "f4"},
		"cmd+space":       {Mods: []string{"win"}, Key: "space"},
		"f24":             {Key: "f24"},
		"ctrl+alt+delete": {Mods: []string{"ctrl", "alt"}, Key: "delete"},
		"win+printscreen": {Mods: []string{"win"}, Key: "printscreen"},
		"control+pgdn":    {Mods: []string{"ctrl"}, Key: "pagedown"},
	}
	for in, want := range ok {
		got, err := ParseHotkey(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("ParseHotkey(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ctrl", "ctrl+shift", "a+b", "f0", "f25", "f01", "ctrl+;", "hyper+x"} {
		if _, err := ParseHotkey(bad); err == nil {
			t.Errorf("ParseHotkey(%q) should fail", bad)
		}
	}
}

func TestPressValidation(t *testing.T) {
	// These must be rejected before anything touches the host.
	bad := []Action{
		{Type: "media", Key: "self_destruct"},
		{Type: "hotkey", Keys: "ctrl"},
		{Type: "url", URL: "file:///C:/Windows/System32/calc.exe"},
		{Type: "url", URL: "javascript:alert(1)"},
		{Type: "url", URL: "https://"},
		{Type: "launch", ID: "not-configured"},
		{Type: "shell", Keys: "rm -rf /"},
	}
	for _, a := range bad {
		if err := Press(a, filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Errorf("Press(%+v) should fail", a)
		}
	}
}

func TestLoadLaunchActions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "actions.json")
	if got, err := LoadLaunchActions(path); err != nil || got != nil {
		t.Fatalf("missing file: got %v, %v", got, err)
	}
	data := `{"launch":[{"id":"obs","name":"OBS","path":"C:\\obs.exe"},{"id":"nopath"},{"id":"calc","path":"calc.exe"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLaunchActions(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "obs" || got[1].Name != "calc" {
		t.Fatalf("unexpected actions: %+v", got)
	}
	// A UTF-8 BOM (common from Windows editors/PowerShell) must be accepted.
	if err := os.WriteFile(path, []byte("\xef\xbb\xbf"+data), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadLaunchActions(path); err != nil || len(got) != 2 {
		t.Fatalf("BOM file: got %v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLaunchActions(path); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

// INPUT must be 40 bytes on 64-bit Windows for SendInput to accept it.
func TestKeyboardInputSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit only")
	}
	type layout struct {
		typ   uint32
		_     uint32
		wVk   uint16
		wScan uint16
		flags uint32
		time  uint32
		_     uint32
		extra uint64
		_     [8]byte
	}
	if s := unsafe.Sizeof(layout{}); s != 40 {
		t.Fatalf("INPUT layout is %d bytes, want 40", s)
	}
}
