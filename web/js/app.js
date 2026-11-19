// Main Application Orchestrator for PC Monitor GitHub Pages Client

import { ServerStore } from './storage.js';
import { MonitorClient } from './client.js?v=6';
import { SparklineChart } from './charts.js';
import { UI } from './ui.js?v=6';
import { QRScanner } from './qr-scanner.js';

class App {
  constructor() {
    this.client = new MonitorClient();
    this.activeServer = null;
    this.charts = {};
    this.qrScanner = null;
    this.currentView = 'dashboard'; // 'dashboard' | 'fleet'
    this.pinServer = null;          // server awaiting PIN entry
    this.audioHoldUntil = 0;        // pause audio re-render while the user adjusts it
    this.volumeTimer = null;
  }

  init() {
    this.initCharts();
    this.initEvents();
    this.handleURLHash();
    this.refreshServerList();
    this.startFleetMonitoring();
  }

  initCharts() {
    this.charts = {
      cpu: new SparklineChart('chart-cpu', { color: '#0a84ff', fillColor: 'rgba(10, 132, 255, 0.28)', maxPoints: 60 }),
      mem: new SparklineChart('chart-mem', { color: '#bf5af2', fillColor: 'rgba(191, 90, 242, 0.28)', maxPoints: 60 }),
      netDown: new SparklineChart('chart-net-down', { color: '#30d158', fillColor: 'rgba(48, 209, 88, 0.28)', autoScale: true, maxPoints: 60 }),
      netUp: new SparklineChart('chart-net-up', { color: '#ff9f0a', fillColor: 'rgba(255, 159, 10, 0.28)', autoScale: true, maxPoints: 60 }),
    };
  }

  initEvents() {
    // Server select dropdown
    const select = document.getElementById('server-select');
    if (select) {
      select.addEventListener('change', (e) => {
        if (e.target.value === '__add__') {
          this.openPairModal();
        } else if (e.target.value === '__fleet__') {
          this.switchView('fleet');
        } else {
          this.switchServer(e.target.value);
        }
      });
    }

    // View toggle buttons
    document.getElementById('btn-nav-dashboard')?.addEventListener('click', () => this.switchView('dashboard'));
    document.getElementById('btn-nav-fleet')?.addEventListener('click', () => this.switchView('fleet'));
    document.getElementById('btn-pair-modal')?.addEventListener('click', () => this.openPairModal());

    // Pair modal form
    document.getElementById('pair-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handlePairFormSubmit();
    });

    // Close modals
    document.querySelectorAll('.modal-close-btn').forEach(btn => {
      btn.addEventListener('click', () => this.closeModals());
    });

    // Camera QR scanner toggle
    document.getElementById('btn-start-scanner')?.addEventListener('click', () => this.startCameraScanner());
    document.getElementById('btn-stop-scanner')?.addEventListener('click', () => this.stopCameraScanner());

    // Delete active server
    document.getElementById('btn-delete-server')?.addEventListener('click', () => {
      if (this.activeServer && confirm(`Unpair server "${this.activeServer.name}"?`)) {
        ServerStore.deleteServer(this.activeServer.id);
        this.refreshServerList();
      }
    });

