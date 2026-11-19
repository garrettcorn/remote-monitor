package telemetry

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"
)

// Collector gathers and caches system performance telemetry.
type Collector struct {
	mu           sync.RWMutex
	lastSnapshot SystemSnapshot
	prevCPUTimes cpuTimes
	prevNetTimes netTimes
	startTime    time.Time
	interval     time.Duration
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

	snap := SystemSnapshot{
		Timestamp: now,
		Host: HostInfo{
			Hostname:     getHostname(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			Kernel:       getKernelVersion(),
			UptimeSec:    uint64(time.Since(c.startTime).Seconds()),
			BootTime:     c.startTime.Format(time.RFC3339),
			ServerTime:   now.Format("15:04:05 MST"),
		},
	}

	// Platform-specific or fallback collections
	snap.Host.OSVersion = getOSVersion()
	snap.CPU = c.collectCPU(now)
	snap.Memory = collectMemory()
	snap.Drives = collectDrives()
	snap.Network = c.collectNetwork(now)
	snap.GPUs = collectGPUs()
	snap.Temperatures = collectTemperatures(snap.GPUs)
	snap.TopProcesses = collectTopProcesses()
	snap.Host.ProcessCount = len(snap.TopProcesses)

	c.mu.Lock()
	c.lastSnapshot = snap
	c.mu.Unlock()
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown-host"
	}
	return name
}
