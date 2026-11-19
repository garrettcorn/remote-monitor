//go:build windows

package telemetry

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// pdhQuery is a reusable PDH query over wildcard counters, e.g.
// `\LogicalDisk(*)\Disk Read Bytes/sec`. Rate counters need two collections
// before they report values, so keep queries open across samples.
type pdhQuery struct {
	q        uintptr
	counters []uintptr
}

func openPDH(paths ...string) *pdhQuery {
	if procPdhOpenQueryW.Find() != nil {
		return nil
	}
	var q uintptr
	if r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return nil
	}
	p := &pdhQuery{q: q}
	for _, path := range paths {
		ptr, _ := windows.UTF16PtrFromString(path)
		var ctr uintptr
		if r, _, _ := procPdhAddEnglishCounterW.Call(q, uintptr(unsafe.Pointer(ptr)), 0, uintptr(unsafe.Pointer(&ctr))); r != 0 {
			return nil
		}
		p.counters = append(p.counters, ctr)
	}
	return p
}

func (p *pdhQuery) collect() bool {
	if p == nil {
		return false
	}
	r, _, _ := procPdhCollectQueryData.Call(p.q)
	return r == 0
}

// values returns instance name -> value for counter i (valid data only).
func (p *pdhQuery) values(i int) map[string]float64 {
	if p == nil || i >= len(p.counters) {
		return nil
	}
	ctr := p.counters[i]
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArrayW.Call(ctr, pdhFmtDouble,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(r) != pdhMoreData || size == 0 {
		return nil
	}
	buf := make([]uint64, (size+7)/8)
	r, _, _ = procPdhGetFormattedCounterArrayW.Call(ctr, pdhFmtDouble,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 || count == 0 {
		return nil
	}
	out := make(map[string]float64, count)
	for _, it := range unsafe.Slice((*pdhCounterItem)(unsafe.Pointer(&buf[0])), count) {
		if it.CStatus <= 1 { // PDH_CSTATUS_VALID_DATA / NEW_DATA
			out[windows.UTF16PtrToString(it.Name)] = it.Value
		}
	}
	runtime.KeepAlive(buf)
	return out
}
