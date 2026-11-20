package buildinfo

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.6.0", "1.5.0", true},
		{"v1.10.0", "1.9.9", true},
		{"2.0.0", "1.99.99", true},
		{"1.5.0", "1.5.0", false},
		{"1.5.0", "1.6.0", false},
		{"1.6.0", "1.6.0-rc1", true},
		{"1.6.0-rc1", "1.6.0", false},
		{"1.5.1", "v1.5.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
