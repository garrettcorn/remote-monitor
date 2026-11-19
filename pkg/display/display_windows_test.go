//go:build windows

package display

import (
	"testing"
	"unsafe"
)

// Windows rejects or misreads these structs if their sizes are wrong.
func TestWindowsStructSizes(t *testing.T) {
	cases := map[string][2]uintptr{
		"DEVMODEW":        {unsafe.Sizeof(devMode{}), 220},
		"DISPLAY_DEVICEW": {unsafe.Sizeof(displayDevice{}), 840},
		"MONITORINFOEXW":  {unsafe.Sizeof(monitorInfoEx{}), 104},
		"BITMAPINFO":      {unsafe.Sizeof(bitmapInfo{}), 44},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s is %d bytes, want %d", name, c[0], c[1])
		}
	}
}

func TestHardwareKey(t *testing.T) {
	got := hardwareKey(`\\?\DISPLAY#GSM5B7F#5&1a2b3c&0&UID4352#{e6f07b5f-ee97-4a90-b076-33f57bf4eaa7}`)
	if want := `DISPLAY\GSM5B7F\5&1A2B3C&0&UID4352`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if hardwareKey("bogus") != "" {
		t.Fatal("expected empty key for malformed path")
	}
}
