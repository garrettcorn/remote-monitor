// Package update checks GitHub Releases for newer versions and installs
// them in place after verifying a signed checksum list.
//
// Trust chain: the release workflow signs checksums.txt with an ed25519 key
// (UPDATE_SIGNING_KEY secret); PublicKey below is compiled into every build.
// An update is installed only if checksums.txt.sig verifies against
// PublicKey AND the downloaded file's SHA-256 matches checksums.txt.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"pc_monitor/pkg/buildinfo"
)

// PublicKey verifies checksums.txt.sig (base64 ed25519 public key).
const PublicKey = "fwI95indDn9KDfoppPrlgLV20Ox4oH9Jcrz18wC4yTM="

const maxDownload = 200 << 20 // 200 MB safety cap

// Install methods.
const (
	MethodSelf     = "self"     // plain binary: replace in place
	MethodAppImage = "appimage" // replace the .AppImage file
	MethodManaged  = "managed"  // package manager owns the files
	MethodDev      = "dev"      // unversioned local build
)

// Status describes the update state shown to users.
type Status struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest,omitempty"`
	Available bool      `json:"available"`
	CanApply  bool      `json:"can_apply"`
	Method    string    `json:"method"`
	Note      string    `json:"note,omitempty"` // e.g. "Run: brew upgrade pc-monitor"
	URL       string    `json:"url,omitempty"`  // release page
	CheckedAt time.Time `json:"checked_at,omitempty"`
	Error     string    `json:"error,omitempty"`
	// AutoUpdate: the server installs updates by itself (config auto_update).
	AutoUpdate bool `json:"auto_update"`
}

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Checker caches the latest release and applies updates.
type Checker struct {
	mu      sync.Mutex
	latest  *release
	status  Status
	applyMu sync.Mutex
	client  *http.Client

	periodic  bool                            // RunPeriodic checks without auto-update
	latestURL string                          // GitHub "latest release" API
	install   func(ctx context.Context) error // Apply; overridable for tests
}

// NewChecker creates a checker for the running build.
func NewChecker() *Checker {
	method, note := InstallMethod()
	c := &Checker{
		client:    &http.Client{Timeout: 60 * time.Second},
		status:    Status{Current: buildinfo.Version, Method: method, Note: note},
		latestURL: "https://api.github.com/repos/" + buildinfo.Repo + "/releases/latest",
	}
	c.install = c.Apply
	return c
}

// Status returns the last known status without network access.
func (c *Checker) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Check queries GitHub for the latest release.
func (c *Checker) Check(ctx context.Context) Status {
	rel, err := c.fetchLatest(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.CheckedAt = time.Now()
	if err != nil {
		c.status.Error = err.Error()
		return c.status
	}
	c.latest = rel
	c.status.Error = ""
	c.status.Latest = strings.TrimPrefix(rel.TagName, "v")
	c.status.URL = rel.HTMLURL
	c.status.Available = !buildinfo.IsDev() && buildinfo.Newer(c.status.Latest, buildinfo.Version)
	c.status.CanApply = c.status.Available && (c.status.Method == MethodSelf || c.status.Method == MethodAppImage)
	return c.status
}

// SetAuto turns automatic installation on or off (see RunPeriodic).
func (c *Checker) SetAuto(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.AutoUpdate = on
}

// SetPeriodicChecks turns RunPeriodic's background checks on or off
// (automatic updates check regardless).
func (c *Checker) SetPeriodicChecks(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.periodic = on
}

func (c *Checker) shouldCheck() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.periodic || c.status.AutoUpdate
}

