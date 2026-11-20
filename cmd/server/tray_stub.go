//go:build !(windows || linux || (darwin && cgo))

package main

import "errors"

const trayAvailable = false

// The macOS tray needs cgo (built on a macOS runner); other builds fall back
// to the settings page.
func trayCommand() error {
	return errors.New("the tray icon isn't available in this build; use `pc_monitor settings`")
}
