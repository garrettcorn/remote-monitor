package pairing

import (
	"fmt"
	"strings"
)

// RenderTerminalQR formats a 2D boolean module matrix (true = dark, false = light)
// using Unicode half-block characters (▀ and ▄) so that 2 vertical rows fit into 1 terminal line.
func RenderTerminalQR(matrix [][]bool) string {
	if len(matrix) == 0 {
		return ""
	}
	size := len(matrix)
	border := 2
	totalSize := size + border*2

	// Build padded matrix
	grid := make([][]bool, totalSize)
	for y := 0; y < totalSize; y++ {
		grid[y] = make([]bool, totalSize)
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			grid[y+border][x+border] = matrix[y][x]
		}
	}

	var sb strings.Builder
	// In standard terminals, background is dark (black) and foreground is light (white).
	// To scan reliably, dark modules should be black and light modules white.
	// Inverting for ANSI: \033[7m (reverse) or block characters with ANSI colors.
	// Using ANSI background:
	// White bg = \033[47m, Black text = \033[30m
	for y := 0; y < totalSize; y += 2 {
		for x := 0; x < totalSize; x++ {
			top := grid[y][x]
			bottom := false
			if y+1 < totalSize {
				bottom = grid[y+1][x]
			}

			// Black = dark module (true), White = light module (false)
			if top && bottom {
				sb.WriteString("\033[30;40m \033[0m") // both dark
			} else if top && !bottom {
				sb.WriteString("\033[30;47m▀\033[0m") // top dark, bottom light
			} else if !top && bottom {
				sb.WriteString("\033[30;47m▄\033[0m") // top light, bottom dark
			} else {
				sb.WriteString("\033[37;47m \033[0m") // both light
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// GenerateSVG converts a QR boolean matrix to an SVG string.
func GenerateSVG(matrix [][]bool, moduleSize int) string {
	size := len(matrix)
	border := 4
	total := (size + border*2) * moduleSize

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, total, total, total, total)
	sb.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	sb.WriteString(`<path fill="#000000" d="`)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if matrix[y][x] {
				px := (x + border) * moduleSize
				py := (y + border) * moduleSize
				fmt.Fprintf(&sb, "M%d %dh%dv%dh-%dz ", px, py, moduleSize, moduleSize, moduleSize)
			}
		}
	}
	sb.WriteString(`"/></svg>`)
	return sb.String()
}
