package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"pc_monitor/pkg/audio"
	"pc_monitor/pkg/control"
	"pc_monitor/pkg/power"
	"pc_monitor/pkg/telemetry"
)

// Options configures access control.
type Options struct {
	// ReadOnly disables all control endpoints; dashboards then need only
	// the pairing token (no PIN).
	ReadOnly bool
	// PIN is the one-time pairing PIN manager. Required unless ReadOnly.
	PIN *control.PIN
	// Sessions stores PIN-issued sessions; nil means in-memory only. Pass a
	// persistent store (control.LoadSessions) to survive restarts.
	Sessions *control.Sessions
}

// Server handles HTTP and WebSocket telemetry requests.
type Server struct {
	serverName string
	token      string
	collector  *telemetry.Collector
	opts       Options
	sessions   *control.Sessions
	clientsMu  sync.Mutex
	clients    map[*websocket.Conn]struct{}

	// Overridable for tests.
	doPower func(power.Action) error
}

// NewServer creates a new API server.
func NewServer(serverName, token string, collector *telemetry.Collector, opts Options) *Server {
	sessions := opts.Sessions
	if sessions == nil {
		sessions = control.NewSessions()
	}
	return &Server{
		serverName: serverName,
		token:      token,
		collector:  collector,
		opts:       opts,
		sessions:   sessions,
		clients:    make(map[*websocket.Conn]struct{}),
		doPower:    power.Do,
	}
}

func (s *Server) controlEnabled() bool { return !s.opts.ReadOnly && s.opts.PIN != nil }

// Handler returns the HTTP multiplexer configured with routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/pair", s.requireToken(s.handlePair))
	mux.HandleFunc("/api/telemetry", s.requireAccess(s.handleTelemetry))
	mux.HandleFunc("/ws", s.requireAccess(s.handleWebSocket))

	mux.HandleFunc("/api/audio/volume", s.requireControl(s.handleAudioVolume))
	mux.HandleFunc("/api/audio/mute", s.requireControl(s.handleAudioMute))
	mux.HandleFunc("/api/audio/default", s.requireControl(s.handleAudioDefault))
	mux.HandleFunc("/api/power", s.requireControl(s.handlePower))

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Pairing-Token, X-Session")

		if r.Method == http.MethodOptions {
			// Lets HTTPS-hosted dashboards reach private LAN addresses (Chrome PNA).
			if r.Header.Get("Access-Control-Request-Private-Network") == "true" {
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errorJSON(w http.ResponseWriter, code int, errCode, msg string) {
	writeJSON(w, code, map[string]string{"error": errCode, "message": msg})
}

// requireToken checks the pairing token only.
func (s *Server) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validateToken(r) {
			errorJSON(w, http.StatusUnauthorized, "unauthorized", "invalid pairing token")
			return
		}
		next(w, r)
	}
}

// requireAccess checks the pairing token, plus a PIN-issued session when
// control is enabled.
func (s *Server) requireAccess(next http.HandlerFunc) http.HandlerFunc {
	return s.requireToken(func(w http.ResponseWriter, r *http.Request) {
		if s.controlEnabled() && !s.sessions.Valid(sessionFrom(r)) {
			errorJSON(w, http.StatusUnauthorized, "pin_required", "enter the PIN shown on the server console")
			return
		}
		next(w, r)
	})
}

// requireControl additionally rejects read-only servers and non-POSTs.
func (s *Server) requireControl(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.controlEnabled() {
			if !s.validateToken(r) {
				errorJSON(w, http.StatusUnauthorized, "unauthorized", "invalid pairing token")
				return
			}
			errorJSON(w, http.StatusForbidden, "read_only", "this server is running in read-only mode")
			return
		}
		if r.Method != http.MethodPost {
			errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
			return
		}
		s.requireAccess(next)(w, r)
	}
}

func sessionFrom(r *http.Request) string {
	if v := r.Header.Get("X-Session"); v != "" {
		return v
	}
	return r.URL.Query().Get("session")
}

func (s *Server) validateToken(r *http.Request) bool {
	if s.token == "" {
		return true // No auth configured
	}

	// 1. Query parameter ?token=...
	if r.URL.Query().Get("token") == s.token {
		return true
	}

	// 2. Header X-Pairing-Token
	if r.Header.Get("X-Pairing-Token") == s.token {
		return true
	}

	// 3. Authorization: Bearer <token>
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " && auth[7:] == s.token {
		return true
	}

	return false
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"server_name":     s.serverName,
		"time":            time.Now().UTC().Format(time.RFC3339),
		"control_enabled": s.controlEnabled(),
		"pin_required":    s.controlEnabled(),
	})
}

// handlePair exchanges the one-time PIN for a session token.
func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !s.controlEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{"session": "", "control_enabled": false})
		return
	}
	var body struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", "expected JSON {\"pin\": \"1234\"}")
		return
	}
	if !s.opts.PIN.Verify(body.PIN) {
		time.Sleep(time.Second) // slow down guessing
		errorJSON(w, http.StatusForbidden, "invalid_pin", "incorrect PIN")
		return
	}
	tok, err := s.sessions.Issue()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal", "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": tok, "control_enabled": true})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.collector.GetSnapshot())
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(v); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return false
	}
	return true
}

// audioResult replies with the fresh audio state after a change.
func (s *Server) audioResult(w http.ResponseWriter, err error) {
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "audio_error", err.Error())
		return
	}
	st, err := audio.Get()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "audio_error", err.Error())
		return
	}
	s.collector.SetAudio(st)
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleAudioVolume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID     string `json:"id"`
		Volume int    `json:"volume"`
	}
	if decodeBody(w, r, &body) {
		s.audioResult(w, audio.SetVolume(body.ID, body.Volume))
	}
}

func (s *Server) handleAudioMute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`
		Muted bool   `json:"muted"`
	}
	if decodeBody(w, r, &body) {
		s.audioResult(w, audio.SetMute(body.ID, body.Muted))
	}
}

func (s *Server) handleAudioDefault(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if decodeBody(w, r, &body) {
		s.audioResult(w, audio.SetDefault(body.ID))
	}
}

func (s *Server) handlePower(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string `json:"action"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	a := power.Action(body.Action)
	if a != power.Shutdown && a != power.Restart {
		errorJSON(w, http.StatusBadRequest, "bad_request", "action must be \"shutdown\" or \"restart\"")
		return
	}
	if err := s.doPower(a); err != nil {
		errorJSON(w, http.StatusInternalServerError, "power_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled", "action": string(a)})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "disconnect")

	s.clientsMu.Lock()
	s.clients[conn] = struct{}{}
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	ctx := r.Context()
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()

	// Send initial snapshot immediately
	initialData, _ := json.Marshal(s.collector.GetSnapshot())
	_ = conn.Write(ctx, websocket.MessageText, initialData)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := s.collector.GetSnapshot()
			data, err := json.Marshal(snap)
			if err != nil {
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
