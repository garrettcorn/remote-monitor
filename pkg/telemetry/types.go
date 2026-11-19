package telemetry

import (
	"time"

	"pc_monitor/pkg/audio"
)

// SystemSnapshot represents the complete telemetry state of a host machine.
type SystemSnapshot struct {
	Timestamp    time.Time         `json:"timestamp"`
	Host         HostInfo          `json:"host"`
	CPU          CPUInfo           `json:"cpu"`
	Memory       MemoryInfo        `json:"memory"`
	GPUs         []GPUInfo         `json:"gpus"`
	Temperatures []TempSensor      `json:"temperatures"`
	Drives       []DriveInfo       `json:"drives"`
	Network      NetworkInfo       `json:"network"`
	TopProcesses []ProcessInfo     `json:"top_processes"`
	Disks        []PhysicalDisk    `json:"physical_disks"`
	SMARTDetails bool              `json:"smart_details_available"` // false when detailed counters need elevation
	Devices      []ConnectedDevice `json:"devices"`
	Audio        *audio.State      `json:"audio,omitempty"`
	Battery      *BatteryInfo      `json:"battery,omitempty"`
	Users        []UserSession     `json:"users"`
}

// PhysicalDisk is a physical drive with health and (when available) SMART data.
type PhysicalDisk struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Serial       string   `json:"serial,omitempty"`
	MediaType    string   `json:"media_type"` // "SSD", "HDD", "Unspecified"
	BusType      string   `json:"bus_type"`   // "NVMe", "SATA", "USB"
	SizeBytes    uint64   `json:"size_bytes"`
	Health       string   `json:"health"` // "Healthy", "Warning", "Unhealthy", "Unknown"
	Status       string   `json:"status,omitempty"`
	TemperatureC *float64 `json:"temperature_c,omitempty"`
	WearPercent  *float64 `json:"wear_percent,omitempty"`
	PowerOnHours *uint64  `json:"power_on_hours,omitempty"`
	ReadErrors   *uint64  `json:"read_errors,omitempty"`
	WriteErrors  *uint64  `json:"write_errors,omitempty"`
}

// ConnectedDevice is a peripheral attached to the host.
type ConnectedDevice struct {
	Name       string `json:"name"`
	Class      string `json:"class"`      // e.g. "Keyboard", "Mouse", "Audio", "Storage"
	Connection string `json:"connection"` // "USB", "Bluetooth", "Display"
	Status     string `json:"status"`     // "OK", "Error", ...
}

// BatteryInfo describes the battery on portable devices.
type BatteryInfo struct {
	Percent     int    `json:"percent"`
	Charging    bool   `json:"charging"`
	PluggedIn   bool   `json:"plugged_in"`
	SecondsLeft int    `json:"seconds_left,omitempty"`
	State       string `json:"state"` // "charging", "discharging", "full"
}

// UserSession is a logged-in user session.
type UserSession struct {
	User    string `json:"user"`
	Session string `json:"session,omitempty"` // e.g. "Console", "RDP-Tcp#1", "tty1"
	State   string `json:"state,omitempty"`   // "Active", "Disconnected"
}

// HostInfo contains machine identity, OS and uptime data.
type HostInfo struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`           // e.g. "windows", "linux", "darwin"
	OSVersion    string `json:"os_version"`   // e.g. "Windows 11 Pro 23H2" or "Ubuntu 24.04 LTS"
	Architecture string `json:"architecture"` // e.g. "amd64", "arm64"
	Kernel       string `json:"kernel"`
	UptimeSec    uint64 `json:"uptime_sec"`
	BootTime     string `json:"boot_time"`
	ProcessCount int    `json:"process_count"`
	ServerTime   string `json:"server_time"`
}

// CPUInfo contains CPU specifications, load and frequency.
type CPUInfo struct {
	ModelName     string    `json:"model_name"`
	Vendor        string    `json:"vendor"`
	PhysicalCores int       `json:"physical_cores"`
	LogicalCores  int       `json:"logical_cores"`
	UsagePercent  float64   `json:"usage_percent"`
	CoreUsages    []float64 `json:"core_usages"` // per-logical-core %
	FrequencyMHz  float64   `json:"frequency_mhz"`
	TemperatureC  *float64  `json:"temperature_c,omitempty"`
}

// MemoryInfo contains RAM and Swap/Pagefile statistics.
type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`

	SwapTotalBytes   uint64  `json:"swap_total_bytes"`
	SwapUsedBytes    uint64  `json:"swap_used_bytes"`
	SwapFreeBytes    uint64  `json:"swap_free_bytes"`
	SwapUsagePercent float64 `json:"swap_usage_percent"`
}

