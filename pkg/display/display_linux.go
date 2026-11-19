//go:build linux

package display

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Linux: display list via xrandr (X11). Screen capture is not implemented.
// Untested on real hardware in this project's CI.

const captureSupported = false

// e.g. "HDMI-1 connected primary 2560x1440+0+0 (normal left inverted ...) 597mm x 336mm"
var outputRe = regexp.MustCompile(`^(\S+) connected( primary)? (\d+)x(\d+)\+(-?\d+)\+(-?\d+)`)

// e.g. "   2560x1440     59.95*+ 143.97"
var modeRe = regexp.MustCompile(`([\d.]+)\*`)

func list() ([]Display, error) {
	out, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return nil, err
	}
	var displays []Display
	var cur *Display
	for _, line := range strings.Split(string(out), "\n") {
		if m := outputRe.FindStringSubmatch(line); m != nil {
			d := Display{Index: len(displays), Name: m[1], Device: m[1], Primary: m[2] != ""}
			d.Width, _ = strconv.Atoi(m[3])
			d.Height, _ = strconv.Atoi(m[4])
			d.X, _ = strconv.Atoi(m[5])
			d.Y, _ = strconv.Atoi(m[6])
			displays = append(displays, d)
			cur = &displays[len(displays)-1]
			continue
		}
		if !strings.HasPrefix(line, " ") {
			cur = nil
			continue
		}
		if cur != nil && cur.RefreshHz == 0 {
			if m := modeRe.FindStringSubmatch(line); m != nil {
				if hz, err := strconv.ParseFloat(m[1], 64); err == nil {
					cur.RefreshHz = int(hz + 0.5)
				}
			}
		}
	}
	return displays, nil
}

func capture(int, int) ([]byte, error) { return nil, ErrUnsupported }
