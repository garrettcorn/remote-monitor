package telemetry

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

const tick = 15625 * time.Microsecond // Windows' process CPU accounting granularity

func TestProcessSmoothing(t *testing.T) {
	c := &Collector{}
	t0 := time.Unix(1000, 0)
	step := 1200 * time.Millisecond
	busy := time.Duration(0)
	quietCPU := time.Duration(0)
	var apps []AppGroup
	for i := 0; i < 25; i++ {
		now := t0.Add(time.Duration(i) * step)
		// "quiet" gets one accounting tick every other sample; "busy" uses
		// half of one core.
		if i%2 == 1 {
			quietCPU += tick
		}
		busy += step / 2
		samples := []procSample{
			{PID: 1, Start: 1, Name: "quiet.exe", CPU: quietCPU, HasCPU: true, Mem: 900 << 20},
			{PID: 2, Start: 1, Name: "busy.exe", CPU: busy, HasCPU: true, Mem: 10 << 20},
		}
		apps, _, _, _ = c.processStats(now, samples, 16<<30)
	}
	want := 50.0 / float64(runtime.NumCPU())
	got := map[string]float64{}
	for _, a := range apps {
		got[a.Name] = a.CPUPercent
	}
	if d := got["busy.exe"] - want; d > want*0.05 || d < -want*0.05 {
		t.Errorf("busy.exe smoothed to %.2f%%, want ~%.2f%%", got["busy.exe"], want)
	}
	// One tick per 2.4 s is ~0.65% of one core; smoothed it must stay well
	// below a single raw tick reading instead of jumping 0 <-> 0.11.
	raw := tick.Seconds() / step.Seconds() / float64(runtime.NumCPU()) * 100
	if got["quiet.exe"] >= raw*0.9 || got["quiet.exe"] <= 0 {
		t.Errorf("quiet.exe smoothed to %.3f%%, want between 0 and %.3f%%", got["quiet.exe"], raw)
	}
	if apps[0].Name != "busy.exe" {
		t.Errorf("busy app should rank first, got %s", apps[0].Name)
	}
}

func TestProcessFirstSampleAndReuse(t *testing.T) {
	c := &Collector{}
	t0 := time.Unix(1000, 0)
	c.processStats(t0, []procSample{{PID: 7, Start: 1, Name: "a", CPU: time.Second, HasCPU: true}}, 0)
	// Same PID, new process instance: no rate from the old one's counters.
	_, top, _, _ := c.processStats(t0.Add(time.Second), []procSample{{PID: 7, Start: 2, Name: "b", CPU: 10 * time.Second, HasCPU: true}}, 0)
	if top[0].CPUPercent != 0 {
		t.Fatalf("reused PID inherited a rate: %v", top[0].CPUPercent)
	}
	if len(c.procs) != 1 {
		t.Fatalf("exited processes not forgotten: %d", len(c.procs))
	}
}

func TestProcessGroupingAndIO(t *testing.T) {
	c := &Collector{}
	t0 := time.Unix(1000, 0)
	mk := func(read uint64) []procSample {
		return []procSample{
			{PID: 1, Start: 1, Name: "brave.exe", Mem: 100, Read: read, HasIO: true, HasCPU: true},
			{PID: 2, Start: 1, Name: "Brave.exe", Mem: 300, HasIO: true, HasCPU: true},
			{PID: 3, Start: 1, Name: "brave.exe", Mem: 200, HasIO: true, HasCPU: true},
			{PID: 4, Start: 1, Name: "code.exe", Mem: 50, HasCPU: true},
		}
	}
	c.processStats(t0, mk(0), 1000)
	apps, _, n, total := c.processStats(t0.Add(time.Second), mk(1<<20), 1000)
	if n != 2 || total != 4 {
		t.Fatalf("app count %d, total %d", n, total)
	}
	var brave AppGroup
	for _, a := range apps {
		if a.Name == "brave.exe" {
			brave = a
		}
	}
	if brave.Count != 3 || brave.MemoryBytes != 600 || brave.MemoryPercent != 60 {
		t.Fatalf("brave group: %+v", brave)
	}
	if brave.Processes[0].PID != 2 { // most memory first when CPU ties
		t.Fatalf("processes not sorted: %+v", brave.Processes)
	}
	if brave.ReadBps == 0 {
		t.Fatal("I/O rate not computed")
	}
}

