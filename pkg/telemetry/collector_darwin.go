//go:build darwin

package telemetry

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (c *Collector) collectCPU(now time.Time) CPUInfo {
	info := CPUInfo{
		LogicalCores:  runtime.NumCPU(),
		PhysicalCores: runtime.NumCPU() / 2,
		Vendor:        "Apple",
	}
	if info.PhysicalCores < 1 {
		info.PhysicalCores = 1
	}

	// Model name via sysctl
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		info.ModelName = strings.TrimSpace(string(out))
	}
	if info.ModelName == "" {
		info.ModelName = "Apple Silicon / Mac CPU"
	}

	// CPU usage via top or host_processor_info
	cmd := exec.Command("top", "-l", "1", "-n", "0")
	if out, err := cmd.Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "CPU usage:") {
				// e.g. "CPU usage: 4.5% user, 6.2% sys, 89.3% idle"
				parts := strings.Split(line, ",")
				for _, part := range parts {
					if strings.Contains(part, "idle") {
						fields := strings.Fields(part)
						if len(fields) >= 2 {
							idleStr := strings.TrimSuffix(fields[0], "%")
							idleVal, err := strconv.ParseFloat(idleStr, 64)
							if err == nil {
								info.UsagePercent = 100.0 - idleVal
							}
						}
					}
				}
			}
		}
	}

	// Per-core mock/estimates
	info.CoreUsages = make([]float64, info.LogicalCores)
	for i := range info.CoreUsages {
		info.CoreUsages[i] = info.UsagePercent
	}

	return info
}

func collectMemory() MemoryInfo {
	info := MemoryInfo{}

	// Total physical memory via sysctl
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		tot, _ := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		info.TotalBytes = tot
	}

	// Free/active/inactive via vm_stat
	cmd := exec.Command("vm_stat")
	if out, err := cmd.Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		var pageSize uint64 = 4096 // default macOS page size
		var freePages, activePages, inactivePages, speculativePages uint64

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "page size of") {
				fields := strings.Fields(line)
				for i, f := range fields {
					if f == "of" && i+1 < len(fields) {
						sz, _ := strconv.ParseUint(fields[i+1], 10, 64)
						if sz > 0 {
							pageSize = sz
						}
					}
				}
			} else if strings.HasPrefix(line, "Pages free:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					freePages, _ = strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
				}
			} else if strings.HasPrefix(line, "Pages active:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					activePages, _ = strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
				}
			} else if strings.HasPrefix(line, "Pages inactive:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					inactivePages, _ = strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
				}
			} else if strings.HasPrefix(line, "Pages speculative:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					speculativePages, _ = strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
				}
			}
		}

		info.FreeBytes = (freePages + speculativePages) * pageSize
		info.AvailableBytes = (freePages + inactivePages) * pageSize
		info.UsedBytes = (activePages) * pageSize
		if info.TotalBytes > 0 {
			info.UsagePercent = float64(info.TotalBytes-info.AvailableBytes) / float64(info.TotalBytes) * 100.0
		}
	}

	return info
}

func collectDrives() []DriveInfo {
	var drives []DriveInfo
	var stat syscall.Statfs_t
	mount := "/"
	if err := syscall.Statfs(mount, &stat); err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		avail := stat.Bavail * uint64(stat.Bsize)
		used := total - free

		drives = append(drives, DriveInfo{
			Device:       "Macintosh HD",
			Mountpoint:   mount,
			Filesystem:   "APFS",
			TotalBytes:   total,
			UsedBytes:    used,
			FreeBytes:    avail,
			UsagePercent: float64(total-avail) / float64(total) * 100.0,
		})
	}
	return drives
}

func (c *Collector) collectNetwork(now time.Time) NetworkInfo {
	netInfo := NetworkInfo{
		Interfaces: []NetInterface{},
	}
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if (iface.Flags&net.FlagLoopback) != 0 && len(ifaces) > 1 {
				continue
			}
			ni := NetInterface{
				Name: iface.Name,
				MAC:  iface.HardwareAddr.String(),
				IsUp: (iface.Flags & net.FlagUp) != 0,
				IPv4: []string{},
				IPv6: []string{},
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
			if len(ni.IPv4) > 0 || len(ni.IPv6) > 0 || ni.IsUp {
				netInfo.Interfaces = append(netInfo.Interfaces, ni)
			}
		}
	}
	return netInfo
}

func collectGPUs() []GPUInfo {
	var gpus []GPUInfo
	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	if out, err := cmd.Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Chipset Model:") {
				name := strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
				gpus = append(gpus, GPUInfo{
					Index:       len(gpus),
					Name:        name,
					Vendor:      "Apple / AMD",
					Utilization: 0,
				})
			}
		}
	}
	return gpus
}

func collectTemperatures(gpus []GPUInfo) []TempSensor {
	var sensors []TempSensor
	// Try osx-cpu-temp if installed
	if out, err := exec.Command("osx-cpu-temp").Output(); err == nil {
		str := strings.TrimSpace(string(out))
		str = strings.TrimSuffix(str, "°C")
		if temp, err := strconv.ParseFloat(str, 64); err == nil {
			sensors = append(sensors, TempSensor{
				Label:        "CPU Core",
				SensorType:   "cpu",
				TemperatureC: temp,
				MaxC:         95,
				CriticalC:    105,
			})
		}
	}
	return sensors
}

func (c *Collector) collectProcesses(now time.Time, totalMem uint64) ([]ProcessInfo, int) {
	out, err := exec.Command("ps", "-arcwwwxo", "pid,%cpu,%mem,rss,comm").Output()
	if err != nil {
		return nil, 0
	}
	return parsePS(out)
}

// systemUptime derives uptime from kern.boottime, e.g.
// "{ sec = 1731990000, usec = 123456 } Tue Nov 19 ...".
func systemUptime() (time.Duration, bool) {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return 0, false
	}
	s := string(out)
	i := strings.Index(s, "sec = ")
	if i < 0 {
		return 0, false
	}
	s = s[i+len("sec = "):]
	if j := strings.IndexAny(s, ", }"); j >= 0 {
		s = s[:j]
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return time.Since(time.Unix(sec, 0)), true
}

func getOSVersion() string {
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		return fmt.Sprintf("macOS %s", strings.TrimSpace(string(out)))
	}
	return "macOS"
}

func getKernelVersion() string {
	if out, err := exec.Command("sysctl", "-n", "kern.osrelease").Output(); err == nil {
		return fmt.Sprintf("Darwin %s", strings.TrimSpace(string(out)))
	}
	return "Darwin"
}
