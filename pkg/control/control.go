// Package control manages one-time pairing PINs and the control sessions
// they unlock.
package control

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
// Only SHA-256 hashes are kept, so a persisted sessions file cannot be used
// to impersonate a paired dashboard.
type Sessions struct {
	mu     sync.Mutex
	hashes map[string]struct{}
	path   string // where to persist; empty = memory only
}

// NewSessions creates an empty in-memory session store.
func NewSessions() *Sessions {
	return &Sessions{hashes: make(map[string]struct{})}
}

type sessionsFile struct {
	Sessions []string `json:"sessions"` // hex SHA-256 of each session token
}

// LoadSessions creates a session store persisted at path, loading any
// sessions saved by a previous run (a missing file means none), so paired
// dashboards keep working across server restarts.
func LoadSessions(path string) (*Sessions, error) {
	s := &Sessions{hashes: make(map[string]struct{}), path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var f sessionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, h := range f.Sessions {
		s.hashes[h] = struct{}{}
	}
	return s, nil
}

func hashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// Issue creates a new random session token (and persists it).
func (s *Sessions) Issue() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hashes[hashToken(tok)] = struct{}{}
	if err := s.saveLocked(); err != nil {
		delete(s.hashes, hashToken(tok))
		return "", err
	}
	return tok, nil
}

// Valid reports whether tok is an issued session token.
func (s *Sessions) Valid(tok string) bool {
	if tok == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.hashes[hashToken(tok)]
	return ok
}

// Revoke forgets one session (a dashboard that unpaired itself). It
// reports whether the session existed.
func (s *Sessions) Revoke(tok string) (bool, error) {
	if tok == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := hashToken(tok)
	if _, ok := s.hashes[h]; !ok {
		return false, nil
	}
	delete(s.hashes, h)
	return true, s.saveLocked()
}

// Count returns the number of paired sessions.
func (s *Sessions) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hashes)
}

// Reset revokes every session (and persists the empty set).
func (s *Sessions) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hashes = make(map[string]struct{})
	return s.saveLocked()
}

func (s *Sessions) saveLocked() error {
	if s.path == "" {
		return nil
	}
	f := sessionsFile{Sessions: make([]string, 0, len(s.hashes))}
	for h := range s.hashes {
		f.Sessions = append(f.Sessions, h)
	}
	sort.Strings(f.Sessions)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(s.path, data)
}

// WriteFileAtomic writes data to path via a temp file and rename, creating
// the directory if needed, readable only by the current user.
func WriteFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// LoadOrCreateToken returns the pairing token stored at path, creating and
// saving one with gen if none exists. created reports a new token.
func LoadOrCreateToken(path string, gen func() (string, error)) (token string, created bool, err error) {
	data, err := os.ReadFile(path)
	if err == nil {
		if tok := strings.TrimSpace(string(data)); tok != "" {
			return tok, false, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	tok, err := gen()
	if err != nil {
		return "", false, err
	}
	if err := WriteFileAtomic(path, []byte(tok+"\n")); err != nil {
		return "", false, err
	}
	return tok, true, nil
}
