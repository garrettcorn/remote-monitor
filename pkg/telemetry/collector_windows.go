//go:build windows

package telemetry

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modkernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes     = modkernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatus = modkernel32.NewProc("GlobalMemoryStatusEx")
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func (c *Collector) collectCPU(now time.Time) CPUInfo {
	info := CPUInfo{
		LogicalCores:  runtime.NumCPU(),
		PhysicalCores: runtime.NumCPU() / 2,
		Vendor:        "Unknown",
	}
	if info.PhysicalCores < 1 {
		info.PhysicalCores = 1
	}

	// Read CPU Name and Frequency from Registry
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if name, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
			info.ModelName = strings.TrimSpace(name)
		}
		if vendor, _, err := k.GetStringValue("VendorIdentifier"); err == nil {
			info.Vendor = strings.TrimSpace(vendor)
		}
		if mhz, _, err := k.GetIntegerValue("~MHz"); err == nil {
			info.FrequencyMHz = float64(mhz)
		}
	}
	if info.ModelName == "" {
		info.ModelName = fmt.Sprintf("CPU (%d cores)", info.LogicalCores)
	}

	// Calculate overall CPU Usage % via GetSystemTimes
	var idleTime, kernelTime, userTime windows.Filetime
	r, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)

	if r != 0 {
		idle := fileTimeToUint64(idleTime)
		kernel := fileTimeToUint64(kernelTime)
		user := fileTimeToUint64(userTime)
		total := kernel + user

		if c.prevCPUTimes.total > 0 && total > c.prevCPUTimes.total {
			deltaTotal := total - c.prevCPUTimes.total
			deltaIdle := idle - c.prevCPUTimes.idle
			if deltaTotal > 0 && deltaTotal >= deltaIdle {
				usage := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100.0
				if usage < 0 {
					usage = 0
				}
				if usage > 100 {
					usage = 100
				}
				info.UsagePercent = usage
			}
		}
		c.prevCPUTimes.idle = idle
		c.prevCPUTimes.total = total
		c.prevCPUTimes.timestamp = now
	}

	info.CoreUsages = c.collectCoreUsages(info.LogicalCores)

	return info
}

// sysProcPerfInfo mirrors SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION.
type sysProcPerfInfo struct {
	IdleTime   int64
	KernelTime int64 // includes idle time
	UserTime   int64
	Reserved1  [2]int64
	Reserved2  uint32
}

const systemProcessorPerformanceInformation = 8

// collectCoreUsages computes real per-logical-core utilisation from
// NtQuerySystemInformation deltas. Returns zeros on the first sample.
func (c *Collector) collectCoreUsages(n int) []float64 {
	if n < 1 {
		return nil
	}
	buf := make([]sysProcPerfInfo, n)
	var retLen uint32
	size := uint32(len(buf)) * uint32(unsafe.Sizeof(buf[0]))
	if err := windows.NtQuerySystemInformation(systemProcessorPerformanceInformation, unsafe.Pointer(&buf[0]), size, &retLen); err != nil {
		return nil
	}
	count := int(retLen / uint32(unsafe.Sizeof(buf[0])))
	if count > n {
		count = n
	}

	usages := make([]float64, count)
	idles := make([]uint64, count)
	totals := make([]uint64, count)
	prevIdle, prevTotal := c.prevCPUTimes.coreIdle, c.prevCPUTimes.coreTotal
	for i := 0; i < count; i++ {
		idles[i] = uint64(buf[i].IdleTime)
		totals[i] = uint64(buf[i].KernelTime + buf[i].UserTime)
		if len(prevTotal) == count && totals[i] > prevTotal[i] {
			dTotal := totals[i] - prevTotal[i]
			dIdle := idles[i] - prevIdle[i]
			if dIdle > dTotal {
				dIdle = dTotal
			}
			usages[i] = float64(dTotal-dIdle) / float64(dTotal) * 100.0
		}
	}
	c.prevCPUTimes.coreIdle = idles
	c.prevCPUTimes.coreTotal = totals
	return usages
}

var procGetTickCount64 = modkernel32.NewProc("GetTickCount64")

// systemUptime returns time since the OS booted (matches Task Manager).
func systemUptime() (time.Duration, bool) {
	r, _, _ := procGetTickCount64.Call()
	if r == 0 {
		return 0, false
	}
	return time.Duration(r) * time.Millisecond, true
}

