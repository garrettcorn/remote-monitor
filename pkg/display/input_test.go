package display

import (
	"math"
	"testing"
)

func TestPointerValidate(t *testing.T) {
	e := PointerEvent{Type: "scroll", X: 0.5, Y: 0.5, DY: -50}
	if err := e.Validate(); err != nil || e.Button != "left" || e.DY != -maxScrollNotches {
		t.Fatalf("normalise: %+v %v", e, err)
	}
	for _, bad := range []PointerEvent{
		{Type: "warp"},
		{Type: "move", X: -0.1},
		{Type: "move", Y: 1.01},
		{Type: "move", X: math.NaN()},
		{Type: "click", Button: "back"},
	} {
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestToPixel(t *testing.T) {
	cases := []struct {
		origin, w int
		f         float64
		want      int
	}{
		{0, 1920, 0, 0}, {0, 1920, 1, 1919}, {-1920, 1920, 0.5, -960}, {100, 10, 1, 109},
	}
	for _, c := range cases {
		if got := toPixel(c.origin, c.w, c.f); got != c.want {
			t.Errorf("toPixel(%d,%d,%v)=%d want %d", c.origin, c.w, c.f, got, c.want)
		}
	}
}
