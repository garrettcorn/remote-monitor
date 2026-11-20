//go:build windows

package appicon

import (
	"bytes"
	"image"
	"image/png"
	"unsafe"

	"golang.org/x/sys/windows"
)

const supported = true

var (
	modshell32         = windows.NewLazySystemDLL("shell32.dll")
	moduser32          = windows.NewLazySystemDLL("user32.dll")
	modgdi32           = windows.NewLazySystemDLL("gdi32.dll")
	procExtractIconExW = modshell32.NewProc("ExtractIconExW")
	procGetIconInfo    = moduser32.NewProc("GetIconInfo")
	procDestroyIcon    = moduser32.NewProc("DestroyIcon")
	procGetDC          = moduser32.NewProc("GetDC")
	procReleaseDC      = moduser32.NewProc("ReleaseDC")
	procGetObjectW     = modgdi32.NewProc("GetObjectW")
	procGetDIBits      = modgdi32.NewProc("GetDIBits")
	procDeleteObject   = modgdi32.NewProc("DeleteObject")
)

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmap struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type bitmapInfoHeader struct {
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
}

// bitmapInfo is BITMAPINFO with room for the 2-entry palette GetDIBits may
// write for monochrome masks.
type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [2]uint32
}

func extract(path string) ([]byte, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var large windows.Handle
	if n, _, _ := procExtractIconExW.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&large)), 0, 1); n == 0 || large == 0 {
		return nil, ErrNoIcon
	}
	defer procDestroyIcon.Call(uintptr(large))
	img, err := iconImage(large)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func iconImage(h windows.Handle) (*image.NRGBA, error) {
	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&ii))); r == 0 {
		return nil, ErrNoIcon
	}
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(uintptr(ii.hbmMask))
	}
	if ii.hbmColor == 0 {
		return nil, ErrNoIcon // monochrome-only icon
	}
	defer procDeleteObject.Call(uintptr(ii.hbmColor))

	var bm bitmap
	if r, _, _ := procGetObjectW.Call(uintptr(ii.hbmColor), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 || bm.bmWidth <= 0 || bm.bmHeight <= 0 || bm.bmWidth > 256 {
		return nil, ErrNoIcon
	}
	w, hgt := int(bm.bmWidth), int(bm.bmHeight)
	color, err := dibits(ii.hbmColor, w, hgt)
	if err != nil {
		return nil, err
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, hgt))
	hasAlpha := false
	for i := 0; i < w*hgt; i++ {
		if color[i*4+3] != 0 {
			hasAlpha = true
			break
		}
	}
	var mask []byte
	if !hasAlpha && ii.hbmMask != 0 {
		mask, _ = dibits(ii.hbmMask, w, hgt) // white = transparent
	}
	for i := 0; i < w*hgt; i++ {
		b, g, r, a := color[i*4], color[i*4+1], color[i*4+2], color[i*4+3]
		if !hasAlpha {
			a = 255
			if mask != nil && mask[i*4] != 0 {
				a = 0
			}
		}
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = r, g, b, a
	}
	return img, nil
}

// dibits reads a bitmap as top-down 32-bit BGRA.
func dibits(bmp windows.Handle, w, h int) ([]byte, error) {
	dc, _, _ := procGetDC.Call(0)
	if dc == 0 {
		return nil, ErrNoIcon
	}
	defer procReleaseDC.Call(0, dc)
	bi := bitmapInfo{Header: bitmapInfoHeader{Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}}
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	pix := make([]byte, w*h*4)
	if r, _, _ := procGetDIBits.Call(dc, uintptr(bmp), 0, uintptr(h), uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&bi)), 0); r == 0 {
		return nil, ErrNoIcon
	}
	return pix, nil
}
