//go:build !windows && !linux && !darwin

package display

const captureSupported = false

func list() ([]Display, error)         { return nil, nil }
func capture(int, int) ([]byte, error) { return nil, ErrUnsupported }
