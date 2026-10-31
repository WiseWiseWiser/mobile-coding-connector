import assert from 'node:assert/strict';
import { register } from 'node:module';
import { test } from 'node:test';

// RemoteScreen imports its helpers without a file extension because the
// bundler resolves them; Node's ESM loader wants the extension. Teach the
// loader instead of adding a build step just for this test.
register(
    'data:text/javascript,' +
        encodeURIComponent(`
export async function resolve(specifier, context, nextResolve) {
    if (specifier.startsWith('.') && !/\\.[mc]?[jt]s$/.test(specifier)) {
        return nextResolve(specifier + '.ts', context);
    }
    return nextResolve(specifier, context);
}
`),
);

const { BrowserStream, CONNECT_PROBE_MS, PROBE_AFTER_FAILURES, STALL_PROBE_MS } = await import(
    new URL('../src/v2/mcc/browser/browserStream.ts', import.meta.url).href
);
const { BrowserPollStream, nextInterval, POLL_INTERVALS } = await import(
    new URL('../src/v2/mcc/browser/browserPollStream.ts', import.meta.url).href
);

// A deterministic clock, so probe thresholds cost no real time.
function fakeClock() {
    let now = 0;
    let seq = 0;
    const timers = new Map();
    const due = (limit) => {
        let best = null;
        for (const entry of timers.entries()) {
            if (entry[1].at > limit) continue;
            if (!best || entry[1].at < best[1].at) best = entry;
        }
        return best;
    };
    return {
        now: () => now,
        setTimeout(fn, ms) {
            const id = ++seq;
            timers.set(id, { fn, at: now + ms, every: 0 });
            return id;
        },
        setInterval(fn, ms) {
            const id = ++seq;
            timers.set(id, { fn, at: now + ms, every: ms });
            return id;
        },
        clearTimeout(id) {
            timers.delete(id);
        },
        clearInterval(id) {
            timers.delete(id);
        },
        advance(ms) {
            const limit = now + ms;
            while (true) {
                const hit = due(limit);
                if (!hit) break;
                const [id, entry] = hit;
                timers.delete(id);
                now = entry.at;
                entry.fn();
                if (entry.every) timers.set(id, { fn: entry.fn, at: now + entry.every, every: entry.every });
            }
            now = limit;
        },
        pending: () => timers.size,
    };
}

// The browser's WebSocket, reduced to what BrowserStream touches.
class MockSocket {
    static sockets = [];

    constructor(url) {
        this.url = url;
        this.closed = false;
        this.sent = [];
        this.onopen = null;
        this.onclose = null;
        this.onerror = null;
        this.onmessage = null;
        MockSocket.sockets.push(this);
    }

    send(data) {
        this.sent.push(data);
    }

    close() {
        if (this.closed) return;
        this.closed = true;
        this.onclose?.({});
    }

    open() {
        this.onopen?.({});
    }
}

function installEnv({ restOk = true } = {}) {
    const clock = fakeClock();
    MockSocket.sockets = [];
    let fetchCalls = 0;
    globalThis.window = {
        location: { protocol: 'http:', host: '127.0.0.1:23712' },
    };
    globalThis.WebSocket = MockSocket;
    globalThis.AbortController = class {
        abort() {}
    };
    globalThis.fetch = async () => {
        fetchCalls++;
        return { ok: restOk, status: restOk ? 200 : 503 };
    };
    globalThis.setTimeout = clock.setTimeout;
    globalThis.clearTimeout = clock.clearTimeout;
    globalThis.setInterval = clock.setInterval;
    globalThis.clearInterval = clock.clearInterval;
    return {
        clock,
        fetchCount: () => fetchCalls,
    };
}

const flush = async () => {
    for (let i = 0; i < 5; i++) await Promise.resolve();
};

test('a handshake that never opens probes REST and fires the banner evidence', async () => {
    const { clock, fetchCount } = installEnv();
    const probes = [];
    const stream = new BrowserStream({ onProbe: (e) => probes.push(e) });
    stream.connect();

    clock.advance(CONNECT_PROBE_MS + 100);
    await flush();

    assert.equal(fetchCount(), 1, 'the probe measured REST once');
    assert.equal(probes.length, 1, 'the connect-timeout probe fired once');
    assert.equal(probes[0].reason, 'connect-timeout');
    assert.ok(probes[0].restMs !== null, 'a 200 REST answer carries an RTT');
    assert.equal(probes[0].attempts, 1);
    assert.equal(stream.probed, true);

    // One offer per episode: more silence does not repeat the banner.
    clock.advance(CONNECT_PROBE_MS * 3);
    await flush();
    assert.equal(probes.length, 1);
});

