//go:build windows

package telemetry

import (
	"encoding/json"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------- battery

var procGetSystemPowerStatus = modkernel32.NewProc("GetSystemPowerStatus")

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

func collectBattery() *BatteryInfo {
	var s systemPowerStatus
	if r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&s))); r == 0 {
		return nil
	}
	// 128 = no system battery, 255 = unknown status.
	if s.BatteryFlag&128 != 0 || s.BatteryFlag == 255 || s.BatteryLifePercent > 100 {
		return nil
	}
	b := &BatteryInfo{
		Percent:   int(s.BatteryLifePercent),
		PluggedIn: s.ACLineStatus == 1,
		Charging:  s.BatteryFlag&8 != 0,
	}
	if s.BatteryLifeTime != 0xFFFFFFFF {
		b.SecondsLeft = int(s.BatteryLifeTime)
	}
	switch {
	case b.Charging:
		b.State = "charging"
	case b.PluggedIn:
		b.State = "full"
	default:
		b.State = "discharging"
	}
	return b
}

// ------------------------------------------------------------- powershell

// runPowerShell runs a script and returns its stdout (hidden window).
func runPowerShell(script string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", "[Console]::OutputEncoding=[Text.Encoding]::UTF8; $ErrorActionPreference='SilentlyContinue'; "+script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Output()
}

// ------------------------------------------------------- physical disks

// Health/type/bus come from Get-PhysicalDisk (no admin needed). Reliability
// counters (temperature, wear, power-on hours, errors) need elevation.
const physicalDiskScript = `
$r = @(Get-PhysicalDisk | ForEach-Object {
  $d = $_; $c = $null
  try { $c = $d | Get-StorageReliabilityCounter -ErrorAction Stop } catch {}
  [pscustomobject]@{
    id = "$($d.DeviceId)"; name = "$($d.FriendlyName)"; serial = "$($d.SerialNumber)".Trim(" .")
    media = "$($d.MediaType)"; bus = "$($d.BusType)"; health = "$($d.HealthStatus)"
    status = "$($d.OperationalStatus)"; size = [uint64]$d.Size; smart = [bool]$c
    temp = $c.Temperature; wear = $c.Wear; hours = $c.PowerOnHours
    rerr = $c.ReadErrorsTotal; werr = $c.WriteErrorsTotal
  }
})
ConvertTo-Json -InputObject $r -Compress`

type psDisk struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Serial string   `json:"serial"`
	Media  string   `json:"media"`
	Bus    string   `json:"bus"`
	Health string   `json:"health"`
	Status string   `json:"status"`
	Size   uint64   `json:"size"`
	SMART  bool     `json:"smart"`
	Temp   *float64 `json:"temp"`
	Wear   *float64 `json:"wear"`
	Hours  *uint64  `json:"hours"`
	RErr   *uint64  `json:"rerr"`
	WErr   *uint64  `json:"werr"`
}

func collectPhysicalDisks() ([]PhysicalDisk, bool) {
	out, err := runPowerShell(physicalDiskScript)
	if err != nil {
		return nil, false
	}
	var raw []psDisk
	if json.Unmarshal(out, &raw) != nil {
		return nil, false
	}
	details := false
	disks := make([]PhysicalDisk, 0, len(raw))
	for _, r := range raw {
		d := PhysicalDisk{
			ID: r.ID, Name: r.Name, Serial: r.Serial, MediaType: r.Media, BusType: r.Bus,
			SizeBytes: r.Size, Health: r.Health, Status: r.Status,
			WearPercent: r.Wear, PowerOnHours: r.Hours, ReadErrors: r.RErr, WriteErrors: r.WErr,
		}
		if d.Health == "" {
			d.Health = "Unknown"
		}
		if r.Temp != nil && *r.Temp > 0 { // 0 means "not reported"
			d.TemperatureC = r.Temp
		}
		if r.SMART {
			details = true
		}
		disks = append(disks, d)
	}
	return disks, details
}

// ------------------------------------------------------------ devices

