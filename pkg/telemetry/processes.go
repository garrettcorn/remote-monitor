package telemetry

import (
	"math"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Process telemetry is gathered in two layers: each platform reports raw,
// cumulative counters per process (procSample), and this file turns them
// into smoothed rates and groups processes into apps, the same way on every
// OS.

// procSample is one process's raw counters at one moment.
type procSample struct {
	PID   int
	Start uint64 // creation time/ticks; with PID it identifies one process instance
	Name  string // executable name, e.g. "brave.exe"
	Path  string // full executable path when known (may be empty)

	CPU    time.Duration // cumulative CPU time (user + kernel)
	HasCPU bool
	Mem    uint64 // resident / working set bytes

	Read, Write uint64 // cumulative I/O bytes
	HasIO       bool

	GPU    float64 // instantaneous GPU % (busiest engine type), Windows only
	HasGPU bool
}

type procID struct {
	pid   int
	start uint64
}

// procState remembers a process between samples.
type procState struct {
	cpu         time.Duration
	read, write uint64
	seen        bool // has a previous sample, so rates can be computed

	cpuPct, gpuPct    float64 // smoothed
	readBps, writeBps float64 // smoothed
	path, title       string
}

const (
	// procSmoothing is the time constant of the moving average for process
	// CPU and I/O. Windows only counts process CPU in ~15.6 ms ticks, so a
	// single 1.2 s sample of a quiet process jumps between 0 and ~0.1%;
	// averaging over a few seconds gives a stable, meaningful ranking.
	procSmoothing = 5 * time.Second
	appsPerMetric = 12 // top apps sent per sort key (CPU, memory, I/O)
	procsPerApp   = 5  // processes listed inside each app
	topProcs      = 10 // legacy top_processes list
)

// processStats updates smoothed per-process rates from a new set of samples
// and returns the apps to show, the legacy top process list, the number of
// distinct apps, and the number of processes.
func (c *Collector) processStats(now time.Time, samples []procSample, totalMem uint64) (apps []AppGroup, top []ProcessInfo, appCount, total int) {
	dt := now.Sub(c.procStamp)
	first := c.procStamp.IsZero() || dt <= 0
	alpha := 1.0
	if !first {
		alpha = 1 - math.Exp(-dt.Seconds()/procSmoothing.Seconds())
	}
	numCPU := float64(runtime.NumCPU())
	next := make(map[procID]*procState, len(samples))
	procs := make([]ProcessInfo, 0, len(samples))
	titles := make([]string, 0, len(samples))
	paths := make([]string, 0, len(samples))

	for _, s := range samples {
		id := procID{s.PID, s.Start}
		st := c.procs[id]
		if st == nil {
			st = &procState{}
		}
		if s.Path != "" {
			st.path = s.Path
		}
		if st.title == "" {
			st.title = c.appTitle(st.path, s.Name)
		}
		if st.seen && !first {
			if s.HasCPU && s.CPU >= st.cpu {
				inst := (s.CPU - st.cpu).Seconds() / dt.Seconds() / numCPU * 100
				st.cpuPct += alpha * (math.Min(inst, 100) - st.cpuPct)
			}
			if s.HasGPU {
				st.gpuPct += alpha * (math.Min(s.GPU, 100) - st.gpuPct)
			}
			if s.HasIO && s.Read >= st.read && s.Write >= st.write {
				st.readBps += alpha * (float64(s.Read-st.read)/dt.Seconds() - st.readBps)
				st.writeBps += alpha * (float64(s.Write-st.write)/dt.Seconds() - st.writeBps)
			}
		}
		st.cpu, st.read, st.write, st.seen = s.CPU, s.Read, s.Write, true
		next[id] = st

		p := ProcessInfo{
			PID:         s.PID,
			Name:        s.Name,
			CPUPercent:  round2(st.cpuPct),
			MemoryBytes: s.Mem,
		}
		if totalMem > 0 {
			p.MemoryPercent = round2(float64(s.Mem) / float64(totalMem) * 100)
		}
		if s.HasGPU {
			p.GPUPercent = round2(st.gpuPct)
		}
		if s.HasIO {
			p.ReadBps, p.WriteBps = uint64(st.readBps), uint64(st.writeBps)
		}
		procs = append(procs, p)
		titles = append(titles, st.title)
		paths = append(paths, st.path)
	}
	c.procs = next
	c.procStamp = now

	all := groupApps(procs, titles, paths, totalMem)
	appCount = len(all)
	c.procMu.Lock()
	c.allApps, c.allTotal = all, len(samples)
	c.procMu.Unlock()
	apps = pickApps(all)
	for i := range apps { // the snapshot lists only the busiest few per app
		if len(apps[i].Processes) > procsPerApp {
			apps[i].Processes = apps[i].Processes[:procsPerApp]
		}
	}

	top = append([]ProcessInfo(nil), procs...)
	sortProcs(top)
	if len(top) > topProcs {
		top = top[:topProcs]
	}
	return apps, top, appCount, len(samples)
}

// AllApps returns every app with all of its processes from the latest
// sample (for search and "Show all"), plus the number of processes. The
// result must not be modified.
func (c *Collector) AllApps() ([]AppGroup, int) {
	c.procMu.Lock()
	defer c.procMu.Unlock()
	return c.allApps, c.allTotal
}

// AppPath returns the executable path of a running app (by program name,
// case-insensitive), or "" if unknown.
func (c *Collector) AppPath(name string) string {
	c.procMu.Lock()
	defer c.procMu.Unlock()
	k := appKey(name)
	for _, a := range c.allApps {
		if appKey(a.Name) == k {
			return a.Path
		}
	}
	return ""
}

// ProcessName returns the program name of a running process from the latest
// sample, or "" if it isn't running.
func (c *Collector) ProcessName(pid int) string {
	c.procMu.Lock()
	defer c.procMu.Unlock()
	for _, a := range c.allApps {
		for _, p := range a.Processes {
			if p.PID == pid {
				return p.Name
			}
		}
	}
	return ""
}

// appKey groups processes of the same program.
func appKey(name string) string { return strings.ToLower(name) }

func groupApps(procs []ProcessInfo, titles, paths []string, totalMem uint64) []AppGroup {
	idx := map[string]int{}
	var apps []AppGroup
	for i, p := range procs {
		k := appKey(p.Name)
		j, ok := idx[k]
		if !ok {
			j = len(apps)
			idx[k] = j
			apps = append(apps, AppGroup{Name: p.Name})
		}
		a := &apps[j]
		if a.Path == "" && paths[i] != "" {
			a.Path = paths[i]
		}
		if a.Title == "" && titles[i] != "" {
			a.Title = titles[i]
		}
		a.Count++
		a.CPUPercent += p.CPUPercent
		a.MemoryBytes += p.MemoryBytes
		a.ReadBps += p.ReadBps
		a.WriteBps += p.WriteBps
		a.GPUPercent += p.GPUPercent
		a.Processes = append(a.Processes, p)
	}
	for i := range apps {
		a := &apps[i]
		a.CPUPercent = round2(math.Min(a.CPUPercent, 100))
		a.GPUPercent = round2(math.Min(a.GPUPercent, 100))
		if totalMem > 0 {
			a.MemoryPercent = round2(float64(a.MemoryBytes) / float64(totalMem) * 100)
		}
		sortProcs(a.Processes)
	}
	return apps
}

// pickApps keeps the top apps by CPU, by memory and by I/O (so the
// dashboard can re-sort by any of them), ordered by CPU then memory.
func pickApps(apps []AppGroup) []AppGroup {
	keep := map[string]bool{}
	// Only apps that actually use the resource qualify, so an idle host
	// doesn't fill the I/O slots with arbitrary zero-I/O apps.
	byKey := func(value func(a AppGroup) float64) {
		s := make([]AppGroup, 0, len(apps))
		for _, a := range apps {
			if value(a) > 0 {
				s = append(s, a)
			}
		}
		sort.SliceStable(s, func(i, j int) bool {
			if value(s[i]) != value(s[j]) {
				return value(s[i]) > value(s[j])
			}
			return s[i].MemoryBytes > s[j].MemoryBytes
		})
		for i := 0; i < len(s) && i < appsPerMetric; i++ {
			keep[appKey(s[i].Name)] = true
		}
	}
	byKey(func(a AppGroup) float64 { return a.CPUPercent })
	byKey(func(a AppGroup) float64 { return float64(a.MemoryBytes) })
	byKey(func(a AppGroup) float64 { return float64(a.ReadBps + a.WriteBps) })
	byKey(func(a AppGroup) float64 { return a.GPUPercent })
	out := make([]AppGroup, 0, len(keep))
	for _, a := range apps {
		if keep[appKey(a.Name)] {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return appLess(out[i], out[j]) })
	return out
}

func appLess(a, b AppGroup) bool {
	if a.CPUPercent != b.CPUPercent {
		return a.CPUPercent > b.CPUPercent
	}
	return a.MemoryBytes > b.MemoryBytes
}

func sortProcs(p []ProcessInfo) {
	sort.SliceStable(p, func(i, j int) bool {
		if p[i].CPUPercent != p[j].CPUPercent {
			return p[i].CPUPercent > p[j].CPUPercent
		}
		return p[i].MemoryBytes > p[j].MemoryBytes
	})
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// appTitle returns a friendly program name, cached per executable path.
func (c *Collector) appTitle(path, name string) string {
	if path == "" {
		return ""
	}
	if t, ok := c.titles[path]; ok {
		return t
	}
	t := c.titleFor(path, name)
	if c.titles == nil {
		c.titles = map[string]string{}
	}
	c.titles[path] = t
	return t
}

// titleFor looks up a friendly name without the cache (safe from any
// goroutine; appTitle's cache belongs to the sampling loop).
func (c *Collector) titleFor(path, name string) string {
	if path == "" {
		return ""
	}
	t := friendlyName(path)
	if t == "" {
		t = bundleName(path)
	}
	if strings.EqualFold(t, name) || strings.EqualFold(t, strings.TrimSuffix(name, filepath.Ext(name))) {
		return "" // no better than the file name
	}
	return t
}

// bundleName extracts "Safari" from ".../Safari.app/Contents/MacOS/Safari".
func bundleName(path string) string {
	p := filepath.ToSlash(path)
	if i := strings.Index(p, ".app/"); i > 0 {
		return filepath.Base(p[:i])
	}
	return ""
}
