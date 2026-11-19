// Dashboard customization (show/hide sections) and the Stream Deck-style
// button grid in the Control Panel.

import { UI } from './ui.js?v=13';
import { MonitorClient } from './client.js?v=13';

const esc = UI.esc;

// ------------------------------------------------------------------ storage

function load(key, fallback) {
  try {
    const v = localStorage.getItem(key);
    return v ? JSON.parse(v) : fallback;
  } catch (e) {
    return fallback;
  }
}

function save(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch (e) { /* storage unavailable: keep working in memory */ }
}

// ------------------------------------------------------------------- layout

export const SECTION_GROUPS = [
  { title: 'Overview', items: [
    ['summary', 'System summary'],
    ['cpu', 'Processor (CPU)'],
    ['memory', 'Memory (RAM)'],
    ['network', 'Network activity'],
    ['thermals', 'Thermals & sensors'],
  ] },
  { title: 'Control Panel', items: [
    ['deck', 'Buttons', true],
    ['audio', 'Audio'],
    ['power', 'Power (restart / shut down)', true],
  ] },
  { title: 'Hardware', items: [
    ['displays', 'Displays & screen preview'],
    ['cores', 'Per-core CPU'],
    ['gpu', 'Graphics (GPU)'],
    ['health', 'Drive health'],
    ['devices', 'Connected devices'],
    ['storage', 'Storage & partitions'],
    ['adapters', 'Network adapters'],
    ['processes', 'Top processes'],
  ] },
];

const HIDDEN_KEY = 'pc_monitor_hidden_sections';

export const Layout = {
  hidden: new Set(load(HIDDEN_KEY, [])),
  controlEnabled: false,

  isHidden(id) { return this.hidden.has(id); },

  setHidden(id, hide) {
    if (hide) this.hidden.add(id); else this.hidden.delete(id);
    save(HIDDEN_KEY, [...this.hidden]);
    this.apply();
  },

  showAll() {
    this.hidden.clear();
    save(HIDDEN_KEY, []);
    this.apply();
  },

  // Apply visibility to every [data-section] and reflow grid rows so hidden
  // cards leave no gaps. control-only parts also need a control-enabled server.
  apply(controlEnabled = this.controlEnabled) {
    this.controlEnabled = controlEnabled;
    document.querySelectorAll('[data-section]').forEach((el) => {
      const id = el.dataset.section;
      const off = this.hidden.has(id) || (el.classList.contains('control-only') && !controlEnabled);
      el.classList.toggle('section-hidden', off);
    });

    // Control Panel: hide when none of its parts are visible; single column
    // when only the side panels or only the deck remain.
    const cp = document.getElementById('control-panel');
    if (cp) {
      const parts = [...cp.querySelectorAll('.cp-panel')];
      const visible = parts.filter(p => !p.classList.contains('section-hidden'));
      const deckOn = visible.some(p => p.dataset.section === 'deck');
      cp.classList.toggle('section-hidden', this.hidden.has('control') || visible.length === 0);
      cp.querySelector('.cp-grid')?.classList.toggle('cp-single', !deckOn || visible.length === 1);
      const side = cp.querySelector('.cp-side');
      side?.classList.toggle('section-hidden', !visible.some(p => side.contains(p)));
    }

    document.querySelectorAll('.grid-hero, .grid-two-col, .grid-equal-col').forEach((row) => {
      const shown = [...row.children].filter(c => !c.classList.contains('section-hidden')).length;
      row.classList.toggle('section-hidden', shown === 0);
      row.classList.toggle('grid-single', shown === 1);
    });
  },

  renderSettings() {
    const list = document.getElementById('settings-list');
    if (!list) return;
    list.innerHTML = SECTION_GROUPS.map(g => `
      <div class="settings-group">
        <div class="audio-label">${esc(g.title)}</div>
        ${g.items.map(([id, label, controlOnly]) => `
          <label class="switch-row">
            <span>${esc(label)}${controlOnly ? ' <small class="sub-text">(needs control)</small>' : ''}</span>
            <input type="checkbox" class="switch" data-id="${id}" ${this.hidden.has(id) ? '' : 'checked'}>
          </label>`).join('')}
      </div>`).join('');
  },
};

// ------------------------------------------------------------- stream deck

export const DECK_COLORS = {
  blue: 'var(--blue)', purple: 'var(--purple)', green: 'var(--green)',
  orange: 'var(--orange)', red: 'var(--red)', gray: 'var(--text-muted)',
};

