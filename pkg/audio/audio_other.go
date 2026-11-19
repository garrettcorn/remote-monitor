//go:build !windows && !linux && !darwin

package audio

func get() (*State, error)        { return nil, ErrUnsupported }
func setVolume(string, int) error { return ErrUnsupported }
func setMute(string, bool) error  { return ErrUnsupported }
func setDefault(string) error     { return ErrUnsupported }
