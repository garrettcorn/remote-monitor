//go:build windows

package display

import (
	"fmt"
	"unsafe"
)

const inputSupported = true

var (
	procSetCursorPos = moduser32.NewProc("SetCursorPos")
	procSendInput    = moduser32.NewProc("SendInput")
)

const (
	inputMouse           = 0
	mouseeventfLeftDown  = 0x0002
	mouseeventfLeftUp    = 0x0004
	mouseeventfRightDown = 0x0008
	mouseeventfRightUp   = 0x0010
	mouseeventfMidDown   = 0x0020
	mouseeventfMidUp     = 0x0040
	mouseeventfWheel     = 0x0800
	mouseeventfHWheel    = 0x1000
	wheelDelta           = 120
)

// mouseInput mirrors INPUT with a MOUSEINPUT union member (64-bit layout,
// 40 bytes).
type mouseInput struct {
	typ         uint32
	_           uint32
	dx, dy      int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	_           uint32
	dwExtraInfo uint64
}

var buttonFlags = map[string][2]uint32{
	"left":   {mouseeventfLeftDown, mouseeventfLeftUp},
	"right":  {mouseeventfRightDown, mouseeventfRightUp},
	"middle": {mouseeventfMidDown, mouseeventfMidUp},
}

func sendMouse(evs ...mouseInput) error {
	if len(evs) == 0 {
		return nil
	}
	n, _, err := procSendInput.Call(uintptr(len(evs)), uintptr(unsafe.Pointer(&evs[0])), unsafe.Sizeof(evs[0]))
	if int(n) != len(evs) {
		return fmt.Errorf("SendInput sent %d of %d events: %v", n, len(evs), err)
	}
	return nil
}

func mouseFlag(flags uint32) mouseInput { return mouseInput{typ: inputMouse, dwFlags: flags} }

func pointer(index int, e PointerEvent) error {
	mons, err := monitors()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(mons) {
		return fmt.Errorf("no display %d", index)
	}
	r := mons[index].info.Monitor
	// The process is per-monitor DPI aware (see init), so monitor rects and
	// SetCursorPos both use physical pixels.
	x := toPixel(int(r.Left), int(r.Right-r.Left), e.X)
	y := toPixel(int(r.Top), int(r.Bottom-r.Top), e.Y)
	if ok, _, err := procSetCursorPos.Call(uintptr(int32(x)), uintptr(int32(y))); ok == 0 {
		return fmt.Errorf("SetCursorPos: %v", err)
	}
	b := buttonFlags[e.Button]
	switch e.Type {
	case "down":
		return sendMouse(mouseFlag(b[0]))
	case "up":
		return sendMouse(mouseFlag(b[1]))
	case "click":
		return sendMouse(mouseFlag(b[0]), mouseFlag(b[1]))
	case "dblclick":
		return sendMouse(mouseFlag(b[0]), mouseFlag(b[1]), mouseFlag(b[0]), mouseFlag(b[1]))
	case "scroll":
		var evs []mouseInput
		if e.DY != 0 {
			ev := mouseFlag(mouseeventfWheel)
			ev.mouseData = uint32(int32(-e.DY * wheelDelta)) // positive wheel = away from user = scroll up
			evs = append(evs, ev)
		}
		if e.DX != 0 {
			ev := mouseFlag(mouseeventfHWheel)
			ev.mouseData = uint32(int32(e.DX * wheelDelta))
			evs = append(evs, ev)
		}
		return sendMouse(evs...)
	}
	return nil // move: SetCursorPos already did it
}
