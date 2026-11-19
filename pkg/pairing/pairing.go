package pairing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// Info holds the pairing credentials and connection endpoints.
type Info struct {
	ServerName     string   `json:"server_name"`
	TailcatAddr    string   `json:"tailcat_addr"`
	TailcatPort    int      `json:"tailcat_port"`
	DirectPort     int      `json:"direct_port"`
	LocalIPs       []string `json:"local_ips"`
	PairingToken   string   `json:"pairing_token"`
	WebClientURL   string   `json:"web_client_url"`
	FullPairingURL string   `json:"full_pairing_url"`
}

// GenerateToken creates a cryptographically secure hex token.
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BuildPairingURL constructs the hash fragment URL for the GitHub Pages web client.
func BuildPairingURL(baseWebURL, serverName, tailcatAddr, token string, tailcatPort, directPort int, localIPs []string) string {
	if baseWebURL == "" {
		baseWebURL = "https://garrettcorn.github.io/remote-monitor/"
	}

	directURL := ""
	if len(localIPs) > 0 && directPort > 0 {
		directURL = fmt.Sprintf("http://%s:%d", localIPs[0], directPort)
	}

	params := url.Values{}
	params.Set("name", serverName)
	if tailcatAddr != "" {
		params.Set("tc", tailcatAddr)
		params.Set("tcport", fmt.Sprintf("%d", tailcatPort))
	}
	params.Set("token", token)
	if directPort > 0 {
		params.Set("port", fmt.Sprintf("%d", directPort))
	}
	if directURL != "" {
		params.Set("direct", directURL)
	}

	return fmt.Sprintf("%s#%s", baseWebURL, params.Encode())
}

// ExportPairingHTML creates a standalone HTML page with the QR code and clickable link.
func ExportPairingHTML(destPath string, info Info, svgQR string) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Pair %s - PC Monitor</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; padding: 20px; }
    .card { background: #1e293b; border: 1px solid #334155; border-radius: 16px; padding: 32px; max-width: 440px; text-align: center; box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5); }
    h1 { margin-top: 0; font-size: 1.5rem; color: #38bdf8; }
    .qr-box { background: #ffffff; padding: 16px; border-radius: 12px; display: inline-block; margin: 20px 0; }
    .qr-box svg { display: block; max-width: 240px; height: auto; }
    .code-box { background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 12px; font-family: monospace; font-size: 0.85rem; word-break: break-all; margin: 16px 0; text-align: left; }
    .btn { display: inline-block; background: #0284c7; color: white; text-decoration: none; padding: 12px 24px; border-radius: 8px; font-weight: 600; margin-top: 8px; transition: background 0.2s; }
    .btn:hover { background: #0369a1; }
    .sub { color: #94a3b8; font-size: 0.85rem; margin-top: 12px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Pairing: %s</h1>
    <p style="color: #cbd5e1;">Scan this QR code with your phone camera or click the pairing link below to connect.</p>
    <div class="qr-box">
      %s
    </div>
    <div class="code-box">
      <strong>Tailcat Addr:</strong> %s<br>
      <strong>Token:</strong> %s
    </div>
    <a href="%s" target="_blank" class="btn">Open in Web Dashboard</a>
    <div class="sub">Valid for this server session. Keep this private.</div>
  </div>
</body>
</html>`, info.ServerName, info.ServerName, svgQR, info.TailcatAddr, info.PairingToken, info.FullPairingURL)

	return os.WriteFile(destPath, []byte(html), 0644)
}
