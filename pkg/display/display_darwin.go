//go:build darwin

package display

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// macOS: displays via system_profiler, capture via the built-in
// `screencapture` (needs Screen Recording permission for the server).
// Untested on real hardware in this project's CI.

const captureSupported = true

var resRe = regexp.MustCompile(`(\d+)\s*x\s*(\d+)(?:\s*@\s*([\d.]+)\s*Hz)?`)

func list() ([]Display, error) {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType", "-json").Output()
	if err != nil {
		return nil, err
	}
	var data struct {
		Items []struct {
			Displays []map[string]any `json:"spdisplays_ndrvs"`
		} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	var displays []Display
	for _, gpu := range data.Items {
		for _, m := range gpu.Displays {
			d := Display{Index: len(displays)}
			d.Name, _ = m["_name"].(string)
			res, _ := m["_spdisplays_resolution"].(string) // "2560 x 1440 @ 60.00Hz"
			if res == "" {
				res, _ = m["_spdisplays_pixels"].(string)
			}
			if mm := resRe.FindStringSubmatch(res); mm != nil {
				d.Width, _ = strconv.Atoi(mm[1])
				d.Height, _ = strconv.Atoi(mm[2])
				if hz, err := strconv.ParseFloat(mm[3], 64); err == nil {
					d.RefreshHz = int(hz + 0.5)
				}
			}
			main, _ := m["spdisplays_main"].(string)
			d.Primary = strings.Contains(main, "yes")
			displays = append(displays, d)
		}
	}
	return displays, nil
}

func capture(index, maxWidth int) ([]byte, error) {
	f := filepath.Join(os.TempDir(), fmt.Sprintf("pcmon-shot-%d.jpg", os.Getpid()))
	defer os.Remove(f)
	// -x: no sound; -D: 1-based display number.
	if out, err := exec.Command("screencapture", "-x", "-t", "jpg", "-D", strconv.Itoa(index+1), f).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("screencapture: %v %s", err, out)
	}
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return downscaleJPEG(data, maxWidth)
}
