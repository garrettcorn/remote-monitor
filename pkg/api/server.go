package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"pc_monitor/pkg/telemetry"
)

// Server handles HTTP and WebSocket telemetry requests.
type Server struct {
	serverName string
	token      string
	collector  *telemetry.Collector
	clientsMu  sync.Mutex
	clients    map[*websocket.Conn]struct{}
}

// NewServer creates a new API server.
func NewServer(serverName, token string, collector *telemetry.Collector) *Server {
	return &Server{
		serverName: serverName,
		token:      token,
		collector:  collector,
		clients:    make(map[*websocket.Conn]struct{}),
	}
}

// Handler returns the HTTP multiplexer configured with routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/telemetry", s.requireAuth(s.handleTelemetry))
	mux.HandleFunc("/ws", s.handleWebSocket)

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Pairing-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validateToken(r) {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"server_name": s.serverName,
		"time":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	snap := s.collector.GetSnapshot()
	_ = json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.validateToken(r) {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

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