// GPUInfo contains dedicated or integrated GPU metrics.
type GPUInfo struct {
	Index        int      `json:"index"`
	Name         string   `json:"name"`
	Vendor       string   `json:"vendor"` // "NVIDIA", "AMD", "Intel", "Apple"
	Utilization  float64  `json:"utilization_percent"`
	VRAMTotal    uint64   `json:"vram_total_bytes"`
	VRAMUsed     uint64   `json:"vram_used_bytes"`
	VRAMFree     uint64   `json:"vram_free_bytes"`
	VRAMPercent  float64  `json:"vram_percent"`
	TemperatureC *float64 `json:"temperature_c,omitempty"`
	PowerWatts   *float64 `json:"power_watts,omitempty"`
	PowerLimitW  *float64 `json:"power_limit_watts,omitempty"`
	FanSpeed     *float64 `json:"fan_speed_percent,omitempty"`
	CoreClockMHz *float64 `json:"core_clock_mhz,omitempty"`
	MemClockMHz  *float64 `json:"mem_clock_mhz,omitempty"`
}

// TempSensor represents any temperature probe (CPU, GPU, NVMe, Motherboard).
type TempSensor struct {
	Label        string  `json:"label"`
	SensorType   string  `json:"sensor_type"` // "cpu", "gpu", "drive", "motherboard", "ambient"
	TemperatureC float64 `json:"temperature_c"`
	MaxC         float64 `json:"max_c,omitempty"`
	CriticalC    float64 `json:"critical_c,omitempty"`
}

// DriveInfo represents a storage volume or partition.
type DriveInfo struct {
	Device        string  `json:"device"`     // e.g. "C:", "/dev/nvme0n1p2"
	Mountpoint    string  `json:"mountpoint"` // e.g. "C:\", "/"
	Filesystem    string  `json:"filesystem"` // e.g. "NTFS", "ext4", "APFS"
	TotalBytes    uint64  `json:"total_bytes"`
	UsedBytes     uint64  `json:"used_bytes"`
	FreeBytes     uint64  `json:"free_bytes"`
	UsagePercent  float64 `json:"usage_percent"`
	ReadSpeedBps  uint64  `json:"read_speed_bps"`  // Bytes per second
	WriteSpeedBps uint64  `json:"write_speed_bps"` // Bytes per second
	BusyPercent   float64 `json:"busy_percent"`    // % of time the volume was busy
}

// NetworkInfo contains all active adapters and network activity rates.
type NetworkInfo struct {
	Interfaces       []NetInterface `json:"interfaces"`
	DownloadSpeedBps uint64         `json:"download_speed_bps"` // Combined instantaneous ingress
	UploadSpeedBps   uint64         `json:"upload_speed_bps"`   // Combined instantaneous egress
	TotalBytesRecv   uint64         `json:"total_bytes_recv"`
	TotalBytesSent   uint64         `json:"total_bytes_sent"`
}

// NetInterface details a physical or virtual network adapter.
type NetInterface struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	IPv4      []string `json:"ipv4"`
	IPv6      []string `json:"ipv6"`
	IsUp      bool     `json:"is_up"`
	BytesRecv uint64   `json:"bytes_recv"`
	BytesSent uint64   `json:"bytes_sent"`
	SpeedRecv uint64   `json:"speed_recv_bps"`
	SpeedSent uint64   `json:"speed_sent_bps"`
	LinkMbps  uint64   `json:"link_speed_mbps,omitempty"`
}

// ProcessInfo represents a high-utilization running process.
type ProcessInfo struct {
	PID           int     `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryBytes   uint64  `json:"memory_bytes"`
}
