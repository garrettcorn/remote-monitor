//go:build linux

package display

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Linux (X11) pointer input via xdotool. Untested on real hardware; Wayland
// sessions generally block synthetic input.

const inputSupported = true

var xButton = map[string]string{"left": "1", "middle": "2", "right": "3"}

func xdotool(args ...string) error {
	if out, err := exec.Command("xdotool", args...).CombinedOutput(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("xdotool is not installed")
		}
		return errors.New(strings.TrimSpace(string(out)) + " (" + err.Error() + ")")
	}
	return nil
}

func pointer(index int, e PointerEvent) error {
	ds, err := list()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(ds) {
		return fmt.Errorf("no display %d", index)
	}
	d := ds[index]
	args := []string{"mousemove", strconv.Itoa(toPixel(d.X, d.Width, e.X)), strconv.Itoa(toPixel(d.Y, d.Height, e.Y))}
	b := xButton[e.Button]
	switch e.Type {
	case "down":
		args = append(args, "mousedown", b)
	case "up":
		args = append(args, "mouseup", b)
	case "click":
		args = append(args, "click", b)
	case "dblclick":
		args = append(args, "click", "--repeat", "2", b)
	case "scroll":
		// X11 buttons 4/5 scroll up/down, 6/7 left/right.
		if e.DY != 0 {
			btn := "5"
			if e.DY < 0 {
				btn = "4"
			}
			args = append(args, "click", "--repeat", strconv.Itoa(abs(e.DY)), btn)
		}
		if e.DX != 0 {
			btn := "7"
			if e.DX < 0 {
				btn = "6"
			}
			args = append(args, "click", "--repeat", strconv.Itoa(abs(e.DX)), btn)
		}
	}
	return xdotool(args...)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
