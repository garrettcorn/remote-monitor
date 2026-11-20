// Dashboard customization (show/hide sections) and the Control Panel
// button grid.

import { UI } from './ui.js?v=21';
import { MonitorClient } from './client.js?v=21';

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

// ------------------------------------------------------------ deck buttons

export const DECK_COLORS = {
  blue: 'var(--blue)', purple: 'var(--purple)', green: 'var(--green)',
  orange: 'var(--orange)', red: 'var(--red)', gray: 'var(--text-muted)',
};

const HEX_COLOR = /^#[0-9a-f]{6}$/i;

// CSS value for a button colour: a preset name or a custom "#rrggbb".
// Anything else falls back to blue, so stored values can't inject CSS.
function colorCss(c) {
  if (DECK_COLORS[c]) return DECK_COLORS[c];
  return HEX_COLOR.test(c || '') ? c : DECK_COLORS.blue;
}

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
  text: '<path d="M3 15l3.5-10h1L11 15M4.3 11.5h5.4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/><path d="M13 9.5c.6-.7 1.4-1 2.3-1 1.3 0 2.2.8 2.2 2V15m0-2.6c-.6.9-1.5 1.4-2.5 1.4-1.1 0-1.9-.6-1.9-1.5 0-1.1 1.1-1.6 4.4-1.6" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>',
  add: '<path d="M10 4v12M4 10h12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>',
};

// Media & system choices (one tap each).
const MEDIA_CHOICES = [
  { type: 'media', key: 'play_pause', label: 'Play / Pause' },
  { type: 'media', key: 'previous', label: 'Previous' },
  { type: 'media', key: 'next', label: 'Next' },
  { type: 'media', key: 'stop', label: 'Stop' },
  { type: 'media', key: 'volume_down', label: 'Volume −' },
  { type: 'media', key: 'volume_up', label: 'Volume +' },
  { type: 'media', key: 'volume_mute', label: 'Mute speakers' },
  { type: 'mic_mute', label: 'Mic mute' },
  { type: 'lock', label: 'Lock screen' },
];

// Common shortcuts. Items are [label, keys, macKeys?]. `os` limits a group
// to one host OS; `cmd` groups use Cmd instead of Ctrl on macOS hosts.
const SHORTCUTS = [
  { group: 'Windows', os: 'windows', items: [
    ['Show desktop', 'win+d'], ['Task Manager', 'ctrl+shift+esc'], ['Screenshot (Snipping Tool)', 'win+shift+s'],
    ['File Explorer', 'win+e'], ['Settings', 'win+i'], ['Run dialog', 'win+r'], ['Clipboard history', 'win+v'],
    ['Minimize all windows', 'win+m'], ['Snap window left', 'win+left'], ['Snap window right', 'win+right'],
    ['Maximize window', 'win+up'], ['Task view', 'win+tab'], ['Next virtual desktop', 'ctrl+win+right'],
    ['Previous virtual desktop', 'ctrl+win+left'], ['Switch app', 'alt+tab'], ['Close window', 'alt+f4'],
    ['Display / projection mode', 'win+p'],
  ] },
  { group: 'Xbox Game Bar', os: 'windows', items: [
    ['Open Game Bar', 'win+g'], ['Record last 30 seconds', 'win+alt+g'], ['Start / stop recording', 'win+alt+r'],
    ['Game screenshot', 'win+alt+printscreen'], ['Game Bar mic on/off', 'win+alt+m'],
  ] },
  { group: 'macOS', os: 'darwin', items: [
    ['Spotlight', 'win+space'], ['Mission Control', 'ctrl+up'], ['Screenshot area', 'win+shift+4'],
    ['Screenshot options', 'win+shift+5'], ['Force Quit', 'win+alt+esc'], ['Switch app', 'win+tab'],
    ['Hide app', 'win+h'], ['Quit app', 'win+q'],
  ] },
  { group: 'Editing', cmd: true, items: [
    ['Copy', 'ctrl+c'], ['Paste', 'ctrl+v'], ['Cut', 'ctrl+x'], ['Undo', 'ctrl+z'], ['Redo', 'ctrl+y', 'win+shift+z'],
    ['Select all', 'ctrl+a'], ['Save', 'ctrl+s'], ['Find', 'ctrl+f'], ['Print', 'ctrl+p'],
  ] },
  { group: 'Browser', cmd: true, items: [
    ['New tab', 'ctrl+t'], ['Close tab', 'ctrl+w'], ['Reopen closed tab', 'ctrl+shift+t'], ['Next tab', 'ctrl+tab', 'ctrl+tab'],
    ['Previous tab', 'ctrl+shift+tab', 'ctrl+shift+tab'], ['Reload page', 'f5', 'win+r'], ['Full screen', 'f11', 'ctrl+win+f'],
    ['Address bar', 'ctrl+l'], ['Reset zoom', 'ctrl+0'],
  ] },
  { group: 'Calls & chat', items: [
    ['Discord: toggle mute', 'ctrl+shift+m', 'win+shift+m'], ['Discord: toggle deafen', 'ctrl+shift+d', 'win+shift+d'],
    ['Teams: toggle mute', 'ctrl+shift+m', 'win+shift+m'], ['Teams: toggle camera', 'ctrl+shift+o', 'win+shift+o'],
    ['Zoom: toggle mute', 'alt+a', 'win+shift+a'], ['Zoom: toggle video', 'alt+v', 'win+shift+v'],
    ['Google Meet: toggle mic', 'ctrl+d', 'win+d'], ['Google Meet: toggle camera', 'ctrl+e', 'win+e'],
  ] },
];

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

