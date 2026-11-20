package telemetry

import (
	"strconv"
	"strings"
)

// windowsProductName corrects the registry's ProductName for Windows 11,
// which Microsoft left as "Windows 10 …" (e.g. "Windows 10 Pro"). Builds
// 22000 and later are Windows 11.
func windowsProductName(product, build string) string {
	n, err := strconv.Atoi(strings.TrimSpace(build))
	if err == nil && n >= 22000 && strings.HasPrefix(product, "Windows 10") {
		return "Windows 11" + strings.TrimPrefix(product, "Windows 10")
	}
	return product
}
