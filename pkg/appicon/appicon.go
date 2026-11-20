// Package appicon extracts a program's icon as a PNG for the dashboard's
// process list. Icons are cached per executable path.
package appicon

import (
	"errors"
	"sync"
)

// ErrNoIcon means the program has no extractable icon (or the platform
// isn't supported); the dashboard shows a letter badge instead.
var ErrNoIcon = errors.New("no icon")

// Supported reports whether icons can be extracted on this platform.
func Supported() bool { return supported }

const maxCache = 512

var (
	mu    sync.Mutex
	cache = map[string][]byte{} // path -> PNG (nil: none)
)

// PNG returns the icon of the executable at path as a 32×32 PNG.
func PNG(path string) ([]byte, error) {
	if path == "" || !supported {
		return nil, ErrNoIcon
	}
	mu.Lock()
	b, ok := cache[path]
	mu.Unlock()
	if !ok {
		var err error
		b, err = extract(path)
		if err != nil {
			b = nil
		}
		mu.Lock()
		if len(cache) >= maxCache {
			cache = map[string][]byte{}
		}
		cache[path] = b
		mu.Unlock()
	}
	if b == nil {
		return nil, ErrNoIcon
	}
	return b, nil
}
