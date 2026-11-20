// First-run "Get started" guide: download the server for this visitor's
// system (from the latest GitHub release) and pair it with this dashboard.

const REPO = 'garrettcorn/remote-monitor';
const RELEASES_URL = `https://github.com/${REPO}/releases/latest`;
const GUIDE_URL = `https://github.com/${REPO}#-quick-start-guide`;
const CACHE_KEY = 'pc_monitor_latest_release';
const CACHE_MS = 60 * 60 * 1000; // GitHub allows 60 unauthenticated API calls/hour

const esc = (v) => String(v ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

// Downloads offered, by system. `match` picks the asset from the release.
export const DOWNLOADS = [
  { id: 'win-x64', os: 'windows', label: 'Windows', detail: 'installer, x64', match: /_windows_amd64_setup\.exe$/ },
  { id: 'win-arm', os: 'windows', label: 'Windows on ARM', detail: 'installer, ARM64', match: /_windows_arm64_setup\.exe$/ },
  { id: 'mac-arm', os: 'mac', label: 'Mac (Apple Silicon)', detail: 'M1 and later', match: /_darwin_arm64\.tar\.gz$/ },
  { id: 'mac-x64', os: 'mac', label: 'Mac (Intel)', detail: '', match: /_darwin_amd64\.tar\.gz$/ },
  { id: 'lin-x64', os: 'linux', label: 'Linux', detail: 'AppImage, x86_64', match: /-x86_64\.AppImage$/ },
  { id: 'lin-arm', os: 'linux', label: 'Linux on ARM', detail: 'AppImage, aarch64', match: /-aarch64\.AppImage$/ },
  { id: 'deb-x64', os: 'linux', label: 'Debian / Ubuntu', detail: '.deb, amd64', match: /_amd64\.deb$/ },
  { id: 'rpm-x64', os: 'linux', label: 'Fedora / RHEL', detail: '.rpm, x86_64', match: /\.x86_64\.rpm$/ },
];

const HINTS = {
  windows: 'Run the installer. If SmartScreen says “Windows protected your PC”, click <strong>More info → Run anyway</strong> (the app isn’t code-signed).',
  mac: `Unpack it and run <code>pc_monitor</code> from Terminal. macOS blocks unsigned downloads at first; the <a href="${GUIDE_URL}" target="_blank" rel="noopener">install guide</a> has a one-line Terminal command that avoids this.`,
  linux: 'Make it executable (<code>chmod +x pc_monitor-*.AppImage</code>) and double-click it.',
  mobile: 'Install it on the <strong>computer</strong> you want to monitor (not this phone). Open this page there, or use one of these downloads.',
  unknown: 'Pick the download for the computer you want to monitor.',
};

// Best guess at the visitor's system; architecture only where the browser
// says (Chromium's client hints), otherwise the common default.
export async function detectPlatform(nav = navigator) {
  const ua = nav.userAgent || '';
  const platform = (nav.userAgentData?.platform || nav.platform || '').toLowerCase();
  let os = 'unknown';
  if (/android|iphone|ipad|ipod/i.test(ua) || nav.userAgentData?.mobile) os = 'mobile';
  else if (platform.includes('win') || /windows/i.test(ua)) os = 'windows';
  else if (platform.includes('mac') || /mac os x/i.test(ua)) os = 'mac';
  else if (platform.includes('linux') || /linux|x11|cros/i.test(ua)) os = 'linux';
  let arm = os === 'mac'; // most Macs in use are Apple Silicon; Safari can't tell
  try {
    const hints = await nav.userAgentData?.getHighEntropyValues?.(['architecture']);
    if (hints?.architecture) arm = hints.architecture === 'arm';
  } catch (e) { /* not available */ }
  return { os, arm };
}

// The download to feature for a platform, e.g. 'win-arm' for Windows on ARM.
export function primaryDownload({ os, arm }) {
  return { windows: arm ? 'win-arm' : 'win-x64', mac: arm ? 'mac-arm' : 'mac-x64', linux: arm ? 'lin-arm' : 'lin-x64' }[os] || null;
}

async function latestRelease() {
  try {
    const c = JSON.parse(sessionStorage.getItem(CACHE_KEY) || 'null');
    if (c && Date.now() - c.t < CACHE_MS) return c.data;
  } catch (e) { /* no cache */ }
  const resp = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: 'application/vnd.github+json' } });
  if (!resp.ok) throw new Error(`GitHub ${resp.status}`);
  const r = await resp.json();
  const data = {
    tag: r.tag_name,
    url: r.html_url,
    assets: (r.assets || []).map((a) => ({ name: a.name, url: a.browser_download_url, size: a.size })),
  };
  try { sessionStorage.setItem(CACHE_KEY, JSON.stringify({ t: Date.now(), data })); } catch (e) { /* ignore */ }
  return data;
}

