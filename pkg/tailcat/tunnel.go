package tailcat

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/tailscale/tailcat"
	"tailscale.com/types/key"
	"tailscale.com/types/logger"
)

// Config holds settings for starting the Tailcat tunnel.
type Config struct {
	Port         int
	KeyFilePath  string
	EphemeralKey bool
	Verbose      bool
}

// Tunnel manages the Tailcat listener and address.
type Tunnel struct {
	server   *tailcat.Server
	listener net.Listener
	addr     string
	port     int
	mu       sync.Mutex
}

type storedKey struct {
	NodeKey      string `json:"node_key"`
	PresharedKey string `json:"preshared_key"`
}

// StartTunnel initializes the Tailcat Server, generates or loads keys,
// and returns a net.Listener accepting incoming peer connections.
func StartTunnel(ctx context.Context, cfg Config) (*Tunnel, error) {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	srv := &tailcat.Server{
		Logf: logger.Discard,
	}
	if cfg.Verbose {
		srv.Logf = func(format string, args ...any) {
			fmt.Printf("[tailcat] "+format+"\n", args...)
		}
	}

	// Key persistence (if not ephemeral)
	if !cfg.EphemeralKey && cfg.KeyFilePath != "" {
		if nodeKey, psk, err := loadKey(cfg.KeyFilePath); err == nil {
			srv.Key = nodeKey
			srv.PresharedKey = psk
		}
	}

	listenAddr := fmt.Sprintf(":%d", cfg.Port)
	ln, err := srv.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to start tailcat listener on %s: %w", listenAddr, err)
	}

	tcAddr := string(srv.TailcatAddr())

	// Save key if persistent
	if !cfg.EphemeralKey && cfg.KeyFilePath != "" && srv.Key.IsZero() == false {
		_ = saveKey(cfg.KeyFilePath, srv.Key, srv.PresharedKey)
	}

	return &Tunnel{
		server:   srv,
		listener: ln,
		addr:     tcAddr,
		port:     cfg.Port,
	}, nil
}

// Listener returns the active Tailcat net.Listener.
func (t *Tunnel) Listener() net.Listener {
	return t.listener
}

// Addr returns the Tailcat pairing address string (tc...).
func (t *Tunnel) Addr() string {
	return t.addr
}

// Port returns the listening TCP port on the Tailcat node.
func (t *Tunnel) Port() int {
	return t.port
}

// Close gracefully closes the Tailcat server and listener.
func (t *Tunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		_ = t.listener.Close()
	}
	if t.server != nil {
		return t.server.Close()
	}
	return nil
}

func loadKey(path string) (key.NodePrivate, tailcat.PresharedKey, error) {
	var k storedKey
	var np key.NodePrivate
	var psk tailcat.PresharedKey

	data, err := os.ReadFile(path)
	if err != nil {
		return np, psk, err
	}
	if err := json.Unmarshal(data, &k); err != nil {
		return np, psk, err
	}
	if err := np.UnmarshalText([]byte(k.NodeKey)); err != nil {
		return np, psk, err
	}
	copy(psk[:], []byte(k.PresharedKey))
	return np, psk, nil
}

func saveKey(path string, np key.NodePrivate, psk tailcat.PresharedKey) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	keyText, err := np.MarshalText()
	if err != nil {
		return err
	}
	k := storedKey{
		NodeKey:      string(keyText),
		PresharedKey: string(psk[:]),
	}
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
