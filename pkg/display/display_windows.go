//go:build windows

package display

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduser32                         = windows.NewLazySystemDLL("user32.dll")
	procEnumDisplayMonitors           = moduser32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW               = moduser32.NewProc("GetMonitorInfoW")
	procEnumDisplaySettingsW          = moduser32.NewProc("EnumDisplaySettingsW")
	procEnumDisplayDevicesW           = moduser32.NewProc("EnumDisplayDevicesW")
	procSetProcessDpiAwarenessContext = moduser32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = moduser32.NewProc("SetProcessDPIAware")
	procGetDC                         = moduser32.NewProc("GetDC")
	procReleaseDC                     = moduser32.NewProc("ReleaseDC")

	modgdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	procCreateCompatibleDC     = modgdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modgdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modgdi32.NewProc("SelectObject")
	procBitBlt                 = modgdi32.NewProc("BitBlt")
	procGetDIBits              = modgdi32.NewProc("GetDIBits")
	procDeleteObject           = modgdi32.NewProc("DeleteObject")
	procDeleteDC               = modgdi32.NewProc("DeleteDC")

	modshcore            = windows.NewLazySystemDLL("shcore.dll")
	procGetDpiForMonitor = modshcore.NewProc("GetDpiForMonitor")
)

const (
	captureSupported = true

	enumCurrentSettings       = 0xFFFFFFFF // ENUM_CURRENT_SETTINGS (-1)
	monitorInfoFPrimary       = 1
	eddGetDeviceInterfaceName = 1
	mdtEffectiveDPI           = 0
	srcCopy                   = 0x00CC0020
	captureBlt                = 0x40000000
	dibRGBColors              = 0
	dpiAwarenessPerMonitorV2  = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
)

// Report real pixels on scaled displays: without DPI awareness, Windows
// virtualises monitor sizes and captures for this process.
func init() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := procSetProcessDpiAwarenessContext.Call(dpiAwarenessPerMonitorV2); r != 0 {
			return
		}
	}
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

type rect struct{ Left, Top, Right, Bottom int32 }

// monitorInfoEx mirrors MONITORINFOEXW.
type monitorInfoEx struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
	Device  [32]uint16
}

// devMode mirrors DEVMODEW (220 bytes) with the display union members.
type devMode struct {
	DeviceName         [32]uint16
	SpecVersion        uint16
	DriverVersion      uint16
	Size               uint16
	DriverExtra        uint16
	Fields             uint32
	PositionX          int32
	PositionY          int32
	DisplayOrientation uint32
	DisplayFixedOutput uint32
	Color              int16
	Duplex             int16
	YResolution        int16
	TTOption           int16
	Collate            int16
	FormName           [32]uint16
	LogPixels          uint16
	BitsPerPel         uint32
	PelsWidth          uint32
	PelsHeight         uint32
	DisplayFlags       uint32
	DisplayFrequency   uint32
	ICMMethod          uint32
	ICMIntent          uint32
	MediaType          uint32
	DitherType         uint32
	Reserved1          uint32
	Reserved2          uint32
	PanningWidth       uint32
	PanningHeight      uint32
}

// displayDevice mirrors DISPLAY_DEVICEW.
type displayDevice struct {
	Cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type bitmapInfo struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
	Colors        [1]uint32
}

type monitor struct {
	handle uintptr
	info   monitorInfoEx
}

// EnumDisplayMonitors needs a callback; NewCallback slots are limited, so
// create it once and collect into a package slice under a mutex.
var (
	enumMu       sync.Mutex
	enumResults  []monitor
	enumCallback = windows.NewCallback(func(hmon, hdc, lprc, lparam uintptr) uintptr {
		m := monitor{handle: hmon}
		m.info.Size = uint32(unsafe.Sizeof(m.info))
		if r, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&m.info))); r != 0 {
			enumResults = append(enumResults, m)
		}
		return 1 // continue
	})
)

func monitors() ([]monitor, error) {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumResults = nil
	if r, _, err := procEnumDisplayMonitors.Call(0, 0, enumCallback, 0); r == 0 {
		return nil, fmt.Errorf("EnumDisplayMonitors: %v", err)
	}
	out := append([]monitor(nil), enumResults...)
	// Primary first, then left-to-right, top-to-bottom.
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := out[i].info.Flags&monitorInfoFPrimary != 0, out[j].info.Flags&monitorInfoFPrimary != 0
		if pi != pj {
			return pi
		}
		if out[i].info.Monitor.Left != out[j].info.Monitor.Left {
			return out[i].info.Monitor.Left < out[j].info.Monitor.Left
		}
		return out[i].info.Monitor.Top < out[j].info.Monitor.Top
	})
	return out, nil
}

