package icon

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestPNG(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(PNG(64)))
	if err != nil || img.Bounds().Dx() != 64 {
		t.Fatalf("bad PNG: %v", err)
	}
}

func TestICOStructure(t *testing.T) {
	ico := ICO(16, 32, 256)
	var hdr [3]uint16
	binary.Read(bytes.NewReader(ico[:6]), binary.LittleEndian, &hdr)
	if hdr != [3]uint16{0, 1, 3} {
		t.Fatalf("header %v", hdr)
	}
	for i := 0; i < 3; i++ {
		e := ico[6+16*i : 6+16*(i+1)]
		size := binary.LittleEndian.Uint32(e[8:12])
		off := binary.LittleEndian.Uint32(e[12:16])
		if int(off+size) > len(ico) {
			t.Fatalf("entry %d out of range", i)
		}
		if _, err := png.Decode(bytes.NewReader(ico[off : off+size])); err != nil {
			t.Fatalf("entry %d not a PNG: %v", i, err)
		}
	}
	if ico[6+16*2] != 0 {
		t.Fatal("256px entry must use width 0")
	}
}
