package display

import (
	"errors"
	"fmt"
	"math"
)

// PointerEvent is a mouse action on one display. X and Y are fractions of
// the display's width and height (0..1), so the dashboard never needs to
// know the host's pixel geometry or scaling.
type PointerEvent struct {
	Type   string  `json:"type"`             // move, down, up, click, dblclick, scroll
	X      float64 `json:"x"`                // 0..1
	Y      float64 `json:"y"`                // 0..1
	Button string  `json:"button,omitempty"` // left (default), right, middle
	DX     int     `json:"dx,omitempty"`     // scroll notches, + = right
	DY     int     `json:"dy,omitempty"`     // scroll notches, + = down
}

// ErrInputUnsupported is returned where remote pointer input isn't available.
var ErrInputUnsupported = errors.New("remote input is not supported on this platform")

// InputSupported reports whether Pointer works on this platform.
func InputSupported() bool { return inputSupported }

const maxScrollNotches = 10

// Validate checks and normalises an event.
func (e *PointerEvent) Validate() error {
	switch e.Type {
	case "move", "down", "up", "click", "dblclick", "scroll":
	default:
		return fmt.Errorf("unknown pointer event %q", e.Type)
	}
	if math.IsNaN(e.X) || math.IsNaN(e.Y) || e.X < 0 || e.X > 1 || e.Y < 0 || e.Y > 1 {
		return errors.New("x and y must be between 0 and 1")
	}
	switch e.Button {
	case "":
		e.Button = "left"
	case "left", "right", "middle":
	default:
		return fmt.Errorf("unknown button %q", e.Button)
	}
	e.DX = max(-maxScrollNotches, min(maxScrollNotches, e.DX))
	e.DY = max(-maxScrollNotches, min(maxScrollNotches, e.DY))
	return nil
}

// toPixel maps a fraction of a w-pixel span starting at origin to a pixel.
func toPixel(origin, w int, f float64) int {
	p := origin + int(math.Round(f*float64(w-1)))
	return max(origin, min(origin+w-1, p))
}

// Pointer moves the mouse to (X, Y) on display index and performs the event.
func Pointer(index int, e PointerEvent) error {
	if !inputSupported {
		return ErrInputUnsupported
	}
	if err := e.Validate(); err != nil {
		return err
	}
	return pointer(index, e)
}