const fmtSize = (b) => (b ? `${(b / 1048576).toFixed(b < 10485760 ? 1 : 0)} MB` : '');

export const GetStarted = {
  render(container) {
    container.innerHTML = `
      <section class="metric-card get-started">
        <h2 class="gs-title">Get started</h2>
        <p class="gs-lead">PC Monitor shows live stats from your computers and lets you control them from this page. Install the small server app on each computer you want to monitor, then pair it here. It takes about a minute.</p>
        <ol class="gs-steps">
          <li>
            <div class="gs-num">1</div>
            <div class="gs-body">
              <h3>Install PC Monitor on the computer</h3>
              <div id="gs-download" class="gs-download"><span class="sub-text">Finding the latest version…</span></div>
              <p id="gs-hint" class="sub-text gs-hint"></p>
            </div>
          </li>
          <li>
            <div class="gs-num">2</div>
            <div class="gs-body">
              <h3>Open it and find the pairing QR code</h3>
              <p class="sub-text"><strong>Windows and Linux:</strong> start <strong>PC Monitor</strong> from the Start menu or app launcher. A tray icon appears; choose <strong>Open settings…</strong> to see the QR code, pairing link and PIN.<br>
              <strong>Mac or terminal:</strong> run <code>pc_monitor</code>. It prints the pairing link, QR code and PIN.</p>
            </div>
          </li>
          <li>
            <div class="gs-num">3</div>
            <div class="gs-body">
              <h3>Pair this browser</h3>
              <p class="sub-text">Scan the QR code with this device’s camera, or click <strong>Pair Server</strong> and paste the pairing link. Then type the <strong>4-digit PIN</strong> shown on that computer. This browser remembers it from then on.</p>
              <button type="button" class="btn btn-primary btn-sm" data-gs-pair>Pair Server</button>
            </div>
          </li>
        </ol>
        <p class="sub-text gs-foot">Same network or VPN needed (the dashboard connects to port 9120). <a href="${GUIDE_URL}" target="_blank" rel="noopener">Full install and pairing guide</a></p>
      </section>`;
    container.querySelector('[data-gs-pair]')?.addEventListener('click', () => document.getElementById('btn-pair-modal')?.click());
    this.fillDownloads(container);
  },

  async fillDownloads(container) {
    const box = container.querySelector('#gs-download');
    const hint = container.querySelector('#gs-hint');
    const plat = await detectPlatform();
    if (hint) hint.innerHTML = HINTS[plat.os] || HINTS.unknown;
    let rel = null;
    try {
      rel = await latestRelease();
    } catch (e) { /* offline or rate-limited: link to the release page */ }
    if (!box?.isConnected) return;
    const find = (d) => rel?.assets.find((a) => d.match.test(a.name));
    const primaryId = primaryDownload(plat);
    const primary = DOWNLOADS.find((d) => d.id === primaryId);
    const pAsset = primary && find(primary);
    let html = '';
    if (primary) {
      const href = pAsset ? pAsset.url : RELEASES_URL;
      html += `<a class="btn btn-primary gs-primary" href="${esc(href)}" rel="noopener">
        Download for ${esc(primary.label)}${pAsset ? ` <span class="gs-meta">${esc(rel.tag)} · ${esc(fmtSize(pAsset.size))}</span>` : ''}</a>`;
    }
    const others = DOWNLOADS.filter((d) => d.id !== primaryId).map((d) => {
      const a = find(d);
      return `<li><a href="${esc(a ? a.url : RELEASES_URL)}" rel="noopener">${esc(d.label)}</a>${d.detail ? ` <span class="sub-text">${esc(d.detail)}</span>` : ''}</li>`;
    }).join('');
    const open = !primary ? ' open' : '';
    html += `<details class="gs-others"${open}><summary>${primary ? 'Other systems' : 'Downloads'}</summary>
      <ul>${others}</ul>
      <a class="sub-text" href="${esc(rel?.url || RELEASES_URL)}" target="_blank" rel="noopener">All files${rel ? ` in ${esc(rel.tag)}` : ''} on GitHub</a>
    </details>`;
    box.innerHTML = html;
  },
};
