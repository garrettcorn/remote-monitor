//go:build darwin

package telemetry

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ProcessDetails uses ps (threads and handles aren't available this way).
func (c *Collector) ProcessDetails(pid int) (ProcessDetail, error) {
	d := ProcessDetail{PID: pid}
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "user=,ppid=,rss=,lstart=,comm=").Output()
	if err != nil {
		return d, fmt.Errorf("process %d is not running", pid)
	}
	user, ppid, rss, started, path, ok := parseDarwinDetail(strings.TrimSpace(string(out)))
	if !ok {
		return d, fmt.Errorf("unexpected ps output for %d", pid)
	}
	d.User, d.ParentPID, d.MemoryBytes, d.Started = user, ppid, rss*1024, &started
	d.Name = filepath.Base(path)
	if strings.HasPrefix(path, "/") {
		d.Path = path
		d.Title = c.titleFor(path, d.Name)
	}
	root := user == "root"
	d.Elevated = &root
	if p, err := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=").Output(); err == nil {
		d.ParentName = filepath.Base(strings.TrimSpace(string(p)))
	}
	return d, nil
}
