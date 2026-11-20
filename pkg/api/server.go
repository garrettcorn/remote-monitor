package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	"pc_monitor/pkg/audio"
	"pc_monitor/pkg/control"
	"pc_monitor/pkg/deck"
	"pc_monitor/pkg/display"
	"pc_monitor/pkg/power"
	"pc_monitor/pkg/telemetry"
	"pc_monitor/pkg/update"
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
	// ActionsFile is the allowlist of apps Control Panel buttons may launch
	// (re-read on each use). Empty disables launch actions.
	ActionsFile string
	// NoScreen disables screen capture previews even when control is enabled.
	NoScreen bool
	// Version is reported in /api/health.
	Version string
	// Updates checks for and installs new releases (nil disables /api/update).
	Updates Updater
	// Restart relaunches the server (after an update).
	Restart func()
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
	doPower   func(power.Action) error
	doDeck    func(deck.Action, string) error
	capture   func(index, maxWidth int) ([]byte, error)
	doPointer func(index int, e display.PointerEvent) error
	listApps  func() ([]deck.App, error)
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
		doDeck:     deck.Press,
		capture:    display.Capture,
		doPointer:  display.Pointer,
		listApps:   deck.ListApps,
	}
}

func (s *Server) controlEnabled() bool { return !s.opts.ReadOnly && s.opts.PIN != nil }

// screenCaptureEnabled: screen images are sensitive, so they need a
// PIN-paired (control) session and can be switched off with -no-screen.
func (s *Server) screenCaptureEnabled() bool {
	return s.controlEnabled() && !s.opts.NoScreen && display.CaptureSupported()
}

// remoteInputEnabled: driving the mouse and keyboard from the screen viewer
// needs everything screen capture needs, plus platform input support.
func (s *Server) remoteInputEnabled() bool {
	return s.screenCaptureEnabled() && display.InputSupported()
}

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
	mux.HandleFunc("/api/deck/actions", s.requireControlMethod(http.MethodGet, s.handleDeckActions))
	mux.HandleFunc("/api/deck/apps", s.requireControlMethod(http.MethodGet, s.handleDeckApps))
	mux.HandleFunc("/api/deck/press", s.requireControl(s.handleDeckPress))
	mux.HandleFunc("/api/display/screenshot", s.requireControlMethod(http.MethodGet, s.handleScreenshot))
	mux.HandleFunc("/api/display/input", s.requireControl(s.handleDisplayInput))
	mux.HandleFunc("/api/update", s.requireControl(s.handleUpdate))

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
	return s.requireControlMethod(http.MethodPost, next)
}

// requireControlMethod is requireControl for a specific HTTP method.
func (s *Server) requireControlMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.controlEnabled() {
			if !s.validateToken(r) {
				errorJSON(w, http.StatusUnauthorized, "unauthorized", "invalid pairing token")
				return
			}
			errorJSON(w, http.StatusForbidden, "read_only", "this server is running in read-only mode")
			return
		}
		if r.Method != method {
			errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "use "+method)
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
		"screen_capture":  s.screenCaptureEnabled(),
		"remote_input":    s.remoteInputEnabled(),
		"version":         s.opts.Version,
		"update":          s.updateStatus(),
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
	if a != power.Shutdown && a != power.Restart && a != power.Sleep {
		errorJSON(w, http.StatusBadRequest, "bad_request", "action must be \"shutdown\", \"restart\" or \"sleep\"")
		return
	}
	if err := s.doPower(a); err != nil {
		errorJSON(w, http.StatusInternalServerError, "power_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled", "action": string(a)})
}

// handleDeckApps lists apps installed on the host (id + name) for the
// button editor. Launching re-validates the id server-side.
func (s *Server) handleDeckApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.listApps()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "apps_error", err.Error())
		return
	}
	if apps == nil {
		apps = []deck.App{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
}

// handleDeckActions lists the launch actions configured on the server
// (names only; paths and arguments stay on the server).
func (s *Server) handleDeckActions(w http.ResponseWriter, r *http.Request) {
	actions, err := deck.LoadLaunchActions(s.opts.ActionsFile)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "actions_error", err.Error())
		return
	}
	launch := make([]deck.LaunchInfo, 0, len(actions))
	for _, a := range actions {
		launch = append(launch, deck.LaunchInfo{ID: a.ID, Name: a.Name})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"launch":       launch,
		"actions_file": s.opts.ActionsFile,
		"media_keys":   deck.MediaKeys,
	})
}

func (s *Server) handleDeckPress(w http.ResponseWriter, r *http.Request) {
	var a deck.Action
	if !decodeBody(w, r, &a) {
		return
	}
	if err := s.doDeck(a, s.opts.ActionsFile); err != nil {
		errorJSON(w, http.StatusBadRequest, "deck_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Updater is implemented by update.Checker.
type Updater interface {
	Status() update.Status
	Check(ctx context.Context) update.Status
	Apply(ctx context.Context) error
}

func (s *Server) updateStatus() any {
	if s.opts.Updates == nil {
		return nil
	}
	return s.opts.Updates.Status()
}

// handleUpdate checks for or installs a new release (PIN-paired dashboards).
func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if s.opts.Updates == nil {
		errorJSON(w, http.StatusNotImplemented, "update_unavailable", "updates are not available on this server")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	switch body.Action {
	case "check":
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, s.opts.Updates.Check(ctx))
	case "apply":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.opts.Updates.Apply(ctx); err != nil {
			errorJSON(w, http.StatusConflict, "update_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "restarting"})
		if s.opts.Restart != nil {
			s.opts.Restart()
		}
	default:
		errorJSON(w, http.StatusBadRequest, "bad_request", "action must be \"check\" or \"apply\"")
	}
}

// handleScreenshot returns a JPEG of one display (?index=N, ?width=max px).
func (s *Server) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	if !s.screenCaptureEnabled() {
		errorJSON(w, http.StatusForbidden, "screen_disabled", "screen capture is disabled on this server")
		return
	}
	index, err := strconv.Atoi(r.URL.Query().Get("index"))
	if err != nil || index < 0 {
		index = 0
	}
	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil || width <= 0 {
		width = 1280
	}
	img, err := s.capture(index, width)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "capture_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(img)
}

// displayInput is one remote-control event from the screen viewer: a
// pointer event on a display, a key press ("ctrl+c", "enter") or text.
type displayInput struct {
	display.PointerEvent
	Index int    `json:"index"`
	Keys  string `json:"keys,omitempty"` // type "key"
	Text  string `json:"text,omitempty"` // type "text"
}

// handleDisplayInput drives the host's mouse and keyboard.
func (s *Server) handleDisplayInput(w http.ResponseWriter, r *http.Request) {
	if !s.remoteInputEnabled() {
		errorJSON(w, http.StatusForbidden, "input_disabled", "remote input is disabled on this server")
		return
	}
	var in displayInput
	if !decodeBody(w, r, &in) {
		return
	}
	var err error
	switch in.Type {
	case "key":
		err = s.doDeck(deck.Action{Type: "hotkey", Keys: in.Keys}, "")
	case "text":
		err = s.doDeck(deck.Action{Type: "text", Text: in.Text}, "")
	default:
		if err = in.PointerEvent.Validate(); err == nil {
			err = s.doPointer(in.Index, in.PointerEvent)
		}
	}
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "input_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
