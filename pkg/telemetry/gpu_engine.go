package telemetry

import (
	"strconv"
	"strings"
)

// gpuByPID turns Windows "GPU Engine" counter values into per-process GPU %,
// the way Task Manager does: utilization is summed per engine type (3D,
// Copy, VideoDecode, ...) on each adapter, and a process's GPU % is its
// busiest engine type. Instance names look like
// "pid_1234_luid_0x00000000_0x0000BC04_phys_0_eng_0_engtype_3D".
func gpuByPID(values map[string]float64) map[int]float64 {
	type key struct {
		pid       int
		luid, eng string
	}
	sums := map[key]float64{}
	for inst, v := range values {
		pid, luid, engType, ok := parseGPUEngine(inst)
		if !ok || v <= 0 {
			continue
		}
		sums[key{pid, luid, engType}] += v
	}
	out := map[int]float64{}
	for k, v := range sums {
		if v > 100 {
			v = 100
		}
		if v > out[k.pid] {
			out[k.pid] = v
		}
	}
	return out
}

func parseGPUEngine(inst string) (pid int, luid, engType string, ok bool) {
	rest, found := strings.CutPrefix(inst, "pid_")
	if !found {
		return 0, "", "", false
	}
	num, rest, found := strings.Cut(rest, "_")
	if !found {
		return 0, "", "", false
	}
	pid, err := strconv.Atoi(num)
	if err != nil {
		return 0, "", "", false
	}
	if i := strings.Index(rest, "luid_"); i >= 0 {
		luid = rest[i+5:]
		if j := strings.Index(luid, "_phys_"); j >= 0 {
			luid = luid[:j]
		}
	}
	i := strings.LastIndex(rest, "engtype_")
	if i < 0 {
		return 0, "", "", false
	}
	return pid, luid, rest[i+len("engtype_"):], true
}
