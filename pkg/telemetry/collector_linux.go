//go:build linux

package telemetry

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
		Vendor:        "Unknown",
	}
	if info.PhysicalCores < 1 {
		info.PhysicalCores = 1
	}

	// Read /proc/cpuinfo
	if f, err := os.Open("/proc/cpuinfo"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "model name") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 && info.ModelName == "" {
					info.ModelName = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "cpu MHz") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 && info.FrequencyMHz == 0 {
					freq, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
					info.FrequencyMHz = freq
				}
			} else if strings.HasPrefix(line, "vendor_id") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 && info.Vendor == "Unknown" {
					info.Vendor = strings.TrimSpace(parts[1])
				}
			}
		}
		f.Close()
	}

	// Read /proc/stat for total and per-core CPU usage
	if f, err := os.Open("/proc/stat"); err == nil {
		scanner := bufio.NewScanner(f)
		var coreUsages []float64

		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}

			if fields[0] == "cpu" {
				// Overall CPU
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				nice, _ := strconv.ParseUint(fields[2], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)
				iowait := uint64(0)
				if len(fields) > 5 {
					iowait, _ = strconv.ParseUint(fields[5], 10, 64)
				}

				total := user + nice + system + idle + iowait
				idleTotal := idle + iowait

				if c.prevCPUTimes.total > 0 && total > c.prevCPUTimes.total {
					deltaTotal := total - c.prevCPUTimes.total
					deltaIdle := idleTotal - c.prevCPUTimes.idle
					if deltaTotal >= deltaIdle {
						info.UsagePercent = float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100.0
					}
				}
				c.prevCPUTimes.total = total
				c.prevCPUTimes.idle = idleTotal
				c.prevCPUTimes.timestamp = now
			} else if strings.HasPrefix(fields[0], "cpu") {
				// Individual core
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)
				total := user + system + idle
				if total > 0 {
					coreUsage := float64(user+system) / float64(total) * 100.0
					coreUsages = append(coreUsages, coreUsage)
				}
			}
		}
		f.Close()
		info.CoreUsages = coreUsages
	}

	return info
}

func collectMemory() MemoryInfo {
	info := MemoryInfo{}
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return info
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	memMap := make(map[string]uint64)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			val, _ := strconv.ParseUint(parts[1], 10, 64)
			memMap[key] = val * 1024 // KB to Bytes
		}
	}

	info.TotalBytes = memMap["MemTotal"]
	info.FreeBytes = memMap["MemFree"]
	info.AvailableBytes = memMap["MemAvailable"]
	if info.AvailableBytes == 0 {
		info.AvailableBytes = info.FreeBytes + memMap["Buffers"] + memMap["Cached"]
	}
	info.UsedBytes = info.TotalBytes - info.AvailableBytes
	if info.TotalBytes > 0 {
		info.UsagePercent = float64(info.UsedBytes) / float64(info.TotalBytes) * 100.0
	}

	info.SwapTotalBytes = memMap["SwapTotal"]
	info.SwapFreeBytes = memMap["SwapFree"]
	info.SwapUsedBytes = info.SwapTotalBytes - info.SwapFreeBytes
	if info.SwapTotalBytes > 0 {
		info.SwapUsagePercent = float64(info.SwapUsedBytes) / float64(info.SwapTotalBytes) * 100.0
	}

	return info
}

func collectDrives() []DriveInfo {
	var drives []DriveInfo
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return drives
	}
	defer f.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		dev := fields[0]
		mount := fields[1]
		fsType := fields[2]

		// Filter virtual filesystems
		if !strings.HasPrefix(dev, "/dev/") && dev != "rootfs" {
			continue
		}
		if seen[mount] {
			continue
		}
		seen[mount] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		avail := stat.Bavail * uint64(stat.Bsize)
		used := total - free

		usagePct := 0.0
		if total > 0 {
			usagePct = float64(total-avail) / float64(total) * 100.0
		}

		drives = append(drives, DriveInfo{
			Device:       dev,
			Mountpoint:   mount,
			Filesystem:   fsType,
			TotalBytes:   total,
			UsedBytes:    used,
			FreeBytes:    avail,
			UsagePercent: usagePct,
		})
	}

	return drives
}

