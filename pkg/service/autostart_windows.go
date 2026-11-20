//go:build windows

package service

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// Per-user Run key: starts at login inside the user's session, no admin needed.
const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValue = "PC Monitor"

func detachedAttr() *syscall.SysProcAttr {
	const createNoWindow = 0x08000000
	const createNewProcessGroup = 0x00000200
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | createNewProcessGroup}
}

func autostartCommand() string {
	exe, gui := Executable()
	if gui {
		return fmt.Sprintf(`"%s"`, exe) // the windowless app runs the server by default
	}
	return fmt.Sprintf(`"%s" start`, exe)
}

// AutostartEnable registers the server to start when the user logs in.
func AutostartEnable() error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runValue, autostartCommand())
}

// AutostartDisable removes the login entry.
func AutostartDisable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	if err := k.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// AutostartEnabled reports whether a login entry exists.
func AutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValue)
	return err == nil
}

// AutostartDescription says where the entry lives.
func AutostartDescription() string {
	return `HKCU\` + runKey + ` → "` + runValue + `"`
}
