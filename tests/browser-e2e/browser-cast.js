// End-to-end check for the /browser page: a real Chrome window driven through
// the browser-agent extension, watched from a second browser that plays the
// iPhone.
//
// Prerequisites (see README.md):
//   * spike browser-agent daemon on DAEMON (env BROWSER_AGENT_PORT on the app)
//   * ai-critic dev server on APP, started with --quick-test (no auth)
//   * static content server on WWW
//
// Run: playwright-debug run browser-cast.js
const BUILD_FALLBACK =
  '/Users/xhd2015/.wrk/worktrees/ai-master-2026-10-27/external/browser-agent/Chrome-Ext-Browser-Agent/build';
const DAEMON = process.env.DAEMON || 'http://127.0.0.1:43799';
const APP = process.env.APP || 'http://127.0.0.1:18555';
const WWW = process.env.WWW || 'http://127.0.0.1:18099';
const SESSION = process.env.SESSION || 'ai-critic';
const PROFILE = process.env.PROFILE || '/tmp/bw-e2e/profile-cast';

const failures = [];
const rawLog = [];
function check(ok, label, extra) {
  console.log(`${ok ? 'PASS' : 'FAIL'} ${label}${extra ? ' :: ' + extra : ''}`);
  if (!ok) failures.push(label + (extra ? ' :: ' + extra : ''));
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function daemon(path, init) {
  const res = await fetch(DAEMON + path, init);
  const text = await res.text();
  try { return JSON.parse(text); } catch { return { text }; }
}
async function status() {
  const res = await fetch(APP + '/api/browser/status');
  return res.json();
}
async function connected() {
  const rows = await daemon('/v1/sessions');
  const list = Array.isArray(rows) ? rows : (rows.sessions || []);
  const row = list.find((x) => x.session_id === SESSION);
  return !!(row && row.extension && row.extension.connected);
}
async function waitFor(label, fn, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try { const v = await fn(); if (v) return v; } catch { /* keep polling */ }
    await sleep(300);
  }
  throw new Error(`timeout: ${label}`);
}

// The session page derives the unpacked extension id from the session's
// extension_install_path (Chrome hashes the absolute directory path). Loading a
// different directory means the page's chrome.runtime.connect(<id>) fails with
// "Receiving end does not exist" and the extension never reports connected, even
// though its service worker is alive. So resolve the directory from the daemon
// unless the caller pinned EXT_BUILD.
async function resolveExtensionDir() {
  if (process.env.EXT_BUILD) return process.env.EXT_BUILD;
  // Creating an existing session returns 409; either way the row now exists.
  await daemon('/v1/sessions', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: SESSION }),
  }).catch(() => {});
  const rows = await daemon('/v1/sessions');
  const list = Array.isArray(rows) ? rows : (rows.sessions || []);
  const row = list.find((x) => x.session_id === SESSION) || {};
  return row.extension_install_path || BUILD_FALLBACK;
}
const BUILD = await resolveExtensionDir();
console.log('extension dir:', BUILD);

// ---------------------------------------------------------------- controller (B)
const viewerCtx = await chromium.launchPersistentContext(PROFILE, {
  headless: false,
  viewport: { width: 900, height: 1100 },
  args: ['--no-first-run', '--no-default-browser-check'],
});
const viewer = viewerCtx.pages()[0] || (await viewerCtx.newPage());

// ------------------------------------------------------------------- target (A)
const castCtx = await chromium.launchPersistentContext(PROFILE + '-chrome', {
  headless: false,
  viewport: null,
  args: [
    `--disable-extensions-except=${BUILD}`,
    `--load-extension=${BUILD}`,
    '--no-first-run',
    '--no-default-browser-check',
  ],
});

async function frameSignature() {
  return viewer.evaluate(() => {
    const img = document.querySelector('img.bw-frame');
    if (!img || !img.naturalWidth) return '';
    // A tiny canvas sample is enough to tell frames apart.
    const c = document.createElement('canvas');
    c.width = 8; c.height = 8;
    c.getContext('2d').drawImage(img, 0, 0, 8, 8);
    return c.toDataURL().length + ':' + img.naturalWidth + 'x' + img.naturalHeight;
  });
}

