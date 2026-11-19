package telemetry

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"pc_monitor/pkg/audio"
)

// Collector gathers and caches system performance telemetry.
type Collector struct {
	mu           sync.RWMutex
	lastSnapshot SystemSnapshot
	prevCPUTimes cpuTimes
	prevNetTimes netTimes
	startTime    time.Time
	interval     time.Duration

	// Per-process CPU time samples (used by platforms that compute
	// process CPU % from deltas, e.g. Windows).
	prevProcTimes map[procKey]uint64
	prevProcStamp time.Time

	// Per-disk I/O counters for platforms that compute rates from deltas.
	prevDisk      map[string]diskSample
	prevDiskStamp time.Time

	// Audio state; refreshed each tick and immediately after control changes.
	audioMu sync.Mutex
	audio   *audio.State

	// Slow-changing data refreshed in the background (see refreshSlow).
	slowMu      sync.Mutex
	slow        slowData
	slowAt      time.Time
	slowRunning bool
}

type diskSample struct {
	readBytes, writeBytes, busyMs uint64
}

// slowData is expensive to gather (e.g. PowerShell on Windows), so it is
// refreshed at most every slowInterval, off the sampling path.
type slowData struct {
	disks        []PhysicalDisk
	smartDetails bool
	devices      []ConnectedDevice
	users        []UserSession
}

const slowInterval = 30 * time.Second

type procKey struct {
	pid     uint32
	created uint64
}

type cpuTimes struct {
	timestamp time.Time
	idle      uint64
	total     uint64
	coreIdle  []uint64
	coreTotal []uint64
}

type netTimes struct {
	timestamp time.Time
	bytesRecv uint64
	bytesSent uint64
	ifaceRecv map[string]uint64
	ifaceSent map[string]uint64
}

// NewCollector creates an initialized telemetry collector.
func NewCollector(sampleInterval time.Duration) *Collector {
	if sampleInterval <= 0 {
		sampleInterval = 1500 * time.Millisecond
	}
	return &Collector{
		startTime: time.Now(),
		interval:  sampleInterval,
		prevNetTimes: netTimes{
			ifaceRecv: make(map[string]uint64),
			ifaceSent: make(map[string]uint64),
		},
	}
}

// Start initiates the background collection loop.
func (c *Collector) Start(ctx context.Context) {
	// Immediate first collection to seed values
	c.collect()

	ticker := time.NewTicker(c.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collect()
			}
		}
	}()
}

// GetSnapshot returns the most recently collected system snapshot.
func (c *Collector) GetSnapshot() SystemSnapshot {
	c.mu.RLock()
	if !c.lastSnapshot.Timestamp.IsZero() {
		defer c.mu.RUnlock()
		return c.lastSnapshot
	}
	c.mu.RUnlock()

	c.collect()
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSnapshot
}

// collect performs one sampling iteration and stores the snapshot.
func (c *Collector) collect() {
	now := time.Now()

	// Prefer real OS uptime; fall back to server uptime if unavailable.
	uptime, ok := systemUptime()
	if !ok {
		uptime = now.Sub(c.startTime)
	}

	snap := SystemSnapshot{
		Timestamp: now,
		Host: HostInfo{
			Hostname:     getHostname(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			Kernel:       getKernelVersion(),
			UptimeSec:    uint64(uptime.Seconds()),
			BootTime:     now.Add(-uptime).Format(time.RFC3339),
			ServerTime:   now.Format("15:04:05 MST"),
		},
	}

	// Platform-specific or fallback collections
	snap.Host.OSVersion = getOSVersion()
	snap.CPU = c.collectCPU(now)
	snap.Memory = collectMemory()
	snap.Drives = c.collectDrives(now)
	snap.Network = c.collectNetwork(now)
	snap.GPUs = collectGPUs()
	snap.Temperatures = collectTemperatures(snap.GPUs)
	snap.TopProcesses, snap.Host.ProcessCount = c.collectProcesses(now, snap.Memory.TotalBytes)
	snap.Battery = collectBattery()

	if st, err := audio.Get(); err == nil {
		c.SetAudio(st)
	}
	c.audioMu.Lock()
	snap.Audio = c.audio
	c.audioMu.Unlock()

	c.slowMu.Lock()
	if !c.slowRunning && now.Sub(c.slowAt) >= slowInterval {
		c.slowRunning = true
		go c.refreshSlow()
	}
	snap.Disks = c.slow.disks
	snap.SMARTDetails = c.slow.smartDetails
	snap.Elevated = serverElevated
	snap.Devices = c.slow.devices
	snap.Users = c.slow.users
	c.slowMu.Unlock()

	c.mu.Lock()
	c.lastSnapshot = snap
	c.mu.Unlock()
}

// refreshSlow gathers slow-changing data in the background.
func (c *Collector) refreshSlow() {
	var d slowData
	d.disks, d.smartDetails = collectPhysicalDisks()
	d.devices = collectDevices()
	d.users = collectUsers()

	c.slowMu.Lock()
	c.slow = d
	c.slowAt = time.Now()
	c.slowRunning = false
	c.slowMu.Unlock()
}

// SetAudio stores a fresh audio state (used after control changes so the
// next snapshot reflects them immediately).
func (c *Collector) SetAudio(st *audio.State) {
	c.audioMu.Lock()
	c.audio = st
	c.audioMu.Unlock()

	c.mu.Lock()
	if !c.lastSnapshot.Timestamp.IsZero() {
		c.lastSnapshot.Audio = st
	}
	c.mu.Unlock()
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown-host"
	}
	return name
}
