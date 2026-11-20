// Package buildinfo holds the version stamped into release builds with
// -ldflags "-X pc_monitor/pkg/buildinfo.Version=1.2.3".
package buildinfo

import (
	"strconv"
	"strings"
)

// Version is the release version without a leading "v", or "dev" for local builds.
var Version = "dev"

// Repo is the GitHub repository releases are published to.
const Repo = "garrettcorn/remote-monitor"

// IsDev reports whether this is an unversioned local build.
func IsDev() bool { return Version == "" || Version == "dev" }

// Newer reports whether version a is newer than b (both "x.y.z", optional
// leading "v"; pre-release suffixes like "-rc1" sort before the release).
func Newer(a, b string) bool {
	pa, preA := parse(a)
	pb, preB := parse(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	// Same numbers: a release beats a pre-release.
	return preA == "" && preB != ""
}

func parse(v string) ([3]int, string) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, pre, _ := strings.Cut(v, "-")
	var out [3]int
	for i, p := range strings.SplitN(v, ".", 3) {
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out, pre
}
