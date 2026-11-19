package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	versionHeaderRegex = regexp.MustCompile(`^## \[(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)\] - (\d{4}-\d{2}-\d{2})$`)
	categoryHeaderRegex = regexp.MustCompile(`^### (Added|Changed|Removed|Fixed)$`)
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	changelogPath := "CHANGELOG.md"

	switch command {
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		filePtr := fs.String("file", changelogPath, "Path to CHANGELOG.md")
		_ = fs.Parse(os.Args[2:])

		if err := validateChangelog(*filePtr); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Changelog does not conform to common-changelog.org:\n%v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ CHANGELOG.md conforms to common-changelog.org standards.")

	case "extract":
		fs := flag.NewFlagSet("extract", flag.ExitOnError)
		filePtr := fs.String("file", changelogPath, "Path to CHANGELOG.md")
		outPtr := fs.String("o", "", "Output file (defaults to stdout)")
		_ = fs.Parse(os.Args[2:])

		args := fs.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: changelog extract [-file CHANGELOG.md] [-o output.md] <version-or-tag>")
			os.Exit(1)
		}

		targetVer := strings.TrimPrefix(args[0], "v")
		notes, err := extractReleaseNotes(*filePtr, targetVer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting notes: %v\n", err)
			os.Exit(1)
		}

		if *outPtr != "" {
			if err := os.WriteFile(*outPtr, []byte(notes), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Release notes for [%s] written to %s\n", targetVer, *outPtr)
		} else {
			fmt.Print(notes)
		}

	case "bump":
		fs := flag.NewFlagSet("bump", flag.ExitOnError)
		filePtr := fs.String("file", changelogPath, "Path to CHANGELOG.md")
		_ = fs.Parse(os.Args[2:])

		args := fs.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: changelog bump [-file CHANGELOG.md] <new-version>")
			os.Exit(1)
		}

		newVer := strings.TrimPrefix(args[0], "v")
		if err := bumpVersion(*filePtr, newVer); err != nil {
			fmt.Fprintf(os.Stderr, "Error bumping changelog: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Added [%s] release section to %s\n", newVer, *filePtr)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`PC Monitor - Common Changelog Tool (https://common-changelog.org)

Usage:
  changelog validate [-file CHANGELOG.md]
  changelog extract  [-file CHANGELOG.md] [-o notes.md] <version>
  changelog bump     [-file CHANGELOG.md] <new-version>

Examples:
  go run ./cmd/changelog validate
  go run ./cmd/changelog extract v1.0.0 -o release_notes.md
  go run ./cmd/changelog bump 1.1.0`)
}

func validateChangelog(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return errors.New("file is empty")
	}

	// 1. Check title
	titleFound := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "# Changelog" {
			titleFound = true
			break
		}
	}
	if !titleFound {
		return errors.New("missing '# Changelog' top-level title")
	}

	// 2. Validate line by line
	var validationErrors []string
	versionCount := 0
	inRelease := false

	for i, rawLine := range lines {
		lineNum := i + 1
		line := strings.TrimRight(rawLine, "\r")

		// Check for unreleased section (forbidden by Common Changelog)
		if strings.Contains(strings.ToLower(line), "[unreleased]") {
			validationErrors = append(validationErrors, fmt.Sprintf("line %d: Common Changelog forbids an '[Unreleased]' section; releases must be dated YYYY-MM-DD", lineNum))
		}

		// Check version header
		if strings.HasPrefix(line, "## ") {
			if versionHeaderRegex.MatchString(line) {
				versionCount++
				inRelease = true
			} else if !strings.HasPrefix(line, "## [") {
				// Might be non-version
				continue
			} else {
				validationErrors = append(validationErrors, fmt.Sprintf("line %d: invalid version header %q (expected format: '## [X.Y.Z] - YYYY-MM-DD')", lineNum, line))
			}
			continue
		}

		// Check category header
		if strings.HasPrefix(line, "### ") {
			if !inRelease {
				validationErrors = append(validationErrors, fmt.Sprintf("line %d: category header outside of release section", lineNum))
			} else if !categoryHeaderRegex.MatchString(line) {
				validationErrors = append(validationErrors, fmt.Sprintf("line %d: invalid category %q (Common Changelog only allows Added, Changed, Removed, Fixed)", lineNum, line))
			}
			continue
		}

		// Check forbidden categories
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "### deprecated") {
			validationErrors = append(validationErrors, fmt.Sprintf("line %d: 'Deprecated' category is deprecated in Common Changelog (use 'Changed' instead)", lineNum))
		}
		if strings.HasPrefix(lower, "### security") {
			validationErrors = append(validationErrors, fmt.Sprintf("line %d: 'Security' category is deprecated in Common Changelog (use 'Changed' instead)", lineNum))
		}
	}

	if versionCount == 0 {
		validationErrors = append(validationErrors, "no valid release sections found (format: ## [X.Y.Z] - YYYY-MM-DD)")
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

func extractReleaseNotes(path, version string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	headerPrefix := fmt.Sprintf("## [%s]", version)

	var output strings.Builder
	capturing := false
	found := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## ") {
			if strings.HasPrefix(line, headerPrefix) {
				capturing = true
				found = true
				output.WriteString(line + "\n")
				continue
			} else if capturing {
				// Reached next version
				break
			}
		}

		// Check if we hit the bottom link references like [1.0.0]:
		if capturing && strings.HasPrefix(line, "[") && strings.Contains(line, "]: ") {
			break
		}

		if capturing {
			output.WriteString(line + "\n")
		}
	}

	if !found {
		return "", fmt.Errorf("version [%s] not found in %s", version, path)
	}

	return strings.TrimSpace(output.String()) + "\n", nil
}

func bumpVersion(path, newVer string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if strings.Contains(content, fmt.Sprintf("## [%s]", newVer)) {
		return fmt.Errorf("version [%s] already exists in %s", newVer, path)
	}

	today := time.Now().UTC().Format("2006-01-02")
	newHeader := fmt.Sprintf("## [%s] - %s\n\n### Added\n- Add new features here (#)\n\n### Changed\n- Detail modified behavior here (#)\n\n### Fixed\n- Document bug fixes here (#)\n", newVer, today)

	lines := strings.Split(content, "\n")
	insertIdx := -1

	// Insert after the description line (under # Changelog)
	for i, l := range lines {
		if strings.HasPrefix(l, "## [") {
			insertIdx = i
			break
		}
	}

	var newLines []string
	if insertIdx >= 0 {
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, newHeader)
		newLines = append(newLines, lines[insertIdx:]...)
	} else {
		newLines = append(lines, "\n"+newHeader)
	}

	// Add link definition at bottom
	repoURL := "https://github.com/your-username/pc_monitor"
	linkLine := fmt.Sprintf("[%s]: %s/releases/tag/v%s", newVer, repoURL, newVer)
	newLines = append(newLines, linkLine)

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}
