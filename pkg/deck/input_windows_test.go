//go:build windows

package deck

import "testing"

// Mapping only: nothing is pressed.
func TestKeyVKLoneModifiers(t *testing.T) {
	for k, want := range map[string]uint16{"win": 0x5B, "ctrl": 0x11, "shift": 0x10, "alt": 0x12, "enter": 0x0D, "a": 'A'} {
		if got, err := keyVK(k); err != nil || got != want {
			t.Errorf("keyVK(%q) = %#x, %v; want %#x", k, got, err, want)
		}
	}
	hk, err := ParseHotkey("win")
	if err != nil {
		t.Fatal(err)
	}
	if vk, _ := keyVK(hk.Key); vk != 0x5B || len(hk.Mods) != 0 {
		t.Fatalf("win parses to %+v (vk %#x)", hk, vk)
	}
	if ev := keyEvent(0x5B, false); ev.dwFlags&keyeventfExtended == 0 {
		t.Fatal("the Windows key must be sent as an extended key")
	}
}
