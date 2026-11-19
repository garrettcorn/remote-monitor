//go:build linux

package audio

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// Linux support via PulseAudio/PipeWire's `pactl` (JSON output needs
// pactl 16+). Device IDs are "sink:<name>" or "source:<name>".
// Untested on real hardware in this project's CI.

type pactlDevice struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mute        bool   `json:"mute"`
	MonitorOf   any    `json:"monitor_of_sink"`
	Volume      map[string]struct {
		ValuePercent string `json:"value_percent"`
	} `json:"volume"`
}

func pactl(args ...string) (string, error) {
	out, err := exec.Command("pactl", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func listPactl(kind, defName string) ([]Device, error) {
	out, err := exec.Command("pactl", "--format=json", "list", kind+"s").Output()
	if err != nil {
		return nil, err
	}
	var raw []pactlDevice
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	var devs []Device
	for _, d := range raw {
		if kind == "source" && (strings.HasSuffix(d.Name, ".monitor") || (d.MonitorOf != nil && d.MonitorOf != "n/a")) {
			continue
		}
		vol, n := 0, 0
		for _, ch := range d.Volume {
			if v, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(ch.ValuePercent), "%")); err == nil {
				vol += v
				n++
			}
		}
		if n > 0 {
			vol /= n
		}
		name := d.Description
		if name == "" {
			name = d.Name
		}
		devs = append(devs, Device{ID: kind + ":" + d.Name, Name: name, Default: d.Name == defName, Volume: vol, Muted: d.Mute})
	}
	return devs, nil
}

func get() (*State, error) {
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil, ErrUnsupported
	}
	defSink, _ := pactl("get-default-sink")
	defSource, _ := pactl("get-default-source")
	outs, err := listPactl("sink", defSink)
	if err != nil {
		return nil, err
	}
	ins, err := listPactl("source", defSource)
	if err != nil {
		return nil, err
	}
	return &State{Outputs: outs, Inputs: ins, CanSwitch: true}, nil
}

func splitID(id string) (kind, name string, err error) {
	kind, name, ok := strings.Cut(id, ":")
	if !ok || (kind != "sink" && kind != "source") || name == "" {
		return "", "", errors.New("invalid device id")
	}
	return kind, name, nil
}

func setVolume(id string, level int) error {
	kind, name, err := splitID(id)
	if err != nil {
		return err
	}
	_, err = pactl("set-"+kind+"-volume", name, strconv.Itoa(level)+"%")
	return err
}

func setMute(id string, muted bool) error {
	kind, name, err := splitID(id)
	if err != nil {
		return err
	}
	v := "0"
	if muted {
		v = "1"
	}
	_, err = pactl("set-"+kind+"-mute", name, v)
	return err
}

func setDefault(id string) error {
	kind, name, err := splitID(id)
	if err != nil {
		return err
	}
	_, err = pactl("set-default-"+kind, name)
	return err
}
