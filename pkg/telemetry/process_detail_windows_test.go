//go:build windows

package telemetry

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Reads details of this test process only.
func TestProcessDetailsSelf(t *testing.T) {
	c := &Collector{}
	d, err := c.ProcessDetails(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(d.Path, os.Args[0]) && !strings.HasSuffix(strings.ToLower(d.Path), ".exe") {
		t.Errorf("path %q", d.Path)
	}
	if d.Threads < 1 || d.ParentPID == 0 || d.Handles < 1 || d.MemoryBytes == 0 || d.PrivateBytes == 0 || d.User == "" || d.Elevated == nil {
		t.Errorf("incomplete: %+v", d)
	}
	if d.Started == nil || time.Since(*d.Started) > time.Hour || time.Since(*d.Started) < 0 {
		t.Errorf("started %v", d.Started)
	}
	if _, err := c.ProcessDetails(0x7ffffff0); err == nil {
		t.Error("expected an error for a missing PID")
	}
}
