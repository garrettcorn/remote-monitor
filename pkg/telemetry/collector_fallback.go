//go:build !windows && !linux && !darwin

package telemetry

import (
	"fmt"
	"net"
	"runtime"
	"time"
)

func (c *Collector) collectCPU(now time.Time) CPUInfo {
	cores := runtime.NumCPU()
	coreUsages := make([]float64, cores)
	for i := range coreUsages {
		coreUsages[i] = 10.0
	}
	return CPUInfo{
		ModelName:     fmt.Sprintf("Generic CPU (%s)", runtime.GOARCH),
		Vendor:        "Unknown",
		LogicalCores:  cores,
		PhysicalCores: cores,
		UsagePercent:  10.0,
		CoreUsages:    coreUsages,
	}
}

func collectMemory() MemoryInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryInfo{
		TotalBytes:     m.Sys,
		UsedBytes:      m.Alloc,
		FreeBytes:      m.Sys - m.Alloc,
		AvailableBytes: m.Sys - m.Alloc,
		UsagePercent:   float64(m.Alloc) / float64(m.Sys) * 100.0,
	}
}

func (c *Collector) collectDrives(now time.Time) []DriveInfo {
	return []DriveInfo{}
}

func (c *Collector) collectNetwork(now time.Time) NetworkInfo {
	netInfo := NetworkInfo{
		Interfaces: []NetInterface{},
	}
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			ni := NetInterface{
				Name: iface.Name,
				MAC:  iface.HardwareAddr.String(),
				IsUp: (iface.Flags & net.FlagUp) != 0,
			}
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				if ipnet.IP.To4() != nil {
					ni.IPv4 = append(ni.IPv4, ipnet.IP.String())
				} else if ipnet.IP.To16() != nil {
					ni.IPv6 = append(ni.IPv6, ipnet.IP.String())
				}
			}
			netInfo.Interfaces = append(netInfo.Interfaces, ni)
		}
	}
	return netInfo
}

func collectGPUs() []GPUInfo {
	return []GPUInfo{}
}

func collectTemperatures(gpus []GPUInfo) []TempSensor {
	return []TempSensor{}
}

func (c *Collector) collectProcesses(now time.Time, totalMem uint64) ([]ProcessInfo, int) {
	return []ProcessInfo{}, 0
}

// systemUptime is unavailable on unsupported platforms; the caller falls
// back to server uptime.
func systemUptime() (time.Duration, bool) {
	return 0, false
}

func getOSVersion() string {
	return runtime.GOOS
}

func getKernelVersion() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
