// Processes section: running programs grouped by app, sortable, searchable,
// with each app expandable to its individual processes and End task on
// PIN-paired servers.

import { UI } from './ui.js?v=23';
import { MonitorClient } from './client.js?v=23';

const esc = UI.esc;
const SORT_KEY = 'pc_monitor_proc_sort';
const TOP_ROWS = 10;        // apps shown before "Show all"
const FULL_REFRESH_MS = 3000; // full list refresh while searching / showing all

function load(key, fallback) {
  try {
    const v = localStorage.getItem(key);
    return v ? JSON.parse(v) : fallback;
  } catch (e) {
    return fallback;
  }
}

function save(key, value) {
  try { localStorage.setItem(key, JSON.stringify(value)); } catch (e) { /* storage unavailable */ }
}

const COLUMNS = {
  name: { defaultDir: 'asc' },
  cpu: { defaultDir: 'desc' },
  gpu: { defaultDir: 'desc' },
  mem: { defaultDir: 'desc' },
  io: { defaultDir: 'desc' },
};

export function fmtCPU(v) {
  if (!v) return '0%';
  if (v < 0.1) return '<0.1%';
  return (v >= 10 ? v.toFixed(0) : v.toFixed(1)) + '%';
}

function fmtRate(v) {
  return v ? UI.formatBps(v) : '—';
}

const ioOf = (p) => (p.read_bps || 0) + (p.write_bps || 0);
const label = (a) => a.title || a.name.replace(/\.exe$/i, '');

export class Processes {
  constructor(app) {
    this.app = app;
    this.server = null;
    this.snap = null;
    this.full = null;        // {apps, total} from /api/processes
    this.fullTimer = null;
    this.fetching = false;
    this.expanded = new Set(); // app keys
    this.showAll = false;
    this.query = '';
    this.busy = new Set();   // app keys / pids being ended
    this.detailPid = null;   // process whose details are open
    this.detail = null;      // {pid, loading} | {pid, data} | {pid, error}
    this.icons = new Map();  // app key -> object URL | null (none) | 'pending'
    this.iconQueue = [];
    this.iconLoading = 0;
    this.renderTimer = null;
    const s = load(SORT_KEY, null);
    this.sort = s && COLUMNS[s.key] ? s : { key: 'cpu', dir: 'desc' };
    this.bind();
  }