const ICON = {
  play_pause: '<path d="M5 4l7 6-7 6zM13 4h2v12h-2z"/>',
  next: '<path d="M4 4l7 6-7 6zM11 4l7 6-7 6z"/>',
  previous: '<path d="M16 4l-7 6 7 6zM9 4l-7 6 7 6z"/>',
  stop: '<rect x="5" y="5" width="10" height="10" rx="1.5"/>',
  volume_up: '<path d="M3 8h3l4-3.5v11L6 12H3z"/><path d="M13 7.5a3.5 3.5 0 0 1 0 5M15 5a7 7 0 0 1 0 10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
  volume_down: '<path d="M3 8h3l4-3.5v11L6 12H3z"/><path d="M13 7.5a3.5 3.5 0 0 1 0 5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
  volume_mute: '<path d="M3 8h3l4-3.5v11L6 12H3z"/><path d="M13 8l4 4m0-4-4 4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
  mic_mute: '<rect x="7.5" y="3" width="5" height="9" rx="2.5"/><path d="M5 10a5 5 0 0 0 10 0M10 15v2.5M4 3l12 14" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
  lock: '<rect x="4.5" y="9" width="11" height="8" rx="2"/><path d="M7 9V6.5a3 3 0 0 1 6 0V9" fill="none" stroke="currentColor" stroke-width="1.6"/>',
  hotkey: '<rect x="2.5" y="5" width="15" height="10" rx="2" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="M5.5 8h1M9.5 8h1M13.5 8h1M6.5 12h7" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
  url: '<circle cx="10" cy="10" r="7" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="M3 10h14M10 3c2.5 2.6 2.5 11.4 0 14M10 3c-2.5 2.6-2.5 11.4 0 14" fill="none" stroke="currentColor" stroke-width="1.4"/>',
  launch: '<rect x="3" y="3" width="6" height="6" rx="1.5"/><rect x="11" y="3" width="6" height="6" rx="1.5"/><rect x="3" y="11" width="6" height="6" rx="1.5"/><rect x="11" y="11" width="6" height="6" rx="1.5"/>',
  add: '<path d="M10 4v12M4 10h12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>',
};

const MEDIA_LABELS = {
  play_pause: 'Play / Pause', next: 'Next', previous: 'Previous', stop: 'Stop',
  volume_up: 'Volume +', volume_down: 'Volume −', volume_mute: 'Mute',
};

const DEFAULT_BUTTONS = [
  { type: 'media', key: 'play_pause', label: 'Play / Pause', color: 'blue' },
  { type: 'media', key: 'previous', label: 'Previous', color: 'blue' },
  { type: 'media', key: 'next', label: 'Next', color: 'blue' },
  { type: 'media', key: 'volume_down', label: 'Volume −', color: 'purple' },
  { type: 'media', key: 'volume_up', label: 'Volume +', color: 'purple' },
  { type: 'media', key: 'volume_mute', label: 'Mute', color: 'purple' },
  { type: 'mic_mute', label: 'Mic mute', color: 'orange' },
  { type: 'lock', label: 'Lock', color: 'gray' },
];

function iconFor(btn) {
  const k = btn.type === 'media' ? btn.key : btn.type;
  return `<svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">${ICON[k] || ICON.hotkey}</svg>`;
}

function subtitle(btn) {
  if (btn.type === 'hotkey') return btn.keys || '';
  if (btn.type === 'url') { try { return new URL(btn.url).host; } catch (e) { return ''; } }
  return '';
}

// Map a KeyboardEvent to our shortcut syntax ("ctrl+shift+m").
function shortcutFromEvent(e) {
  const named = {
    ' ': 'space', Enter: 'enter', Tab: 'tab', Escape: 'esc', Backspace: 'backspace',
    Delete: 'delete', Insert: 'insert', Home: 'home', End: 'end', PageUp: 'pageup',
    PageDown: 'pagedown', ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left',
    ArrowRight: 'right', PrintScreen: 'printscreen',
  };
  const mods = [];
  if (e.ctrlKey) mods.push('ctrl');
  if (e.shiftKey) mods.push('shift');
  if (e.altKey) mods.push('alt');
  if (e.metaKey) mods.push('win');
  let key = null;
  if (named[e.key]) key = named[e.key];
  else if (/^F([1-9]|1\d|2[0-4])$/.test(e.key)) key = e.key.toLowerCase();
  else if (/^Key[A-Z]$/.test(e.code)) key = e.code.slice(3).toLowerCase();
  else if (/^Digit[0-9]$/.test(e.code)) key = e.code.slice(5);
  return { mods, key };
}

