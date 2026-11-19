package pairing

import (
	"errors"
)

// EncodeQR generates a boolean matrix (true = dark) representing a QR code
// for data, using byte mode and error-correction level L. It follows
// ISO/IEC 18004: Reed-Solomon ECC, block interleaving, format/version info,
// and automatic mask selection. Versions 1–40 are supported.
func EncodeQR(data string) ([][]bool, error) {
	payload := []byte(data)

	version := 0
	for v := 1; v <= 40; v++ {
		ccBits := 8
		if v >= 10 {
			ccBits = 16
		}
		if 4+ccBits+len(payload)*8 <= numDataCodewords(v)*8 {
			version = v
			break
		}
	}
	if version == 0 {
		return nil, errors.New("data too long for QR code")
	}

	q := newQR(version)
	q.drawFunctionPatterns()
	codewords := addECCAndInterleave(encodeBytes(payload, version), version)
	q.drawCodewords(codewords)

	// Pick the mask with the lowest penalty score.
	bestMask, bestPenalty := 0, -1
	for m := 0; m < 8; m++ {
		q.applyMask(m)
		q.drawFormatBits(m)
		p := q.penalty()
		if bestPenalty < 0 || p < bestPenalty {
			bestMask, bestPenalty = m, p
		}
		q.applyMask(m) // XOR again to undo
	}
	q.applyMask(bestMask)
	q.drawFormatBits(bestMask)

	return q.modules, nil
}

// EC level L tables, indexed by version (index 0 unused).
var eccCodewordsPerBlockL = [41]int{-1,
	7, 10, 15, 20, 26, 18, 20, 24, 30, 18,
	20, 24, 26, 30, 22, 24, 28, 30, 28, 28,
	28, 28, 30, 30, 26, 28, 30, 30, 30, 30,
	30, 30, 30, 30, 30, 30, 30, 30, 30, 30}

var numECCBlocksL = [41]int{-1,
	1, 1, 1, 1, 1, 2, 2, 2, 2, 4,
	4, 4, 4, 4, 6, 6, 6, 6, 7, 8,
	8, 9, 9, 10, 12, 12, 12, 13, 14, 15,
	16, 17, 18, 19, 19, 20, 21, 22, 24, 25}

func numRawDataModules(ver int) int {
	result := (16*ver+128)*ver + 64
	if ver >= 2 {
		numAlign := ver/7 + 2
		result -= (25*numAlign-10)*numAlign - 55
		if ver >= 7 {
			result -= 36
		}
	}
	return result
}

func numDataCodewords(ver int) int {
	return numRawDataModules(ver)/8 - eccCodewordsPerBlockL[ver]*numECCBlocksL[ver]
}

func alignmentPositions(ver int) []int {
	if ver == 1 {
		return nil
	}
	numAlign := ver/7 + 2
	step := (ver*8 + numAlign*3 + 5) / (numAlign*4 - 4) * 2
	result := make([]int, numAlign)
	result[0] = 6
	for i, pos := numAlign-1, ver*4+10; i >= 1; i, pos = i-1, pos-step {
		result[i] = pos
	}
	return result
}

// encodeBytes builds the data codeword sequence (byte mode, padded to capacity).
func encodeBytes(data []byte, ver int) []byte {
	var bits []bool
	appendBits(&bits, 0x4, 4) // byte mode
	ccBits := uint(8)
	if ver >= 10 {
		ccBits = 16
	}
	appendBits(&bits, uint32(len(data)), ccBits)
	for _, b := range data {
		appendBits(&bits, uint32(b), 8)
	}

	capBits := numDataCodewords(ver) * 8
	term := capBits - len(bits)
	if term > 4 {
		term = 4
	}
	appendBits(&bits, 0, uint(term))
	appendBits(&bits, 0, uint((8-len(bits)%8)%8))
	for pad := uint32(0xEC); len(bits) < capBits; pad ^= 0xEC ^ 0x11 {
		appendBits(&bits, pad, 8)
	}

	out := make([]byte, len(bits)/8)
	for i, b := range bits {
		if b {
			out[i>>3] |= 1 << (7 - uint(i&7))
		}
	}
	return out
}

func appendBits(bits *[]bool, val uint32, count uint) {
	for i := int(count) - 1; i >= 0; i-- {
		*bits = append(*bits, (val>>uint(i))&1 == 1)
	}
}

