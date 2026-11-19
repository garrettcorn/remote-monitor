package display

import (
	"bytes"
	"image/jpeg"
	"testing"
)

func TestEncodeBGRADownscales(t *testing.T) {
	w, h := 2000, 1000
	pix := make([]byte, w*h*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2] = 255, 0, 0 // pure blue in BGRA
	}
	out, err := encodeBGRA(pix, w, h, 640)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 640 || b.Dy() != 320 {
		t.Fatalf("got %dx%d, want 640x320", b.Dx(), b.Dy())
	}
	r, g, b, _ := img.At(100, 100).RGBA()
	if b>>8 < 200 || r>>8 > 40 || g>>8 > 40 {
		t.Fatalf("colour channels swapped: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
	if _, err := encodeBGRA(pix[:10], w, h, 640); err == nil {
		t.Fatal("short buffer should fail")
	}
}

func TestCaptureRejectsBadIndex(t *testing.T) {
	if !CaptureSupported() {
		t.Skip("capture unsupported here")
	}
	if _, err := Capture(9999, 640); err == nil {
		t.Fatal("expected error for missing display")
	}
}