func (c *Collector) collectNetwork(now time.Time) NetworkInfo {
	netInfo := NetworkInfo{
		Interfaces: []NetInterface{},
	}

	// Read /proc/net/dev for accurate packet/byte counters
	ifaceStats := make(map[string][2]uint64)
	if f, err := os.Open("/proc/net/dev"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, ":") {
				continue
			}
			parts := strings.Split(line, ":")
			iface := strings.TrimSpace(parts[0])
			fields := strings.Fields(parts[1])
			if len(fields) >= 9 {
				rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
				txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
				ifaceStats[iface] = [2]uint64{rxBytes, txBytes}
			}
		}
		f.Close()
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

			if stats, ok := ifaceStats[iface.Name]; ok {
				ni.BytesRecv = stats[0]
				ni.BytesSent = stats[1]
				netInfo.TotalBytesRecv += stats[0]
				netInfo.TotalBytesSent += stats[1]
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

	elapsed := now.Sub(c.prevNetTimes.timestamp).Seconds()
	if elapsed <= 0 {
		elapsed = 1.0
	}

	if c.prevNetTimes.bytesRecv > 0 && netInfo.TotalBytesRecv >= c.prevNetTimes.bytesRecv {
		netInfo.DownloadSpeedBps = uint64(float64(netInfo.TotalBytesRecv-c.prevNetTimes.bytesRecv) / elapsed)
		netInfo.UploadSpeedBps = uint64(float64(netInfo.TotalBytesSent-c.prevNetTimes.bytesSent) / elapsed)
	}

	c.prevNetTimes.timestamp = now
	c.prevNetTimes.bytesRecv = netInfo.TotalBytesRecv
	c.prevNetTimes.bytesSent = netInfo.TotalBytesSent

	return netInfo
}

func collectGPUs() []GPUInfo {
	var gpus []GPUInfo

	// Check nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,utilization.gpu,memory.total,memory.used,memory.free,temperature.gpu", "--format=csv,noheader,nounits")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		for _, line := range lines {
			parts := strings.Split(line, ",")
			if len(parts) >= 7 {
				idx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				name := strings.TrimSpace(parts[1])
				util, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
				memTot, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
				memUsed, _ := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
				memFree, _ := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
				temp, _ := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)

				totalBytes := uint64(memTot * 1024 * 1024)
				usedBytes := uint64(memUsed * 1024 * 1024)
				freeBytes := uint64(memFree * 1024 * 1024)

				gpus = append(gpus, GPUInfo{
					Index:        idx,
					Name:         name,
					Vendor:       "NVIDIA",
					Utilization:  util,
					VRAMTotal:    totalBytes,
					VRAMUsed:     usedBytes,
					VRAMFree:     freeBytes,
					VRAMPercent:  float64(usedBytes) / float64(totalBytes) * 100.0,
					TemperatureC: &temp,
				})
			}
		}
	}

	return gpus
}

func collectTemperatures(gpus []GPUInfo) []TempSensor {
	var sensors []TempSensor

	for _, gpu := range gpus {
		if gpu.TemperatureC != nil {
			sensors = append(sensors, TempSensor{
				Label:        gpu.Name + " Temp",
				SensorType:   "gpu",
				TemperatureC: *gpu.TemperatureC,
				MaxC:         90,
				CriticalC:    100,
			})
		}
	}

	// Read Linux sysfs thermal zones
	files, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err == nil {
			raw, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
			tempC := raw / 1000.0
			if tempC > 0 && tempC < 130 {
				typeFile := filepath.Join(filepath.Dir(f), "type")
				label := "Thermal Zone"
				if tData, err := os.ReadFile(typeFile); err == nil {
					label = strings.TrimSpace(string(tData))
				}
				sensors = append(sensors, TempSensor{
					Label:        label,
					SensorType:   "cpu",
					TemperatureC: tempC,
					MaxC:         85,
					CriticalC:    100,
				})
			}
		}
	}

	return sensors
}

func collectTopProcesses() []ProcessInfo {
	var procs []ProcessInfo
	cmd := exec.Command("ps", "-eo", "pid,%cpu,%mem,comm", "--sort=-%cpu")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		scanner := bufio.NewScanner(&out)
		header := true
		count := 0
		for scanner.Scan() && count < 10 {
			if header {
				header = false
				continue
			}
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 4 {
				pid, _ := strconv.Atoi(fields[0])
				cpu, _ := strconv.ParseFloat(fields[1], 64)
				mem, _ := strconv.ParseFloat(fields[2], 64)
				name := fields[3]
				procs = append(procs, ProcessInfo{
					PID:           pid,
					Name:          name,
					CPUPercent:    cpu,
					MemoryPercent: mem,
				})
				count++
			}
		}
	}
	return procs
}

func getOSVersion() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	return "Linux"
}

func getKernelVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		var buf []byte
		for _, b := range uname.Release {
			if b == 0 {
				break
			}
			buf = append(buf, byte(b))
		}
		return string(buf)
	}
	return "Unknown Kernel"
}
