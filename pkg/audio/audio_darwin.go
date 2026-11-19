//go:build darwin

package audio

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// macOS support via AppleScript: system output/input volume and output
// mute only. Switching devices is not available without extra tools.
// Untested on real hardware in this project's CI.

const (
	idOutput = "default-output"
	idInput  = "default-input"
)

func osascript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	return strings.TrimSpace(string(out)), err
}

func get() (*State, error) {
	// e.g. "output volume:50, input volume:75, alert volume:100, output muted:false"
	out, err := osascript("get volume settings")
	if err != nil {
		return nil, err
	}
	vals := map[string]string{}
	for _, part := range strings.Split(out, ",") {
		if k, v, ok := strings.Cut(strings.TrimSpace(part), ":"); ok {
			vals[k] = v
		}
	}
	outVol, _ := strconv.Atoi(vals["output volume"])
	inVol, _ := strconv.Atoi(vals["input volume"])
	return &State{
		Outputs: []Device{{ID: idOutput, Name: "System Output", Default: true, Volume: outVol, Muted: vals["output muted"] == "true"}},
		Inputs:  []Device{{ID: idInput, Name: "System Input", Default: true, Volume: inVol}},
	}, nil
}

func setVolume(id string, level int) error {
	switch id {
	case idOutput:
		_, err := osascript("set volume output volume " + strconv.Itoa(level))
		return err
	case idInput:
		_, err := osascript("set volume input volume " + strconv.Itoa(level))
		return err
	}
	return errors.New("unknown device")
}

func setMute(id string, muted bool) error {
	if id != idOutput {
		return errors.New("mute is only supported for the output device on macOS")
	}
	script := "set volume without output muted"
	if muted {
		script = "set volume with output muted"
	}
	_, err := osascript(script)
	return err
}

func setDefault(string) error {
	return errors.New("switching audio devices is not supported on macOS")
}