func list() ([]Display, error) {
	mons, err := monitors()
	if err != nil {
		return nil, err
	}
	displays := make([]Display, 0, len(mons))
	for i, m := range mons {
		dev := windows.UTF16ToString(m.info.Device[:])
		d := Display{
			Index:   i,
			Device:  dev,
			Primary: m.info.Flags&monitorInfoFPrimary != 0,
			X:       int(m.info.Monitor.Left),
			Y:       int(m.info.Monitor.Top),
			Width:   int(m.info.Monitor.Right - m.info.Monitor.Left),
			Height:  int(m.info.Monitor.Bottom - m.info.Monitor.Top),
		}
		devPtr, _ := windows.UTF16PtrFromString(dev)
		var dm devMode
		dm.Size = uint16(unsafe.Sizeof(dm))
		if r, _, _ := procEnumDisplaySettingsW.Call(uintptr(unsafe.Pointer(devPtr)), enumCurrentSettings, uintptr(unsafe.Pointer(&dm))); r != 0 {
			d.Width, d.Height = int(dm.PelsWidth), int(dm.PelsHeight)
			d.RefreshHz = int(dm.DisplayFrequency)
			d.BitsPerPixel = int(dm.BitsPerPel)
		}
		// Monitor attached to this display output: generic name + PnP interface path.
		var dd displayDevice
		dd.Cb = uint32(unsafe.Sizeof(dd))
		if r, _, _ := procEnumDisplayDevicesW.Call(uintptr(unsafe.Pointer(devPtr)), 0, uintptr(unsafe.Pointer(&dd)), eddGetDeviceInterfaceName); r != 0 {
			d.Name = windows.UTF16ToString(dd.DeviceString[:])
			d.HardwareID = hardwareKey(windows.UTF16ToString(dd.DeviceID[:]))
		}
		if d.Name == "" {
			d.Name = fmt.Sprintf("Display %d", i+1)
		}
		if procGetDpiForMonitor.Find() == nil {
			var dpiX, dpiY uint32
			if r, _, _ := procGetDpiForMonitor.Call(m.handle, mdtEffectiveDPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); r == 0 && dpiX > 0 {
				d.ScalePercent = int(dpiX * 100 / 96)
			}
		}
		displays = append(displays, d)
	}
	return displays, nil
}

// hardwareKey turns a monitor interface path such as
// `\\?\DISPLAY#GSM5B7F#5&1a2b&0&UID4352#{e6f07b5f-...}` into
// `DISPLAY\GSM5B7F\5&1A2B&0&UID4352`, matching WmiMonitorID.InstanceName
// (minus its `_0` suffix).
func hardwareKey(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, `\\?\`), "#")
	if len(parts) < 3 {
		return ""
	}
	return strings.ToUpper(parts[0] + `\` + parts[1] + `\` + parts[2])
}

func capture(index, maxWidth int) ([]byte, error) {
	mons, err := monitors()
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(mons) {
		return nil, fmt.Errorf("no display %d", index)
	}
	r := mons[index].info.Monitor
	x, y := int32(r.Left), int32(r.Top)
	w, h := int32(r.Right-r.Left), int32(r.Bottom-r.Top)
	if w <= 0 || h <= 0 {
		return nil, errors.New("display has no area")
	}

	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, errors.New("GetDC failed")
	}
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return nil, errors.New("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(mem)
	bmp, _, _ := procCreateCompatibleBitmap.Call(screen, uintptr(w), uintptr(h))
	if bmp == 0 {
		return nil, errors.New("CreateCompatibleBitmap failed")
	}
	defer procDeleteObject.Call(bmp)

	old, _, _ := procSelectObject.Call(mem, bmp)
	ok, _, _ := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(x), uintptr(y), srcCopy|captureBlt)
	procSelectObject.Call(mem, old) // GetDIBits needs the bitmap deselected
	if ok == 0 {
		return nil, errors.New("screen capture failed (the desktop may be locked or on a secure screen)")
	}

	bi := bitmapInfo{Size: 40, Width: w, Height: -h, Planes: 1, BitCount: 32} // negative height: top-down
	pix := make([]byte, int(w)*int(h)*4)
	lines, _, _ := procGetDIBits.Call(mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors)
	if int32(lines) != h {
		return nil, errors.New("GetDIBits failed")
	}
	return encodeBGRA(pix, int(w), int(h), maxWidth)
}
