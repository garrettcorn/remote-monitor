package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"pc_monitor/pkg/buildinfo"
)

func TestVerifySignatureRejectsForgeries(t *testing.T) {
	sums := []byte("abc123  pc_monitor_1.6.0_windows_amd64.zip\n")
	// A signature from some other key must be rejected.
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	forged := base64.StdEncoding.EncodeToString(ed25519.Sign(otherKey, sums))
	if err := VerifySignature(sums, []byte(forged)); err == nil {
		t.Fatal("accepted a signature from a different key")
	}
	if err := VerifySignature(sums, []byte("not base64!!")); err == nil {
		t.Fatal("accepted garbage signature")
	}
	if err := VerifySignature(sums, nil); err == nil {
		t.Fatal("accepted empty signature")
	}
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("AAAA  pc_monitor_1.6.0_linux_amd64.tar.gz\nbbbb *pc_monitor_1.6.0_windows_amd64.zip\n")
	if got, err := checksumFor(sums, "pc_monitor_1.6.0_linux_amd64.tar.gz"); err != nil || got != "aaaa" {
		t.Fatalf("got %q %v", got, err)
	}
	if got, err := checksumFor(sums, "pc_monitor_1.6.0_windows_amd64.zip"); err != nil || got != "bbbb" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := checksumFor(sums, "missing.zip"); err == nil {
		t.Fatal("expected error for unlisted file")
	}
}

func TestExtract(t *testing.T) {
	// zip
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	w, _ := zw.Create("pc_monitor_windows_amd64.exe")
	w.Write([]byte("EXE"))
	zw.Close()
	if got, err := extract(zb.Bytes(), "x.zip", "pc_monitor_windows_amd64.exe"); err != nil || string(got) != "EXE" {
		t.Fatalf("zip: %q %v", got, err)
	}
	// tar.gz
	var tb bytes.Buffer
	gz := gzip.NewWriter(&tb)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "pc_monitor_linux_amd64", Mode: 0o755, Size: 3, Typeflag: tar.TypeReg})
	tw.Write([]byte("BIN"))
	tw.Close()
	gz.Close()
	if got, err := extract(tb.Bytes(), "x.tar.gz", "pc_monitor_linux_amd64"); err != nil || string(got) != "BIN" {
		t.Fatalf("tar: %q %v", got, err)
	}
	if _, err := extract(tb.Bytes(), "x.tar.gz", "missing"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReplaceFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "pc_monitor")
	os.WriteFile(target, []byte("old"), 0o755)
	if err := replaceFile(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "new" {
		t.Fatalf("target has %q", b)
	}
}

func TestReplacements(t *testing.T) {
	dir := t.TempDir()
	console := filepath.Join(dir, "pc_monitor.exe")
	app := filepath.Join(dir, "pc_monitor-app.exe")
	os.WriteFile(console, []byte("c"), 0o755)
	if runtime.GOOS != "windows" {
		if r := replacements(console); len(r) != 1 || r[0].inner != "pc_monitor" {
			t.Fatalf("unix: %+v", r)
		}
		return
	}
	if r := replacements(console); len(r) != 1 || r[0].target != console {
		t.Fatalf("console only: %+v", r)
	}
	os.WriteFile(app, []byte("a"), 0o755)
	r := replacements(app) // running as the tray app
	if len(r) != 2 || r[0].target != console || r[1].target != app {
		t.Fatalf("both: %+v", r)
	}
}

func TestAutoUpdate(t *testing.T) {
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v99.0.0","html_url":"https://example.invalid/r"}`))
	}))
	defer gh.Close()
	old := buildinfo.Version
	buildinfo.Version = "1.0.0"
	defer func() { buildinfo.Version = old }()

	newChecker := func(method string) (*Checker, *int) {
		c := NewChecker()
		c.latestURL = gh.URL
		c.status.Method = method
		installs := 0
		c.install = func(context.Context) error { installs++; return nil } // never touch real files
		return c, &installs
	}
	ctx := context.Background()

	// Off: only checks.
	c, n := newChecker(MethodSelf)
	if c.CheckAuto(ctx) || *n != 0 || !c.Status().Available {
		t.Fatalf("auto off: installed=%d status=%+v", *n, c.Status())
	}
	// On: installs.
	c.SetAuto(true)
	if !c.CheckAuto(ctx) || *n != 1 || !c.Status().AutoUpdate {
		t.Fatalf("auto on: installed=%d", *n)
	}
	// On, but a package manager owns the files: never installs.
	c, n = newChecker(MethodManaged)
	c.SetAuto(true)
	if c.CheckAuto(ctx) || *n != 0 {
		t.Fatalf("managed: installed=%d", *n)
	}
	// Install failure is reported, not retried in a loop.
	c, _ = newChecker(MethodSelf)
	c.SetAuto(true)
	c.install = func(context.Context) error { return errors.New("disk full") }
	if c.CheckAuto(ctx) || !strings.Contains(c.Status().Error, "disk full") {
		t.Fatalf("failure: %+v", c.Status())
	}
	// RunPeriodic calls installed() after an automatic install.
	c, _ = newChecker(MethodSelf)
	c.SetAuto(true)
	done := make(chan struct{})
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.RunPeriodic(rctx, time.Hour, func() { close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("installed callback not called")
	}
}
