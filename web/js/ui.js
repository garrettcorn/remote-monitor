// UI Renderer for dashboard widgets, gauges, drives, and process tables.

export const UI = {
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
    this.setText('host-os', `${snap.host.os} ${snap.host.os_version || ''}`.trim());
    this.setText('host-uptime', this.formatUptime(snap.host.uptime_sec));
    this.setText('host-kernel', snap.host.kernel || snap.host.architecture);
    this.setText('host-procs', snap.host.process_count || snap.top_processes?.length || 0);

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

    // 9. Top Processes
    this.renderProcesses(snap.top_processes);
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
        <div class="empty-state">
          <span class="icon">🎮</span> No dedicated GPU metrics detected.
        </div>`;
      return;
    }

    let html = '';
    gpus.forEach((gpu) => {
      const util = gpu.utilization_percent || 0;
      const vramPct = gpu.vram_percent || 0;
      const temp = gpu.temperature_c ? `${gpu.temperature_c.toFixed(0)}°C` : 'N/A';
      const pwr = gpu.power_watts ? `${gpu.power_watts.toFixed(1)}W` : '';

      html += `
        <div class="metric-card gpu-card">
          <div class="card-header">
            <div>
              <div class="card-title">${gpu.name || 'GPU ' + gpu.index}</div>
              <div class="sub-text">${gpu.vendor || 'Graphics Device'} ${pwr ? '• ' + pwr : ''}</div>
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
            <span class="temp-icon">🌡️</span>
            <span>${s.label}</span>
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
            <div class="drive-icon">💾</div>
            <div class="drive-title-box">
              <div class="drive-name">${d.device} (${d.mountpoint})</div>
              <div class="sub-text">${d.filesystem} • Free: ${this.formatBytes(d.free_bytes)} of ${this.formatBytes(d.total_bytes)}</div>
            </div>
            <div class="drive-pct">${pct.toFixed(0)}%</div>
          </div>
          <div class="progress-track">
            <div class="progress-fill ${barClass}" style="width: ${pct}%"></div>
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
              <strong>${iface.name}</strong>
            </div>
            <div class="sub-text">${iface.mac || ''}</div>
          </div>
          <div class="iface-ips">
            ${ips.map(ip => `<span class="ip-pill">${ip}</span>`).join('')}
          </div>
        </div>
      `;
    });
    container.innerHTML = html;
  },

  renderProcesses(procs = []) {
    const tbody = document.getElementById('process-table-body');
    if (!tbody) return;

    if (!procs || procs.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="text-center sub-text">No process data available</td></tr>';
      return;
    }

    let html = '';
    procs.forEach((p) => {
      html += `
        <tr>
          <td><span class="pid-badge">${p.pid}</span></td>
          <td class="proc-name">${p.name}</td>
          <td>${(p.cpu_percent || 0).toFixed(1)}%</td>
          <td>${this.formatBytes(p.memory_bytes)}</td>
        </tr>
      `;
    });
    tbody.innerHTML = html;
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
      const osName = snap?.host?.os || 'System';

      html += `
        <div class="fleet-card" data-server-id="${s.id}">
          <div class="fleet-card-header">
            <div>
              <div class="fleet-server-name">${s.name}</div>
              <div class="sub-text">${osName} • ${s.tailcatAddr ? 'Tailcat' : 'Direct LAN'}</div>
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