test('three failed reconnects probe even when each handshake fails fast', async () => {
    const { clock } = installEnv();
    const probes = [];
    const stream = new BrowserStream({ onProbe: (e) => probes.push(e) });
    stream.connect();

    for (let i = 0; i < PROBE_AFTER_FAILURES; i++) {
        MockSocket.sockets[MockSocket.sockets.length - 1].close();
        clock.advance(8000);
        await flush();
    }

    assert.equal(probes.length, 1, 'the broken transport probes exactly once');
    assert.ok(
        probes[0].reason === 'reconnect-failures' || probes[0].reason === 'connect-timeout',
        `reason = ${probes[0].reason}`,
    );
    assert.ok(probes[0].attempts >= 1, `attempts = ${probes[0].attempts}`);
});

test('resetProbe re-arms the offer for a fresh episode', async () => {
    const { clock } = installEnv();
    const probes = [];
    const stream = new BrowserStream({ onProbe: (e) => probes.push(e) });
    stream.connect();

    clock.advance(CONNECT_PROBE_MS + 100);
    await flush();
    assert.equal(probes.length, 1);

    stream.resetProbe();
    // A fresh episode: the user tapped Retry, the socket reopens and stalls again.
    MockSocket.sockets[MockSocket.sockets.length - 1].close();
    clock.advance(1000 + CONNECT_PROBE_MS + 100);
    await flush();
    assert.equal(probes.length, 2, 'a new episode may probe again');
});

test('a stalled open socket probes after the silence threshold', async () => {
    const { clock } = installEnv();
    const probes = [];
    const stream = new BrowserStream({ onProbe: (e) => probes.push(e) });
    stream.connect();
    MockSocket.sockets[0].open();

    // The frame clock stopped 31 s ago while keepalives keep flowing.
    stream.lastFrameAt = Date.now() - STALL_PROBE_MS - 1000;
    stream.lastMessageAt = Date.now();
    clock.advance(5000);
    await flush();

    assert.equal(probes.length, 1);
    assert.equal(probes[0].reason, 'stalled');
});

test('a probe whose REST also fails still offers the choice, with no RTT', async () => {
    const { clock } = installEnv({ restOk: false });
    const probes = [];
    const stream = new BrowserStream({ onProbe: (e) => probes.push(e) });
    stream.connect();

    clock.advance(CONNECT_PROBE_MS + 100);
    await flush();

    assert.equal(probes.length, 1);
    assert.equal(probes[0].restMs, null);
});

// ---- poll transport -------------------------------------------------------

function installPollEnv({ status = 200, seq = 7 } = {}) {
    const clock = fakeClock();
    const requests = [];
    globalThis.window = { location: { protocol: 'http:', host: '127.0.0.1:23712' } };
    globalThis.fetch = async (path) => {
        requests.push(String(path));
        if (String(path).startsWith('/api/browser/frame')) {
            if (status === 304) return { status: 304 };
            return { status: 200, headers: new Map([['X-Frame-Seq', String(seq)]]), blob: () => Promise.resolve(new Blob([new Uint8Array([0xff, 0xd9])])) };
        }
        return { ok: true, status: 200, json: () => Promise.resolve({ ok: true }) };
    };
    globalThis.setTimeout = clock.setTimeout;
    globalThis.clearTimeout = clock.clearTimeout;
    globalThis.setInterval = clock.setInterval;
    globalThis.clearInterval = clock.clearInterval;
    globalThis.URL.createObjectURL = () => 'blob:fake';
    globalThis.URL.revokeObjectURL = () => {};
    return { clock, requests };
}

test('poll mode paints 200 frames and stays quiet on 304', async () => {
    const { clock, requests } = installPollEnv();
    const painted = [];
    const poll = new BrowserPollStream({
        onFrame: () => painted.push(1),
        onConnection: () => {},
    });
    poll.attachImage({ set src(_url) { this.onload?.(); } });
    poll.connect();
    await flush();
    clock.advance(100);
    await flush();

    assert.equal(painted.length, 1, 'the 200 frame painted');
    assert.equal(poll.seq, 7, 'the seq header advanced the cursor');
    assert.ok(requests.some((path) => path.includes('after=0')), 'the first poll asked from zero');

    // Next tick: 304, nothing painted, request carries the seen seq.
    globalThis.fetch = async (path) => {
        requests.push(String(path));
        return { status: 304 };
    };
    clock.advance(3100);
    await flush();

    assert.equal(painted.length, 1, '304 painted nothing');
    assert.ok(requests.some((path) => path.includes('after=7')), 'the cursor rides the request');
    poll.dispose();
});

test('a slow round trip backs the interval off and never speeds back up', () => {
    assert.equal(nextInterval(1000, 3500), 3000);
    assert.equal(nextInterval(3000, 3500), 5000);
    assert.equal(nextInterval(5000, 9000), 5000, 'already at the slowest step');
    assert.equal(nextInterval(3000, 800), 3000, 'a healed link keeps the chosen interval');
    assert.ok(POLL_INTERVALS.includes(1000) && POLL_INTERVALS.includes(5000));
});
