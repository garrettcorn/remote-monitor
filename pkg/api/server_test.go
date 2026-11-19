package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pc_monitor/pkg/control"
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
