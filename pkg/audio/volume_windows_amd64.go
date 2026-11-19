//go:build windows && amd64

package audio

import "math"

// setMasterScalar calls SetMasterVolumeLevelScalar directly. On amd64 the
// Go Windows call path mirrors integer args into XMM registers, so the
// float32 bit pattern reaches the callee correctly.
func setMasterScalar(v *comIface, level float32) error {
	return hresult(v.call(volSetMasterScalar, uintptr(math.Float32bits(level)), 0), "SetMasterVolumeLevelScalar")
}