// USB devices use the name the device reports itself (BusReportedDeviceDesc)
// instead of Windows' generic "USB Composite Device" / "HID Keyboard Device".
const devicesScript = `
$all = Get-PnpDevice -PresentOnly
$out = New-Object System.Collections.ArrayList
$usb = @($all | Where-Object { $_.InstanceId -match '^USB\\VID_[0-9A-F]{4}&PID_[0-9A-F]{4}\\' -and $_.InstanceId -notmatch '&MI_' })
foreach ($d in $usb) {
  if ($d.FriendlyName -match 'Hub|Host Controller|Port Policy') { continue }
  # Query one device at a time: batched -InstanceId arrays return mismatched IDs.
  $n = (Get-PnpDeviceProperty -InstanceId $d.InstanceId -KeyName 'DEVPKEY_Device_BusReportedDeviceDesc').Data
  if (-not $n) { $n = $d.FriendlyName }
  [void]$out.Add([pscustomobject]@{ name = "$n"; class = "$($d.Class)"; conn = 'USB'; status = "$($d.Status)" })
}
foreach ($d in @($all | Where-Object { $_.InstanceId -match '^BTH(ENUM|LE)\\DEV_' })) {
  [void]$out.Add([pscustomobject]@{ name = "$($d.FriendlyName)"; class = 'Bluetooth'; conn = 'Bluetooth'; status = "$($d.Status)" })
}
foreach ($d in @($all | Where-Object { $_.Class -eq 'Monitor' })) {
  [void]$out.Add([pscustomobject]@{ name = "$($d.FriendlyName)"; class = 'Monitor'; conn = 'Display'; status = "$($d.Status)" })
}
ConvertTo-Json -InputObject @($out) -Compress`

// Friendlier labels for Windows device setup classes.
var deviceClassLabels = map[string]string{
	"Keyboard": "Keyboard", "Mouse": "Mouse", "HIDClass": "Input device", "MEDIA": "Audio",
	"AudioEndpoint": "Audio", "XnaComposite": "Game controller", "SCSIAdapter": "Storage",
	"DiskDrive": "Storage", "USBDevice": "USB device", "USB": "USB device", "Image": "Camera",
	"Camera": "Camera", "Printer": "Printer", "WPD": "Portable device", "Bluetooth": "Bluetooth",
	"Monitor": "Display", "Net": "Network adapter", "Biometric": "Biometric", "SmartCardReader": "Card reader",
}

func collectDevices() []ConnectedDevice {
	out, err := runPowerShell(devicesScript)
	if err != nil {
		return nil
	}
	var raw []struct {
		Name   string `json:"name"`
		Class  string `json:"class"`
		Conn   string `json:"conn"`
		Status string `json:"status"`
	}
	if json.Unmarshal(out, &raw) != nil {
		return nil
	}
	seen := map[string]bool{}
	devs := make([]ConnectedDevice, 0, len(raw))
	for _, r := range raw {
		name := strings.TrimSpace(r.Name)
		key := r.Conn + "|" + name
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		class := deviceClassLabels[r.Class]
		if class == "" {
			class = r.Class
		}
		devs = append(devs, ConnectedDevice{Name: name, Class: class, Connection: r.Conn, Status: r.Status})
	}
	return devs
}

// -------------------------------------------------------------- users

var (
	modwtsapi32                     = windows.NewLazySystemDLL("wtsapi32.dll")
	procWTSQuerySessionInformationW = modwtsapi32.NewProc("WTSQuerySessionInformationW")
)

const (
	wtsUserName     = 5
	wtsActive       = 0
	wtsDisconnected = 4
)

func collectUsers() []UserSession {
	var sessions *windows.WTS_SESSION_INFO
	var count uint32
	if err := windows.WTSEnumerateSessions(0, 0, 1, &sessions, &count); err != nil {
		return nil
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(sessions)))

	var users []UserSession
	for _, s := range unsafe.Slice(sessions, count) {
		var buf *uint16
		var n uint32
		r, _, _ := procWTSQuerySessionInformationW.Call(0, uintptr(s.SessionID), wtsUserName,
			uintptr(unsafe.Pointer(&buf)), uintptr(unsafe.Pointer(&n)))
		if r == 0 || buf == nil {
			continue
		}
		name := windows.UTF16PtrToString(buf)
		windows.WTSFreeMemory(uintptr(unsafe.Pointer(buf)))
		if name == "" {
			continue
		}
		state := "Other"
		switch s.State {
		case wtsActive:
			state = "Active"
		case wtsDisconnected:
			state = "Disconnected"
		}
		users = append(users, UserSession{User: name, Session: windows.UTF16PtrToString(s.WindowStationName), State: state})
	}
	return users
}
