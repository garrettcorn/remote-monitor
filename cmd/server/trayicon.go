package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

// trayIconPNG draws the tray icon: a blue rounded square with a white
// heartbeat line (no image files to ship).
func trayIconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	blue := color.RGBA{10, 132, 255, 255}
	white := color.RGBA{255, 255, 255, 255}
	r := float64(size) * 0.22 // corner radius
	inside := func(x, y float64) bool {
		s := float64(size)
		cx, cy := x, y
		if x < r {
			cx = r
		} else if x > s-r {
			cx = s - r
		}
		if y < r {
			cy = r
		} else if y > s-r {
			cy = s - r
		}
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= r*r
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if inside(float64(x)+0.5, float64(y)+0.5) {
				img.Set(x, y, blue)
			}
		}
	}
	// Heartbeat polyline in unit coordinates.
	pts := [][2]float64{{0.14, 0.55}, {0.36, 0.55}, {0.45, 0.30}, {0.56, 0.75}, {0.64, 0.50}, {0.86, 0.50}}
	thick := float64(size) / 14
	for i := 0; i+1 < len(pts); i++ {
		x0, y0 := pts[i][0]*float64(size), pts[i][1]*float64(size)
		x1, y1 := pts[i+1][0]*float64(size), pts[i+1][1]*float64(size)
		steps := size * 2
		for s := 0; s <= steps; s++ {
			t := float64(s) / float64(steps)
			px, py := x0+(x1-x0)*t, y0+(y1-y0)*t
			for dy := -thick; dy <= thick; dy++ {
				for dx := -thick; dx <= thick; dx++ {
					if dx*dx+dy*dy <= thick*thick/1.2 {
						img.Set(int(px+dx), int(py+dy), white)
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// trayIconICO wraps a PNG in an ICO container (supported since Windows Vista).
func trayIconICO(size int) []byte {
	p := trayIconPNG(size)
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, []uint16{0, 1, 1}) // reserved, type=icon, count
	dim := byte(size)
	if size >= 256 {
		dim = 0
	}
	buf.Write([]byte{dim, dim, 0, 0})                                         // width, height, colours, reserved
	_ = binary.Write(&buf, binary.LittleEndian, []uint16{1, 32})              // planes, bpp
	_ = binary.Write(&buf, binary.LittleEndian, []uint32{uint32(len(p)), 22}) // size, offset
	buf.Write(p)
	return buf.Bytes()
}
