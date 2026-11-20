//go:build windows

package service

import (
	"strings"
	"testing"
)

// Doesn't touch the registry: checks the command that would be registered.
func TestAutostartCommand(t *testing.T) {
	cmd := autostartCommand()
	exe, gui := Executable()
	if !strings.HasPrefix(cmd, `"`+exe+`"`) {
		t.Fatalf("command %q doesn't start with quoted exe %q", cmd, exe)
	}
	if !gui && !strings.HasSuffix(cmd, " start") {
		t.Fatalf("console exe should be registered with `start`, got %q", cmd)
	}
}
