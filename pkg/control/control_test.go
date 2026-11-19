package control

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionsPersistAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "sessions.json")

	s1, err := LoadSessions(path) // missing file -> empty store
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s1.Issue()
	if err != nil {
		t.Fatal(err)
	}

	// The file must hold only a hash, never the usable token.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), tok) {
		t.Fatal("sessions file contains the raw session token")
	}

	// A "restarted" server loading the same file accepts the session.
	s2, err := LoadSessions(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Valid(tok) {
		t.Fatal("session not valid after reload")
	}
	if s2.Valid("not-a-session") || s2.Valid("") {
		t.Fatal("unknown session accepted")
	}
	if s2.Count() != 1 {
		t.Fatalf("expected 1 session, got %d", s2.Count())
	}

	// Reset revokes everything, persistently.
	if err := s2.Reset(); err != nil {
		t.Fatal(err)
	}
	s3, err := LoadSessions(path)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Valid(tok) {
		t.Fatal("session still valid after reset")
	}
}

func TestLoadOrCreateToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing_token")
	calls := 0
	gen := func() (string, error) { calls++; return "tok-abc", nil }

	tok, created, err := LoadOrCreateToken(path, gen)
	if err != nil || !created || tok != "tok-abc" {
		t.Fatalf("first load: tok=%q created=%v err=%v", tok, created, err)
	}
	tok, created, err = LoadOrCreateToken(path, gen)
	if err != nil || created || tok != "tok-abc" || calls != 1 {
		t.Fatalf("second load should reuse saved token: tok=%q created=%v calls=%d err=%v", tok, created, calls, err)
	}
}