func fileTimeToUint64(ft windows.Filetime) uint64 {
	return (uint64(ft.HighDateTime) << 32) | uint64(ft.LowDateTime)
}

func collectMemory() MemoryInfo {
	var ms memoryStatusEx
	ms.cbSize = uint32(unsafe.Sizeof(ms))
	r, _, _ := procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&ms)))
	if r == 0 {
		return MemoryInfo{}
	}

	used := ms.ullTotalPhys - ms.ullAvailPhys
	usagePct := 0.0
	if ms.ullTotalPhys > 0 {
		usagePct = float64(used) / float64(ms.ullTotalPhys) * 100.0
	}

	swapTotal := ms.ullTotalPageFile
	swapFree := ms.ullAvailPageFile
	swapUsed := swapTotal - swapFree
	swapPct := 0.0
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100.0
	}

	return MemoryInfo{
		TotalBytes:       ms.ullTotalPhys,
		UsedBytes:        used,
		FreeBytes:        ms.ullAvailPhys,
		AvailableBytes:   ms.ullAvailPhys,
		UsagePercent:     usagePct,
		SwapTotalBytes:   swapTotal,
		SwapUsedBytes:    swapUsed,
		SwapFreeBytes:    swapFree,
		SwapUsagePercent: swapPct,
	}
}

var (
	diskIOOnce  sync.Once
	diskIOQuery *pdhQuery
)

func (c *Collector) collectDrives(now time.Time) []DriveInfo {
	drives := collectDriveList()

	// Per-volume throughput and busy % from the LogicalDisk perf counters.
	diskIOOnce.Do(func() {
		diskIOQuery = openPDH(
			`\LogicalDisk(*)\Disk Read Bytes/sec`,
			`\LogicalDisk(*)\Disk Write Bytes/sec`,
			`\LogicalDisk(*)\% Idle Time`,
		)
	})
	if diskIOQuery.collect() {
		reads, writes, idle := diskIOQuery.values(0), diskIOQuery.values(1), diskIOQuery.values(2)
		for i := range drives {
			inst := drives[i].Device // "C:"
			drives[i].ReadSpeedBps = uint64(reads[inst])
			drives[i].WriteSpeedBps = uint64(writes[inst])
			// Sleeping/stalled drives can report 0% idle with no transfers;
			// only report busy time while data is actually moving.
			if v, ok := idle[inst]; ok && drives[i].ReadSpeedBps+drives[i].WriteSpeedBps > 0 {
				drives[i].BusyPercent = max(0, min(100, 100-v))
			}
		}
	}
	return drives
}

func collectDriveList() []DriveInfo {
	var drives []DriveInfo
	bitmask, err := windows.GetLogicalDrives()
	if err != nil {
		return drives
	}

	for i := 0; i < 26; i++ {
		if (bitmask & (1 << i)) != 0 {
			letter := string(rune('A' + i))
			root := letter + ":\\"

			// Check drive type
			rootPtr, _ := windows.UTF16PtrFromString(root)
			driveType := windows.GetDriveType(rootPtr)
			// DRIVE_FIXED = 3, DRIVE_REMOVABLE = 2, DRIVE_REMOTE = 4
			if driveType != windows.DRIVE_FIXED && driveType != windows.DRIVE_REMOVABLE && driveType != windows.DRIVE_REMOTE {
				continue
			}

			var freeBytesAvail, totalBytes, totalFreeBytes uint64
			err := windows.GetDiskFreeSpaceEx(
				rootPtr,
				&freeBytesAvail,
				&totalBytes,
				&totalFreeBytes,
			)
			if err != nil || totalBytes == 0 {
				continue
			}

			usedBytes := totalBytes - totalFreeBytes
			usagePct := float64(usedBytes) / float64(totalBytes) * 100.0

			// Get Volume and Filesystem name
			var fsNameBuf [256]uint16
			var volNameBuf [256]uint16
			fsType := "NTFS"
			_ = windows.GetVolumeInformation(
				rootPtr,
				&volNameBuf[0], uint32(len(volNameBuf)),
				nil, nil, nil,
				&fsNameBuf[0], uint32(len(fsNameBuf)),
			)
			if name := windows.UTF16ToString(fsNameBuf[:]); name != "" {
				fsType = name
			}

			drives = append(drives, DriveInfo{
				Device:       letter + ":",
				Mountpoint:   root,
				Filesystem:   fsType,
				TotalBytes:   totalBytes,
				UsedBytes:    usedBytes,
				FreeBytes:    totalFreeBytes,
				UsagePercent: usagePct,
			})
		}
	}
	return drives
}

