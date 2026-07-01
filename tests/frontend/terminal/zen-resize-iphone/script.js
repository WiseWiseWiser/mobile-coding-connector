const path = require('path');

await page.setViewportSize({ width: 390, height: 844 });

await page.addInitScript(() => {
  const NativeWebSocket = window.WebSocket;
  const terminalMessages = [];
  const terminalSockets = [];

  function normalizeData(data) {
    if (typeof data === 'string') return data;
    if (data instanceof ArrayBuffer) return '[arraybuffer]';
    if (ArrayBuffer.isView(data)) return '[arraybuffer-view]';
    if (data instanceof Blob) return '[blob]';
    return String(data);
  }

  function RecordingWebSocket(url, protocols) {
    const socket = protocols === undefined
      ? new NativeWebSocket(url)
      : new NativeWebSocket(url, protocols);
    const socketInfo = {
      url: String(url),
      openedAt: performance.now(),
      sentCount: 0,
    };
    terminalSockets.push(socketInfo);

    const nativeSend = socket.send.bind(socket);
    socket.send = (data) => {
      const text = normalizeData(data);
      let parsed = null;
      try {
        parsed = JSON.parse(text);
      } catch {
        parsed = null;
      }
      socketInfo.sentCount += 1;
      if (String(url).includes('/api/terminal')) {
        terminalMessages.push({
          url: String(url),
          data: text,
          parsed,
          sentAt: performance.now(),
          sentCount: socketInfo.sentCount,
        });
      }
      return nativeSend(data);
    };
    return socket;
  }

  RecordingWebSocket.prototype = NativeWebSocket.prototype;
  Object.setPrototypeOf(RecordingWebSocket, NativeWebSocket);
  RecordingWebSocket.CONNECTING = NativeWebSocket.CONNECTING;
  RecordingWebSocket.OPEN = NativeWebSocket.OPEN;
  RecordingWebSocket.CLOSING = NativeWebSocket.CLOSING;
  RecordingWebSocket.CLOSED = NativeWebSocket.CLOSED;

  window.WebSocket = RecordingWebSocket;
  window.__terminalMessages = terminalMessages;
  window.__terminalSockets = terminalSockets;
});

const repoRoot = path.resolve(CASE_DIR, '../../../..');
const projectName = `zen-resize-${Date.now()}`;
const addProjectResponse = await page.request.post(BASE_URL + '/api/projects', {
  data: {
    name: projectName,
    dir: repoRoot,
  },
});
if (!addProjectResponse.ok()) {
  throw new Error(`failed to add project: ${addProjectResponse.status()} ${await addProjectResponse.text()}`);
}

function sleep(ms) {
  return page.waitForTimeout(ms);
}

async function resizeMessages() {
  return await page.evaluate(() => {
    return (window.__terminalMessages || [])
      .filter((msg) => msg.parsed && msg.parsed.type === 'resize')
      .map((msg) => ({
        url: msg.url,
        cols: msg.parsed.cols,
        rows: msg.parsed.rows,
        sentAt: msg.sentAt,
        sentCount: msg.sentCount,
      }));
  });
}

async function terminalSocketCount() {
  return await page.evaluate(() => {
    return (window.__terminalSockets || [])
      .filter((socket) => String(socket.url).includes('/api/terminal'))
      .length;
  });
}

async function waitForResizeCountGreaterThan(previousCount, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let current = await resizeMessages();
  while (Date.now() < deadline) {
    current = await resizeMessages();
    if (current.length > previousCount) return current;
    await sleep(100);
  }
  return current;
}

async function waitForStableTerminalBox() {
  await page.waitForSelector('.terminal-instance.active .xterm', { timeout: 60000 });
  await page.waitForFunction(() => {
    const el = document.querySelector('.terminal-instance.active .xterm');
    if (!el) return false;
    const rect = el.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  }, { timeout: 60000 });
}

const projectPath = `/project/${encodeURIComponent(projectName)}/terminal`;
await page.goto(BASE_URL + projectPath, { waitUntil: 'domcontentloaded' });
await page.waitForSelector('.terminal-manager', { timeout: 60000 });
await page.waitForSelector('.terminal-tab-item.active', { timeout: 60000 });
await waitForStableTerminalBox();

const initialResizeMessages = await waitForResizeCountGreaterThan(0, 30000);
const beforeEntryCount = initialResizeMessages.length;

const zenButton = page.locator('button.terminal-zen-btn').filter({ hasText: /^Zen$/ }).first();
await zenButton.waitFor({ state: 'visible', timeout: 30000 });
await zenButton.click();
await page.waitForSelector('.terminal-manager.zen-mode', { timeout: 30000 });
await waitForStableTerminalBox();
const afterEntryMessages = await waitForResizeCountGreaterThan(beforeEntryCount, 5000);
const entryResizeCount = afterEntryMessages.length - beforeEntryCount;

const exitButton = page.locator('button.terminal-zen-btn').filter({ hasText: /^Exit Zen$/ }).first();
await exitButton.waitFor({ state: 'visible', timeout: 30000 });
const beforeExitCount = afterEntryMessages.length;
await exitButton.click();
await page.waitForFunction(() => {
  return !document.querySelector('.terminal-manager')?.classList.contains('zen-mode');
}, { timeout: 30000 });
await waitForStableTerminalBox();
const afterExitMessages = await waitForResizeCountGreaterThan(beforeExitCount, 5000);
const exitResizeCount = afterExitMessages.length - beforeExitCount;

const lastResize = afterExitMessages[afterExitMessages.length - 1] || null;
const ok = beforeEntryCount >= 1
  && entryResizeCount >= 1
  && exitResizeCount >= 1
  && !!lastResize
  && Number(lastResize.cols) > 0
  && Number(lastResize.rows) > 0;

console.log(JSON.stringify({
  ok,
  url: page.url(),
  viewport: { width: 390, height: 844 },
  projectName,
  terminalSocketCount: await terminalSocketCount(),
  initialResizeCount: beforeEntryCount,
  entryResizeCount,
  exitResizeCount,
  resizeMessages: afterExitMessages,
  lastResize,
  zenClassPresent: await page.locator('.terminal-manager.zen-mode').count(),
}));