const CATEGORY_OF = { app: 'app', launch: 'app', hotkey: 'shortcut', media: 'media', mic_mute: 'media', lock: 'media', text: 'text', url: 'url' };

function iconFor(btn) {
  if (btn.type === 'app' || btn.type === 'launch') {
    const ch = (btn.appName || btn.label || '?').trim().charAt(0).toUpperCase();
    return `<span class="deck-initial">${esc(ch)}</span>`;
  }
  const k = btn.type === 'media' ? btn.key : btn.type;
  return `<svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">${ICON[k] || ICON.hotkey}</svg>`;
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

// Human-readable shortcut for the host OS ("Ctrl+Shift+M", "Cmd+Shift+M").
function prettyKeys(s, os) {
  const names = os === 'darwin'
    ? { win: 'Cmd', alt: 'Option', ctrl: 'Ctrl' }
    : { win: 'Win', alt: 'Alt', ctrl: 'Ctrl' };
  const special = { esc: 'Esc', pageup: 'PgUp', pagedown: 'PgDn', printscreen: 'PrtSc', left: '←', right: '→', up: '↑', down: '↓' };
  return s.split('+').map(p => names[p] || special[p] || (p.length <= 1 ? p.toUpperCase() : p[0].toUpperCase() + p.slice(1))).join('+');
}

function shortcutsFor(os) {
  const out = [];
  for (const g of SHORTCUTS) {
    if (g.os && g.os !== os) continue;
    out.push({
      group: g.group,
      items: g.items.map(([label, keys, mac]) => ({
        label,
        keys: os === 'darwin' ? (mac || (g.cmd ? keys.replace(/\bctrl\b/, 'win') : keys)) : keys,
      })),
    });
  }
  return out;
}

export class Deck {
  constructor(app) {
    this.app = app;
    this.server = null;
    this.buttons = [];
    this.editing = false;
    this.editIndex = -1;
    this.hostOS = 'windows';
    this.live = { speakerMuted: false, micMuted: false };
    this.apps = null;       // installed apps on the server (cached per server)
    this.launchList = null; // custom apps from actions.json
    this.dragFrom = -1;
    this.draft = null;      // button being edited
    this.cat = 'app';
    this.labelTouched = false;
    this.bind();
  }

  storageKey() { return 'pc_monitor_deck_' + (this.server?.id || 'none'); }

  setServer(server) {
    if (this.server?.id !== server?.id) {
      this.apps = null;
      this.launchList = null;
    }
    this.server = server;
    this.buttons = load(this.storageKey(), null) || DEFAULT_BUTTONS.map(b => ({ ...b }));
    this.editing = false;
    this.render();
  }

  persist() { save(this.storageKey(), this.buttons); }

  // Light up mute / mic-mute buttons from the live audio state.
  setLiveState(snap) {
    if (snap?.host?.os) this.hostOS = snap.host.os;
    const out = snap?.audio?.outputs?.find(d => d.default);
    const inp = snap?.audio?.inputs?.find(d => d.default);
    const next = { speakerMuted: !!out?.muted, micMuted: !!inp?.muted };
    if (next.speakerMuted !== this.live.speakerMuted || next.micMuted !== this.live.micMuted) {
      this.live = next;
      this.applyLiveState();
    }
  }

  applyLiveState() {
    document.querySelectorAll('#deck-grid .deck-btn[data-live]').forEach((b) => {
      const on = b.dataset.live === 'speaker' ? this.live.speakerMuted : this.live.micMuted;
      b.classList.toggle('is-on', on);
      b.setAttribute('aria-pressed', String(on));
    });
  }

  render() {
    const grid = document.getElementById('deck-grid');
    const editBtn = document.getElementById('btn-deck-edit');
    if (!grid) return;
    if (editBtn) {
      editBtn.textContent = this.editing ? 'Done' : 'Edit';
      editBtn.setAttribute('aria-pressed', String(this.editing));
    }
    grid.classList.toggle('is-editing', this.editing);
    let html = this.buttons.map((b, i) => {
      const live = b.type === 'mic_mute' ? 'mic' : (b.type === 'media' && b.key === 'volume_mute' ? 'speaker' : '');
      const hint = b.type === 'hotkey' ? ` · ${prettyKeys(b.keys || '', this.hostOS)}`
        : b.type === 'url' ? ` · ${(() => { try { return new URL(b.url).host; } catch (e) { return ''; } })()}`
        : (b.type === 'app' || b.type === 'launch') && b.appName && b.appName !== b.label ? ` · ${b.appName}` : '';
      return `
      <button class="deck-btn" data-index="${i}" ${live ? `data-live="${live}"` : ''} style="--accent: ${colorCss(b.color)}"
              ${this.editing ? 'draggable="true"' : ''} title="${esc(b.label + hint)}">
        <span class="deck-icon">${iconFor(b)}</span>
        <span class="deck-label">${esc(b.label)}</span>
        ${this.editing ? '<span class="deck-edit-badge" aria-hidden="true">Edit</span>' : ''}
      </button>`;
    }).join('');
    if (this.editing) {
      html += `<button class="deck-btn deck-add" data-add="1"><span class="deck-icon"><svg viewBox="0 0 20 20" aria-hidden="true">${ICON.add}</svg></span><span class="deck-label">Add</span></button>`;
    }
    if (!this.buttons.length && !this.editing) {
      html = '<div class="sub-text">No buttons yet. Click Edit to add some.</div>';
    }
    grid.innerHTML = html;
    this.applyLiveState();
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

    // ---- editor
    document.getElementById('deck-cats')?.addEventListener('click', (e) => {
      const tile = e.target.closest('.deck-cat');
      if (tile) this.showCategory(tile.dataset.cat);
    });

    document.getElementById('deck-app-search')?.addEventListener('input', () => this.renderApps());
    document.getElementById('deck-app-list')?.addEventListener('click', (e) => {
      const item = e.target.closest('.pick-item');
      if (!item) return;
      this.choose({ type: item.dataset.type, id: item.dataset.id, appName: item.dataset.name }, item.dataset.name);
      this.renderApps();
    });

    document.getElementById('deck-shortcut-search')?.addEventListener('input', () => this.renderShortcuts());
    document.getElementById('deck-shortcut-list')?.addEventListener('click', (e) => {
      const item = e.target.closest('.pick-item');
      if (!item) return;
      this.choose({ type: 'hotkey', keys: item.dataset.keys }, item.dataset.label);
      this.showRecorded();
      this.renderShortcuts();
    });

    const keys = document.getElementById('deck-keys');
    keys?.addEventListener('keydown', (e) => {
      e.preventDefault();
      const { mods, key } = shortcutFromEvent(e);
      if (!key) {
        keys.value = mods.length ? prettyKeys(mods.join('+'), this.hostOS) + '+…' : '';
        return;
      }
      const combo = [...mods, key].join('+');
      this.choose({ type: 'hotkey', keys: combo }, prettyKeys(combo, this.hostOS));
      this.showRecorded();
      this.renderShortcuts();
    });

    document.getElementById('deck-media-list')?.addEventListener('click', (e) => {
      const chip = e.target.closest('.chip');
      if (!chip) return;
      const c = MEDIA_CHOICES[parseInt(chip.dataset.i, 10)];
      this.choose({ type: c.type, key: c.key }, c.label);
      this.renderMedia();
    });

    const text = document.getElementById('deck-text');
    text?.addEventListener('input', () => {
      document.getElementById('deck-text-count').textContent = String(text.value.length);
      this.choose({ type: 'text', text: text.value }, text.value.split('\n')[0]);
    });

    const url = document.getElementById('deck-url');
    url?.addEventListener('input', () => {
      let host = '';
      try { host = new URL(url.value.trim()).host.replace(/^www\./, ''); } catch (e) { /* incomplete */ }
      this.choose({ type: 'url', url: url.value.trim() }, host);
    });

    document.getElementById('deck-label')?.addEventListener('input', () => { this.labelTouched = true; });

    const colors = document.getElementById('deck-colors');
    const selectSwatch = (sw) => colors.querySelectorAll('.swatch').forEach(s => s.setAttribute('aria-checked', String(s === sw)));
    colors?.addEventListener('click', (e) => {
      const sw = e.target.closest('.swatch');
      if (sw) selectSwatch(sw); // the custom swatch's click also opens the colour picker
    });
    // Custom colour wheel: update its swatch live as the user drags in the picker.
    colors?.addEventListener('input', (e) => {
      if (e.target.id !== 'deck-color-custom') return;
      const sw = e.target.closest('.swatch');
      sw.style.setProperty('--accent', e.target.value);
      sw.classList.add('has-color');
      selectSwatch(sw);
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
    const action = { type: b.type, key: b.key, keys: b.keys, url: b.url, id: b.id, text: b.text };
    try {
      await MonitorClient.post(this.server, '/api/deck/press', action);
      el.classList.add('ok');
      setTimeout(() => el.classList.remove('ok'), 600);
    } catch (e) {
      this.app.handleControlError(this.server, e);
    }
  }

  // ---------------------------------------------------------------- editor

  openEditor(i) {
    this.editIndex = i;
    const b = i >= 0 ? { ...this.buttons[i] } : { type: 'app', color: 'blue' };
    this.draft = { ...b };
    this.labelTouched = i >= 0;
    document.getElementById('deck-modal-title').textContent = i >= 0 ? 'Edit button' : 'Add button';
    document.getElementById('btn-deck-delete').classList.toggle('hidden', i < 0);
    document.getElementById('deck-error').classList.add('hidden');
    document.getElementById('deck-label').value = b.label || '';
    document.getElementById('deck-app-search').value = '';
    document.getElementById('deck-shortcut-search').value = '';
    document.getElementById('deck-text').value = b.type === 'text' ? b.text || '' : '';
    document.getElementById('deck-text-count').textContent = String(document.getElementById('deck-text').value.length);
    document.getElementById('deck-url').value = b.type === 'url' ? b.url || '' : '';
    const current = b.color || 'blue';
    const custom = HEX_COLOR.test(current) ? current : '';
    document.getElementById('deck-colors').innerHTML = Object.entries(DECK_COLORS).map(([name, css]) =>
      `<button type="button" class="swatch" role="radio" aria-label="${name}" aria-checked="${name === current}" data-color="${name}" style="--accent: ${css}"></button>`).join('') +
      // Custom colour: a rainbow wheel until a colour is picked, then that colour.
      `<label class="swatch swatch-custom ${custom ? 'has-color' : ''}" role="radio" aria-label="Custom colour" aria-checked="${!!custom}"
              title="Custom colour" ${custom ? `style="--accent: ${custom}"` : ''}>
         <input id="deck-color-custom" type="color" value="${custom || '#0a84ff'}" aria-label="Pick a custom colour">
       </label>`;
    this.showRecorded();
    document.getElementById('deck-modal').classList.remove('hidden');
    this.showCategory(CATEGORY_OF[b.type] || 'app');
  }

  showCategory(cat) {
    this.cat = cat;
    document.querySelectorAll('#deck-cats .deck-cat').forEach(t => t.setAttribute('aria-selected', String(t.dataset.cat === cat)));
    document.querySelectorAll('#deck-form .deck-pane').forEach(p => p.classList.toggle('hidden', p.dataset.pane !== cat));
    if (cat === 'app') { this.loadApps(); setTimeout(() => document.getElementById('deck-app-search')?.focus(), 30); }
    if (cat === 'shortcut') { this.renderShortcuts(); setTimeout(() => document.getElementById('deck-shortcut-search')?.focus(), 30); }
    if (cat === 'media') this.renderMedia();
  }

  // Select an action; suggest a label until the user types their own.
  choose(fields, suggestedLabel) {
    this.draft = { color: this.draft?.color, ...fields };
    const label = document.getElementById('deck-label');
    if (!this.labelTouched && suggestedLabel) label.value = String(suggestedLabel).slice(0, 18);
    document.getElementById('deck-error').classList.add('hidden');
  }

  showRecorded() {
    const keys = document.getElementById('deck-keys');
    if (keys) keys.value = this.draft?.type === 'hotkey' && this.draft.keys ? prettyKeys(this.draft.keys, this.hostOS) : '';
  }

  async loadApps() {
    const list = document.getElementById('deck-app-list');
    const help = document.getElementById('deck-app-help');
    if (!this.apps) {
      list.innerHTML = '<div class="sub-text pick-empty">Loading apps from the host…</div>';
      try {
        const [apps, custom] = await Promise.all([
          fetch(`${MonitorClient.baseURL(this.server)}/api/deck/apps`, { headers: MonitorClient.authHeaders(this.server) }).then(r => r.ok ? r.json() : Promise.reject(r)),
          fetch(`${MonitorClient.baseURL(this.server)}/api/deck/actions`, { headers: MonitorClient.authHeaders(this.server) }).then(r => r.ok ? r.json() : { launch: [] }).catch(() => ({ launch: [] })),
        ]);
        this.apps = apps.apps || [];
        this.launchList = custom.launch || [];
        help.textContent = `${this.apps.length} apps found on ${this.server.name}.` +
          (custom.actions_file ? ` Apps that need arguments can be added to ${custom.actions_file}.` : '');
      } catch (e) {
        this.apps = null;
        list.innerHTML = '<div class="sub-text pick-empty">Couldn\'t load apps. This server may be running an older version.</div>';
        help.textContent = '';
        return;
      }
    }
    this.renderApps();
  }

  renderApps() {
    const list = document.getElementById('deck-app-list');
    if (!list || !this.apps) return;
    const q = document.getElementById('deck-app-search').value.trim().toLowerCase();
    const match = (n) => !q || n.toLowerCase().includes(q);
    const selected = (type, id) => this.draft?.type === type && this.draft.id === id;
    const item = (type, a) => `
      <button type="button" class="pick-item ${selected(type, a.id) ? 'is-selected' : ''}" role="option"
              aria-selected="${selected(type, a.id)}" data-type="${type}" data-id="${esc(a.id)}" data-name="${esc(a.name)}">
        <span class="pick-avatar">${esc(a.name.trim().charAt(0).toUpperCase())}</span>
        <span class="pick-name">${esc(a.name)}</span>
      </button>`;
    const custom = (this.launchList || []).filter(a => match(a.name));
    const apps = this.apps.filter(a => match(a.name)).slice(0, 250);
    let html = '';
    if (custom.length) html += '<div class="pick-group">Custom (actions.json)</div>' + custom.map(a => item('launch', a)).join('');
    if (apps.length) html += (custom.length ? '<div class="pick-group">Installed apps</div>' : '') + apps.map(a => item('app', a)).join('');
    list.innerHTML = html || `<div class="sub-text pick-empty">No apps match "${esc(q)}".</div>`;
  }

  renderShortcuts() {
    const list = document.getElementById('deck-shortcut-list');
    if (!list) return;
    const q = document.getElementById('deck-shortcut-search').value.trim().toLowerCase();
    let html = '';
    for (const g of shortcutsFor(this.hostOS)) {
      const items = g.items.filter(it => !q || it.label.toLowerCase().includes(q) || g.group.toLowerCase().includes(q) || it.keys.includes(q));
      if (!items.length) continue;
      html += `<div class="pick-group">${esc(g.group)}</div>` + items.map((it) => {
        const sel = this.draft?.type === 'hotkey' && this.draft.keys === it.keys;
        // "Discord: toggle mute" -> "Discord mute" for the button label.
        const short = it.label.replace(/:\s*(toggle\s+)?/, ' ');
        return `
        <button type="button" class="pick-item ${sel ? 'is-selected' : ''}" role="option" aria-selected="${sel}"
                data-keys="${esc(it.keys)}" data-label="${esc(short)}">
          <span class="pick-name">${esc(it.label)}</span>
          <kbd>${esc(prettyKeys(it.keys, this.hostOS))}</kbd>
        </button>`;
      }).join('');
    }
    list.innerHTML = html || `<div class="sub-text pick-empty">No shortcuts match "${esc(q)}". Record your own below.</div>`;
  }

  renderMedia() {
    const list = document.getElementById('deck-media-list');
    if (!list) return;
    list.innerHTML = MEDIA_CHOICES.map((c, i) => {
      const sel = this.draft?.type === c.type && (c.type !== 'media' || this.draft.key === c.key);
      return `<button type="button" class="chip ${sel ? 'is-selected' : ''}" role="option" aria-selected="${sel}" data-i="${i}">
        <span class="chip-icon">${iconFor({ type: c.type, key: c.key })}</span>${esc(c.label)}</button>`;
    }).join('');
  }

  selectedColor() {
    const sw = document.querySelector('#deck-colors .swatch[aria-checked="true"]');
    if (sw?.classList.contains('swatch-custom')) {
      const v = document.getElementById('deck-color-custom').value;
      return HEX_COLOR.test(v) ? v.toLowerCase() : 'blue';
    }
    return sw?.dataset.color || 'blue';
  }

  saveEditor() {
    const err = document.getElementById('deck-error');
    const fail = (m) => { err.textContent = m; err.classList.remove('hidden'); };
    const d = this.draft || {};
    const cat = this.cat;
    const b = {
      type: d.type,
      label: document.getElementById('deck-label').value.trim(),
      color: this.selectedColor(),
    };
    if (CATEGORY_OF[d.type] !== cat) {
      return fail({
        app: 'Pick an app from the list.', shortcut: 'Pick a shortcut or record your own.',
        media: 'Pick an action.', text: 'Type the text to send.', url: 'Enter a web address.',
      }[cat]);
    }
    switch (d.type) {
      case 'media': b.key = d.key; break;
      case 'hotkey':
        if (!d.keys) return fail('Pick a shortcut or record your own.');
        b.keys = d.keys; break;
      case 'app':
      case 'launch':
        if (!d.id) return fail('Pick an app from the list.');
        b.id = d.id; b.appName = d.appName; break;
      case 'text':
        if (!d.text) return fail('Type the text to send.');
        b.text = d.text; break;
      case 'url':
        if (!/^https?:\/\/[^/\s]+/i.test(d.url || '')) return fail('Enter a web address starting with http:// or https://');
        b.url = d.url; break;
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
    this.inputEnabled = false;
    this.displays = [];
    this.signature = '';
    this.images = {};     // display index -> object URL
    this.capturedAt = {}; // display index -> Date
    this.busy = false;
    this.viewing = -1;    // index shown in the full-size viewer
    this.controlling = false;
    this.inputQueue = [];
    this.sending = false;
    this.wheelAccum = 0;
    this.lastMoveAt = 0;
    this.liveTimer = null;
    this.fullscreen = false;
    this.keysLocked = false;
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
      const saveBtn = e.target.closest('[data-save]');
      if (saveBtn) {
        e.stopPropagation();
        this.saveCapture(parseInt(saveBtn.dataset.save, 10));
        return;
      }
      const prev = e.target.closest('.screen-preview[data-index]');
      if (prev && this.images[prev.dataset.index]) this.openViewer(parseInt(prev.dataset.index, 10));
    });
    document.querySelectorAll('#screen-modal .modal-close-btn').forEach(b => b.addEventListener('click', () => this.closeViewer()));

    // Viewer toolbar
    document.getElementById('btn-screen-save')?.addEventListener('click', () => this.saveCapture(this.viewing));
    document.getElementById('btn-screen-refresh')?.addEventListener('click', () => this.refreshViewing());
    document.getElementById('screen-control')?.addEventListener('change', (e) => this.setControlling(e.target.checked));
    document.getElementById('btn-screen-fullscreen')?.addEventListener('click', () => this.setFullscreen(!this.fullscreen));
    // Leaving browser full screen (Esc, F11, swipe) ends our full-screen mode.
    const fsChange = () => {
      if (this.fullscreen && !(document.fullscreenElement || document.webkitFullscreenElement)) this.setFullscreen(false);
      else { this.updateKeyLock(); this.updateHint(); }
    };
    document.addEventListener('fullscreenchange', fsChange);
    document.addEventListener('webkitfullscreenchange', fsChange);
    document.querySelectorAll('#screen-control-bar [data-send-keys]').forEach(b =>
      b.addEventListener('click', () => this.queueInput({ type: 'key', keys: b.dataset.sendKeys })));

    // Remote control: pointer + wheel on the image, keys anywhere in the viewer.
    const img = document.getElementById('screen-modal-img');
    if (img) {
      img.addEventListener('pointerdown', (e) => this.onPointer(e, 'down'));
      img.addEventListener('pointerup', (e) => this.onPointer(e, 'up'));
      img.addEventListener('pointermove', (e) => this.onPointer(e, 'move'));
      img.addEventListener('contextmenu', (e) => { if (this.controlling) e.preventDefault(); });
      img.addEventListener('wheel', (e) => this.onWheel(e), { passive: false });
      img.addEventListener('dblclick', (e) => { if (this.controlling) e.preventDefault(); });
    }
    // Capture phase so the dashboard's own shortcuts never see forwarded keys.
    window.addEventListener('keydown', (e) => this.onKey(e), true);
    const typeBox = document.getElementById('screen-type');
    typeBox?.addEventListener('input', () => {
      // Soft keyboards (no usable keydown) land here; desktop keys are sent by onKey.
      if (typeBox.value) this.queueInput({ type: 'text', text: typeBox.value });
      typeBox.value = '';
    });
    document.addEventListener('visibilitychange', () => { if (!document.hidden) this.scheduleLive(0); });
  }

  clearImages() {
    Object.values(this.images).forEach(u => URL.revokeObjectURL(u));
    this.images = {};
    this.capturedAt = {};
  }

  setServer(server, captureEnabled, inputEnabled) {
    if (this.server?.id !== server?.id) {
      this.clearImages();
      this.closeViewer();
    }
    this.server = server;
    this.captureEnabled = !!captureEnabled;
    this.inputEnabled = !!captureEnabled && !!inputEnabled;
    if (!this.inputEnabled) this.setControlling(false);
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
    if (this.captureEnabled) {
      return this.inputEnabled
        ? 'Click Capture to see this display, then open it to save or control it'
        : 'Click Capture to see what is on this display';
    }
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
      const title = this.inputEnabled ? 'Open larger, save or control' : 'Open larger or save';
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
               ${img ? `role="button" tabindex="0" title="${title}"` : ''}>
            ${img ? `<img src="${img}" alt="Screen of ${esc(d.name)}">` : `<span class="sub-text">${esc(this.placeholder())}</span>`}
            ${img ? `<button class="screen-save-btn" data-save="${d.index}" title="Save a full-resolution capture">Save</button>` : ''}
            ${at ? `<span class="screen-time">${at.toLocaleTimeString()}</span>` : ''}
          </div>
        </div>`;
    }).join('');
  }

  // Auto-refresh tick: only while enabled, visible and on the dashboard.
  tick() {
    if (!this.autoRefresh || !this.captureEnabled || document.hidden || this.controlling) return;
    if (Layout.isHidden('displays') || this.app.currentView !== 'dashboard') return;
    this.captureAll();
  }

  // Fetch one JPEG; throws Error with .code on failure.
  async fetchCapture(server, index, width) {
    const url = `${MonitorClient.baseURL(server)}/api/display/screenshot?index=${index}&width=${width}`;
    const resp = await fetch(url, { headers: MonitorClient.authHeaders(server), cache: 'no-store' });
    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}));
      const err = new Error(body.message || `HTTP ${resp.status}`);
      err.code = body.error;
      throw err;
    }
    return resp.blob();
  }

  setImage(index, blob) {
    if (this.images[index]) URL.revokeObjectURL(this.images[index]);
    this.images[index] = URL.createObjectURL(blob);
    this.capturedAt[index] = new Date();
  }

  captureFailed(server, e) {
    if (e.code === 'screen_disabled') {
      this.captureEnabled = false;
      this.autoRefresh = false;
      this.setServer(server, false, false);
    }
    this.app.handleControlError(server, e);
  }

  async captureAll() {
    if (!this.server || !this.captureEnabled || this.busy || !this.displays.length) return;
    this.busy = true;
    const btn = document.getElementById('btn-screen-capture');
    if (btn) btn.textContent = 'Capturing…';
    const server = this.server;
    try {
      for (const d of this.displays) {
        const blob = await this.fetchCapture(server, d.index, 1280);
        if (this.server !== server) return; // switched servers meanwhile
        this.setImage(d.index, blob);
        if (this.viewing === d.index) this.showViewerImage();
      }
      this.render();
    } catch (e) {
      this.captureFailed(server, e);
    } finally {
      this.busy = false;
      if (btn) btn.textContent = 'Capture';
    }
  }

  // Capture just the display in the viewer (Refresh button, live control).
  async refreshViewing() {
    const index = this.viewing;
    if (index < 0 || !this.server || !this.captureEnabled || this.busy) return false;
    this.busy = true;
    const server = this.server;
    const d = this.displays.find(x => x.index === index);
    // Live frames are sized for the viewer (sharper in full screen, up to
    // 2560 px); the card previews stay at 1280.
    let width = Math.min(d?.width || 1600, 1600);
    if (this.fullscreen) {
      const stage = document.getElementById('screen-stage');
      const want = Math.round((stage?.clientWidth || 1600) * (window.devicePixelRatio || 1));
      width = Math.min(d?.width || 2560, Math.max(1600, want), 2560);
    }
    try {
      const blob = await this.fetchCapture(server, index, width);
      if (this.server !== server || this.viewing !== index) return false;
      this.setImage(index, blob);
      this.showViewerImage();
      return true;
    } catch (e) {
      this.setControlling(false);
      this.captureFailed(server, e);
      return false;
    } finally {
      this.busy = false;
    }
  }

  // Download a fresh capture at the display's native resolution.
  async saveCapture(index) {
    if (index < 0 || !this.server || !this.captureEnabled) return;
    const server = this.server;
    const d = this.displays.find(x => x.index === index);
    try {
      const blob = await this.fetchCapture(server, index, d?.width || 3840);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = captureFilename(server.name, d?.name || `display${index + 1}`, new Date());
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 10000);
      this.app.showToast('Capture saved');
    } catch (e) {
      this.captureFailed(server, e);
    }
  }

  openViewer(index) {
    const d = this.displays.find(x => x.index === index);
    this.viewing = index;
    document.getElementById('screen-modal-title').textContent = d ? d.name : 'Display';
    const label = document.getElementById('screen-control-label');
    if (label) label.classList.toggle('hidden', !this.inputEnabled);
    const ctl = document.getElementById('screen-control');
    if (ctl) ctl.checked = this.controlling;
    this.showViewerImage();
    document.getElementById('screen-modal').classList.remove('hidden');
  }

  // Full screen: the viewer fills the display (Fullscreen API, or a fixed
  // overlay where the API isn't available, e.g. iPhone Safari). While
  // controlling in real full screen, Chromium's Keyboard Lock sends Esc,
  // Alt+Tab, Ctrl+W and friends to the host (hold Esc to leave).
  setFullscreen(on) {
    on = !!on && this.viewing >= 0;
    if (on === this.fullscreen) return;
    this.fullscreen = on;
    const card = document.querySelector('#screen-modal .screen-card');
    card?.classList.toggle('is-fullscreen', on);
    const btn = document.getElementById('btn-screen-fullscreen');
    if (btn) btn.textContent = on ? 'Exit full screen' : 'Full screen';
    // Never wait on these: some embedders leave the promise pending. The
    // CSS layout works either way; key lock follows fullscreenchange.
    const fsEl = document.fullscreenElement || document.webkitFullscreenElement;
    if (on && card && !fsEl) {
      const req = card.requestFullscreen || card.webkitRequestFullscreen;
      try { Promise.resolve(req?.call(card, { navigationUI: 'hide' })).catch(() => {}); } catch (e) { /* fixed overlay fallback */ }
    } else if (!on && fsEl) {
      const exit = document.exitFullscreen || document.webkitExitFullscreen;
      try { Promise.resolve(exit?.call(document)).catch(() => {}); } catch (e) { /* already gone */ }
    }
    this.updateKeyLock();
    this.updateHint();
    // A sharper frame for the bigger view.
    if (on) {
      if (this.controlling) this.scheduleLive(0);
      else this.refreshViewing();
    }
  }

  updateKeyLock() {
    const want = this.controlling && this.fullscreen && !!(document.fullscreenElement || document.webkitFullscreenElement);
    if (want === this.keysLocked || !navigator.keyboard?.lock) return;
    this.keysLocked = want;
    if (want) navigator.keyboard.lock().catch(() => { this.keysLocked = false; this.updateHint(); });
    else navigator.keyboard.unlock();
  }

  updateHint() {
    const hint = document.getElementById('screen-control-hint');
    if (!hint) return;
    hint.textContent = this.keysLocked
      ? 'Clicks, scrolling and all keys (including Esc and Alt+Tab) go to the host. Hold Esc to leave full screen.'
      : 'Clicks, scrolling and keys go to the host while Remote control is on. Some browser shortcuts (like Ctrl+W) can\'t be captured; use the buttons above, or Full screen in Chrome or Edge.';
  }

  closeViewer() {
    this.setFullscreen(false);
    this.setControlling(false);
    this.viewing = -1;
    document.getElementById('screen-modal')?.classList.add('hidden');
  }

  showViewerImage() {
    const index = this.viewing;
    if (index < 0) return;
    const img = document.getElementById('screen-modal-img');
    if (img && this.images[index]) img.src = this.images[index];
    const at = this.capturedAt[index];
    let note = at ? `Captured ${at.toLocaleTimeString()}` : '';
    if (this.controlling) note = at ? `Live · ${at.toLocaleTimeString()}` : 'Live';
    else if (at && this.autoRefresh) note += ' · refreshing every 10 s';
    document.getElementById('screen-modal-time').textContent = note;
  }

  // ------------------------------------------------------- remote control

  setControlling(on) {
    on = !!on && this.inputEnabled && this.viewing >= 0;
    this.controlling = on;
    const ctl = document.getElementById('screen-control');
    if (ctl) ctl.checked = on;
    document.getElementById('screen-stage')?.classList.toggle('is-controlling', on);
    document.getElementById('screen-control-bar')?.classList.toggle('hidden', !on);
    document.getElementById('screen-control-hint')?.classList.toggle('hidden', !on);
    this.updateKeyLock();
    this.updateHint();
    if (!on) {
      this.inputQueue = [];
      clearTimeout(this.liveTimer);
      this.liveTimer = null;
    } else {
      document.getElementById('screen-stage')?.focus?.();
      this.scheduleLive(0);
    }
    this.showViewerImage();
  }

  // Live view while controlling: a new frame as soon as the last one lands
  // (a little sooner after input), paused while the tab is hidden.
  scheduleLive(delay) {
    if (!this.controlling) return;
    clearTimeout(this.liveTimer);
    this.liveTimer = setTimeout(async () => {
      this.liveTimer = null;
      if (!this.controlling || document.hidden) return;
      if (this.busy) { this.scheduleLive(100); return; }
      await this.refreshViewing();
      this.scheduleLive(LIVE_FRAME_MS);
    }, delay);
  }

  // Map a pointer event to 0..1 coordinates on the screen image, allowing
  // for object-fit: contain letterboxing.
  pointToScreen(e) {
    const img = e.currentTarget;
    const r = img.getBoundingClientRect();
    const nw = img.naturalWidth || r.width;
    const nh = img.naturalHeight || r.height;
    const scale = Math.min(r.width / nw, r.height / nh);
    const w = nw * scale, h = nh * scale;
    const x = (e.clientX - r.left - (r.width - w) / 2) / w;
    const y = (e.clientY - r.top - (r.height - h) / 2) / h;
    if (x < 0 || x > 1 || y < 0 || y > 1) return null;
    return { x, y };
  }

  onPointer(e, type) {
    if (!this.controlling) return;
    const p = this.pointToScreen(e);
    if (!p) return;
    e.preventDefault();
    if (type === 'move') {
      const now = performance.now();
      if (now - this.lastMoveAt < MOVE_INTERVAL_MS) return;
      this.lastMoveAt = now;
    } else if (type === 'down') {
      e.currentTarget.setPointerCapture?.(e.pointerId);
    }
    const button = ['left', 'middle', 'right'][e.button] || 'left';
    this.queueInput({ index: this.viewing, type, x: p.x, y: p.y, button });
  }

  onWheel(e) {
    if (!this.controlling) return;
    const p = this.pointToScreen(e);
    if (!p) return;
    e.preventDefault();
    // ~100 px (or 3 lines) per wheel notch.
    this.wheelAccum += e.deltaMode === 1 ? e.deltaY * 33 : e.deltaMode === 2 ? e.deltaY * 800 : e.deltaY;
    const notches = Math.trunc(this.wheelAccum / 100);
    if (!notches) return;
    this.wheelAccum -= notches * 100;
    this.queueInput({ index: this.viewing, type: 'scroll', x: p.x, y: p.y, dy: notches });
  }

  onKey(e) {
    // Viewer shortcuts while not controlling: F toggles full screen, Esc
    // leaves the fallback (non-API) full screen.
    if (!this.controlling && this.viewing >= 0 && !e.ctrlKey && !e.altKey && !e.metaKey
        && !isTextField(e.target)) {
      if (e.key === 'f' || e.key === 'F') {
        e.preventDefault();
        this.setFullscreen(!this.fullscreen);
        return;
      }
      if (e.key === 'Escape' && this.fullscreen) {
        e.preventDefault();
        e.stopPropagation();
        this.setFullscreen(false);
        return;
      }
    }
    if (!this.controlling || e.isComposing) return;
    if (['Control', 'Shift', 'Alt', 'Meta', 'AltGraph', 'CapsLock', 'Unidentified', 'Process', 'Dead'].includes(e.key)) return;
    const typeBox = document.getElementById('screen-type');
    // Plain printable characters go as text so layouts and symbols survive.
    if (e.key.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) {
      e.preventDefault();
      e.stopPropagation();
      const last = this.inputQueue[this.inputQueue.length - 1];
      if (last && last.type === 'text' && last.text.length < 200) last.text += e.key;
      else this.queueInput({ type: 'text', text: e.key });
      return;
    }
    const { mods, key } = shortcutFromEvent(e);
    if (!key) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.target === typeBox) typeBox.value = '';
    this.queueInput({ type: 'key', keys: [...mods, key].join('+') });
  }

  // Inputs are sent one at a time, in order. Consecutive moves collapse to
  // the latest so a slow link never builds a backlog.
  queueInput(ev) {
    if (!this.controlling || !this.server) return;
    const last = this.inputQueue[this.inputQueue.length - 1];
    if (ev.type === 'move' && last && last.type === 'move') this.inputQueue[this.inputQueue.length - 1] = ev;
    else this.inputQueue.push(ev);
    this.pumpInput();
  }

  async pumpInput() {
    if (this.sending) return;
    this.sending = true;
    const server = this.server;
    try {
      while (this.inputQueue.length && this.controlling && this.server === server) {
        const ev = this.inputQueue.shift();
        try {
          await MonitorClient.post(server, '/api/display/input', ev);
        } catch (e) {
          // The host refused this one event (e.g. a key it can't press):
          // say so and keep controlling. Anything else ends control below.
          if (e.code !== 'input_error') throw e;
          this.app.showToast(`Couldn't send that: ${e.message}`);
        }
        if (ev.type !== 'move') this.scheduleLive(INPUT_FRAME_MS);
      }
    } catch (e) {
      this.setControlling(false);
      if (e.code === 'input_disabled') {
        this.inputEnabled = false;
        document.getElementById('screen-control-label')?.classList.add('hidden');
        this.app.showToast('Remote control is disabled on this server');
      } else {
        this.app.handleControlError(server, e);
      }
    } finally {
      this.sending = false;
    }
  }
}

const LIVE_FRAME_MS = 250;   // pause between live frames while controlling
const INPUT_FRAME_MS = 120;  // next frame this soon after a click or key
const MOVE_INTERVAL_MS = 50; // at most ~20 pointer moves per second

// "Office PC-DELL U2720Q-2026-11-20-153012.jpg"
function captureFilename(serverName, displayName, at) {
  const clean = (s) => String(s || '').replace(/[^A-Za-z0-9 ._-]+/g, ' ').trim().replace(/\s+/g, ' ').slice(0, 60);
  const pad = (n) => String(n).padStart(2, '0');
  const stamp = `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}-${pad(at.getHours())}${pad(at.getMinutes())}${pad(at.getSeconds())}`;
  return [clean(serverName), clean(displayName), stamp].filter(Boolean).join('-') + '.jpg';
}

// Typing targets where single-letter shortcuts must not fire.
function isTextField(el) {
  if (!el) return false;
  if (el.isContentEditable || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT') return true;
  return el.tagName === 'INPUT' && !['checkbox', 'radio', 'button', 'range', 'color'].includes(el.type);
}
