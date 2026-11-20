package telemetry

import "testing"

func TestWindowsProductName(t *testing.T) {
	cases := []struct{ product, build, want string }{
		{"Windows 10 Pro", "26200", "Windows 11 Pro"},
		{"Windows 10 Home", "22000", "Windows 11 Home"},
		{"Windows 10 Enterprise LTSC 2021", "19044", "Windows 10 Enterprise LTSC 2021"},
		{"Windows 10 Pro", "19045", "Windows 10 Pro"},
		{"Windows Server 2022 Datacenter", "20348", "Windows Server 2022 Datacenter"},
		{"Windows Server 2025 Standard", "26100", "Windows Server 2025 Standard"},
		{"Windows 11 Pro", "26100", "Windows 11 Pro"}, // in case Microsoft ever fixes it
		{"Windows 10 Pro", "", "Windows 10 Pro"},
	}
	for _, c := range cases {
		if got := windowsProductName(c.product, c.build); got != c.want {
			t.Errorf("windowsProductName(%q, %q) = %q, want %q", c.product, c.build, got, c.want)
		}
	}
}
