package pairing

import (
	"errors"
)

// EncodeQR generates a boolean matrix representing the QR code for data.
// It supports alphanumeric and byte payloads up to Version 6 (up to ~130 bytes).
func EncodeQR(data string) ([][]bool, error) {
	bytes := []byte(data)
	version := pickVersion(len(bytes))
	if version > 6 {
		return nil, errors.New("data too long for embedded QR encoder")
	}

	size := 17 + 4*version
	matrix := make([][]bool, size)
	isFunction := make([][]bool, size)
	for i := range matrix {
		matrix[i] = make([]bool, size)
		isFunction[i] = make([]bool, size)
	}

	// 1. Draw Finder Patterns (top-left, top-right, bottom-left)
	drawFinder(matrix, isFunction, 0, 0)
	drawFinder(matrix, isFunction, size-7, 0)
	drawFinder(matrix, isFunction, 0, size-7)

	// 2. Draw Separators
	drawSeparators(matrix, isFunction, size)

	// 3. Draw Timing Patterns (row 6 and col 6)
	for i := 8; i < size-8; i++ {
		val := (i % 2) == 0
		matrix[6][i] = val
		isFunction[6][i] = true
		matrix[i][6] = val
		isFunction[i][6] = true
	}

	// 4. Draw Alignment Patterns (for version >= 2)
	if version >= 2 {
		pos := getAlignmentPositions(version)
		for _, r := range pos {
			for _, c := range pos {
				// Don't overlap finders
				if (r < 9 && c < 9) || (r < 9 && c >= size-9) || (r >= size-9 && c < 9) {
					continue
				}
				drawAlignment(matrix, isFunction, r, c)
			}
		}
	}

	// 5. Reserve Format Information Area
	reserveFormatInfo(isFunction, size)

	// 6. Encode Data into Bitstream
	bitstream := encodePayload(bytes, version)

	// 7. Place Data Bits in Matrix
	placeDataBits(matrix, isFunction, bitstream, size)

	// 8. Apply Mask Pattern (Mask 0: (row + col) % 2 == 0)
	applyMask(matrix, isFunction, size)

	// 9. Write Format Information (Mask 0, EC Level L = 01)
	writeFormatInfo(matrix, size)

	return matrix, nil
}

func pickVersion(byteLen int) int {
	// EC Level L capacities:
	// V1: 17 bytes, V2: 32 bytes, V3: 53 bytes, V4: 78 bytes, V5: 106 bytes, V6: 134 bytes
	if byteLen <= 17 {
		return 1
	} else if byteLen <= 32 {
		return 2
	} else if byteLen <= 53 {
		return 3
	} else if byteLen <= 78 {
		return 4
	} else if byteLen <= 106 {
		return 5
	}
	return 6
}

func drawFinder(matrix, isFunc [][]bool, r, c int) {
	for dr := 0; dr < 7; dr++ {
		for dc := 0; dc < 7; dc++ {
			isFunc[r+dr][c+dc] = true
			if dr == 0 || dr == 6 || dc == 0 || dc == 6 || (dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4) {
				matrix[r+dr][c+dc] = true
			} else {
				matrix[r+dr][c+dc] = false
			}
		}
	}
}

func drawSeparators(matrix, isFunc [][]bool, size int) {
	// Top-left
	for i := 0; i < 8; i++ {
		markFunc(matrix, isFunc, 7, i, false)
		markFunc(matrix, isFunc, i, 7, false)
	}
	// Top-right
	for i := 0; i < 8; i++ {
		markFunc(matrix, isFunc, 7, size-1-i, false)
		markFunc(matrix, isFunc, i, size-8, false)
	}
	// Bottom-left
	for i := 0; i < 8; i++ {
		markFunc(matrix, isFunc, size-8, i, false)
		markFunc(matrix, isFunc, size-1-i, 7, false)
	}
}

func markFunc(matrix, isFunc [][]bool, r, c int, val bool) {
	if r >= 0 && r < len(matrix) && c >= 0 && c < len(matrix) {
		matrix[r][c] = val
		isFunc[r][c] = true
	}
}

func getAlignmentPositions(version int) []int {
	switch version {
	case 2:
		return []int{6, 18}
	case 3:
		return []int{6, 22}
	case 4:
		return []int{6, 26}
	case 5:
		return []int{6, 30}
	case 6:
		return []int{6, 34}
	default:
		return []int{}
	}
}

func drawAlignment(matrix, isFunc [][]bool, centerR, centerC int) {
	for dr := -2; dr <= 2; dr++ {
		for dc := -2; dc <= 2; dc++ {
			r, c := centerR+dr, centerC+dc
			isFunc[r][c] = true
			if dr == -2 || dr == 2 || dc == -2 || dc == 2 || (dr == 0 && dc == 0) {
				matrix[r][c] = true
			} else {
				matrix[r][c] = false
			}
		}
	}
}

