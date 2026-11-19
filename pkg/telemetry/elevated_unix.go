//go:build !windows

package telemetry

import "os"

// serverElevated reports whether the server runs as root. It cannot change
// while the process runs.
var serverElevated = os.Geteuid() == 0
