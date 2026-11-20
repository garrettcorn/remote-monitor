package procctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestHelperProcess is the throwaway process the End test ends. It only
// runs when started by the test below.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("PROCCTL_HELPER") != "1" {
		t.Skip("helper")
	}
	time.Sleep(60 * time.Second)
	os.Exit(0)
}

func TestEndEndsOnlyTheExpectedProgram(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "PROCCTL_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer cmd.Process.Kill()
	pid := cmd.Process.Pid
	self := filepath.Base(os.Args[0])

	// The PID now belongs to a different program: refuse, leave it running.
	if err := End(pid, "notepad.exe", self); err == nil {
		t.Fatal("ended a process whose name didn't match")
	}
	if err := End(pid, self, ""); err == nil {
		t.Fatal("accepted a process that isn't running")
	}
	select {
	case <-done:
		t.Fatal("helper exited although End refused")
	case <-time.After(200 * time.Millisecond):
	}

	if err := End(pid, self, self); err != nil {
		t.Fatalf("End: %v", err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("helper still running after End")
	}
}

func TestProtected(t *testing.T) {
	if !Protected(os.Getpid(), "anything") || !Protected(4, "x") {
		t.Fatal("own process / system PIDs must be protected")
	}
	for _, n := range []string{"pc_monitor.exe", "PC_Monitor-app.exe", "pc-monitor"} {
		if !Protected(5000, n) {
			t.Errorf("%s not protected", n)
		}
	}
	if runtime.GOOS == "windows" {
		for _, n := range []string{"csrss.exe", "LSASS.EXE", "svchost.exe", "winlogon.exe", "System"} {
			if !Protected(5000, n) {
				t.Errorf("%s not protected", n)
			}
		}
	}
	if Protected(5000, "notepad.exe") {
		t.Error("an ordinary app is protected")
	}
	if err := End(5000, "csrss.exe", "csrss.exe"); runtime.GOOS == "windows" && err != ErrProtected {
		t.Errorf("End(csrss) = %v, want ErrProtected", err)
	}
}
