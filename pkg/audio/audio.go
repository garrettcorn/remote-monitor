// Package audio reports and controls the host's audio endpoints
// (output/input devices, volume, mute, default device).
package audio

import "errors"

// Device is one audio endpoint.
type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
	Volume  int    `json:"volume"` // 0-100
	Muted   bool   `json:"muted"`
}

// State is the full audio picture of the host.
type State struct {
	Outputs   []Device `json:"outputs"`
	Inputs    []Device `json:"inputs"`
	CanSwitch bool     `json:"can_switch"` // default device can be changed
}

// ErrUnsupported is returned on platforms without audio support.
var ErrUnsupported = errors.New("audio control is not supported on this platform")

// Get returns the current audio state.
func Get() (*State, error) { return get() }

// SetVolume sets a device's volume (0-100).
func SetVolume(id string, level int) error {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	return setVolume(id, level)
}

// SetMute mutes or unmutes a device.
func SetMute(id string, muted bool) error { return setMute(id, muted) }

// SetDefault makes a device the default for its direction.
func SetDefault(id string) error { return setDefault(id) }