const pretty = (s) => s.split('+').map(p => p.length <= 1 ? p.toUpperCase() : p[0].toUpperCase() + p.slice(1)).join('+');

export class Deck {
  constructor(app) {
    this.app = app;
    this.server = null;
    this.buttons = [];
    this.editing = false;
    this.editIndex = -1;
    this.launchList = null;
    this.dragFrom = -1;
    this.bind();
  }

  storageKey() { return 'pc_monitor_deck_' + (this.server?.id || 'none'); }

  setServer(server) {
    if (this.server?.id !== server?.id) this.launchList = null;
    this.server = server;
    this.buttons = load(this.storageKey(), null) || DEFAULT_BUTTONS.map(b => ({ ...b }));
    this.editing = false;
    this.render();
  }

  persist() { save(this.storageKey(), this.buttons); }

  render() {
    const grid = document.getElementById('deck-grid');
    const editBtn = document.getElementById('btn-deck-edit');
    if (!grid) return;
    if (editBtn) {
      editBtn.textContent = this.editing ? 'Done' : 'Edit';
      editBtn.setAttribute('aria-pressed', String(this.editing));
    }
    grid.classList.toggle('is-editing', this.editing);
    let html = this.buttons.map((b, i) => `
      <button class="deck-btn" data-index="${i}" style="--accent: ${DECK_COLORS[b.color] || DECK_COLORS.blue}"
              ${this.editing ? 'draggable="true"' : ''} title="${esc(b.label)}${subtitle(b) ? ' · ' + esc(subtitle(b)) : ''}">
        <span class="deck-icon">${iconFor(b)}</span>
        <span class="deck-label">${esc(b.label)}</span>
        ${this.editing ? '<span class="deck-edit-badge" aria-hidden="true">Edit</span>' : ''}
      </button>`).join('');
    if (this.editing) {
      html += `<button class="deck-btn deck-add" data-add="1"><span class="deck-icon"><svg viewBox="0 0 20 20" aria-hidden="true">${ICON.add}</svg></span><span class="deck-label">Add</span></button>`;
    }
    if (!this.buttons.length && !this.editing) {
      html = '<div class="sub-text">No buttons yet. Click Edit to add some.</div>';
    }
    grid.innerHTML = html;
  }

