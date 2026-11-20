package telemetry

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Parsers for Unix process sources. They have no build tags so they are
// unit-tested on every platform.

// parseProcStat parses /proc/<pid>/stat. It returns the command name,
// user+system CPU in clock ticks, the start time in ticks since boot and the
// resident set in pages.
func parseProcStat(data string) (name string, cpuTicks, startTicks, rssPages uint64, ok bool) {
	open := strings.IndexByte(data, '(')
	closeP := strings.LastIndexByte(data, ')') // names may contain ')'
	if open < 0 || closeP < open {
		return "", 0, 0, 0, false
	}
	name = data[open+1 : closeP]
	// Fields after ")" start at field 3 (state); see proc(5).
	f := strings.Fields(data[closeP+1:])
	if len(f) < 22 {
		return "", 0, 0, 0, false
	}
	utime, err1 := strconv.ParseUint(f[11], 10, 64) // field 14
	stime, err2 := strconv.ParseUint(f[12], 10, 64) // field 15
	start, err3 := strconv.ParseUint(f[19], 10, 64) // field 22
	rss, _ := strconv.ParseInt(f[21], 10, 64)       // field 24
	if err1 != nil || err2 != nil || err3 != nil {
		return "", 0, 0, 0, false
	}
	if rss < 0 {
		rss = 0
	}
	return name, utime + stime, start, uint64(rss), true
}

// parseProcIO parses /proc/<pid>/io (read_bytes / write_bytes: real disk I/O).
func parseProcIO(data string) (read, write uint64, ok bool) {
	var gotR, gotW bool
	for _, line := range strings.Split(data, "\n") {
		k, v, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(k) {
		case "read_bytes":
			read, gotR = n, true
		case "write_bytes":
			write, gotW = n, true
		}
	}
	return read, write, gotR && gotW
}

// parsePSTime parses ps cumulative CPU time: "[[dd-]hh:]mm:ss[.cc]"
// (e.g. "0:01.23", "12:34.56", "1:02:03", "2-03:04:05").
func parsePSTime(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	var days int64
	if d, rest, found := strings.Cut(s, "-"); found {
		n, err := strconv.ParseInt(d, 10, 64)
		if err != nil {
			return 0, false
		}
		days, s = n, rest
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	sec, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return 0, false
	}
	total := sec
	mult := 60.0
	for i := len(parts) - 2; i >= 0; i-- {
		n, err := strconv.ParseInt(parts[i], 10, 64)
		if err != nil {
			return 0, false
		}
		total += float64(n) * mult
		mult *= 60
	}
	total += float64(days) * 86400
	return time.Duration(total * float64(time.Second)), true
}

// parseDarwinPS parses `ps -axo pid=,time=,rss=,comm=` (comm is the full
// executable path on macOS and may contain spaces).
func parseDarwinPS(out []byte) []procSample {
	var res []procSample
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 4 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		cpu, ok := parsePSTime(f[1])
		rssKB, _ := strconv.ParseUint(f[2], 10, 64)
		path := strings.Join(f[3:], " ")
		s := procSample{PID: pid, Name: filepath.Base(path), CPU: cpu, HasCPU: ok, Mem: rssKB * 1024}
		if strings.HasPrefix(path, "/") {
			s.Path = path
		}
		res = append(res, s)
	}
	return res
}

// procStatus holds the /proc/<pid>/status fields the details view uses.
type procStatus struct {
	Name          string
	PPid, Threads int
	UID           string
	RSSKB, HWMKB  uint64
}

// parseProcStatus parses /proc/<pid>/status ("Key:\tvalue" lines).
func parseProcStatus(data string) procStatus {
	var st procStatus
	for _, line := range strings.Split(data, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch k {
		case "Name":
			st.Name = v
		case "PPid":
			st.PPid, _ = strconv.Atoi(v)
		case "Threads":
			st.Threads, _ = strconv.Atoi(v)
		case "Uid":
			if f := strings.Fields(v); len(f) > 0 {
				st.UID = f[0] // real UID
			}
		case "VmRSS":
			st.RSSKB, _ = strconv.ParseUint(strings.TrimSuffix(v, " kB"), 10, 64)
		case "VmHWM":
			st.HWMKB, _ = strconv.ParseUint(strings.TrimSuffix(v, " kB"), 10, 64)
		}
	}
	return st
}

// parseDarwinDetail parses `ps -p <pid> -o user=,ppid=,rss=,lstart=,comm=`:
// "jon   1  51200 Thu Nov 20 09:11:08 2026 /Applications/Foo.app/Contents/MacOS/Foo".
func parseDarwinDetail(line string) (user string, ppid int, rssKB uint64, started time.Time, path string, ok bool) {
	f := strings.Fields(line)
	if len(f) < 9 {
		return "", 0, 0, time.Time{}, "", false
	}
	ppid, _ = strconv.Atoi(f[1])
	rssKB, _ = strconv.ParseUint(f[2], 10, 64)
	started, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", strings.Join(f[3:8], " "), time.Local)
	if err != nil {
		return "", 0, 0, time.Time{}, "", false
	}
	return f[0], ppid, rssKB, started, strings.Join(f[8:], " "), true
}