// RunPeriodic checks now and then every interval until ctx ends. When
// automatic updates are on (SetAuto) and an update can be applied, it is
// installed and installed() is called so the caller can restart.
func (c *Checker) RunPeriodic(ctx context.Context, interval time.Duration, installed func()) {
	go func() {
		for {
			if c.shouldCheck() && c.CheckAuto(ctx) && installed != nil {
				installed()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()
}

// CheckAuto runs one check and, if automatic updates are
// on and possible, installs the update. It reports whether one was installed.
func (c *Checker) CheckAuto(ctx context.Context) bool {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	st := c.Check(cctx)
	cancel()
	if !st.AutoUpdate || !st.CanApply || ctx.Err() != nil {
		return false
	}
	actx, acancel := context.WithTimeout(ctx, 5*time.Minute)
	defer acancel()
	if err := c.install(actx); err != nil {
		c.mu.Lock()
		c.status.Error = "automatic update failed: " + err.Error()
		c.mu.Unlock()
		return false
	}
	return true
}

func (c *Checker) fetchLatest(ctx context.Context) (*release, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.latestURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pc-monitor-updater")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" || rel.Draft {
		return nil, errors.New("no published release found")
	}
	return &rel, nil
}

// InstallMethod works out how this copy was installed.
func InstallMethod() (method, note string) {
	if buildinfo.IsDev() {
		return MethodDev, "Development build: updates are disabled"
	}
	if os.Getenv("APPIMAGE") != "" {
		return MethodAppImage, ""
	}
	exe, err := os.Executable()
	if err != nil {
		return MethodManaged, "Can't locate the running program"
	}
	exe, _ = filepath.EvalSymlinks(exe)
	p := filepath.ToSlash(strings.ToLower(exe))
	switch {
	case strings.Contains(p, "/cellar/") || strings.Contains(p, "/homebrew/"):
		return MethodManaged, "Installed with Homebrew. Run: brew upgrade pc-monitor"
	case strings.Contains(p, "/scoop/apps/"):
		return MethodManaged, "Installed with Scoop. Run: scoop update pc-monitor"
	case runtime.GOOS == "linux" && (strings.HasPrefix(p, "/usr/bin/") || strings.HasPrefix(p, "/usr/lib/")):
		return MethodManaged, "Installed from a .deb/.rpm package. Download the new package from the release page"
	}
	return MethodSelf, ""
}

// assetName is the release file to download for this platform and method.
func assetName(version, method string) string {
	if method == MethodAppImage {
		arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
		return fmt.Sprintf("pc_monitor-%s-%s.AppImage", version, arch)
	}
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("pc_monitor_%s_%s_%s.zip", version, runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("pc_monitor_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

// replacement maps a file inside the archive to the installed file it replaces.
type replacement struct{ inner, target string }

// replacements works out which installed files to update. On Windows the
// console program (pc_monitor.exe) and the windowless tray app
// (pc_monitor-app.exe) sit side by side and the server may be running as
// either, so both are updated when present.
func replacements(exe string) []replacement {
	if runtime.GOOS != "windows" {
		return []replacement{{"pc_monitor", exe}}
	}
	dir := filepath.Dir(exe)
	console, app := exe, filepath.Join(dir, "pc_monitor-app.exe")
	if strings.EqualFold(filepath.Base(exe), "pc_monitor-app.exe") {
		console, app = filepath.Join(dir, "pc_monitor.exe"), exe
	}
	var out []replacement
	for _, r := range []replacement{{"pc_monitor.exe", console}, {"pc_monitor-app.exe", app}} {
		if _, err := os.Stat(r.target); err == nil {
			out = append(out, r)
		}
	}
	return out
}

// Apply downloads, verifies and installs the latest release. The caller
// should restart the program afterwards.
func (c *Checker) Apply(ctx context.Context) error {
	c.applyMu.Lock()
	defer c.applyMu.Unlock()

	st := c.Check(ctx)
	if st.Error != "" {
		return errors.New(st.Error)
	}
	if !st.Available {
		return errors.New("already up to date")
	}
	if !st.CanApply {
		if st.Note != "" {
			return errors.New(st.Note)
		}
		return errors.New("this installation can't update itself")
	}
	c.mu.Lock()
	rel := c.latest
	c.mu.Unlock()

	assetURL := map[string]string{}
	for _, a := range rel.Assets {
		assetURL[a.Name] = a.URL
	}
	asset := assetName(st.Latest, st.Method)
	if assetURL[asset] == "" {
		return fmt.Errorf("release %s has no %s", rel.TagName, asset)
	}
	if assetURL["checksums.txt"] == "" || assetURL["checksums.txt.sig"] == "" {
		return errors.New("release is not signed; refusing to update")
	}

	sums, err := c.download(ctx, assetURL["checksums.txt"])
	if err != nil {
		return err
	}
	sig, err := c.download(ctx, assetURL["checksums.txt.sig"])
	if err != nil {
		return err
	}
	if err := VerifySignature(sums, sig); err != nil {
		return err
	}
	want, err := checksumFor(sums, asset)
	if err != nil {
		return err
	}
	data, err := c.download(ctx, assetURL[asset])
	if err != nil {
		return err
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}

	if st.Method == MethodAppImage {
		return replaceFile(os.Getenv("APPIMAGE"), data)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	// Extract everything first so a bad archive changes nothing.
	reps := replacements(exe)
	bins := make([][]byte, len(reps))
	for i, r := range reps {
		if bins[i], err = extract(data, asset, r.inner); err != nil {
			return err
		}
	}
	files := make([]newFile, len(reps))
	for i, r := range reps {
		files[i] = newFile{r.target, bins[i]}
	}
	return replaceFiles(files)
}

// VerifySignature checks checksums.txt against its base64 signature.
func VerifySignature(sums, sigB64 []byte) error {
	pub, err := base64.StdEncoding.DecodeString(PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid embedded public key")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil || !ed25519.Verify(pub, sums, sig) {
		return errors.New("release signature is invalid; refusing to update")
	}
	return nil
}

// checksumFor finds a file's hash in sha256sum-style output.
func checksumFor(sums []byte, name string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("%s is not listed in checksums.txt", name)
}

func (c *Checker) download(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "pc-monitor-updater")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, errors.New("download too large")
	}
	return data, nil
}

// extract pulls one file out of a .zip or .tar.gz archive.
func extract(data []byte, archiveName, inner string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == inner {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, maxDownload))
			}
		}
		return nil, fmt.Errorf("%s not found in %s", inner, archiveName)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s not found in %s", inner, archiveName)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == inner && h.Typeflag == tar.TypeReg {
			return io.ReadAll(io.LimitReader(tr, maxDownload))
		}
	}
}