  bind() {
    document.getElementById('btn-deck-edit')?.addEventListener('click', () => {
      this.editing = !this.editing;
      this.render();
    });

    const grid = document.getElementById('deck-grid');
    grid?.addEventListener('click', (e) => {
      const btn = e.target.closest('.deck-btn');
      if (!btn) return;
      if (btn.dataset.add) return this.openEditor(-1);
      const i = parseInt(btn.dataset.index, 10);
      if (this.editing) this.openEditor(i);
      else this.press(i, btn);
    });

    // Drag to reorder (edit mode).
    grid?.addEventListener('dragstart', (e) => {
      const btn = e.target.closest('.deck-btn[data-index]');
      if (!btn) return;
      this.dragFrom = parseInt(btn.dataset.index, 10);
      e.dataTransfer.effectAllowed = 'move';
      btn.classList.add('dragging');
    });
    grid?.addEventListener('dragover', (e) => {
      if (this.dragFrom >= 0 && e.target.closest('.deck-btn[data-index]')) e.preventDefault();
    });
    grid?.addEventListener('drop', (e) => {
      const target = e.target.closest('.deck-btn[data-index]');
      if (!target || this.dragFrom < 0) return;
      e.preventDefault();
      const to = parseInt(target.dataset.index, 10);
      const [moved] = this.buttons.splice(this.dragFrom, 1);
      this.buttons.splice(to, 0, moved);
      this.dragFrom = -1;
      this.persist();
      this.render();
    });
    grid?.addEventListener('dragend', () => { this.dragFrom = -1; this.render(); });

    // Editor form.
    document.getElementById('deck-type')?.addEventListener('change', () => this.syncFields(true));
    document.getElementById('deck-media')?.addEventListener('change', () => this.autoLabel());
    document.getElementById('deck-launch')?.addEventListener('change', () => this.autoLabel());
    const keys = document.getElementById('deck-keys');
    keys?.addEventListener('keydown', (e) => {
      e.preventDefault();
      const { mods, key } = shortcutFromEvent(e);
      if (!key) {
        keys.value = mods.length ? pretty(mods.join('+')) + '+…' : '';
        keys.dataset.value = '';
        return;
      }
      const combo = [...mods, key].join('+');
      keys.dataset.value = combo;
      keys.value = pretty(combo);
      this.autoLabel();
    });
    document.getElementById('deck-colors')?.addEventListener('click', (e) => {
      const sw = e.target.closest('.swatch');
      if (!sw) return;
      document.querySelectorAll('#deck-colors .swatch').forEach(s => s.setAttribute('aria-checked', String(s === sw)));
    });
    document.getElementById('deck-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.saveEditor();
    });
    document.getElementById('btn-deck-delete')?.addEventListener('click', () => {
      if (this.editIndex >= 0) {
        this.buttons.splice(this.editIndex, 1);
        this.persist();
        this.render();
      }
      document.getElementById('deck-modal')?.classList.add('hidden');
    });
  }

  async press(i, el) {
    const b = this.buttons[i];
    if (!b || !this.server) return;
    el.classList.add('pressed');
    setTimeout(() => el.classList.remove('pressed'), 220);
    const action = { type: b.type, key: b.key, keys: b.keys, url: b.url, id: b.id };
    try {
      await MonitorClient.post(this.server, '/api/deck/press', action);
      el.classList.add('ok');
      setTimeout(() => el.classList.remove('ok'), 600);
    } catch (e) {
      this.app.handleControlError(this.server, e);
    }
  }

  async loadLaunchList() {
    if (this.launchList) return this.launchList;
    try {
      const resp = await fetch(`${MonitorClient.baseURL(this.server)}/api/deck/actions`, { headers: MonitorClient.authHeaders(this.server) });
      const data = await resp.json();
      this.launchList = resp.ok ? data : { launch: [], error: data.message };
    } catch (e) {
      this.launchList = { launch: [], error: 'Server unreachable' };
    }
    return this.launchList;
  }

  async openEditor(i) {
    this.editIndex = i;
    const b = i >= 0 ? this.buttons[i] : { type: 'media', key: 'play_pause', label: '', color: 'blue' };
    document.getElementById('deck-modal-title').textContent = i >= 0 ? 'Edit button' : 'Add button';
    document.getElementById('btn-deck-delete').classList.toggle('hidden', i < 0);
    document.getElementById('deck-error').classList.add('hidden');
    document.getElementById('deck-type').value = b.type;
    document.getElementById('deck-media').value = b.key || 'play_pause';
    const keys = document.getElementById('deck-keys');
    keys.dataset.value = b.keys || '';
    keys.value = b.keys ? pretty(b.keys) : '';
    document.getElementById('deck-url').value = b.url || '';
    document.getElementById('deck-label').value = b.label || '';
    this.labelTouched = i >= 0;
    document.getElementById('deck-colors').innerHTML = Object.entries(DECK_COLORS).map(([name, css]) =>
      `<button type="button" class="swatch" role="radio" aria-label="${name}" aria-checked="${name === (b.color || 'blue')}" data-color="${name}" style="--accent: ${css}"></button>`).join('');
    document.getElementById('deck-modal').classList.remove('hidden');

    const list = await this.loadLaunchList();
    const sel = document.getElementById('deck-launch');
    sel.innerHTML = (list.launch || []).map(a => `<option value="${esc(a.id)}">${esc(a.name)}</option>`).join('');
    if (b.id) sel.value = b.id;
    const help = document.getElementById('deck-launch-help');
    help.textContent = list.error
      ? list.error
      : (list.launch?.length ? 'Apps are configured on the server' : 'No apps configured yet. ')
        + (list.actions_file ? ` (${list.actions_file})` : '') + (list.launch?.length ? '.' : ' Add them there to make them available here; see the README.');
    this.syncFields(false);
  }

  syncFields(fromTypeChange) {
    const type = document.getElementById('deck-type').value;
    document.querySelectorAll('#deck-form .deck-field').forEach(f => f.classList.toggle('hidden', f.dataset.for !== type));
    if (fromTypeChange && !this.labelTouched) document.getElementById('deck-label').value = '';
    this.autoLabel();
  }

  // Suggest a label until the user types their own.
  autoLabel() {
    const label = document.getElementById('deck-label');
    if (this.labelTouched && label.value) return;
    const type = document.getElementById('deck-type').value;
    let s = '';
    if (type === 'media') s = MEDIA_LABELS[document.getElementById('deck-media').value];
    else if (type === 'mic_mute') s = 'Mic mute';
    else if (type === 'lock') s = 'Lock';
    else if (type === 'hotkey') s = document.getElementById('deck-keys').value.replace('…', '');
    else if (type === 'launch') s = document.getElementById('deck-launch').selectedOptions[0]?.text || '';
    if (s) label.value = s.slice(0, 18);
    label.oninput = () => { this.labelTouched = true; };
  }

  saveEditor() {
    const err = document.getElementById('deck-error');
    const fail = (m) => { err.textContent = m; err.classList.remove('hidden'); };
    const type = document.getElementById('deck-type').value;
    const b = {
      type,
      label: document.getElementById('deck-label').value.trim(),
      color: document.querySelector('#deck-colors .swatch[aria-checked="true"]')?.dataset.color || 'blue',
    };
    if (type === 'media') b.key = document.getElementById('deck-media').value;
    if (type === 'hotkey') {
      b.keys = document.getElementById('deck-keys').dataset.value;
      if (!b.keys) return fail('Click the shortcut field and press a key combination.');
    }
    if (type === 'url') {
      b.url = document.getElementById('deck-url').value.trim();
      if (!/^https?:\/\/[^/\s]+/i.test(b.url)) return fail('Enter a web address starting with http:// or https://');
    }
    if (type === 'launch') {
      b.id = document.getElementById('deck-launch').value;
      if (!b.id) return fail('No apps are configured on the server yet.');
    }
    if (!b.label) return fail('Give the button a label.');
    if (this.editIndex >= 0) this.buttons[this.editIndex] = b;
    else this.buttons.push(b);
    this.persist();
    this.render();
    document.getElementById('deck-modal').classList.add('hidden');
  }
}

