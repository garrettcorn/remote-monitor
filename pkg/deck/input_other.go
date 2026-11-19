//go:build !windows && !linux && !darwin

package deck

import "errors"

var errUnsupported = errors.New("button actions are not supported on this platform")

func pressMedia(string) error   { return errUnsupported }
func pressHotkey(Hotkey) error  { return errUnsupported }
func lockScreen() error         { return errUnsupported }
func openURL(string) error      { return errUnsupported }
func launch(LaunchAction) error { return errUnsupported }