func (c *Collector) collectNetwork(now time.Time) NetworkInfo {
	netInfo := NetworkInfo{
		Interfaces: []NetInterface{},
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return netInfo
	}

	var totalRecv, totalSent uint64
	elapsed := now.Sub(c.prevNetTimes.timestamp).Seconds()
	if elapsed <= 0 {
		elapsed = 1.0
	}

	for _, iface := range ifaces {
		// Ignore down or loopback interfaces for primary list
		isLoopback := (iface.Flags & net.FlagLoopback) != 0
		isUp := (iface.Flags & net.FlagUp) != 0
		if isLoopback && len(ifaces) > 1 {
			continue
		}

		ni := NetInterface{
			Name: iface.Name,
			MAC:  iface.HardwareAddr.String(),
			IsUp: isUp,
			IPv4: []string{},
			IPv6: []string{},
		}

		addrs, err := iface.Addrs()
		if err == nil {
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
		}

		// Keep only interfaces that have IP addresses or are active
		if !(len(ni.IPv4) > 0 || len(ni.IPv6) > 0 || ni.IsUp) {
			continue
		}

		// Read byte counters via GetIfEntry2Ex (MIB_IF_ROW2)
		row := windows.MibIfRow2{InterfaceIndex: uint32(iface.Index)}
		if err := windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &row); err == nil {
			ni.BytesRecv = row.InOctets
			ni.BytesSent = row.OutOctets
			if row.ReceiveLinkSpeed > 0 && row.ReceiveLinkSpeed != ^uint64(0) {
				ni.LinkMbps = row.ReceiveLinkSpeed / 1_000_000
			}
			if prev, ok := c.prevNetTimes.ifaceRecv[ni.Name]; ok && ni.BytesRecv >= prev {
				ni.SpeedRecv = uint64(float64(ni.BytesRecv-prev) / elapsed)
			}
			if prev, ok := c.prevNetTimes.ifaceSent[ni.Name]; ok && ni.BytesSent >= prev {
				ni.SpeedSent = uint64(float64(ni.BytesSent-prev) / elapsed)
			}
			c.prevNetTimes.ifaceRecv[ni.Name] = ni.BytesRecv
			c.prevNetTimes.ifaceSent[ni.Name] = ni.BytesSent
			totalRecv += ni.BytesRecv
			totalSent += ni.BytesSent
		}

		netInfo.Interfaces = append(netInfo.Interfaces, ni)
	}

	netInfo.TotalBytesRecv = totalRecv
	netInfo.TotalBytesSent = totalSent

	if c.prevNetTimes.bytesRecv > 0 && totalRecv >= c.prevNetTimes.bytesRecv {
		netInfo.DownloadSpeedBps = uint64(float64(totalRecv-c.prevNetTimes.bytesRecv) / elapsed)
		netInfo.UploadSpeedBps = uint64(float64(totalSent-c.prevNetTimes.bytesSent) / elapsed)
	}

	c.prevNetTimes.timestamp = now
	c.prevNetTimes.bytesRecv = totalRecv
	c.prevNetTimes.bytesSent = totalSent

	return netInfo
}