// ----------------------------------------------------------------- displays

const AUTOREFRESH_KEY = 'pc_monitor_screen_autorefresh';
const REFRESH_MS = 10000;

// Displays card: connected displays plus on-demand / auto-refreshing screen
// previews. Nothing is captured until the user clicks Capture or turns on
// auto-refresh (off by default).
export class Screens {
  constructor(app) {
    this.app = app;
    this.server = null;
    this.captureEnabled = false;
    this.displays = [];
    this.signature = '';
    this.images = {};     // display index -> object URL
    this.capturedAt = {}; // display index -> Date
    this.busy = false;
    this.viewing = -1;    // index shown in the full-size viewer
    this.autoRefresh = load(AUTOREFRESH_KEY, false) === true;
    this.timer = setInterval(() => this.tick(), REFRESH_MS);
    this.bind();
  }

  bind() {
    const toggle = document.getElementById('screen-autorefresh');
    if (toggle) {
      toggle.checked = this.autoRefresh;
      toggle.addEventListener('change', () => {
        this.autoRefresh = toggle.checked;
        save(AUTOREFRESH_KEY, this.autoRefresh);
        if (this.autoRefresh) this.captureAll(); // start immediately, then every 10 s
      });
    }
    document.getElementById('btn-screen-capture')?.addEventListener('click', () => this.captureAll());
    document.getElementById('displays-container')?.addEventListener('click', (e) => {
      const prev = e.target.closest('.screen-preview[data-index]');
      if (prev && this.images[prev.dataset.index]) this.openViewer(parseInt(prev.dataset.index, 10));
    });
    document.querySelectorAll('#screen-modal .modal-close-btn').forEach(b => b.addEventListener('click', () => { this.viewing = -1; }));
  }

  clearImages() {
    Object.values(this.images).forEach(u => URL.revokeObjectURL(u));
    this.images = {};
    this.capturedAt = {};
  }

  setServer(server, captureEnabled) {
    if (this.server?.id !== server?.id) this.clearImages();
    this.server = server;
    this.captureEnabled = !!captureEnabled;
    this.signature = '';
    const toggle = document.getElementById('screen-autorefresh');
    const btn = document.getElementById('btn-screen-capture');
    if (toggle) toggle.disabled = !this.captureEnabled;
    if (btn) btn.disabled = !this.captureEnabled;
    this.render();
  }

