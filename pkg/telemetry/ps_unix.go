//go:build linux || darwin

package telemetry

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

// parsePS parses `ps` output with columns pid,%cpu,%mem,rss,comm (with a
// header line, already sorted by CPU descending). It returns the top 10
// processes and the total number of processes listed.
func parsePS(out []byte) ([]ProcessInfo, int) {
	var procs []ProcessInfo
	total := 0
	scanner := bufio.NewScanner(bytes.NewReader(out))
	header := true
	for scanner.Scan() {
		if header {
			header = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		total++
		if len(procs) >= 10 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[1], 64)
		mem, _ := strconv.ParseFloat(fields[2], 64)
		rssKB, _ := strconv.ParseUint(fields[3], 10, 64)
		procs = append(procs, ProcessInfo{
			PID:           pid,
			Name:          strings.Join(fields[4:], " "),
			CPUPercent:    cpu,
			MemoryPercent: mem,
			MemoryBytes:   rssKB * 1024,
		})
	}
	return procs, total
}

// parseWho parses `who` output: "user tty date time (host)".
func parseWho() []UserSession {
	out, err := exec.Command("who").Output()
	if err != nil {
		return nil
	}
	var users []UserSession
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		users = append(users, UserSession{User: f[0], Session: f[1], State: "Active"})
	}
	return users
}