func collectGPUs() []GPUInfo {
	var gpus []GPUInfo

	// 1. Try nvidia-smi if available
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,utilization.gpu,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,fan.speed,clocks.gr,clocks.mem", "--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		r := csv.NewReader(&out)
		records, err := r.ReadAll()
		if err == nil {
			for _, rec := range records {
				if len(rec) >= 7 {
					idx, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
					name := strings.TrimSpace(rec[1])
					util, _ := strconv.ParseFloat(strings.TrimSpace(rec[2]), 64)
					memTotalMB, _ := strconv.ParseFloat(strings.TrimSpace(rec[3]), 64)
					memUsedMB, _ := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
					memFreeMB, _ := strconv.ParseFloat(strings.TrimSpace(rec[5]), 64)
					temp, _ := strconv.ParseFloat(strings.TrimSpace(rec[6]), 64)

					totalBytes := uint64(memTotalMB * 1024 * 1024)
					usedBytes := uint64(memUsedMB * 1024 * 1024)
					freeBytes := uint64(memFreeMB * 1024 * 1024)
					vramPct := 0.0
					if totalBytes > 0 {
						vramPct = float64(usedBytes) / float64(totalBytes) * 100.0
					}

					gpu := GPUInfo{
						Index:        idx,
						Name:         name,
						Vendor:       "NVIDIA",
						Utilization:  util,
						VRAMTotal:    totalBytes,
						VRAMUsed:     usedBytes,
						VRAMFree:     freeBytes,
						VRAMPercent:  vramPct,
						TemperatureC: &temp,
					}
					// Optional fields; nvidia-smi reports "[N/A]" when unsupported.
					optional := func(i int) *float64 {
						if i >= len(rec) {
							return nil
						}
						v, err := strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
						if err != nil {
							return nil
						}
						return &v
					}
					gpu.PowerWatts = optional(7)
					gpu.PowerLimitW = optional(8)
					gpu.FanSpeed = optional(9)
					gpu.CoreClockMHz = optional(10)
					gpu.MemClockMHz = optional(11)
					gpus = append(gpus, gpu)
				}
			}
		}
	}

	// 2. If no NVIDIA GPU was found, query registry or WMI for video controller
	if len(gpus) == 0 {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\0000`, registry.QUERY_VALUE)
		if err == nil {
			defer k.Close()
			if desc, _, err := k.GetStringValue("DriverDesc"); err == nil && desc != "" {
				vendor := "Intel / Integrated"
				if strings.Contains(strings.ToLower(desc), "amd") || strings.Contains(strings.ToLower(desc), "radeon") {
					vendor = "AMD"
				} else if strings.Contains(strings.ToLower(desc), "nvidia") || strings.Contains(strings.ToLower(desc), "geforce") {
					vendor = "NVIDIA"
				} else if strings.Contains(strings.ToLower(desc), "intel") {
					vendor = "Intel"
				}
				gpus = append(gpus, GPUInfo{
					Index:       0,
					Name:        desc,
					Vendor:      vendor,
					Utilization: 0,
				})
			}
		}
	}

	return gpus
}

func collectTemperatures(gpus []GPUInfo) []TempSensor {
	var sensors []TempSensor

	// Include GPU temperatures if detected
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

	// ACPI thermal zones via the "Thermal Zone Information" perf counter
	// (no admin rights required). Only real readings are reported.
	sensors = append(sensors, readThermalZones()...)

	return sensors
}

var (
	modpdh                           = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQueryW                = modpdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW        = modpdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData          = modpdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArrayW = modpdh.NewProc("PdhGetFormattedCounterArrayW")

	thermalOnce    sync.Once
	thermalOK      bool
	thermalQuery   uintptr
	thermalCounter uintptr
)

const (
	pdhFmtDouble = 0x00000200
	pdhMoreData  = 0x800007D2
)

// pdhCounterItem mirrors PDH_FMT_COUNTERVALUE_ITEM_W with a double value (64-bit layout).
type pdhCounterItem struct {
	Name    *uint16
	CStatus uint32
	_       uint32
	Value   float64
}

func initThermalQuery() {
	if procPdhOpenQueryW.Find() != nil {
		return
	}
	var q uintptr
	if r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return
	}
	path, _ := windows.UTF16PtrFromString(`\Thermal Zone Information(*)\Temperature`)
	var ctr uintptr
	if r, _, _ := procPdhAddEnglishCounterW.Call(q, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&ctr))); r != 0 {
		return
	}
	thermalQuery, thermalCounter, thermalOK = q, ctr, true
}

func readThermalZones() []TempSensor {
	thermalOnce.Do(initThermalQuery)
	if !thermalOK {
		return nil
	}
	if r, _, _ := procPdhCollectQueryData.Call(thermalQuery); r != 0 {
		return nil
	}
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArrayW.Call(thermalCounter, pdhFmtDouble,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(r) != pdhMoreData || size == 0 {
		return nil
	}
	buf := make([]uint64, (size+7)/8) // 8-byte aligned
	r, _, _ = procPdhGetFormattedCounterArrayW.Call(thermalCounter, pdhFmtDouble,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 || count == 0 {
		return nil
	}

	var sensors []TempSensor
	items := unsafe.Slice((*pdhCounterItem)(unsafe.Pointer(&buf[0])), count)
	for _, it := range items {
		if it.CStatus > 1 { // PDH_CSTATUS_VALID_DATA / NEW_DATA
			continue
		}
		tempC := it.Value - 273.15 // counter reports Kelvin
		if tempC <= 0 || tempC >= 130 {
			continue
		}
		zone := windows.UTF16PtrToString(it.Name)
		if i := strings.LastIndex(zone, "."); i >= 0 {
			zone = zone[i+1:]
		}
		sensors = append(sensors, TempSensor{
			Label:        "ACPI Thermal Zone " + strings.ToUpper(zone),
			SensorType:   "motherboard",
			TemperatureC: tempC,
			MaxC:         85,
			CriticalC:    100,
		})
	}
	runtime.KeepAlive(buf)
	return sensors
}

var procK32GetProcessMemoryInfo = modkernel32.NewProc("K32GetProcessMemoryInfo")

// processMemoryCounters mirrors PROCESS_MEMORY_COUNTERS.
type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// collectProcesses enumerates all processes, computes CPU % from deltas of
// kernel+user time (normalised to all cores, like Task Manager), and returns
// the top 10 by CPU then memory, plus the total process count.
func (c *Collector) collectProcesses(now time.Time, totalMem uint64) ([]ProcessInfo, int) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, 0
	}
	defer windows.CloseHandle(snapshot)

	elapsed := now.Sub(c.prevProcStamp).Seconds()
	havePrev := c.prevProcTimes != nil && elapsed > 0
	numCPU := float64(runtime.NumCPU())
	current := make(map[procKey]uint64)

	var procs []ProcessInfo
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if entry.ProcessID == 0 { // System Idle Process
			continue
		}
		p := ProcessInfo{
			PID:  int(entry.ProcessID),
			Name: windows.UTF16ToString(entry.ExeFile[:]),
		}

		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if err == nil {
			var created, exited, kernel, user windows.Filetime
			if windows.GetProcessTimes(h, &created, &exited, &kernel, &user) == nil {
				key := procKey{entry.ProcessID, fileTimeToUint64(created)}
				cpuTime := fileTimeToUint64(kernel) + fileTimeToUint64(user) // 100ns units
				current[key] = cpuTime
				if prev, ok := c.prevProcTimes[key]; ok && havePrev && cpuTime >= prev {
					pct := float64(cpuTime-prev) / (elapsed * 1e7) / numCPU * 100.0
					if pct > 100 {
						pct = 100
					}
					p.CPUPercent = pct
				}
			}
			var pmc processMemoryCounters
			pmc.cb = uint32(unsafe.Sizeof(pmc))
			if r, _, _ := procK32GetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.cb)); r != 0 {
				p.MemoryBytes = uint64(pmc.WorkingSetSize)
				if totalMem > 0 {
					p.MemoryPercent = float64(p.MemoryBytes) / float64(totalMem) * 100.0
				}
			}
			windows.CloseHandle(h)
		}
		procs = append(procs, p)
	}

	c.prevProcTimes = current
	c.prevProcStamp = now

	total := len(procs)
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPUPercent != procs[j].CPUPercent {
			return procs[i].CPUPercent > procs[j].CPUPercent
		}
		return procs[i].MemoryBytes > procs[j].MemoryBytes
	})
	if len(procs) > 10 {
		procs = procs[:10]
	}
	return procs, total
}

func getOSVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows"
	}
	defer k.Close()

	prod, _, _ := k.GetStringValue("ProductName")
	build, _, _ := k.GetStringValue("CurrentBuild")
	displayVer, _, _ := k.GetStringValue("DisplayVersion")

	if displayVer != "" {
		return fmt.Sprintf("%s (%s, Build %s)", prod, displayVer, build)
	}
	if prod != "" {
		return fmt.Sprintf("%s (Build %s)", prod, build)
	}
	return "Windows"
}

func getKernelVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "NT Kernel"
	}
	defer k.Close()

	major, _, _ := k.GetIntegerValue("CurrentMajorVersionNumber")
	minor, _, _ := k.GetIntegerValue("CurrentMinorVersionNumber")
	build, _, _ := k.GetStringValue("CurrentBuild")
	return fmt.Sprintf("NT %d.%d (Build %s)", major, minor, build)
}
