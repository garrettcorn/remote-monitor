//go:build !windows && !linux && !darwin

package telemetry

import "errors"

func (c *Collector) ProcessDetails(pid int) (ProcessDetail, error) {
	return ProcessDetail{PID: pid}, errors.New("process details aren't available on this platform")
}
