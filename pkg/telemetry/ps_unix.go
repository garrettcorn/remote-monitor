//go:build linux || darwin

package telemetry

import (
	"os/exec"
	"strings"
)

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
