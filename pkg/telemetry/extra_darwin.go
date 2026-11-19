//go:build darwin

package telemetry

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// macOS implementations: battery via pmset, users via who. Physical disk
// health and connected devices are not collected on macOS yet.
// Best effort; not exercised by this project's CI on real hardware.

// e.g. " -InternalBattery-0 (id=1234)	87%; charging; 1:02 remaining present: true"
var pmsetBatt = regexp.MustCompile(`(\d+)%;\s*([a-zA-Z ]+);`)

func collectBattery() *BatteryInfo {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return nil
	}
	m := pmsetBatt.FindStringSubmatch(string(out))
	if m == nil {
		return nil
	}
	pct, _ := strconv.Atoi(m[1])
	state := strings.TrimSpace(m[2]) // charging, discharging, charged, AC attached
	ac := strings.Contains(string(out), "AC Power")
	return &BatteryInfo{Percent: pct, State: state, Charging: state == "charging", PluggedIn: ac}
}

func collectPhysicalDisks() ([]PhysicalDisk, bool) { return nil, false }

func collectDevices() []ConnectedDevice { return nil }

func collectUsers() []UserSession { return parseWho() }
