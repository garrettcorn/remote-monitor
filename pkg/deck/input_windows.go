//go:build windows

package deck

import (
	"fmt"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduser32           = windows.NewLazySystemDLL("user32.dll")
	procSendInput       = moduser32.NewProc("SendInput")
	procLockWorkStation = moduser32.NewProc("LockWorkStation")
)

const (
	inputKeyboard     = 1
	keyeventfExtended = 0x0001
	keyeventfKeyUp    = 0x0002
	swShowNormal      = 1
)

// keyboardInput mirrors INPUT with a KEYBDINPUT union member (64-bit layout,
// 40 bytes: the union is sized by MOUSEINPUT).
type keyboardInput struct {
	typ         uint32
	_           uint32
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	_           uint32
	dwExtraInfo uint64
	_           [8]byte
}

var mediaVK = map[string]uint16{
	"play_pause": 0xB3, "next": 0xB0, "previous": 0xB1, "stop": 0xB2,
	"volume_mute": 0xAD, "volume_down": 0xAE, "volume_up": 0xAF,
}

var modVK = map[string]uint16{"ctrl": 0x11, "shift": 0x10, "alt": 0x12, "win": 0x5B}

var namedVK = map[string]uint16{
	"space": 0x20, "enter": 0x0D, "tab": 0x09, "esc": 0x1B, "backspace": 0x08,
	"delete": 0x2E, "insert": 0x2D, "home": 0x24, "end": 0x23, "pageup": 0x21,
	"pagedown": 0x22, "left": 0x25, "up": 0x26, "right": 0x27, "down": 0x28,
	"printscreen": 0x2C,
}

// Keys that need KEYEVENTF_EXTENDEDKEY to be distinguished from the numpad.
var extendedVK = map[uint16]bool{0x2E: true, 0x2D: true, 0x24: true, 0x23: true, 0x21: true, 0x22: true, 0x25: true, 0x26: true, 0x27: true, 0x28: true, 0x5B: true}

func keyVK(k string) (uint16, error) {
	if vk, ok := namedVK[k]; ok {
		return vk, nil
	}
	if len(k) == 1 {
		c := k[0]
		if c >= 'a' && c <= 'z' {
			return uint16(c - 'a' + 'A'), nil
		}
		if c >= '0' && c <= '9' {
			return uint16(c), nil
		}
	}
	if len(k) >= 2 && k[0] == 'f' {
		if n, err := strconv.Atoi(k[1:]); err == nil && n >= 1 && n <= 24 {
			return uint16(0x70 + n - 1), nil
		}
	}
	return 0, fmt.Errorf("unsupported key %q", k)
}

func keyEvent(vk uint16, up bool) keyboardInput {
	ev := keyboardInput{typ: inputKeyboard, wVk: vk}
	if up {
		ev.dwFlags |= keyeventfKeyUp
	}
	if extendedVK[vk] {
		ev.dwFlags |= keyeventfExtended
	}
	return ev
}

func sendInputs(evs []keyboardInput) error {
	if len(evs) == 0 {
		return nil
	}
	n, _, err := procSendInput.Call(uintptr(len(evs)), uintptr(unsafe.Pointer(&evs[0])), unsafe.Sizeof(evs[0]))
	if int(n) != len(evs) {
		return fmt.Errorf("SendInput sent %d of %d events: %v", n, len(evs), err)
	}
	return nil
}

func pressMedia(key string) error {
	vk := mediaVK[key]
	return sendInputs([]keyboardInput{keyEvent(vk, false), keyEvent(vk, true)})
}

func pressHotkey(hk Hotkey) error {
	vk, err := keyVK(hk.Key)
	if err != nil {
		return err
	}
	var evs []keyboardInput
	for _, m := range hk.Mods {
		evs = append(evs, keyEvent(modVK[m], false))
	}
	evs = append(evs, keyEvent(vk, false), keyEvent(vk, true))
	for i := len(hk.Mods) - 1; i >= 0; i-- {
		evs = append(evs, keyEvent(modVK[hk.Mods[i]], true))
	}
	return sendInputs(evs)
}

func lockScreen() error {
	if r, _, err := procLockWorkStation.Call(); r == 0 {
		return fmt.Errorf("LockWorkStation: %v", err)
	}
	return nil
}

func shellOpen(file, args, dir string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	f, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	var a, d *uint16
	if args != "" {
		a, _ = windows.UTF16PtrFromString(args)
	}
	if dir != "" {
		d, _ = windows.UTF16PtrFromString(dir)
	}
	return windows.ShellExecute(0, verb, f, a, d, swShowNormal)
}

func openURL(u string) error { return shellOpen(u, "", "") }

func launch(a LaunchAction) error { return shellOpen(a.Path, a.Args, a.Dir) }