func TestPickAppsKeepsEachMetricsLeaders(t *testing.T) {
	var apps []AppGroup
	for i := 0; i < 40; i++ {
		apps = append(apps, AppGroup{Name: fmt.Sprintf("p%02d", i), CPUPercent: float64(40 - i)})
	}
	apps[39].MemoryBytes = 8 << 30 // idle but huge
	apps[38].WriteBps = 50 << 20   // idle but writing hard
	got := pickApps(apps)
	has := map[string]bool{}
	for _, a := range got {
		has[a.Name] = true
	}
	if !has["p39"] || !has["p38"] || !has["p00"] || has["p20"] {
		t.Fatalf("picked %v", got)
	}
	if len(got) > 3*appsPerMetric {
		t.Fatalf("too many apps: %d", len(got))
	}
	// An idle app with no memory or I/O never fills a slot.
	idle := pickApps([]AppGroup{{Name: "busy", CPUPercent: 1}, {Name: "System"}})
	if len(idle) != 1 || idle[0].Name != "busy" {
		t.Fatalf("idle app picked: %+v", idle)
	}
}

func TestParseProcStat(t *testing.T) {
	line := "1234 (Web Content (x)) S 1 1234 1234 0 -1 4194560 100 0 0 0 250 50 0 0 20 0 30 0 98765 1000000 2048 18446744073709551615"
	name, cpu, start, rss, ok := parseProcStat(line)
	if !ok || name != "Web Content (x)" || cpu != 300 || start != 98765 || rss != 2048 {
		t.Fatalf("got %q %d %d %d %v", name, cpu, start, rss, ok)
	}
	if _, _, _, _, ok := parseProcStat("garbage"); ok {
		t.Fatal("accepted garbage")
	}
}

func TestParseProcIO(t *testing.T) {
	r, w, ok := parseProcIO("rchar: 1\nwchar: 2\nread_bytes: 4096\nwrite_bytes: 8192\ncancelled_write_bytes: 0\n")
	if !ok || r != 4096 || w != 8192 {
		t.Fatalf("got %d %d %v", r, w, ok)
	}
}

func TestParsePSTime(t *testing.T) {
	cases := map[string]time.Duration{
		"0:01.23":    1230 * time.Millisecond,
		"12:34.50":   12*time.Minute + 34500*time.Millisecond,
		"1:02:03":    time.Hour + 2*time.Minute + 3*time.Second,
		"2-03:04:05": 51*time.Hour + 4*time.Minute + 5*time.Second,
	}
	for in, want := range cases {
		if got, ok := parsePSTime(in); !ok || (got-want).Abs() > time.Millisecond {
			t.Errorf("parsePSTime(%q) = %v %v, want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "abc", "1", "x:10"} {
		if _, ok := parsePSTime(bad); ok {
			t.Errorf("parsePSTime(%q) accepted", bad)
		}
	}
}

func TestParseDarwinPS(t *testing.T) {
	out := "  101   0:12.50  51200 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome\n  202   0:00.01    800 launchd\n"
	got := parseDarwinPS([]byte(out))
	if len(got) != 2 || got[0].Name != "Google Chrome" || got[0].Mem != 51200*1024 || got[0].CPU != 12500*time.Millisecond || got[0].Path == "" {
		t.Fatalf("got %+v", got)
	}
	if bundleName(got[0].Path) != "Google Chrome" {
		t.Fatalf("bundle name %q", bundleName(got[0].Path))
	}
	if got[1].Path != "" {
		t.Fatalf("relative comm treated as a path: %+v", got[1])
	}
}

func TestGPUByPID(t *testing.T) {
	v := map[string]float64{
		"pid_10_luid_0x0_0xBC04_phys_0_eng_0_engtype_3D":          20,
		"pid_10_luid_0x0_0xBC04_phys_0_eng_1_engtype_3D":          15, // same type: summed -> 35
		"pid_10_luid_0x0_0xBC04_phys_0_eng_5_engtype_VideoDecode": 30,
		"pid_11_luid_0x0_0xBC04_phys_0_eng_0_engtype_Copy":        2,
		"pid_12_luid_0x0_0xBC04_phys_0_eng_0_engtype_3D":          0,
		"garbage": 50,
	}
	got := gpuByPID(v)
	if got[10] != 35 || got[11] != 2 || len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestParseProcStatus(t *testing.T) {
	st := parseProcStatus("Name:\tfirefox\nState:\tS (sleeping)\nPPid:\t1234\nUid:\t1000\t1000\t1000\t1000\nVmHWM:\t  204800 kB\nVmRSS:\t  102400 kB\nThreads:\t87\n")
	if st.Name != "firefox" || st.PPid != 1234 || st.UID != "1000" || st.Threads != 87 || st.RSSKB != 102400 || st.HWMKB != 204800 {
		t.Fatalf("got %+v", st)
	}
}

func TestParseDarwinDetail(t *testing.T) {
	user, ppid, rss, started, path, ok := parseDarwinDetail("jon     1  51200 Thu Nov 20 09:11:08 2026 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	if !ok || user != "jon" || ppid != 1 || rss != 51200 || started.Hour() != 9 || started.Day() != 20 || path != "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
		t.Fatalf("got %q %d %d %v %q %v", user, ppid, rss, started, path, ok)
	}
	if _, _, _, _, _, ok := parseDarwinDetail("short line"); ok {
		t.Fatal("accepted a short line")
	}
}
