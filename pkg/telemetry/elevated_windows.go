//go:build windows

package telemetry

import "golang.org/x/sys/windows"

// serverElevated reports whether the server runs with an elevated
// (administrator) token. It cannot change while the process runs.
var serverElevated = windows.GetCurrentProcessToken().IsElevated()