// addECCAndInterleave splits data into blocks, appends Reed-Solomon ECC to
// each, and interleaves them into the final codeword sequence.
func addECCAndInterleave(data []byte, ver int) []byte {
	numBlocks := numECCBlocksL[ver]
	blockECCLen := eccCodewordsPerBlockL[ver]
	rawCodewords := numRawDataModules(ver) / 8
	numShortBlocks := numBlocks - rawCodewords%numBlocks
	shortBlockLen := rawCodewords / numBlocks

	divisor := rsComputeDivisor(blockECCLen)
	blocks := make([][]byte, 0, numBlocks)
	k := 0
	for i := 0; i < numBlocks; i++ {
		datLen := shortBlockLen - blockECCLen
		if i >= numShortBlocks {
			datLen++
		}
		dat := append([]byte(nil), data[k:k+datLen]...)
		k += datLen
		ecc := rsComputeRemainder(dat, divisor)
		if i < numShortBlocks {
			dat = append(dat, 0) // placeholder, skipped below
		}
		blocks = append(blocks, append(dat, ecc...))
	}

	result := make([]byte, 0, rawCodewords)
	for i := range blocks[0] {
		for j, blk := range blocks {
			if i != shortBlockLen-blockECCLen || j >= numShortBlocks {
				result = append(result, blk[i])
			}
		}
	}
	return result
}

func rsComputeDivisor(degree int) []byte {
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := 0; j < degree; j++ {
			result[j] = rsMultiply(result[j], root)
			if j+1 < degree {
				result[j] ^= result[j+1]
			}
		}
		root = rsMultiply(root, 0x02)
	}
	return result
}

func rsComputeRemainder(data, divisor []byte) []byte {
	result := make([]byte, len(divisor))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i, d := range divisor {
			result[i] ^= rsMultiply(d, factor)
		}
	}
	return result
}

// rsMultiply multiplies in GF(2^8) modulo x^8 + x^4 + x^3 + x^2 + 1.
func rsMultiply(x, y byte) byte {
	z := 0
	for i := 7; i >= 0; i-- {
		z = (z << 1) ^ ((z >> 7) * 0x11D)
		z ^= int((y>>uint(i))&1) * int(x)
	}
	return byte(z)
}

// qr holds the module grid; modules[y][x] with y = row, x = column.
type qr struct {
	version    int
	size       int
	modules    [][]bool
	isFunction [][]bool
}

func newQR(ver int) *qr {
	size := ver*4 + 17
	q := &qr{version: ver, size: size}
	q.modules = make([][]bool, size)
	q.isFunction = make([][]bool, size)
	for i := range q.modules {
		q.modules[i] = make([]bool, size)
		q.isFunction[i] = make([]bool, size)
	}
	return q
}

func (q *qr) setFunction(x, y int, dark bool) {
	q.modules[y][x] = dark
	q.isFunction[y][x] = true
}

func (q *qr) drawFunctionPatterns() {
	// Timing patterns
	for i := 0; i < q.size; i++ {
		q.setFunction(6, i, i%2 == 0)
		q.setFunction(i, 6, i%2 == 0)
	}
	// Finder patterns (with separators)
	q.drawFinder(3, 3)
	q.drawFinder(q.size-4, 3)
	q.drawFinder(3, q.size-4)
	// Alignment patterns
	pos := alignmentPositions(q.version)
	n := len(pos)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (i == 0 && j == 0) || (i == 0 && j == n-1) || (i == n-1 && j == 0) {
				continue
			}
			q.drawAlignment(pos[i], pos[j])
		}
	}
	// Reserve format area (real bits drawn later) and draw version info
	q.drawFormatBits(0)
	q.drawVersion()
}

func (q *qr) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x, y := cx+dx, cy+dy
			if x < 0 || x >= q.size || y < 0 || y >= q.size {
				continue
			}
			dist := max(abs(dx), abs(dy))
			q.setFunction(x, y, dist != 2 && dist != 4)
		}
	}
}

func (q *qr) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			q.setFunction(cx+dx, cy+dy, max(abs(dx), abs(dy)) != 1)
		}
	}
}