  // Called with every snapshot; rebuilds cards only when the setup changes.
  update(displays = []) {
    this.displays = displays || [];
    const sig = JSON.stringify(this.displays.map(d => [d.index, d.name, d.width, d.height, d.refresh_hz, d.scale_percent, d.primary]));
    if (sig !== this.signature) {
      this.signature = sig;
      this.render();
    }
  }

  placeholder() {
    if (!this.server) return '';
    if (this.captureEnabled) return 'Click Capture to see what is on this display';
    if (!this.server.controlEnabled) return 'Screen preview needs a PIN-paired server (not available in read-only mode)';
    return 'Screen preview is turned off or not supported on this server';
  }

  render() {
    const box = document.getElementById('displays-container');
    if (!box) return;
    if (!this.displays.length) {
      box.innerHTML = '<div class="sub-text">No displays reported by this host.</div>';
      return;
    }
    box.innerHTML = this.displays.map((d) => {
      const facts = [`${d.width} × ${d.height}`];
      if (d.refresh_hz) facts.push(`${d.refresh_hz} Hz`);
      if (d.scale_percent && d.scale_percent !== 100) facts.push(`${d.scale_percent}% scale`);
      // Windows device names look like \\.\DISPLAY1; show just DISPLAY1.
      const device = (d.device || '').split('\\').filter(Boolean).pop() || '';
      const img = this.images[d.index];
      const at = this.capturedAt[d.index];
      return `
        <div class="display-card">
          <div class="display-head">
            <div class="display-title">
              <span class="device-name" title="${esc(d.name)}">${esc(d.name)}</span>
              ${d.primary ? '<span class="conn-pill">Primary</span>' : ''}
            </div>
            <div class="sub-text">${facts.join(' · ')}${device ? ' · ' + esc(device) : ''}</div>
          </div>
          <div class="screen-preview ${img ? 'has-image' : ''}" data-index="${d.index}" style="aspect-ratio: ${d.width || 16} / ${d.height || 9}"
               ${img ? `role="button" tabindex="0" title="Open larger"` : ''}>
            ${img ? `<img src="${img}" alt="Screen of ${esc(d.name)}">` : `<span class="sub-text">${esc(this.placeholder())}</span>`}
            ${at ? `<span class="screen-time">${at.toLocaleTimeString()}</span>` : ''}
          </div>
        </div>`;
    }).join('');
  }

  // Auto-refresh tick: only while enabled, visible and on the dashboard.
  tick() {
    if (!this.autoRefresh || !this.captureEnabled || document.hidden) return;
    if (Layout.isHidden('displays') || this.app.currentView !== 'dashboard') return;
    this.captureAll();
  }

  async captureAll() {
    if (!this.server || !this.captureEnabled || this.busy || !this.displays.length) return;
    this.busy = true;
    const btn = document.getElementById('btn-screen-capture');
    if (btn) btn.textContent = 'Capturing…';
    const server = this.server;
    try {
      for (const d of this.displays) {
        const url = `${MonitorClient.baseURL(server)}/api/display/screenshot?index=${d.index}&width=1280`;
        const resp = await fetch(url, { headers: MonitorClient.authHeaders(server), cache: 'no-store' });
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({}));
          const err = new Error(body.message || `HTTP ${resp.status}`);
          err.code = body.error;
          throw err;
        }
        const blob = await resp.blob();
        if (this.server !== server) return; // switched servers meanwhile
        if (this.images[d.index]) URL.revokeObjectURL(this.images[d.index]);
        this.images[d.index] = URL.createObjectURL(blob);
        this.capturedAt[d.index] = new Date();
        if (this.viewing === d.index) this.openViewer(d.index);
      }
      this.render();
    } catch (e) {
      if (e.code === 'screen_disabled') {
        this.captureEnabled = false;
        this.autoRefresh = false;
        this.setServer(server, false);
      }
      this.app.handleControlError(server, e);
    } finally {
      this.busy = false;
      if (btn) btn.textContent = 'Capture';
    }
  }

  openViewer(index) {
    const d = this.displays.find(x => x.index === index);
    this.viewing = index;
    document.getElementById('screen-modal-title').textContent = d ? d.name : 'Display';
    document.getElementById('screen-modal-img').src = this.images[index];
    const at = this.capturedAt[index];
    document.getElementById('screen-modal-time').textContent = at
      ? `Captured ${at.toLocaleTimeString()}${this.autoRefresh ? ' · refreshing every 10 s' : ''}` : '';
    document.getElementById('screen-modal').classList.remove('hidden');
  }
}
