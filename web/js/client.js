// Network client managing connections via WebSocket, HTTP polling, and Tailcat WASM.

export class MonitorClient {
  constructor() {
    this.activeServer = null;
    this.ws = null;
    this.pollTimer = null;
    this.reconnectTimer = null;
    this.isConnecting = false;
    this.tailcatConn = null;
    this.onSnapshotCallback = null;
    this.onStatusCallback = null;
  }

  connect(server, onSnapshot, onStatus) {
    this.disconnect();
    this.activeServer = server;
    this.onSnapshotCallback = onSnapshot;
    this.onStatusCallback = onStatus;

    this.setStatus('connecting');

    // Strategy 1: If directURL is provided, connect directly via WebSocket or HTTP
    if (server.directURL) {
      this.connectDirect(server);
      return;
    }

    // Strategy 2: If only tailcatAddr is available, use Tailcat WASM tunnel
    if (server.tailcatAddr) {
      this.connectTailcat(server);
      return;
    }

    this.setStatus('offline', 'No valid connection endpoint (direct URL or Tailcat address) provided.');
  }

  disconnect() {
    clearTimeout(this.reconnectTimer);
    clearInterval(this.pollTimer);
    if (this.ws) {
      try {
        this.ws.close();
      } catch (e) {}
      this.ws = null;
    }
    if (this.tailcatConn) {
      try {
        this.tailcatConn.close();
      } catch (e) {}
      this.tailcatConn = null;
    }
    this.isConnecting = false;
  }

  setStatus(status, message = '') {
    if (this.onStatusCallback) {
      this.onStatusCallback(status, message);
    }
  }

  connectDirect(server) {
    let wsUrl = '';
    const base = server.directURL.replace(/\/+$/, '');
    if (base.startsWith('https://')) {
      wsUrl = base.replace('https://', 'wss://') + '/ws?token=' + encodeURIComponent(server.token || '');
    } else if (base.startsWith('http://')) {
      wsUrl = base.replace('http://', 'ws://') + '/ws?token=' + encodeURIComponent(server.token || '');
    } else {
      wsUrl = 'ws://' + base + '/ws?token=' + encodeURIComponent(server.token || '');
    }

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.setStatus('online');
      };

      this.ws.onmessage = (event) => {
        let snapshot;
        try {
          snapshot = JSON.parse(event.data);
        } catch (e) {
          console.warn('Failed to parse snapshot JSON:', e);
          return;
        }
        try {
          if (this.onSnapshotCallback) {
            this.onSnapshotCallback(snapshot);
          }
        } catch (e) {
          console.error('Failed to render snapshot:', e);
        }
      };

      this.ws.onerror = (err) => {
        console.warn('WebSocket error, falling back to HTTP polling...', err);
        this.startHttpPolling(server);
      };

      this.ws.onclose = () => {
        if (this.activeServer && this.activeServer.id === server.id) {
          this.setStatus('connecting', 'Reconnecting...');
          this.scheduleReconnect(server);
        }
      };
    } catch (e) {
      console.warn('Failed to construct WebSocket, trying HTTP polling:', e);
      this.startHttpPolling(server);
    }
  }

  startHttpPolling(server) {
    if (this.pollTimer) clearInterval(this.pollTimer);

    const poll = async () => {
      try {
        const url = `${server.directURL.replace(/\/+$/, '')}/api/telemetry?token=${encodeURIComponent(server.token || '')}`;
        const resp = await fetch(url, { headers: { 'Accept': 'application/json' } });
        if (!resp.ok) {
          throw new Error(`HTTP ${resp.status}`);
        }
        const data = await resp.json();
        this.setStatus('online');
        if (this.onSnapshotCallback) {
          this.onSnapshotCallback(data);
        }
      } catch (e) {
        this.setStatus('offline', e.message);
      }
    };

    poll();
    this.pollTimer = setInterval(poll, 1500);
  }

  async connectTailcat(server) {
    if (typeof window.tailcatDial !== 'function') {
      this.setStatus('connecting', 'Loading Tailcat WebAssembly engine...');
      // Wait or retry if WASM is still initializing
      let attempts = 0;
      const checkWasm = setInterval(async () => {
        attempts++;
        if (typeof window.tailcatDial === 'function') {
          clearInterval(checkWasm);
          this.dialTailcatStream(server);
        } else if (attempts > 10) {
          clearInterval(checkWasm);
          this.setStatus('offline', 'Tailcat WASM runtime not available in this browser session.');
        }
      }, 500);
      return;
    }

    this.dialTailcatStream(server);
  }

  async dialTailcatStream(server) {
    try {
      this.setStatus('connecting', 'Dialing Tailcat peer ' + server.tailcatAddr.substring(0, 12) + '…');
      const conn = await window.tailcatDial({
        addr: server.tailcatAddr,
        port: server.tailcatPort || 8080,
      });

      this.tailcatConn = conn;
      this.setStatus('online');

      // Loop reading HTTP or telemetry frames from tunneled stream
      this.readTailcatLoop(conn, server);
    } catch (e) {
      console.error('Tailcat dial error:', e);
      this.setStatus('offline', 'Tailcat connection failed: ' + e.message);
      this.scheduleReconnect(server);
    }
  }

  async readTailcatLoop(conn, server) {
    const textDecoder = new TextDecoder();
    // Send initial HTTP GET request over the Tailcat TCP stream
    const req = `GET /api/telemetry?token=${encodeURIComponent(server.token || '')} HTTP/1.1\r\nHost: tailcat\r\nConnection: keep-alive\r\n\r\n`;
    await conn.write(new TextEncoder().encode(req));

    while (this.tailcatConn === conn) {
      try {
        const chunk = await conn.read();
        if (!chunk) break; // EOF
        const text = textDecoder.decode(chunk);
        // Extract JSON body
        const jsonStart = text.indexOf('{');
        const jsonEnd = text.lastIndexOf('}');
        if (jsonStart >= 0 && jsonEnd > jsonStart) {
          const jsonStr = text.substring(jsonStart, jsonEnd + 1);
          const snap = JSON.parse(jsonStr);
          if (this.onSnapshotCallback) {
            this.onSnapshotCallback(snap);
          }
        }
      } catch (e) {
        break;
      }
    }
  }

  scheduleReconnect(server) {
    clearTimeout(this.reconnectTimer);
    this.reconnectTimer = setTimeout(() => {
      if (this.activeServer && this.activeServer.id === server.id) {
        this.connect(server, this.onSnapshotCallback, this.onStatusCallback);
      }
    }, 3000);
  }

  // Probe single server status for multi-server fleet overview
  static async pingServer(server) {
    if (!server.directURL) {
      return { status: server.status || 'unknown' };
    }
    try {
      const url = `${server.directURL.replace(/\/+$/, '')}/api/health?token=${encodeURIComponent(server.token || '')}`;
      const ctrl = new AbortController();
      const tid = setTimeout(() => ctrl.abort(), 2000);
      const resp = await fetch(url, { signal: ctrl.signal });
      clearTimeout(tid);
      if (resp.ok) {
        return { status: 'online' };
      }
      return { status: 'offline' };
    } catch (e) {
      return { status: 'offline' };
    }
  }
}