func (q *qr) drawFormatBits(mask int) {
	const eclFormatBitsL = 1
	data := eclFormatBitsL<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	bits := (data<<10 | rem) ^ 0x5412
	bit := func(i int) bool { return (bits>>uint(i))&1 != 0 }

	// First copy, around the top-left finder
	for i := 0; i <= 5; i++ {
		q.setFunction(8, i, bit(i))
	}
	q.setFunction(8, 7, bit(6))
	q.setFunction(8, 8, bit(7))
	q.setFunction(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		q.setFunction(14-i, 8, bit(i))
	}
	// Second copy, split between top-right and bottom-left
	for i := 0; i < 8; i++ {
		q.setFunction(q.size-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		q.setFunction(8, q.size-15+i, bit(i))
	}
	q.setFunction(8, q.size-8, true) // dark module
}

func (q *qr) drawVersion() {
	if q.version < 7 {
		return
	}
	rem := q.version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	bits := q.version<<12 | rem
	for i := 0; i < 18; i++ {
		dark := (bits>>uint(i))&1 != 0
		a := q.size - 11 + i%3
		b := i / 3
		q.setFunction(a, b, dark)
		q.setFunction(b, a, dark)
	}
}

func (q *qr) drawCodewords(data []byte) {
	i := 0
	for right := q.size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < q.size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				upward := (right+1)&2 == 0
				y := vert
				if upward {
					y = q.size - 1 - vert
				}
				if !q.isFunction[y][x] && i < len(data)*8 {
					q.modules[y][x] = (data[i>>3]>>(7-uint(i&7)))&1 != 0
					i++
				}
			}
		}
	}
}

func (q *qr) applyMask(mask int) {
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			var invert bool
			switch mask {
			case 0:
				invert = (x+y)%2 == 0
			case 1:
				invert = y%2 == 0
			case 2:
				invert = x%3 == 0
			case 3:
				invert = (x+y)%3 == 0
			case 4:
				invert = (x/3+y/2)%2 == 0
			case 5:
				invert = x*y%2+x*y%3 == 0
			case 6:
				invert = (x*y%2+x*y%3)%2 == 0
			case 7:
				invert = ((x+y)%2+x*y%3)%2 == 0
			}
			if invert && !q.isFunction[y][x] {
				q.modules[y][x] = !q.modules[y][x]
			}
		}
	}
}

// penalty computes the ISO 18004 mask penalty score.
func (q *qr) penalty() int {
	n := q.size
	result := 0
	at := func(x, y int, horiz bool) bool {
		if horiz {
			return q.modules[y][x]
		}
		return q.modules[x][y]
	}

	finderA := []bool{true, false, true, true, true, false, true, false, false, false, false}
	finderB := []bool{false, false, false, false, true, false, true, true, true, false, true}

	for _, horiz := range []bool{true, false} {
		for y := 0; y < n; y++ {
			// Rule 1: runs of 5+ same-colour modules
			runLen := 1
			for x := 1; x <= n; x++ {
				if x < n && at(x, y, horiz) == at(x-1, y, horiz) {
					runLen++
					continue
				}
				if runLen >= 5 {
					result += 3 + runLen - 5
				}
				runLen = 1
			}
			// Rule 3: finder-like 1:1:3:1:1 patterns with light padding
			line := make([]bool, n+8) // 4 light modules either side
			for x := 0; x < n; x++ {
				line[x+4] = at(x, y, horiz)
			}
			for s := 0; s+11 <= len(line); s++ {
				if matchAt(line, s, finderA) || matchAt(line, s, finderB) {
					result += 40
				}
			}
		}
	}

	// Rule 2: 2x2 blocks of the same colour
	for y := 0; y < n-1; y++ {
		for x := 0; x < n-1; x++ {
			c := q.modules[y][x]
			if c == q.modules[y][x+1] && c == q.modules[y+1][x] && c == q.modules[y+1][x+1] {
				result += 3
			}
		}
	}

	// Rule 4: dark/light balance
	dark := 0
	for _, row := range q.modules {
		for _, m := range row {
			if m {
				dark++
			}
		}
	}
	total := n * n
	k := (abs(dark*20-total*10)+total-1)/total - 1
	result += k * 10
	return result
}

func matchAt(line []bool, s int, pat []bool) bool {
	for i, p := range pat {
		if line[s+i] != p {
			return false
		}
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