// newFile is one file an update installs.
type newFile struct {
	target string
	data   []byte
}

// replaceFile installs one file (AppImage, single binaries).
func replaceFile(target string, data []byte) error {
	return replaceFiles([]newFile{{target, data}})
}

// replaceFiles installs all files or none. Every new file is written next to
// its target first; then each target is swapped. A running executable can't
// be overwritten on Windows but can be renamed, so the old file moves aside
// (see asideName; removed on a later start by CleanupOld). If any swap
// fails, the ones already done are put back.
func replaceFiles(files []newFile) (err error) {
	type staged struct{ target, tmp, aside string }
	var st []staged
	defer func() {
		for _, s := range st {
			_ = os.Remove(s.tmp) // no-op once renamed into place
		}
	}()
	for _, f := range files {
		tmp, err := writeTemp(f.target, f.data)
		if err != nil {
			return err
		}
		st = append(st, staged{target: f.target, tmp: tmp})
	}

	done := 0
	defer func() {
		if err == nil {
			return
		}
		// Roll back in reverse so the install is never half-updated.
		for i := done - 1; i >= 0; i-- {
			s := st[i]
			if s.aside == "" {
				continue
			}
			_ = os.Remove(s.target)
			_ = os.Rename(s.aside, s.target)
		}
	}()
	for i := range st {
		s := &st[i]
		if runtime.GOOS == "windows" {
			s.aside = asideName(s.target)
			if err := os.Rename(s.target, s.aside); err != nil {
				return fmt.Errorf("can't move %s aside: %w", filepath.Base(s.target), err)
			}
			if err := os.Rename(s.tmp, s.target); err != nil {
				_ = os.Rename(s.aside, s.target)
				return err
			}
		} else if err := os.Rename(s.tmp, s.target); err != nil {
			return err
		}
		done = i + 1
	}
	return nil
}

func writeTemp(target string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".new-*")
	if err != nil {
		return "", fmt.Errorf("can't write next to %s (try running as the user who installed it): %w", target, err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	if err := os.Chmod(name, 0o755); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// asideName picks where a replaced file goes: <name>.old, or a unique
// <name>.old-<n> when an earlier .old is still locked (e.g. a tray icon
// still running from a previous version).
func asideName(target string) string {
	old := target + ".old"
	if err := os.Remove(old); err == nil || errors.Is(err, os.ErrNotExist) {
		return old
	}
	return fmt.Sprintf("%s.old-%d", target, time.Now().UnixNano())
}

// CleanupOld removes files set aside by earlier updates (and any staged
// files an interrupted update left), skipping those still in use.
func CleanupOld() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cleanupDir(filepath.Dir(exe))
}

func cleanupDir(dir string) {
	for _, pat := range []string{"pc_monitor*.old", "pc_monitor*.old-*", "pc_monitor*.new-*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pat))
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}
