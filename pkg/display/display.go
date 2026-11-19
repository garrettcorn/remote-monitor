// Package display lists the host's connected displays and captures their
// contents as downscaled JPEG previews.
package display

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"sync"
)

// Display describes one connected display.
type Display struct {
	Index        int    `json:"index"`
	Name         string `json:"name"`
	Device       string `json:"device,omitempty"` // e.g. \\.\DISPLAY1, HDMI-1
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RefreshHz    int    `json:"refresh_hz,omitempty"`
	BitsPerPixel int    `json:"bits_per_pixel,omitempty"`
	ScalePercent int    `json:"scale_percent,omitempty"`
	Primary      bool   `json:"primary"`
	X            int    `json:"x"`
	Y            int    `json:"y"`

	// HardwareID links a display to its monitor's PnP identity (Windows),
	// used to look up a friendly model name. Not sent to clients.
	HardwareID string `json:"-"`
}

// ErrUnsupported is returned where capture isn't available.
var ErrUnsupported = errors.New("screen capture is not supported on this platform")

// List returns the connected displays (primary first).
func List() ([]Display, error) { return list() }

// CaptureSupported reports whether Capture works on this platform.
func CaptureSupported() bool { return captureSupported }

var captureMu sync.Mutex // one capture at a time keeps CPU use bounded

// Capture grabs display index and returns a JPEG no wider than maxWidth.
func Capture(index, maxWidth int) ([]byte, error) {
	if !captureSupported {
		return nil, ErrUnsupported
	}
	if maxWidth < 320 {
		maxWidth = 320
	}
	if maxWidth > 3840 {
		maxWidth = 3840
	}
	captureMu.Lock()
	defer captureMu.Unlock()
	return capture(index, maxWidth)
}

// encodeBGRA downsizes a top-down BGRA (or BGRX) buffer with a box filter
// and encodes it as JPEG.
func encodeBGRA(pix []byte, w, h, maxWidth int) ([]byte, error) {
	if w <= 0 || h <= 0 || len(pix) < w*h*4 {
		return nil, errors.New("invalid capture buffer")
	}
	tw := w
	if tw > maxWidth {
		tw = maxWidth
	}
	th := h * tw / w
	if th < 1 {
		th = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, tw, th))
	for ty := 0; ty < th; ty++ {
		y0, y1 := ty*h/th, (ty+1)*h/th
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for tx := 0; tx < tw; tx++ {
			x0, x1 := tx*w/tw, (tx+1)*w/tw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, n uint32
			for y := y0; y < y1; y++ {
				row := y * w * 4
				for x := x0; x < x1; x++ {
					i := row + x*4
					b += uint32(pix[i])
					g += uint32(pix[i+1])
					r += uint32(pix[i+2])
					n++
				}
			}
			o := img.PixOffset(tx, ty)
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 72}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// downscaleJPEG re-encodes an existing image (used where the OS hands us a
// file, e.g. macOS screencapture).
func downscaleJPEG(data []byte, maxWidth int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			i := (y*w + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = uint8(bl>>8), uint8(g>>8), uint8(r>>8), 255
		}
	}
	return encodeBGRA(pix, w, h, maxWidth)
}
