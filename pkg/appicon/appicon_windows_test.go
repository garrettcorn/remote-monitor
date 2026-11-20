//go:build windows

package appicon

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

func TestStructSizes(t *testing.T) {
	if s := unsafe.Sizeof(bitmapInfoHeader{}); s != 40 {
		t.Errorf("BITMAPINFOHEADER is %d bytes, want 40", s)
	}
	if s := unsafe.Sizeof(bitmap{}); s != 32 {
		t.Errorf("BITMAP is %d bytes, want 32", s)
	}
	if s := unsafe.Sizeof(iconInfo{}); s != 32 {
		t.Errorf("ICONINFO is %d bytes, want 32", s)
	}
}

func TestPNGFromSystemExe(t *testing.T) {
	notepad := filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe")
	b, err := PNG(notepad)
	if err != nil {
		t.Skipf("no icon for %s: %v", notepad, err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() < 16 || img.Bounds().Dx() > 256 {
		t.Fatalf("size %v", img.Bounds())
	}
	// Some pixels visible: a real icon, not an empty image.
	opaque := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				opaque++
			}
		}
	}
	if opaque == 0 {
		t.Fatal("icon is fully transparent")
	}
	if _, err := PNG(filepath.Join(t.TempDir(), "missing.exe")); err == nil {
		t.Fatal("expected no icon for a missing file")
	}
}
