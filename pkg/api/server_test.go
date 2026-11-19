package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pc_monitor/pkg/telemetry"
)

func TestAPIServer(t *testing.T) {
	collector := telemetry.NewCollector(500 * time.Millisecond)
	srv := NewServer("test-server", "secret-token", collector)
	handler := srv.Handler()

	// 1. Health endpoint (no auth needed)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Telemetry endpoint without auth -> 401
	req = httptest.NewRequest("GET", "/api/telemetry", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}

	// 3. Telemetry endpoint with auth -> 200
	req = httptest.NewRequest("GET", "/api/telemetry?token=secret-token", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var snap telemetry.SystemSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatalf("failed to parse snapshot JSON: %v", err)
	}

	if snap.Host.Hostname == "" {
		t.Errorf("expected hostname, got empty")
	}
	t.Logf("Snapshot retrieved successfully! Host: %s, OS: %s, CPU: %s (%.1f%%), RAM Total: %d MB",
		snap.Host.Hostname, snap.Host.OS, snap.CPU.ModelName, snap.CPU.UsagePercent, snap.Memory.TotalBytes/(1024*1024))
}
