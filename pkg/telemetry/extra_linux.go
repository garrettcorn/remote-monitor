//go:build linux

package telemetry

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Linux implementations of battery, physical disks (SMART via smartctl when
// available and permitted), connected devices (lsusb) and users (who).
// Best effort; not exercised by this project's CI on real hardware.

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func collectBattery() *BatteryInfo {
	bats, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	if len(bats) == 0 {
		return nil
	}
	pct, err := strconv.Atoi(readTrim(filepath.Join(bats[0], "capacity")))
	if err != nil {
		return nil
	}
	status := strings.ToLower(readTrim(filepath.Join(bats[0], "status"))) // Charging, Discharging, Full, Not charging
	b := &BatteryInfo{Percent: pct, State: status}
	b.Charging = status == "charging"
	b.PluggedIn = status != "discharging"
	return b
}

// smartctl JSON subset.
type smartctlOut struct {
	ModelName    string `json:"model_name"`
	SerialNumber string `json:"serial_number"`
	SmartStatus  *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature *struct {
		Current float64 `json:"current"`
	} `json:"temperature"`
	PowerOnTime *struct {
		Hours uint64 `json:"hours"`
	} `json:"power_on_time"`
	NVMeHealth *struct {
		PercentageUsed float64 `json:"percentage_used"`
		MediaErrors    uint64  `json:"media_errors"`
	} `json:"nvme_smart_health_information_log"`
}

func collectPhysicalDisks() ([]PhysicalDisk, bool) {
	entries, _ := os.ReadDir("/sys/block")
	_, smartErr := exec.LookPath("smartctl")
	details := false
	var disks []PhysicalDisk
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "zram") || strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "sr") {
			continue
		}
		base := filepath.Join("/sys/block", name)
		sectors, _ := strconv.ParseUint(readTrim(filepath.Join(base, "size")), 10, 64)
		if sectors == 0 {
			continue
		}
		d := PhysicalDisk{ID: name, Name: readTrim(filepath.Join(base, "device/model")), SizeBytes: sectors * 512, Health: "Unknown"}
		if d.Name == "" {
			d.Name = name
		}
		d.MediaType = "SSD"
		if readTrim(filepath.Join(base, "queue/rotational")) == "1" {
			d.MediaType = "HDD"
		}
		switch {
		case strings.HasPrefix(name, "nvme"):
			d.BusType = "NVMe"
		case strings.HasPrefix(name, "mmcblk"):
			d.BusType = "MMC"
		default:
			if link, err := os.Readlink(base); err == nil && strings.Contains(link, "/usb") {
				d.BusType = "USB"
			} else {
				d.BusType = "SATA"
			}
		}

		if smartErr == nil {
			out, _ := exec.Command("smartctl", "-a", "-j", "/dev/"+name).Output()
			var s smartctlOut
			if json.Unmarshal(out, &s) == nil && s.SmartStatus != nil {
				details = true
				d.Health = "Healthy"
				if !s.SmartStatus.Passed {
					d.Health = "Unhealthy"
				}
				if s.SerialNumber != "" {
					d.Serial = s.SerialNumber
				}
				if s.Temperature != nil {
					t := s.Temperature.Current
					d.TemperatureC = &t
				}
				if s.PowerOnTime != nil {
					h := s.PowerOnTime.Hours
					d.PowerOnHours = &h
				}
				if s.NVMeHealth != nil {
					w := s.NVMeHealth.PercentageUsed
					d.WearPercent = &w
					me := s.NVMeHealth.MediaErrors
					d.ReadErrors = &me
				}
			}
		}
		disks = append(disks, d)
	}
	return disks, details
}

// e.g. "Bus 001 Device 003: ID 046d:c52b Logitech, Inc. Unifying Receiver"
var lsusbLine = regexp.MustCompile(`^Bus \d+ Device \d+: ID [0-9a-f]{4}:[0-9a-f]{4} (.+)$`)

func collectDevices() []ConnectedDevice {
	out, err := exec.Command("lsusb").Output()
	if err != nil {
		return nil
	}
	var devs []ConnectedDevice
	for _, line := range strings.Split(string(out), "\n") {
		m := lsusbLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil || strings.Contains(strings.ToLower(m[1]), "hub") {
			continue
		}
		devs = append(devs, ConnectedDevice{Name: m[1], Class: "USB", Connection: "USB", Status: "OK"})
	}
	return devs
}

func collectUsers() []UserSession {
	return parseWho()
}
