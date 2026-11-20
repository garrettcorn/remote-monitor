// UI Renderer for dashboard widgets, gauges, drives, and process tables.

const ICONS = {
  drive: '<svg viewBox="0 0 20 20" aria-hidden="true"><rect x="2.5" y="5" width="15" height="10" rx="2.5" fill="none" stroke="currentColor" stroke-width="1.6"/><circle cx="14" cy="10" r="1.2" fill="currentColor"/><path d="M5.5 10h5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>',
  thermo: '<svg viewBox="0 0 20 20" aria-hidden="true"><path d="M8 4a2 2 0 1 1 4 0v7.3a3.5 3.5 0 1 1-4 0z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/><circle cx="10" cy="14" r="1.6" fill="currentColor"/><path d="M10 13V7" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>',
};

// Escape text from the server before inserting it into HTML.
const esc = (v) => String(v ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

const SPEAKER_ICON = '<svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 8h3l4-3.5v11L6 12H3z" fill="currentColor"/><path d="M13 7.5a3.5 3.5 0 0 1 0 5M15 5a7 7 0 0 1 0 10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>';
const MUTED_ICON = '<svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 8h3l4-3.5v11L6 12H3z" fill="currentColor"/><path d="M13 8l4 4m0-4-4 4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>';
const MIC_ICON = '<svg viewBox="0 0 20 20" aria-hidden="true"><rect x="7.5" y="3" width="5" height="9" rx="2.5" fill="currentColor"/><path d="M5 10a5 5 0 0 0 10 0M10 15v2.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>';
const MIC_MUTED_ICON = '<svg viewBox="0 0 20 20" aria-hidden="true"><rect x="7.5" y="3" width="5" height="9" rx="2.5" fill="currentColor"/><path d="M5 10a5 5 0 0 0 10 0M10 15v2.5M4 3l12 14" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>';

export const UI = {
  esc,

  formatBytes(bytes, decimals = 1) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  },

  formatBps(bytesPerSec) {
    return this.formatBytes(bytesPerSec) + '/s';
  },

  formatUptime(seconds) {
    if (!seconds) return '0m';
    const d = Math.floor(seconds / (3600 * 24));
    const h = Math.floor((seconds % (3600 * 24)) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const parts = [];
    if (d > 0) parts.push(`${d}d`);
    if (h > 0) parts.push(`${h}h`);
    parts.push(`${m}m`);
    return parts.join(' ');
  },

  updateGauge(elementId, value, max = 100) {
    const el = document.getElementById(elementId);
    if (!el) return;
    const pct = Math.min(100, Math.max(0, (value / max) * 100));
    // SVG circle circumference for r=36 is 2 * PI * 36 = 226.2
    const circumference = 226.2;
    const offset = circumference - (pct / 100) * circumference;
    el.style.strokeDashoffset = offset;
  },

  renderSnapshot(snap, charts) {
    if (!snap) return;

    // 1. Host Info
    this.setText('host-name', snap.host.hostname || 'Unknown');
    // os_version is the full name ("Windows 11 Pro (25H2, Build 26200)");
    // the short os id ("windows") is only a fallback.
    this.setText('host-os', snap.host.os_version || snap.host.os || '');
    this.setText('host-uptime', this.formatUptime(snap.host.uptime_sec));
    this.setText('host-kernel', snap.host.kernel || snap.host.architecture);
    this.setText('host-procs', snap.host.process_count || snap.top_processes?.length || 0);
    const users = (snap.users || []).filter(u => u.state === 'Active').map(u => u.user);
    this.setText('host-users', users.length ? [...new Set(users)].join(', ') : 'Nobody');
    const batItem = document.getElementById('host-battery-item');
    if (snap.battery) {
      batItem?.classList.remove('hidden');
      const b = snap.battery;
      let txt = `${b.percent}% · ${b.charging ? 'Charging' : b.plugged_in ? 'Plugged in' : 'On battery'}`;
      if (!b.plugged_in && b.seconds_left > 0) {
        txt += ` · ${Math.floor(b.seconds_left / 3600)}h ${Math.floor((b.seconds_left % 3600) / 60)}m left`;
      }
      this.setText('host-battery', txt);
    } else {
      batItem?.classList.add('hidden');
    }

    // 2. CPU
    const cpuPct = snap.cpu.usage_percent || 0;
    this.setText('cpu-pct', `${cpuPct.toFixed(1)}%`);
    this.setText('cpu-model', snap.cpu.model_name || 'CPU');
    this.setText('cpu-cores', `${snap.cpu.logical_cores} Cores`);
    if (snap.cpu.frequency_mhz > 0) {
      const ghz = (snap.cpu.frequency_mhz / 1000).toFixed(2);
      this.setText('cpu-freq', `${ghz} GHz`);
    }
    this.updateGauge('cpu-gauge-circle', cpuPct);
    if (charts?.cpu) charts.cpu.push(cpuPct);

    // Per-core bars
    this.renderCoreBars(snap.cpu.core_usages);

    // 3. Memory
    const memPct = snap.memory.usage_percent || 0;
    this.setText('mem-pct', `${memPct.toFixed(1)}%`);
    this.setText('mem-used', this.formatBytes(snap.memory.used_bytes));
    this.setText('mem-total', this.formatBytes(snap.memory.total_bytes));
    this.setText('mem-avail', this.formatBytes(snap.memory.available_bytes));
    this.updateGauge('mem-gauge-circle', memPct);
    if (charts?.mem) charts.mem.push(memPct);

    // Swap
    if (snap.memory.swap_total_bytes > 0) {
      this.setText('swap-used', this.formatBytes(snap.memory.swap_used_bytes));
      this.setText('swap-total', this.formatBytes(snap.memory.swap_total_bytes));
      this.setText('swap-pct', `${snap.memory.swap_usage_percent.toFixed(0)}%`);
    } else {
      this.setText('swap-used', '0 B');
      this.setText('swap-total', 'Disabled');
      this.setText('swap-pct', '0%');
    }

    // 4. Network
    const downSpeed = snap.network.download_speed_bps || 0;
    const upSpeed = snap.network.upload_speed_bps || 0;
    this.setText('net-down', this.formatBps(downSpeed));
    this.setText('net-up', this.formatBps(upSpeed));
    this.setText('net-total-recv', this.formatBytes(snap.network.total_bytes_recv));
    this.setText('net-total-sent', this.formatBytes(snap.network.total_bytes_sent));
    if (charts?.netDown) charts.netDown.push(downSpeed / (1024 * 1024)); // MB/s
    if (charts?.netUp) charts.netUp.push(upSpeed / (1024 * 1024));

    // 5. GPUs
    this.renderGPUs(snap.gpus);

    // 6. Temperatures
    this.renderTemperatures(snap.temperatures);

    // 7. Drives
    this.renderDrives(snap.drives);

    // 8. Network Interfaces
    this.renderInterfaces(snap.network.interfaces);


    // 10. Drive health / SMART and connected devices
    this.renderDiskHealth(snap.physical_disks, snap.smart_details_available, snap.server_elevated);
    this.renderDevices(snap.devices);
  },

  // elevated: whether the server runs as admin/root (undefined on servers
  // older than v1.3.1, where we fall back to "no drive returned details").
  renderDiskHealth(disks = [], smartDetails = false, elevated) {
    const container = document.getElementById('disk-health-container');
    if (!container) return;
    if (!disks || disks.length === 0) {
      container.innerHTML = '<div class="sub-text">Drive health is not available on this host yet.</div>';
      return;
    }
    const healthClass = (h) => ({ Healthy: 'temp-normal', Warning: 'temp-high', Unhealthy: 'temp-crit' }[h] || 'badge');
    let html = '';
    disks.forEach((d) => {
      const facts = [];
      if (d.temperature_c != null) facts.push(`<span><small>Temp</small>${d.temperature_c.toFixed(0)}°C</span>`);
      if (d.wear_percent != null) facts.push(`<span><small>Wear</small>${d.wear_percent.toFixed(0)}%</span>`);
      if (d.power_on_hours != null) facts.push(`<span><small>Power-on</small>${d.power_on_hours.toLocaleString()} h</span>`);
      if (d.read_errors != null) facts.push(`<span><small>Read errors</small>${d.read_errors}</span>`);
      if (d.write_errors != null) facts.push(`<span><small>Write errors</small>${d.write_errors}</span>`);
      html += `
        <div class="disk-health-item">
          <div class="drive-icon">${ICONS.drive}</div>
          <div class="drive-title-box">
            <div class="drive-name">${esc(d.name)}</div>
            <div class="sub-text">${esc(d.media_type === 'Unspecified' ? 'Drive' : d.media_type)} · ${esc(d.bus_type)} · ${this.formatBytes(d.size_bytes)}</div>
            ${facts.length ? `<div class="smart-facts">${facts.join('')}</div>` : ''}
          </div>
          <div class="temp-badge ${healthClass(d.health)}">${esc(d.health)}</div>
        </div>`;
    });
    const needsElevation = elevated === undefined ? !smartDetails : !elevated;
    if (needsElevation) {
      html += '<div class="sub-text smart-note">Temperature, wear and error counts need the server to run as administrator (or root with smartctl on Linux).</div>';
    }
    container.innerHTML = html;
  },

  renderDevices(devices = []) {
    const container = document.getElementById('devices-container');
    if (!container) return;
    if (!devices || devices.length === 0) {
      container.innerHTML = '<div class="sub-text">No external devices detected (or not supported on this host).</div>';
      return;
    }
    const order = { USB: 0, Bluetooth: 1, Display: 2 };
    const sorted = [...devices].sort((a, b) => (order[a.connection] ?? 9) - (order[b.connection] ?? 9) || a.name.localeCompare(b.name));
    container.innerHTML = sorted.map((d) => `
      <div class="device-item">
        <span class="status-dot ${d.status === 'OK' ? 'dot-online' : 'dot-offline'}" title="${esc(d.status)}"></span>
        <div class="device-text">
          <div class="device-name" title="${esc(d.name)}">${esc(d.name)}</div>
          <div class="sub-text">${esc(d.class)}</div>
        </div>
        <span class="conn-pill">${esc(d.connection)}</span>
      </div>`).join('');
  },

  // Audio card. controlEnabled=false renders it read-only.
  renderAudio(state, controlEnabled) {
    const container = document.getElementById('audio-container');
    if (!container) return;
    document.getElementById('audio-mode-badge')?.classList.toggle('hidden', controlEnabled || !state);
    if (!state || (!state.outputs?.length && !state.inputs?.length)) {
      container.innerHTML = '<div class="sub-text">Audio devices are not available on this host.</div>';
      return;
    }
    const section = (flow, label, devices, icons) => {
      if (!devices?.length) return '';
      const def = devices.find(d => d.default) || devices[0];
      const disabled = controlEnabled ? '' : 'disabled';
      const switchDisabled = controlEnabled && state.can_switch ? '' : 'disabled';
      const options = devices.map(d => `<option value="${esc(d.id)}" ${d.id === def.id ? 'selected' : ''}>${esc(d.name)}</option>`).join('');
      return `
        <div class="audio-section" data-flow="${flow}">
          <div class="audio-label">${label}</div>
          <div class="server-selector-box audio-select-box">
            <select class="server-select audio-device-select" data-flow="${flow}" ${switchDisabled} aria-label="${label} device">${options}</select>
          </div>
          <div class="audio-row">
            <button class="audio-mute-btn ${def.muted ? 'is-muted' : ''}" data-id="${esc(def.id)}" data-muted="${def.muted}" ${disabled}
                    aria-label="${def.muted ? 'Unmute' : 'Mute'}" title="${def.muted ? 'Unmute' : 'Mute'}">${def.muted ? icons[1] : icons[0]}</button>
            <input type="range" class="audio-slider" min="0" max="100" step="1" value="${def.volume}" data-id="${esc(def.id)}" ${disabled}
                   style="--val: ${def.volume}%" aria-label="${label} volume">
            <span class="audio-value">${def.volume}%</span>
          </div>
        </div>`;
    };
    container.innerHTML =
      section('output', 'Output', state.outputs, [SPEAKER_ICON, MUTED_ICON]) +
      section('input', 'Input', state.inputs, [MIC_ICON, MIC_MUTED_ICON]);
  },

  renderCoreBars(usages = []) {
    const container = document.getElementById('core-bars-container');
    if (!container) return;

    if (!usages || usages.length === 0) {
      container.innerHTML = '<div class="sub-text">Per-core telemetry idle</div>';
      return;
    }

    let html = '';
    usages.forEach((usage, idx) => {
      const pct = Math.min(100, Math.max(0, usage));
      html += `
        <div class="core-bar-item">
          <div class="core-label"><span>Core ${idx}</span><span>${pct.toFixed(0)}%</span></div>
          <div class="progress-track">
            <div class="progress-fill core-fill" style="width: ${pct}%"></div>
          </div>
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderGPUs(gpus = []) {
    const container = document.getElementById('gpu-cards-container');
    if (!container) return;

    if (!gpus || gpus.length === 0) {
      container.innerHTML = `
        <div class="empty-state">No dedicated GPU metrics detected.</div>`;
      return;
    }

    let html = '';
    gpus.forEach((gpu) => {
      const util = gpu.utilization_percent || 0;
      const vramPct = gpu.vram_percent || 0;
      const temp = gpu.temperature_c ? `${gpu.temperature_c.toFixed(0)}°C` : 'N/A';
      let pwr = gpu.power_watts ? `${gpu.power_watts.toFixed(1)} W` : '';
      if (pwr && gpu.power_limit_watts) pwr += ` / ${gpu.power_limit_watts.toFixed(0)} W`;
      const extras = [];
      if (gpu.fan_speed_percent != null) extras.push(`<span><small>Fan</small>${gpu.fan_speed_percent.toFixed(0)}%</span>`);
      if (gpu.core_clock_mhz != null) extras.push(`<span><small>Core clock</small>${gpu.core_clock_mhz.toFixed(0)} MHz</span>`);
      if (gpu.mem_clock_mhz != null) extras.push(`<span><small>Mem clock</small>${gpu.mem_clock_mhz.toFixed(0)} MHz</span>`);

      html += `
        <div class="metric-card gpu-card">
          <div class="card-header">
            <div>
              <div class="card-title">${esc(gpu.name || 'GPU ' + gpu.index)}</div>
              <div class="sub-text">${esc(gpu.vendor || 'Graphics Device')} ${pwr ? '• ' + pwr : ''}</div>
            </div>
            <div class="badge badge-temp">${temp}</div>
          </div>
          <div class="gpu-metric-row">
            <div class="metric-stat">
              <span class="stat-label">Core Load</span>
              <span class="stat-val">${util.toFixed(1)}%</span>
              <div class="progress-track"><div class="progress-fill" style="width: ${util}%"></div></div>
            </div>
            <div class="metric-stat">
              <span class="stat-label">VRAM Usage</span>
              <span class="stat-val">${vramPct.toFixed(1)}% <small>(${this.formatBytes(gpu.vram_used_bytes)} / ${this.formatBytes(gpu.vram_total_bytes)})</small></span>
              <div class="progress-track"><div class="progress-fill" style="width: ${vramPct}%"></div></div>
            </div>
            ${extras.length ? `<div class="smart-facts">${extras.join('')}</div>` : ''}
          </div>
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderTemperatures(sensors = []) {
    const container = document.getElementById('temp-list-container');
    if (!container) return;

    if (!sensors || sensors.length === 0) {
      container.innerHTML = '<div class="sub-text">No temperature sensors available</div>';
      return;
    }

    let html = '';
    sensors.forEach((s) => {
      const temp = s.temperature_c || 0;
      let colorClass = 'temp-normal';
      if (temp > 80) colorClass = 'temp-high';
      if (temp > 92) colorClass = 'temp-crit';

      html += `
        <div class="temp-item">
          <div class="temp-label">
            <span class="temp-icon">${ICONS.thermo}</span>
            <span>${esc(s.label)}</span>
          </div>
          <div class="temp-badge ${colorClass}">${temp.toFixed(1)}°C</div>
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderDrives(drives = []) {
    const container = document.getElementById('drives-container');
    if (!container) return;

    if (!drives || drives.length === 0) {
      container.innerHTML = '<div class="sub-text">No drives detected</div>';
      return;
    }

    let html = '';
    drives.forEach((d) => {
      const pct = d.usage_percent || 0;
      let barClass = '';
      if (pct > 85) barClass = 'progress-warning';
      if (pct > 95) barClass = 'progress-danger';

      html += `
        <div class="drive-card">
          <div class="drive-header">
            <div class="drive-icon">${ICONS.drive}</div>
            <div class="drive-title-box">
              <div class="drive-name">${esc(d.device)} (${esc(d.mountpoint)})</div>
              <div class="sub-text">${esc(d.filesystem)} • Free: ${this.formatBytes(d.free_bytes)} of ${this.formatBytes(d.total_bytes)}</div>
            </div>
            <div class="drive-pct">${pct.toFixed(0)}%</div>
          </div>
          <div class="progress-track">
            <div class="progress-fill ${barClass}" style="width: ${pct}%"></div>
          </div>
          <div class="io-row">
            <span>Read <strong>${this.formatBps(d.read_speed_bps)}</strong></span>
            <span>Write <strong>${this.formatBps(d.write_speed_bps)}</strong></span>
            <span>Active <strong>${(d.busy_percent || 0).toFixed(0)}%</strong></span>
          </div>
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderInterfaces(ifaces = []) {
    const container = document.getElementById('ifaces-container');
    if (!container) return;

    if (!ifaces || ifaces.length === 0) {
      container.innerHTML = '<div class="sub-text">No network adapters detected</div>';
      return;
    }

    let html = '';
    ifaces.forEach((iface) => {
      const ips = [...(iface.ipv4 || []), ...(iface.ipv6 || [])];
      html += `
        <div class="iface-card">
          <div class="iface-header">
            <div class="iface-title">
              <span class="status-dot ${iface.is_up ? 'dot-online' : 'dot-offline'}"></span>
              <strong>${esc(iface.name)}</strong>
            </div>
            <div class="sub-text">${esc(iface.mac || '')}</div>
          </div>
          <div class="iface-ips">
            ${ips.map(ip => `<span class="ip-pill">${esc(ip)}</span>`).join('')}
          </div>
          ${iface.is_up ? `
          <div class="io-row">
            <span><span class="net-arrow" style="color: var(--green);">↓</span><strong>${this.formatBps(iface.speed_recv_bps)}</strong></span>
            <span><span class="net-arrow" style="color: var(--orange);">↑</span><strong>${this.formatBps(iface.speed_sent_bps)}</strong></span>
            ${iface.link_speed_mbps ? `<span>Link <strong>${iface.link_speed_mbps >= 1000 ? (iface.link_speed_mbps / 1000) + ' Gbps' : iface.link_speed_mbps + ' Mbps'}</strong></span>` : ''}
          </div>` : ''}
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderFleetOverview(servers, onSelectServer) {
    const container = document.getElementById('fleet-grid-container');
    if (!container) return;

    if (!servers || servers.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <h3>No servers paired yet</h3>
          <p>Click "Pair New Server" to connect to your first machine.</p>
        </div>`;
      return;
    }

    let html = '';
    servers.forEach((s) => {
      const snap = s.lastSnapshot;
      const cpu = snap?.cpu?.usage_percent !== undefined ? `${snap.cpu.usage_percent.toFixed(0)}%` : '--';
      const mem = snap?.memory?.usage_percent !== undefined ? `${snap.memory.usage_percent.toFixed(0)}%` : '--';
      // "Windows 11 Pro (25H2, Build 26200)" -> "Windows 11 Pro"
      const osName = (snap?.host?.os_version || '').replace(/\s*\(.*\)\s*$/, '') || snap?.host?.os || 'System';

      html += `
        <div class="fleet-card" data-server-id="${s.id}">
          <div class="fleet-card-header">
            <div>
              <div class="fleet-server-name">${esc(s.name)}</div>
              <div class="sub-text">${esc(osName)} • ${s.tailcatAddr ? 'Tailcat' : 'Direct LAN'}</div>
            </div>
            <span class="status-badge ${s.status === 'online' ? 'status-online' : 'status-offline'}">
              ${s.status || 'Offline'}
            </span>
          </div>
          <div class="fleet-metrics-row">
            <div class="fleet-metric">
              <span class="sub-text">CPU</span>
              <span class="fleet-val">${cpu}</span>
            </div>
            <div class="fleet-metric">
              <span class="sub-text">RAM</span>
              <span class="fleet-val">${mem}</span>
            </div>
            <div class="fleet-metric">
              <span class="sub-text">Uptime</span>
              <span class="fleet-val">${snap ? this.formatUptime(snap.host.uptime_sec) : '--'}</span>
            </div>
          </div>
          <button class="btn btn-primary btn-sm btn-block view-btn" data-id="${s.id}">View Dashboard →</button>
        </div>
      `;
    });
    container.innerHTML = html;

    // Attach click handlers
    container.querySelectorAll('.view-btn').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const id = e.target.getAttribute('data-id');
        onSelectServer(id);
      });
    });
  },

  setText(id, text) {
    const el = document.getElementById(id);
    if (el) el.textContent = text;
  }
};
