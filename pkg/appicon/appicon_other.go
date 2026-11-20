//go:build !windows

package appicon

// Linux (.desktop icon themes) and macOS (.icns in bundles) icons aren't
// extracted yet; the dashboard shows letter badges there.

const supported = false

func extract(string) ([]byte, error) { return nil, ErrNoIcon }