  bind() {
    document.querySelectorAll('#process-table thead [data-sort]').forEach((b) =>
      b.addEventListener('click', () => this.setSort(b.dataset.sort)));
    const body = document.getElementById('process-table-body');
    body?.addEventListener('click', (e) => {
      const btn = e.target.closest('button[data-action]');
      if (btn) {
        e.stopPropagation();
        this.onAction(btn);
        return;
      }
      const row = e.target.closest('tr.proc-app[data-key]');
      if (row) this.toggle(row.dataset.key);
      const proc = e.target.closest('tr.proc-child[data-pid]');
      if (proc) this.toggleDetail(parseInt(proc.dataset.pid, 10));
    });
    body?.addEventListener('keydown', (e) => {
      if (e.target.closest('button') || (e.key !== 'Enter' && e.key !== ' ')) return;
      const row = e.target.closest('tr.proc-app[data-key]');
      const proc = e.target.closest('tr.proc-child[data-pid]');
      if (row || proc) e.preventDefault();
      if (row) this.toggle(row.dataset.key);
      if (proc) this.toggleDetail(parseInt(proc.dataset.pid, 10));
    });
    const search = document.getElementById('proc-search');
    search?.addEventListener('input', () => {
      this.query = search.value.trim().toLowerCase();
      this.syncFull();
      this.render();
    });
    search?.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && search.value) {
        search.value = '';
        search.dispatchEvent(new Event('input'));
      }
    });
    document.addEventListener('visibilitychange', () => this.syncFull());
  }

  setServer(server) {
    if (this.server?.id !== server?.id) {
      this.expanded.clear();
      this.detailPid = null;
      this.clearIcons();
      this.full = null;
    }
    this.server = server;
    this.snap = null;
    this.syncFull();
    this.render();
  }

  update(snap) {
    this.snap = snap;
    this.render();
  }

  setSort(key) {
    if (this.sort.key === key) this.sort.dir = this.sort.dir === 'asc' ? 'desc' : 'asc';
    else this.sort = { key, dir: COLUMNS[key].defaultDir };
    save(SORT_KEY, this.sort);
    this.render();
  }

  toggle(key) {
    if (this.expanded.has(key)) this.expanded.delete(key);
    else this.expanded.add(key);
    this.render();
    document.querySelector(`#process-table-body tr.proc-app[data-key="${CSS.escape(key)}"]`)?.focus();
  }

  setShowAll(on) {
    this.showAll = on;
    this.syncFull();
    this.render();
  }

  // ------------------------------------------------------------ full list

  // The full list (every app, every process) is only fetched while it's
  // needed: searching or "Show all", with the dashboard visible.
  wantsFull() {
    return !!(this.server?.processList && (this.showAll || this.query) && !document.hidden
      && this.app.currentView === 'dashboard');
  }

  syncFull() {
    clearTimeout(this.fullTimer);
    this.fullTimer = null;
    if (!this.wantsFull()) return;
    this.fetchFull().finally(() => {
      if (this.wantsFull()) this.fullTimer = setTimeout(() => this.syncFull(), FULL_REFRESH_MS);
    });
  }

  async fetchFull() {
    const server = this.server;
    if (!server || this.fetching) return;
    this.fetching = true;
    try {
      const resp = await fetch(`${MonitorClient.baseURL(server)}/api/processes`, { headers: MonitorClient.authHeaders(server), cache: 'no-store' });
      if (!resp.ok) return;
      const data = await resp.json();
      if (this.server === server) {
        this.full = data;
        this.render();
      }
    } catch (e) {
      /* offline: keep showing the snapshot */
    } finally {
      this.fetching = false;
    }
  }

  // -------------------------------------------------------------- end task

  async onAction(btn) {
    const action = btn.dataset.action;
    if (action === 'show-all' || action === 'show-top') {
      if (btn.dataset.app) this.expanded.add(btn.dataset.app);
      this.setShowAll(action === 'show-all');
      return;
    }
    const server = this.server;
    if (!server?.processEnd) return;
    const app = this.findApp(btn.dataset.app);
    if (!app) return;
    let pids, what;
    if (btn.dataset.action === 'end-app') {
      // The snapshot only lists a few processes per app; get all of them.
      if (app.count > (app.processes || []).length) await this.fetchFull();
      const full = this.findApp(btn.dataset.app) || app;
      pids = (full.processes || []).map((p) => p.pid);
      what = `"${label(app)}" (${pids.length} process${pids.length === 1 ? '' : 'es'})`;
    } else {
      pids = [parseInt(btn.dataset.pid, 10)];
      what = `${label(app)} process ${pids[0]}`;
    }
    if (!pids.length) return;
    if (!confirm(`End ${what} on ${server.name}?\n\nThe program closes immediately, like End task in Task Manager. Unsaved work in it will be lost.`)) return;
    const busyKey = btn.dataset.action === 'end-app' ? 'app:' + btn.dataset.app : 'pid:' + pids[0];
    this.busy.add(busyKey);
    this.render();
    try {
      const r = await MonitorClient.post(server, '/api/processes/end', { name: app.name, pids });
      const failed = r.failed || [];
      if (failed.length) {
        this.app.showToast(`Ended ${r.ended} of ${pids.length}; ${failed.length} couldn't be ended: ${failed[0].error}`);
      } else {
        this.app.showToast(`Ended ${what}`);
      }
    } catch (e) {
      if (e.code === 'end_failed') this.app.showToast(`Couldn't end ${what}: ${e.message}`);
      else this.app.handleControlError(server, e);
    } finally {
      this.busy.delete(busyKey);
      // Give the server one sampling interval to notice the change.
      await new Promise((r) => setTimeout(r, 1500));
      await this.fetchFull();
      this.render();
    }
  }

  // ---------------------------------------------------------------- details

  toggleDetail(pid) {
    this.detailPid = this.detailPid === pid ? null : pid;
    if (this.detailPid) this.fetchDetail(pid);
    this.render();
    document.querySelector(`#process-table-body tr.proc-child[data-pid="${pid}"]`)?.focus();
  }

  async fetchDetail(pid) {
    const server = this.server;
    this.detail = { pid, loading: true };
    try {
      const resp = await fetch(`${MonitorClient.baseURL(server)}/api/processes/detail?pid=${pid}`, { headers: MonitorClient.authHeaders(server), cache: 'no-store' });
      const data = await resp.json().catch(() => ({}));
      if (this.detailPid !== pid || this.server !== server) return;
      this.detail = resp.ok ? { pid, data } : { pid, error: data.message || `HTTP ${resp.status}` };
    } catch (e) {
      if (this.detailPid === pid) this.detail = { pid, error: 'Server unreachable' };
    }
    this.render();
  }

  renderDetail(pid, cols) {
    const d = this.detail;
    let inner;
    if (!d || d.pid !== pid || d.loading) inner = '<span class="sub-text">Loading details…</span>';
    else if (d.error) inner = `<span class="sub-text">${esc(d.error)}</span>`;
    else {
      const x = d.data;
      const rows = [];
      const add = (k, v) => { if (v !== undefined && v !== null && v !== '') rows.push(`<dt>${k}</dt><dd>${v}</dd>`); };
      add('Program', x.title ? esc(x.title) : '');
      add('Path', x.path ? `<span class="proc-path">${esc(x.path)}</span>` : '<span class="sub-text">Not readable by the server (protected or another user\'s process)</span>');
      add('User', esc(x.user || ''));
      if (x.elevated !== undefined) add('Administrator', x.elevated ? 'Yes' : 'No');
      if (x.started) {
        const t = new Date(x.started);
        add('Started', `${esc(t.toLocaleString())} <span class="sub-text">(${esc(UI.formatUptime(Math.max(0, (Date.now() - t) / 1000)))} ago)</span>`);
      }
      if (x.parent_pid) add('Parent', `${esc(x.parent_name || 'unknown')} <span class="pid-badge">PID ${x.parent_pid}</span>`);
      add('Threads', x.threads || '');
      add(this.snap?.host?.os === 'windows' ? 'Handles' : 'Open files', x.handles || '');
      add('Memory', x.memory_bytes ? UI.formatBytes(x.memory_bytes) : '');
      add('Private memory', x.private_bytes ? UI.formatBytes(x.private_bytes) : '');
      add('Peak memory', x.peak_bytes ? UI.formatBytes(x.peak_bytes) : '');
      inner = `<dl class="proc-dl">${rows.join('')}</dl>`;
    }
    return `<tr class="proc-detail"><td colspan="${cols}">${inner}</td></tr>`;
  }

  // ------------------------------------------------------------------ icons

  // Icon <img> when loaded, else a letter badge (and queue a fetch on hosts
  // that serve icons).
  iconHTML(a) {
    const key = a.name.toLowerCase();
    const url = this.icons.get(key);
    if (typeof url === 'string' && url !== 'pending') return `<img class="proc-icon" src="${url}" alt="" aria-hidden="true">`;
    if (url === undefined && this.server?.processIcons && !a.legacy) this.queueIcon(key, a.name);
    const text = label(a);
    let hue = 0;
    for (const ch of key) hue = (hue * 31 + ch.charCodeAt(0)) % 360;
    return `<span class="proc-avatar" style="--hue:${hue}" aria-hidden="true">${esc(text.charAt(0).toUpperCase() || '?')}</span>`;
  }

  queueIcon(key, name) {
    this.icons.set(key, 'pending');
    this.iconQueue.push({ key, name });
    this.pumpIcons();
  }

  async pumpIcons() {
    while (this.iconQueue.length && this.iconLoading < 3) {
      const { key, name } = this.iconQueue.shift();
      const server = this.server;
      this.iconLoading++;
      (async () => {
        let url = null;
        try {
          const resp = await fetch(`${MonitorClient.baseURL(server)}/api/processes/icon?name=${encodeURIComponent(name)}`, { headers: MonitorClient.authHeaders(server) });
          if (resp.status === 200) url = URL.createObjectURL(await resp.blob());
        } catch (e) { /* no icon */ }
        if (this.server !== server) {
          if (url) URL.revokeObjectURL(url);
        } else {
          this.icons.set(key, url);
          if (url) this.scheduleRender();
        }
      })().finally(() => {
        this.iconLoading--;
        this.pumpIcons();
      });
    }
  }

  clearIcons() {
    for (const u of this.icons.values()) if (typeof u === 'string' && u !== 'pending') URL.revokeObjectURL(u);
    this.icons.clear();
    this.iconQueue = [];
  }

  scheduleRender() {
    if (this.renderTimer) return;
    this.renderTimer = setTimeout(() => { this.renderTimer = null; this.render(); }, 120);
  }

  findApp(key) {
    const lists = [this.full?.apps, this.snap?.apps];
    for (const l of lists) {
      const a = (l || []).find((x) => x.name.toLowerCase() === key);
      if (a) return a;
    }
    return null;
  }

  // ------------------------------------------------------------- rendering

  // Apps to show: the full list while searching / showing all (once
  // fetched), otherwise the snapshot's. Servers older than 1.7.4 only send
  // top_processes, shown one process per row.
  apps() {
    const s = this.snap;
    if ((this.showAll || this.query) && this.full?.apps) return this.full.apps;
    if (!s) return [];
    if (Array.isArray(s.apps)) return s.apps;
    return (s.top_processes || []).map((p) => ({ ...p, count: 1, processes: [p], legacy: true }));
  }

  filter(list) {
    const q = this.query;
    if (!q) return list;
    return list.filter((a) => label(a).toLowerCase().includes(q) || a.name.toLowerCase().includes(q)
      || (a.processes || []).some((p) => String(p.pid).startsWith(q)));
  }

  sorted(list) {
    const { key, dir } = this.sort;
    const val = {
      name: (a) => label(a).toLowerCase(),
      cpu: (a) => a.cpu_percent || 0,
      gpu: (a) => a.gpu_percent || 0,
      mem: (a) => a.memory_bytes || 0,
      io: (a) => ioOf(a),
    }[key];
    const sign = dir === 'asc' ? 1 : -1;
    return [...list].sort((a, b) => {
      const x = val(a), y = val(b);
      if (x < y) return -sign;
      if (x > y) return sign;
      return (b.cpu_percent || 0) - (a.cpu_percent || 0) || (b.memory_bytes || 0) - (a.memory_bytes || 0);
    });
  }

  renderHeader(showIO, showGPU) {
    document.querySelectorAll('#process-table thead th[data-col]').forEach((th) => {
      const k = th.dataset.col;
      const active = this.sort.key === k;
      th.setAttribute('aria-sort', active ? (this.sort.dir === 'asc' ? 'ascending' : 'descending') : 'none');
      th.classList.toggle('is-sorted', active);
      const arrow = th.querySelector('.sort-arrow');
      if (arrow) arrow.textContent = active ? (this.sort.dir === 'asc' ? '▲' : '▼') : '';
      if (k === 'io') th.classList.toggle('hidden', !showIO);
      if (k === 'gpu') th.classList.toggle('hidden', !showGPU);
    });
  }

  renderSummary(apps) {
    const el = document.getElementById('proc-summary');
    if (!el) return;
    const s = this.snap;
    if (!s) { el.textContent = 'Waiting for data…'; return; }
    const parts = [`${s.host?.process_count || 0} processes`];
    if (s.app_count) parts.push(`${s.app_count} apps`);
    if (!apps[0]?.legacy) parts.push('CPU averaged over 5 s');
    el.textContent = parts.join(' · ');
  }

  render() {
    const body = document.getElementById('process-table-body');
    if (!body) return;
    const search = document.getElementById('proc-search');
    if (search) search.disabled = !this.snap;
    const all = this.apps();
    const showIO = !!this.snap?.process_io;
    const showGPU = !!this.snap?.process_gpu;
    const canEnd = !!this.server?.processEnd;
    this.renderHeader(showIO, showGPU);
    this.renderSummary(all);
    const cols = 3 + (showIO ? 1 : 0) + (showGPU ? 1 : 0);
    if (!this.snap) {
      body.innerHTML = `<tr><td colspan="${cols}" class="text-center sub-text">Waiting for process data…</td></tr>`;
      return;
    }
    const matched = this.sorted(this.filter(all));
    if (!matched.length) {
      const msg = this.query ? `No apps match “${esc(this.query)}”` : 'No process data from this host';
      body.innerHTML = `<tr><td colspan="${cols}" class="text-center sub-text">${msg}</td></tr>`;
      return;
    }
    const limited = !this.showAll && !this.query && matched.length > TOP_ROWS;
    const list = limited ? matched.slice(0, TOP_ROWS) : matched;
    const maxCPU = Math.max(...list.map((a) => a.cpu_percent || 0), 1);
    const maxMem = Math.max(...list.map((a) => a.memory_bytes || 0), 1);
    const maxGPU = Math.max(...list.map((a) => a.gpu_percent || 0), 1);
    const bar = (v, max, cls) => `<span class="proc-bar ${cls}"><span style="width:${v > 0 ? Math.max(2, Math.round((v / max) * 100)) : 0}%"></span></span>`;
    // A numeric search opens the apps whose PIDs match.
    const pidQuery = /^\d+$/.test(this.query) ? this.query : '';

    let html = list.map((a) => {
      const key = a.name.toLowerCase();
      const procs = a.processes || [];
      const pidHit = pidQuery && procs.some((p) => String(p.pid).startsWith(pidQuery));
      const open = this.expanded.has(key) || pidHit;
      const io = showIO ? `<td class="num proc-io" title="Read ${fmtRate(a.read_bps)} · Write ${fmtRate(a.write_bps)}">${fmtRate(ioOf(a))}</td>` : '';
      const head = `
        <tr class="proc-app${open ? ' is-open' : ''}" data-key="${esc(key)}" tabindex="0" aria-expanded="${open}">
          <td class="proc-app-name"><div class="proc-name-row">
            <span class="proc-chevron" aria-hidden="true">${open ? '▾' : '▸'}</span>
            ${this.iconHTML(a)}
            <span class="proc-label">
              <span class="proc-title">${esc(label(a))}</span>
              ${a.title ? `<span class="proc-exe">${esc(a.name)}</span>` : ''}
            </span>
            ${a.count > 1 ? `<span class="proc-count" title="${a.count} processes">${a.count}</span>` : ''}
          </div></td>
          <td class="num"><span class="proc-val">${fmtCPU(a.cpu_percent)}</span>${bar(a.cpu_percent || 0, maxCPU, 'cpu')}</td>
          ${showGPU ? `<td class="num proc-gpu"><span class="proc-val">${fmtCPU(a.gpu_percent)}</span>${bar(a.gpu_percent || 0, maxGPU, 'gpu')}</td>` : ''}
          <td class="num"><span class="proc-val">${UI.formatBytes(a.memory_bytes)}</span>${bar(a.memory_bytes || 0, maxMem, 'mem')}</td>
          ${io}
        </tr>`;
      if (!open) return head;
      const appBusy = this.busy.has('app:' + key);
      const actions = canEnd && !a.legacy ? `
        <tr class="proc-child proc-actions"><td colspan="${cols}">
          <button class="btn btn-outline btn-sm btn-danger-outline" data-action="end-app" data-app="${esc(key)}" ${appBusy ? 'disabled' : ''}>
            ${appBusy ? 'Ending…' : a.count > 1 ? `End task (${a.count} processes)` : 'End task'}</button>
        </td></tr>` : '';
      const kids = procs.map((p) => {
        const hit = pidQuery && String(p.pid).startsWith(pidQuery);
        const pBusy = this.busy.has('pid:' + p.pid);
        const end = canEnd && !a.legacy && a.count > 1
          ? `<button class="proc-end-btn" data-action="end-pid" data-app="${esc(key)}" data-pid="${p.pid}" ${pBusy ? 'disabled' : ''} title="End this process">${pBusy ? '…' : 'End'}</button>` : '';
        // Details need a 1.7.4+ server (same release as /api/processes).
        const canDetail = !!this.server?.processList && !a.legacy;
        const openDetail = canDetail && this.detailPid === p.pid;
        return `
        <tr class="proc-child${hit ? ' is-hit' : ''}${canDetail ? ' is-clickable' : ''}${openDetail ? ' is-open' : ''}"${canDetail ? ` data-pid="${p.pid}" tabindex="0" aria-expanded="${openDetail}" title="Show details"` : ''}>
          <td class="proc-app-name"><span class="pid-badge">PID ${p.pid}</span> <span class="proc-exe">${esc(p.name)}</span>${end}</td>
          <td class="num">${fmtCPU(p.cpu_percent)}</td>
          ${showGPU ? `<td class="num proc-gpu">${fmtCPU(p.gpu_percent)}</td>` : ''}
          <td class="num">${UI.formatBytes(p.memory_bytes)}</td>
          ${showIO ? `<td class="num proc-io">${fmtRate(ioOf(p))}</td>` : ''}
        </tr>${openDetail ? this.renderDetail(p.pid, cols) : ''}`;
      }).join('');
      const hidden = a.count - procs.length;
      const more = hidden > 0
        ? `<tr class="proc-child proc-more"><td colspan="${cols}">${hidden} more ${esc(a.name)} process${hidden === 1 ? '' : 'es'}${this.server?.processList ? ' · <button class="link-btn" data-action="show-all" data-app="' + esc(key) + '">Show all</button>' : ''}</td></tr>`
        : '';
      return head + actions + kids + more;
    }).join('');

    if (limited) {
      html += `<tr class="proc-footer"><td colspan="${cols}"><button class="link-btn" data-action="show-all" data-app="">Show all ${this.snap.app_count || matched.length} apps</button></td></tr>`;
    } else if (this.showAll && !this.query) {
      html += `<tr class="proc-footer"><td colspan="${cols}"><button class="link-btn" data-action="show-top" data-app="">Show top ${TOP_ROWS} only</button></td></tr>`;
    }
    body.innerHTML = html;
  }
}
