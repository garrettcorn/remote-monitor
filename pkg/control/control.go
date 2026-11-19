// Package control manages one-time pairing PINs and the control sessions
// they unlock.
package control

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
)

// MaxFailures is how many wrong PIN attempts are allowed before the PIN
// is replaced, which caps guessing at 5 tries per 10,000-value PIN.
const MaxFailures = 5

// PIN is a rotating one-time 4-digit pairing PIN. Each successful use, or
// MaxFailures wrong attempts, replaces it and reports the new PIN via onNew
// (shown on the server console only).
type PIN struct {
	mu       sync.Mutex
	pin      string
	failures int
	onNew    func(pin, reason string)
}

// NewPIN creates a PIN manager with a fresh random PIN.
func NewPIN(onNew func(pin, reason string)) (*PIN, error) {
	pin, err := randomPIN()
	if err != nil {
		return nil, err
	}
	return &PIN{pin: pin, onNew: onNew}, nil
}

// Current returns the PIN to display on the server console.
func (p *PIN) Current() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pin
}

// Verify checks an attempt. A correct PIN is consumed and replaced.
func (p *PIN) Verify(attempt string) bool {
	p.mu.Lock()
	ok := len(attempt) == len(p.pin) && subtle.ConstantTimeCompare([]byte(attempt), []byte(p.pin)) == 1
	reason := ""
	if ok {
		reason = "PIN used"
	} else {
		p.failures++
		if p.failures >= MaxFailures {
			reason = fmt.Sprintf("%d wrong attempts", p.failures)
		}
	}
	var newPIN string
	if reason != "" {
		if pin, err := randomPIN(); err == nil {
			p.pin = pin
		}
		p.failures = 0
		newPIN = p.pin
	}
	onNew := p.onNew
	p.mu.Unlock()

	if newPIN != "" && onNew != nil {
		onNew(newPIN, reason)
	}
	return ok
}

func randomPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// Sessions holds control session tokens issued after a successful PIN.
// They live in memory, so restarting the server requires re-pairing.
type Sessions struct {
	mu     sync.Mutex
	tokens map[string]struct{}
}

// NewSessions creates an empty session store.
func NewSessions() *Sessions {
	return &Sessions{tokens: make(map[string]struct{})}
}

// Issue creates a new random session token.
func (s *Sessions) Issue() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	s.tokens[tok] = struct{}{}
	s.mu.Unlock()
	return tok, nil
}

// Valid reports whether tok is an issued session token.
func (s *Sessions) Valid(tok string) bool {
	if tok == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.tokens[tok]
	return ok
}
