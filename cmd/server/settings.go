package main

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"pc_monitor/pkg/buildinfo"
	"pc_monitor/pkg/config"
	"pc_monitor/pkg/service"
)

//go:embed settings.html
var settingsPage []byte

// settingsHandler serves the local settings page on 127.0.0.1. It refuses
// non-localhost Host headers (DNS rebinding) and requires the key from
// ~/.pc_monitor/settings_key for everything except the page and /api/ping.
func (s *server) settingsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		_, _ = w.Write(settingsPage)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSONResp(w, http.StatusOK, map[string]any{"app": "pc-monitor", "version": buildinfo.Version, "pid": os.Getpid()})
	})
	mux.HandleFunc("/api/state", s.keyed(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		writeJSONResp(w, http.StatusOK, s.state())
	}))
	mux.HandleFunc("/api/config", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var c config.Config
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&c); err != nil {
			writeJSONResp(w, http.StatusBadRequest, map[string]string{"error": "invalid settings"})
			return
		}
		if err := c.Save(); err != nil {
			writeJSONResp(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSONResp(w, http.StatusOK, map[string]string{"status": "restarting"})
		s.restart()
	}))
	mux.HandleFunc("/api/autostart", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body)
		var err error
		if body.Enabled {
			err = service.AutostartEnable()
		} else {
			err = service.AutostartDisable()
		}
		if err != nil {
			writeJSONResp(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSONResp(w, http.StatusOK, map[string]bool{"enabled": service.AutostartEnabled()})
	}))
	mux.HandleFunc("/api/update", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Action string `json:"action"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body)
		if body.Action == "apply" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := s.updates.Apply(ctx); err != nil {
				writeJSONResp(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSONResp(w, http.StatusOK, map[string]string{"status": "restarting"})
			s.restart()
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		writeJSONResp(w, http.StatusOK, s.updates.Check(ctx))
	}))
	// /api/auto-update saves auto_update and applies it without a restart;
	// turning it on checks (and installs) right away.
	mux.HandleFunc("/api/auto-update", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil {
			writeJSONResp(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		c, err := config.Load()
		if err == nil {
			c.AutoUpdate = body.Enabled
			err = c.Save()
		}
		if err != nil {
			writeJSONResp(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.cfg.AutoUpdate = body.Enabled
		s.updates.SetAuto(body.Enabled)
		if body.Enabled && !buildinfo.IsDev() {
			go func() {
				if s.updates.CheckAuto(context.Background()) {
					s.autoInstalled()
				}
			}()
		}
		writeJSONResp(w, http.StatusOK, s.updates.Status())
	}))
	mux.HandleFunc("/api/pairing/reset", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		if err := resetPairingFiles(); err != nil {
			writeJSONResp(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSONResp(w, http.StatusOK, map[string]string{"status": "restarting"})
		s.restart()
	}))
	mux.HandleFunc("/api/restart", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		writeJSONResp(w, http.StatusOK, map[string]string{"status": "restarting"})
		s.restart()
	}))
	mux.HandleFunc("/api/stop", s.keyed(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		writeJSONResp(w, http.StatusOK, map[string]string{"status": "stopping"})
		go func() { time.Sleep(300 * time.Millisecond); s.stop(false) }()
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "[::1]" && host != "::1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *server) keyed(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			writeJSONResp(w, http.StatusMethodNotAllowed, map[string]string{"error": "use " + method})
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Settings-Key")), []byte(s.settingsKey)) != 1 {
			writeJSONResp(w, http.StatusUnauthorized, map[string]string{"error": "open this page with `pc_monitor settings`"})
			return
		}
		next(w, r)
	}
}

// state is everything the settings page and `pc_monitor status/pin` show.
func (s *server) state() map[string]any {
	saved, _ := config.Load()
	pin := ""
	paired := 0
	if s.pin != nil {
		pin = s.pin.Current()
	}
	if s.sessions != nil {
		paired = s.sessions.Count()
	}
	dashboardURL := ""
	if len(s.localIPs) > 0 && s.cfg.Port > 0 {
		dashboardURL = fmt.Sprintf("http://%s:%d", s.localIPs[0], s.cfg.Port)
	}
	return map[string]any{
		"version":   buildinfo.Version,
		"os":        runtimeOS(),
		"pid":       os.Getpid(),
		"config":    saved,
		"running":   s.cfg,
		"autostart": map[string]any{"enabled": service.AutostartEnabled(), "where": service.AutostartDescription()},
		"update":    s.updates.Status(),
		"pairing": map[string]any{
			"url": s.pairingURL, "qr_svg": s.svgQR, "pin": pin, "paired": paired,
			"control": s.pin != nil, "tailcat_addr": s.tailcatAddr, "direct_url": dashboardURL,
		},
		"log": service.LogPath(),
	}
}

func writeJSONResp(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------------------------------------------------------------- client

func settingsBase(port int) string { return fmt.Sprintf("http://127.0.0.1:%d", port) }

func readSettingsKey() string {
	b, err := os.ReadFile(config.Dir() + string(os.PathSeparator) + "settings_key")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// pingSettings reports whether a PC Monitor server answers on the port.
func pingSettings(port int) (bool, string) {
	c := http.Client{Timeout: time.Second}
	resp, err := c.Get(settingsBase(port) + "/api/ping")
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	var p struct {
		App     string `json:"app"`
		Version string `json:"version"`
	}
	if json.NewDecoder(resp.Body).Decode(&p) != nil || p.App != "pc-monitor" {
		return false, ""
	}
	return true, p.Version
}

// settingsCall calls the running server's settings API.
func settingsCall(port int, method, path string, body any, out any) error {
	var rd *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = strings.NewReader(string(b))
	} else {
		rd = strings.NewReader("")
	}
	req, _ := http.NewRequest(method, settingsBase(port)+path, rd)
	req.Header.Set("X-Settings-Key", readSettingsKey())
	req.Header.Set("Content-Type", "application/json")
	c := http.Client{Timeout: 6 * time.Minute}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return fmt.Errorf("%s", e.Error)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
