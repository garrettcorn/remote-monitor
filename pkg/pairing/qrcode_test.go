package pairing

import "testing"

func TestEncodeQR(t *testing.T) {
	matrix, err := EncodeQR("https://example.com/#pair=tc123456")
	if err != nil {
		t.Fatalf("EncodeQR failed: %v", err)
	}
	if len(matrix) == 0 {
		t.Fatal("empty matrix")
	}
	ansi := RenderTerminalQR(matrix)
	if len(ansi) == 0 {
		t.Fatal("empty ANSI output")
	}
	svg := GenerateSVG(matrix, 4)
	if len(svg) == 0 {
		t.Fatal("empty SVG output")
	}
}