try {
  await viewer.goto(APP + '/browser');
  // Status is reachable before Chrome is: /browser reports ok:false until a
  // content tab exists, so only wait for the config to be served.
  const st0 = await waitFor('status from /browser socket', async () => {
    const s = await status();
    return s && s.presets && s.presets.length ? s : null;
  }, 20000);
  check(st0.daemon.port === Number(new URL(DAEMON).port), 'app talks to the test daemon', `port ${st0.daemon.port}`);
  check(!!st0.presets.find((p) => p.name === 'iphone'), 'status advertises presets');

  // Chrome with the extension joins the session.
  let sw = castCtx.serviceWorkers()[0];
  if (!sw) { try { sw = await castCtx.waitForEvent('serviceworker', { timeout: 20000 }); } catch { /* logged below */ } }
  console.log('extension service worker:', sw ? sw.url() : 'none');
  const go = castCtx.pages()[0] || (await castCtx.newPage());
  await go.goto(DAEMON + '/go?session=' + SESSION);
  await waitFor('extension connected', connected, 40000);
  check(true, 'extension connected to session ' + SESSION);

  // Watch the raw socket for a while: server-side ensure errors are only
  // reported as {"type":"error"} messages, not in the polled status.
  const WebSocket = require('/tmp/wsprobe/node_modules/ws');
  const raw = new WebSocket(APP.replace('http', 'ws') + '/api/browser/ws');
  raw.on('message', (d, bin) => {
    if (bin) return;
    rawLog.push(String(d).slice(0, 400));
  });

  // When the session window only holds /go, the server creates a placeholder
  // content tab (a data: document, because about:blank is not capturable).
  // The URL bar then drives that tab.
  const contentPage = await waitFor('content tab in Chrome', async () => {
    const pages = castCtx.pages().filter((p) => {
      const u = p.url();
      return !u.includes('/go?session=') && (u.startsWith('http') || u.startsWith('data:'));
    });
    return pages[pages.length - 1] || null;
  }, 30000);
  console.log('content tab:', contentPage.url().slice(0, 60));

  await viewer.fill('.bw-url-input', WWW + '/');
  await viewer.press('.bw-url-input', 'Enter');
  await waitFor('navigate', async () => (await status()).session.url.startsWith(WWW), 20000);
  check(authority(await status()), 'navigate reached the Chrome tab', (await status()).session.url);

  // First painted frame.
  await waitFor('first frame', async () => (await frameSignature()).length > 0, 45000);
  const sig1 = await frameSignature();
  check(!!sig1, 'frame painted in /browser', sig1);

  const pageState = await waitFor('page rendered', async () => {
    const state = await contentPage.evaluate(() => ({
      title: document.title,
      vp: [window.innerWidth, window.innerHeight],
      box: (() => { const el = document.getElementById('box'); if (!el) return null; const r = el.getBoundingClientRect(); return { bg: getComputedStyle(el).backgroundColor, w: r.width }; })(),
      btn: (() => { const el = document.getElementById('btn'); if (!el) return null; const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2, w: r.width, h: r.height }; })(),
    }));
    return state.box && state.btn ? state : null;
  }, 20000);
  check(pageState.title === 'spike', 'target page loaded', pageState.title);

  // Tap the button through the mirror. Normalized coordinates mean the CSS size
  // of the cast image does not matter.
  const box = await viewer.locator('img.bw-frame').boundingBox();
  const [vw, vh] = pageState.vp;
  const nx = pageState.btn.x / vw;
  const ny = pageState.btn.y / vh;
  await viewer.mouse.click(box.x + nx * box.width, box.y + ny * box.height);
  const tapped = await waitFor('button click landed', async () => {
    const bg = await contentPage.evaluate(() => getComputedStyle(document.getElementById('box')).backgroundColor);
    return bg !== pageState.box.bg ? bg : null;
  }, 15000);
  check(tapped === 'rgb(255, 136, 136)', 'tap landed on the Chrome tab', tapped);

  // Frames must keep flowing after the click (input wakes the capture loop).
  const before = (await status()).capture.frames;
  await waitFor('frame after click', async () => (await status()).capture.frames > before, 15000);
  check(true, 'capture still streaming after input');

  // iPhone preset: a real window resize, reported as achieved (macOS clamps).
  const beforeIphone = await status();
  await viewer.click('.bw-chip:has-text("iphone")');
  const iphone = await waitFor('iphone preset applied', async () => {
    const s = await status();
    return s.preset === 'iphone' && s.viewport.width > 0 && s.viewport.width !== beforeIphone.viewport.width ? s : null;
  }, 30000);
  const iw = await contentPage.evaluate(() => [window.innerWidth, window.innerHeight]);
  check(iphone.viewport.width === iw[0] && iphone.viewport.height === iw[1],
    'reported viewport matches the real window', `${JSON.stringify(iphone.viewport)} vs ${JSON.stringify(iw)}`);
  const pill = await viewer.textContent('.bw-pill');
  check(/\d+×\d+/.test(pill || ''), 'toolbar shows the achieved size', pill);

  await viewer.click('.bw-chip:has-text("ipad")');
  const ipad = await waitFor('ipad preset applied', async () => {
    const s = await status();
    return s.preset === 'ipad' ? s : null;
  }, 30000);
  check(ipad.viewport.width > iphone.viewport.width, 'iPad preset is wider', JSON.stringify(ipad.viewport));

  await viewer.click('.bw-chip:has-text("mac")');
  await waitFor('mac preset restored', async () => (await status()).preset === 'mac', 30000);
  const mac = await status();
  // "Mac" means the window the user originally had, not a hard-coded size: the
  // preset restores the bounds captured before the first shrink.
  const restored = mac.viewport.width === beforeIphone.viewport.width &&
    Math.abs(mac.viewport.height - beforeIphone.viewport.height) <= 2;
  check(restored, 'Mac preset restores the original size', `${JSON.stringify(mac.viewport)} vs ${JSON.stringify(beforeIphone.viewport)}`);
  check(mac.viewport.width > ipad.viewport.width, 'Mac preset is wider than iPad', `${mac.viewport.width} > ${ipad.viewport.width}`);

  // Shot mode: slower cadence, still updates after input.
  await viewer.click('.bw-chip:has-text("SHOT")');
  await waitFor('shot mode', async () => (await status()).mode === 'shot', 10000);
  const shots0 = (await status()).capture.frames;
  await sleep(1500);
  const shots1 = (await status()).capture.frames;
  check(shots1 > shots0, 'shot mode keeps a slow keepalive', `${shots0} -> ${shots1}`);
  await viewer.click('.bw-chip:has-text("CAST")');
  await waitFor('cast mode', async () => (await status()).mode === 'cast', 10000);
  check(true, 'switched back to cast');

  await viewer.screenshot({ path: '/tmp/bw-e2e/page.png', fullPage: false });
  console.log('artifact: /tmp/bw-e2e/page.png');
} catch (err) {
  try { console.log('status at failure:', JSON.stringify(await status())); } catch { /* ignore */ }
  try { console.log('recent socket messages:', JSON.stringify(rawLog.slice(-8))); } catch { /* ignore */ }
  check(false, 'unexpected error', err && err.stack ? err.stack.split('\n')[0] : String(err));
} finally {
  await viewerCtx.close().catch(() => {});
  await castCtx.close().catch(() => {});
}

function authority(s) {
  try { return new URL(s.session.url).host !== new URL(WWW).host ? null : 'ok'; } catch { return null; }
}

console.log(failures.length ? `\n${failures.length} FAILURE(S)` : '\nALL PASS');
process.exit(failures.length ? 1 : 0);
