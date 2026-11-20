//go:build !windows && !linux

package display

// macOS needs cgo (CGEvent) or Accessibility-granted helpers for synthetic
// pointer input; not implemented.

const inputSupported = false

func pointer(int, PointerEvent) error { return ErrInputUnsupported }