func reserveFormatInfo(isFunc [][]bool, size int) {
	// Around top-left
	for i := 0; i < 9; i++ {
		isFunc[8][i] = true
		isFunc[i][8] = true
	}
	// Around top-right
	for i := 0; i < 8; i++ {
		isFunc[8][size-1-i] = true
	}
	// Around bottom-left
	for i := 0; i < 8; i++ {
		isFunc[size-1-i][8] = true
	}
	// Dark module
	isFunc[size-8][8] = true
}

func encodePayload(data []byte, version int) []bool {
	// Mode indicator: 0100 (Byte mode)
	var bits []bool
	appendBits(&bits, 4, 4) // 0100

	// Character count indicator (8 bits for Version 1-9)
	appendBits(&bits, uint32(len(data)), 8)

	// Data bits
	for _, b := range data {
		appendBits(&bits, uint32(b), 8)
	}

	// Total capacity in bits for EC Level L
	totalCapBits := getCapacityBits(version)

	// Terminator (up to 4 zeroes)
	termLen := 4
	if totalCapBits-len(bits) < 4 {
		termLen = totalCapBits - len(bits)
	}
	appendBits(&bits, 0, uint(termLen))

	// Byte padding to multiple of 8
	padBits := (8 - (len(bits) % 8)) % 8
	appendBits(&bits, 0, uint(padBits))

	// Pad bytes: 0xEC, 0x11 alternating
	padBytes := []byte{0xEC, 0x11}
	padIdx := 0
	for len(bits) < totalCapBits {
		appendBits(&bits, uint32(padBytes[padIdx]), 8)
		padIdx = (padIdx + 1) % 2
	}

	return bits
}

func getCapacityBits(version int) int {
	// EC Level L data codewords
	codewords := []int{0, 19, 34, 55, 80, 108, 136}[version]
	return codewords * 8
}

func appendBits(bits *[]bool, val uint32, count uint) {
	for i := int(count) - 1; i >= 0; i-- {
		*bits = append(*bits, ((val>>i)&1) == 1)
	}
}

func placeDataBits(matrix, isFunc [][]bool, bits []bool, size int) {
	bitIdx := 0
	upwards := true

	for col := size - 1; col > 0; col -= 2 {
		if col == 6 {
			col-- // skip vertical timing column
		}

		rows := make([]int, size)
		if upwards {
			for i := 0; i < size; i++ {
				rows[i] = size - 1 - i
			}
		} else {
			for i := 0; i < size; i++ {
				rows[i] = i
			}
		}

		for _, row := range rows {
			for c := 0; c < 2; c++ {
				currCol := col - c
				if !isFunc[row][currCol] {
					if bitIdx < len(bits) {
						matrix[row][currCol] = bits[bitIdx]
						bitIdx++
					} else {
						matrix[row][currCol] = false
					}
				}
			}
		}
		upwards = !upwards
	}
}

func applyMask(matrix, isFunc [][]bool, size int) {
	// Mask 0: (row + col) % 2 == 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !isFunc[r][c] {
				if (r+c)%2 == 0 {
					matrix[r][c] = !matrix[r][c]
				}
			}
		}
	}
}

func writeFormatInfo(matrix [][]bool, size int) {
	// Format for Level L, Mask 0: 0b111011111000100 (15 bits with BCH code)
	formatBits := uint32(0x77C4)

	bits := make([]bool, 15)
	for i := 0; i < 15; i++ {
		bits[i] = ((formatBits >> (14 - i)) & 1) == 1
	}

	// Top-left format bits:
	// bit 0..5: (8, 0) to (8, 5)
	for i := 0; i <= 5; i++ {
		matrix[8][i] = bits[i]
	}
	// bit 6: (8, 7)
	matrix[8][7] = bits[6]
	// bit 7: (8, 8)
	matrix[8][8] = bits[7]
	// bit 8: (7, 8)
	matrix[7][8] = bits[8]
	// bit 9..14: (5, 8) down to (0, 8)
	matrix[5][8] = bits[9]
	matrix[4][8] = bits[10]
	matrix[3][8] = bits[11]
	matrix[2][8] = bits[12]
	matrix[1][8] = bits[13]
	matrix[0][8] = bits[14]

	// Bottom-left: bits 0..6: (size-1, 8) up to (size-7, 8)
	for i := 0; i < 7; i++ {
		matrix[size-1-i][8] = bits[i]
	}
	// Top-right: bits 7..14: (8, size-8) to (8, size-1)
	for i := 0; i < 8; i++ {
		matrix[8][size-8+i] = bits[7+i]
	}

	// Dark module always at (4*version + 9, 8)
	matrix[size-8][8] = true
}
