package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateChangelog(t *testing.T) {
	valid := `# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-11-19

### Added
- Add initial feature (#1)

[1.0.0]: https://github.com/example/repo/releases/tag/v1.0.0
`
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "CHANGELOG.md")
	if err := os.WriteFile(validPath, []byte(valid), 0644); err != nil {
		t.Fatal(err)
	}

	if err := validateChangelog(validPath); err != nil {
		t.Fatalf("expected valid changelog, got error: %v", err)
	}

	// Test unreleased section rejection (Common Changelog forbids [Unreleased])
	invalidUnreleased := strings.Replace(valid, "## [1.0.0] - 2026-11-19", "## [Unreleased]", 1)
	invalidPath := filepath.Join(tmpDir, "INVALID.md")
	_ = os.WriteFile(invalidPath, []byte(invalidUnreleased), 0644)
	if err := validateChangelog(invalidPath); err == nil {
		t.Fatal("expected error for [Unreleased] section, but got nil")
	}

	// Test extractReleaseNotes
	notes, err := extractReleaseNotes(validPath, "1.0.0")
	if err != nil {
		t.Fatalf("failed to extract notes: %v", err)
	}
	if !strings.Contains(notes, "Add initial feature") {
		t.Fatalf("expected notes to contain feature, got: %s", notes)
	}
}
