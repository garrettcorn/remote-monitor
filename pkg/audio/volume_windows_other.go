//go:build windows && !amd64

package audio

// setMasterScalar uses volume steps because SyscallN on this architecture
// does not pass float arguments in floating-point registers.
func setMasterScalar(v *comIface, level float32) error {
	return stepToLevel(v, level)
}