    // Control PIN form
    document.getElementById('pin-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.submitPIN();
    });

    // Power actions
    document.getElementById('btn-restart')?.addEventListener('click', () => this.powerAction('restart'));
    document.getElementById('btn-shutdown')?.addEventListener('click', () => this.powerAction('shutdown'));

    // Audio controls (delegated; the card is re-rendered with live data)
    const audio = document.getElementById('audio-container');
    if (audio) {
      const hold = () => { this.audioHoldUntil = Date.now() + 2500; };
      audio.addEventListener('pointerdown', hold);
      audio.addEventListener('focusin', hold);
      audio.addEventListener('input', (e) => {
        if (!e.target.classList.contains('audio-slider')) return;
        hold();
        const v = e.target.value;
        e.target.style.setProperty('--val', v + '%');
        const label = e.target.parentElement.querySelector('.audio-value');
        if (label) label.textContent = v + '%';
        clearTimeout(this.volumeTimer);
        const id = e.target.dataset.id;
        this.volumeTimer = setTimeout(() => this.audioControl('/api/audio/volume', { id, volume: parseInt(v, 10) }), 150);
      });
      audio.addEventListener('change', (e) => {
        if (e.target.classList.contains('audio-device-select')) {
          hold();
          this.audioControl('/api/audio/default', { id: e.target.value });
        }
      });
      audio.addEventListener('click', (e) => {
        const btn = e.target.closest('.audio-mute-btn');
        if (!btn || btn.disabled) return;
        hold();
        this.audioControl('/api/audio/mute', { id: btn.dataset.id, muted: btn.dataset.muted !== 'true' });
      });
    }
  }

  // ---------------------------------------------------------------- control

  setControlUI(enabled) {
    document.querySelectorAll('.control-only').forEach(el => el.classList.toggle('hidden', !enabled));
  }

  renderAudio(state, force = false) {
    if (!force && Date.now() < this.audioHoldUntil) return;
    UI.renderAudio(state, !!this.activeServer?.controlEnabled);
  }

  async audioControl(path, body) {
    const server = this.activeServer;
    if (!server) return;
    try {
      const state = await MonitorClient.post(server, path, body);
      this.audioHoldUntil = 0;
      this.renderAudio(state, true);
    } catch (e) {
      this.handleControlError(server, e);
    }
  }

  async powerAction(action) {
    const server = this.activeServer;
    if (!server) return;
    const verb = action === 'restart' ? 'Restart' : 'Shut down';
    if (!confirm(`${verb} "${server.name}"?\n\nThe computer will ${action === 'restart' ? 'restart' : 'shut down'} in about 5 seconds. Unsaved work on it may be lost.`)) {
      return;
    }
    try {
      await MonitorClient.post(server, '/api/power', { action });
      this.showToast(`${verb} requested for ${server.name}`);
    } catch (e) {
      this.handleControlError(server, e);
    }
  }

  handleControlError(server, e) {
    if (e.code === 'pin_required') {
      this.requirePIN(server);
    } else if (e.code === 'read_only') {
      server.controlEnabled = false;
      ServerStore.saveServer(server);
      this.setControlUI(false);
      this.showToast('This server is read-only');
    } else {
      this.showToast(`Failed: ${e.message}`);
    }
  }

  // Ask for the one-time PIN shown on the server console.
  requirePIN(server) {
    server.session = '';
    ServerStore.saveServer(server);
    this.client.disconnect();
    this.updateConnectionStatus('offline', 'PIN required');
    this.pinServer = server;
    document.getElementById('pin-server-name').textContent = server.name;
    document.getElementById('pin-error')?.classList.add('hidden');
    const input = document.getElementById('input-pin');
    if (input) input.value = '';
    document.getElementById('pin-modal')?.classList.remove('hidden');
    setTimeout(() => input?.focus(), 50);
  }

  async submitPIN() {
    const server = this.pinServer;
    const input = document.getElementById('input-pin');
    const errEl = document.getElementById('pin-error');
    const btn = document.getElementById('btn-pin-submit');
    const pin = (input?.value || '').trim();
    if (!server || !/^\d{4}$/.test(pin)) {
      errEl.textContent = 'Enter the 4-digit PIN.';
      errEl.classList.remove('hidden');
      return;
    }
    btn.disabled = true;
    try {
      server.session = await MonitorClient.pair(server, pin);
      server.controlEnabled = true;
      ServerStore.saveServer(server);
      this.pinServer = null;
      document.getElementById('pin-modal')?.classList.add('hidden');
      this.showToast(`Connected to ${server.name}`);
      this.connectServer(server);
    } catch (e) {
      errEl.textContent = e.code === 'invalid_pin'
        ? 'Incorrect PIN. After 5 wrong attempts the server shows a new PIN.'
        : `Could not pair: ${e.message}`;
      errEl.classList.remove('hidden');
      input?.select();
    } finally {
      btn.disabled = false;
    }
  }

  handleURLHash() {
    const hash = window.location.hash;
    if (!hash || hash.length < 2) return;

    const parsed = ServerStore.parsePairingParams(hash);
    if (parsed) {
      // Save or update server
      const existing = ServerStore.getServers().find(s =>
        (s.tailcatAddr && s.tailcatAddr === parsed.tailcatAddr) ||
        (s.directURL && s.directURL === parsed.directURL)
      );

      let saved;
      if (existing) {
        saved = ServerStore.saveServer({ ...existing, ...parsed });
      } else {
        saved = ServerStore.saveServer(parsed);
      }

      ServerStore.setActiveServerId(saved.id);
      // Clean URL hash to keep clean view
      history.replaceState(null, '', window.location.pathname);
      this.showToast(`Paired successfully with ${saved.name}!`);
    }
  }

  refreshServerList() {
    const servers = ServerStore.getServers();
    const select = document.getElementById('server-select');
    if (!select) return;

    select.innerHTML = '';

    // Fleet Overview option
    const fleetOpt = document.createElement('option');
    fleetOpt.value = '__fleet__';
    fleetOpt.textContent = `📊 Fleet Overview (${servers.length} servers)`;
    select.appendChild(fleetOpt);

    // Individual servers
    servers.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = `🖥️ ${s.name}`;
      select.appendChild(opt);
    });

    // Add new option
    const addOpt = document.createElement('option');
    addOpt.value = '__add__';
    addOpt.textContent = '➕ Pair New Server...';
    select.appendChild(addOpt);

    // Select active
    const activeId = ServerStore.getActiveServerId();
    if (activeId && ServerStore.getServer(activeId)) {
      select.value = activeId;
      this.switchServer(activeId);
    } else if (servers.length > 0) {
      select.value = servers[0].id;
      this.switchServer(servers[0].id);
    } else {
      this.switchView('fleet');
      select.value = '__fleet__';
    }

    UI.renderFleetOverview(servers, (id) => this.switchServer(id));
  }

  async switchServer(id) {
    const server = ServerStore.getServer(id);
    if (!server) return;

    this.activeServer = server;
    ServerStore.setActiveServerId(id);
    document.getElementById('server-select').value = id;
    this.switchView('dashboard');

    // Reset charts
    Object.values(this.charts).forEach(c => c.clear());
    this.client.disconnect();
    this.updateConnectionStatus('connecting', 'Connecting...');

    // Learn whether this server has remote control (and so needs a PIN).
    const health = await MonitorClient.fetchHealth(server);
    if (this.activeServer !== server) return; // user switched again meanwhile
    if (health) {
      server.controlEnabled = !!health.control_enabled;
      ServerStore.saveServer(server);
    }
    this.setControlUI(!!server.controlEnabled);
    if (health?.pin_required && !server.session) {
      this.requirePIN(server);
      return;
    }
    this.connectServer(server);
  }

  connectServer(server) {
    this.client.connect(
      server,
      (snapshot) => {
        // Cache snapshot (without bulky fields) for the fleet view
        server.lastSnapshot = { host: snapshot.host, cpu: { usage_percent: snapshot.cpu?.usage_percent }, memory: { usage_percent: snapshot.memory?.usage_percent } };
        server.status = 'online';
        ServerStore.saveServer(server);
        UI.renderSnapshot(snapshot, this.charts);
        this.renderAudio(snapshot.audio);
      },
      (status, message) => {
        this.updateConnectionStatus(status, message);
      },
      (srv) => this.requirePIN(srv)
    );
  }

  switchView(viewName) {
    this.currentView = viewName;
    const dashboardView = document.getElementById('view-dashboard');
    const fleetView = document.getElementById('view-fleet');
    const btnDash = document.getElementById('btn-nav-dashboard');
    const btnFleet = document.getElementById('btn-nav-fleet');

    if (viewName === 'dashboard') {
      dashboardView?.classList.remove('hidden');
      fleetView?.classList.add('hidden');
      btnDash?.classList.add('nav-active');
      btnFleet?.classList.remove('nav-active');
    } else {
      dashboardView?.classList.add('hidden');
      fleetView?.classList.remove('hidden');
      btnDash?.classList.remove('nav-active');
      btnFleet?.classList.add('nav-active');
      UI.renderFleetOverview(ServerStore.getServers(), (id) => this.switchServer(id));
    }
  }

  updateConnectionStatus(status, message) {
    const dot = document.getElementById('header-status-dot');
    const text = document.getElementById('header-status-text');
    if (!dot || !text) return;

    dot.className = 'status-dot';
    if (status === 'online') {
      dot.classList.add('dot-online');
      text.textContent = 'Live Connected';
    } else if (status === 'connecting') {
      dot.classList.add('dot-connecting');
      text.textContent = message || 'Connecting...';
    } else {
      dot.classList.add('dot-offline');
      text.textContent = message || 'Offline';
    }
  }

  openPairModal() {
    document.getElementById('pair-modal')?.classList.remove('hidden');
  }

  closeModals() {
    document.querySelectorAll('.modal-overlay').forEach(m => m.classList.add('hidden'));
    this.stopCameraScanner();
  }

  handlePairFormSubmit() {
    const rawInput = document.getElementById('input-pairing-code')?.value.trim();
    const customName = document.getElementById('input-server-name')?.value.trim();

    if (!rawInput) {
      alert('Please enter a pairing link, code, or IP address.');
      return;
    }

    let parsed = ServerStore.parsePairingParams(rawInput);
    if (!parsed) {
      // Treat as direct URL or raw Tailcat address
      if (rawInput.startsWith('http://') || rawInput.startsWith('https://')) {
        parsed = { directURL: rawInput, name: customName || 'Server' };
      } else if (rawInput.startsWith('tc')) {
        parsed = { tailcatAddr: rawInput, name: customName || 'Tailcat Server' };
      } else {
        alert('Invalid pairing format. Paste a full pairing link, tailcat address (tc...), or direct URL.');
        return;
      }
    }

    if (customName) {
      parsed.name = customName;
    }

    const saved = ServerStore.saveServer(parsed);
    this.closeModals();
    this.refreshServerList();
    this.switchServer(saved.id);
    this.showToast(`Paired with ${saved.name}!`);
  }

  startCameraScanner() {
    const video = document.getElementById('qr-video');
    const canvas = document.getElementById('qr-canvas');
    const box = document.getElementById('camera-scanner-box');
    box?.classList.remove('hidden');

    this.qrScanner = new QRScanner(video, canvas, (code) => {
      document.getElementById('input-pairing-code').value = code;
      this.stopCameraScanner();
      this.handlePairFormSubmit();
    });

    this.qrScanner.start().then(ok => {
      if (!ok) {
        alert('Could not access camera for QR scanning. Please paste the code manually.');
        box?.classList.add('hidden');
      }
    });
  }

  stopCameraScanner() {
    if (this.qrScanner) {
      this.qrScanner.stop();
      this.qrScanner = null;
    }
    document.getElementById('camera-scanner-box')?.classList.add('hidden');
  }

  startFleetMonitoring() {
    // Ping all servers every 6 seconds to keep fleet status up-to-date
    setInterval(async () => {
      const servers = ServerStore.getServers();
      for (const s of servers) {
        if (this.activeServer && s.id === this.activeServer.id) continue;
        const res = await MonitorClient.pingServer(s);
        s.status = res.status;
        ServerStore.saveServer(s);
      }
      if (this.currentView === 'fleet') {
        UI.renderFleetOverview(servers, (id) => this.switchServer(id));
      }
    }, 6000);
  }

  showToast(msg) {
    const toast = document.createElement('div');
    toast.className = 'toast-alert';
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(() => toast.classList.add('toast-visible'), 10);
    setTimeout(() => {
      toast.classList.remove('toast-visible');
      setTimeout(() => toast.remove(), 300);
    }, 3000);
  }
}

// Bootstrap application on DOM load
window.addEventListener('DOMContentLoaded', () => {
  window.app = new App();
  window.app.init();
});
