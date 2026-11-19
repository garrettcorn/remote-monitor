package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"pc_monitor/pkg/control"
	"pc_monitor/pkg/deck"
	"pc_monitor/pkg/display"
	"pc_monitor/pkg/power"
	"pc_monitor/pkg/telemetry"
)

func do(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func errCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	return e.Error
}

func TestReadOnlyServer(t *testing.T) {
	collector := telemetry.NewCollector(500 * time.Millisecond)
	srv := NewServer("test-server", "secret-token", collector, Options{ReadOnly: true})
	h := srv.Handler()

	// Health is public and reports control disabled.
	w := do(t, h, "GET", "/api/health", "", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"control_enabled":false`) {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}

	// Telemetry needs the token, but no PIN/session in read-only mode.
	if w := do(t, h, "GET", "/api/telemetry", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
	w = do(t, h, "GET", "/api/telemetry?token=secret-token", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d: %s", w.Code, w.Body.String())
	}
	var snap telemetry.SystemSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatalf("failed to parse snapshot JSON: %v", err)
	}
	if snap.Host.Hostname == "" {
		t.Errorf("expected hostname, got empty")
	}

	// Control endpoints are refused.
	w = do(t, h, "POST", "/api/power?token=secret-token", `{"action":"shutdown"}`, nil)
	if w.Code != http.StatusForbidden || errCode(t, w) != "read_only" {
		t.Fatalf("expected 403 read_only, got %d %s", w.Code, w.Body.String())
	}
}

func TestPINPairingAndControl(t *testing.T) {
	var announced []string
	pin, err := control.NewPIN(func(p, _ string) { announced = append(announced, p) })
	if err != nil {
		t.Fatal(err)
	}
	collector := telemetry.NewCollector(500 * time.Millisecond)
	srv := NewServer("test-server", "secret-token", collector, Options{PIN: pin})
	var powered []power.Action
	srv.doPower = func(a power.Action) error { powered = append(powered, a); return nil }
	h := srv.Handler()
	tok := map[string]string{"X-Pairing-Token": "secret-token"}

	// Token alone is not enough when control is enabled.
	w := do(t, h, "GET", "/api/telemetry", "", tok)
	if w.Code != http.StatusUnauthorized || errCode(t, w) != "pin_required" {
		t.Fatalf("expected 401 pin_required, got %d %s", w.Code, w.Body.String())
	}

	// Pairing needs the token too.
	if w := do(t, h, "POST", "/api/pair", `{"pin":"`+pin.Current()+`"}`, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for pairing without token, got %d", w.Code)
	}

	// Correct PIN yields a session, and the PIN is replaced.
	first := pin.Current()
	w = do(t, h, "POST", "/api/pair", `{"pin":"`+first+`"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("pair failed: %d %s", w.Code, w.Body.String())
	}
	var pair struct {
		Session string `json:"session"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &pair)
	if pair.Session == "" {
		t.Fatal("expected a session token")
	}
	if len(announced) != 1 || announced[0] != pin.Current() {
		t.Fatalf("expected new PIN to be announced once, got %v", announced)
	}

	// The used PIN cannot be reused (unless the new random PIN collides).
	if pin.Current() != first {
		w = do(t, h, "POST", "/api/pair", `{"pin":"`+first+`"}`, tok)
		if w.Code != http.StatusForbidden || errCode(t, w) != "invalid_pin" {
			t.Fatalf("expected reused PIN to fail, got %d %s", w.Code, w.Body.String())
		}
	}

	// Session grants telemetry and control.
	auth := map[string]string{"X-Pairing-Token": "secret-token", "X-Session": pair.Session}
	if w := do(t, h, "GET", "/api/telemetry", "", auth); w.Code != http.StatusOK {
		t.Fatalf("expected telemetry with session, got %d", w.Code)
	}
	if w := do(t, h, "POST", "/api/power", `{"action":"reboot-now"}`, auth); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown action, got %d", w.Code)
	}
	if w := do(t, h, "GET", "/api/power", "", auth); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", w.Code)
	}
	if w := do(t, h, "POST", "/api/power", `{"action":"restart"}`, auth); w.Code != http.StatusOK {
		t.Fatalf("expected restart to be accepted, got %d %s", w.Code, w.Body.String())
	}
	if len(powered) != 1 || powered[0] != power.Restart {
		t.Fatalf("expected one restart call, got %v", powered)
	}

	// A bogus session is rejected.
	bad := map[string]string{"X-Pairing-Token": "secret-token", "X-Session": "nope"}
	if w := do(t, h, "POST", "/api/power", `{"action":"shutdown"}`, bad); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad session, got %d", w.Code)
	}
	if len(powered) != 1 {
		t.Fatalf("power must not run without a valid session, got %v", powered)
	}
}

func TestPINRotatesAfterFailures(t *testing.T) {
	rotations := 0
	pin, _ := control.NewPIN(func(string, string) { rotations++ })
	for i := 0; i < control.MaxFailures; i++ {
		pin.Verify("not-a-pin")
	}
	if rotations != 1 {
		t.Fatalf("expected PIN to rotate after %d failures, rotations=%d", control.MaxFailures, rotations)
	}
}

func TestDeckEndpoints(t *testing.T) {
	pin, _ := control.NewPIN(nil)
	collector := telemetry.NewCollector(500 * time.Millisecond)
	actionsFile := t.TempDir() + "/actions.json"
	_ = os.WriteFile(actionsFile, []byte(`{"launch":[{"id":"notes","name":"Notepad","path":"notepad.exe","args":"secret.txt"}]}`), 0o600)
	srv := NewServer("test-server", "secret-token", collector, Options{PIN: pin, ActionsFile: actionsFile})
	var pressed []deck.Action
	srv.doDeck = func(a deck.Action, _ string) error { pressed = append(pressed, a); return nil }
	h := srv.Handler()

	w := do(t, h, "POST", "/api/pair", `{"pin":"`+pin.Current()+`"}`, map[string]string{"X-Pairing-Token": "secret-token"})
	var pair struct {
		Session string `json:"session"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &pair)
	auth := map[string]string{"X-Pairing-Token": "secret-token", "X-Session": pair.Session}

	// Listing exposes names only, never paths/args.
	w = do(t, h, "GET", "/api/deck/actions", "", auth)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"Notepad"`) || strings.Contains(w.Body.String(), "secret.txt") || strings.Contains(w.Body.String(), "notepad.exe") {
		t.Fatalf("actions listing: %d %s", w.Code, w.Body.String())
	}

	// Pressing needs a session.
	if w := do(t, h, "POST", "/api/deck/press", `{"type":"lock"}`, map[string]string{"X-Pairing-Token": "secret-token"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("press without session: %d", w.Code)
	}
	if w := do(t, h, "POST", "/api/deck/press", `{"type":"media","key":"play_pause"}`, auth); w.Code != http.StatusOK {
		t.Fatalf("press: %d %s", w.Code, w.Body.String())
	}
	if len(pressed) != 1 || pressed[0].Key != "play_pause" {
		t.Fatalf("expected one press, got %+v", pressed)
	}

	// Read-only servers refuse deck actions.
	ro := NewServer("ro", "secret-token", collector, Options{ReadOnly: true})
	if w := do(t, ro.Handler(), "POST", "/api/deck/press?token=secret-token", `{"type":"lock"}`, nil); w.Code != http.StatusForbidden {
		t.Fatalf("read-only press: %d", w.Code)
	}
}

func TestScreenshotEndpoint(t *testing.T) {
	collector := telemetry.NewCollector(500 * time.Millisecond)
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	newSrv := func(opts Options) (*Server, http.Handler, map[string]string) {
		srv := NewServer("test-server", "secret-token", collector, opts)
		srv.capture = func(int, int) ([]byte, error) { return fakeJPEG, nil }
		h := srv.Handler()
		auth := map[string]string{"X-Pairing-Token": "secret-token"}
		if opts.PIN != nil {
			w := do(t, h, "POST", "/api/pair", `{"pin":"`+opts.PIN.Current()+`"}`, auth)
			var pair struct {
				Session string `json:"session"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &pair)
			auth["X-Session"] = pair.Session
		}
		return srv, h, auth
	}

	pin, _ := control.NewPIN(nil)
	_, h, auth := newSrv(Options{PIN: pin})
	if !display.CaptureSupported() {
		t.Skip("capture unsupported on this platform")
	}
	w := do(t, h, "GET", "/api/display/screenshot?index=0&width=640", "", auth)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/jpeg" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("screenshot: %d %v", w.Code, w.Header())
	}
	// Needs a PIN session.
	if w := do(t, h, "GET", "/api/display/screenshot", "", map[string]string{"X-Pairing-Token": "secret-token"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("screenshot without session: %d", w.Code)
	}
	// -no-screen disables it even with a session.
	pin2, _ := control.NewPIN(nil)
	_, h2, auth2 := newSrv(Options{PIN: pin2, NoScreen: true})
	if w := do(t, h2, "GET", "/api/display/screenshot", "", auth2); w.Code != http.StatusForbidden {
		t.Fatalf("screenshot with -no-screen: %d", w.Code)
	}
	if w := do(t, h2, "GET", "/api/health", "", nil); !strings.Contains(w.Body.String(), `"screen_capture":false`) {
		t.Fatalf("health should report screen_capture false: %s", w.Body.String())
	}
	// Read-only servers never expose the screen.
	_, h3, _ := newSrv(Options{ReadOnly: true})
	if w := do(t, h3, "GET", "/api/display/screenshot?token=secret-token", "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("screenshot on read-only: %d", w.Code)
	}
}
