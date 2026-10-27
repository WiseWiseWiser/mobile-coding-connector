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

const { RemoteScreen } = await import(
    new URL('../src/v2/mcc/screen/RemoteScreen.ts', import.meta.url).href
);

// The server starts capture for the first viewer and stops it shortly after the
// last one leaves. A hidden page therefore has to close its socket: leaving it
// open kept the host screen recording and the display awake with nobody
// looking. These cases pin that contract, plus the reconnect rules that keep a
// return to the page cheap and the watchdog honest.

// A deterministic clock, so "no reconnect for a minute" costs no real time.
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
            for (;;) {
                const next = due(limit);
                if (!next) break;
                const [id, timer] = next;
                now = timer.at;
                if (timer.every) timer.at += timer.every;
                else timers.delete(id);
                timer.fn();
            }
            now = limit;
        },
        pending: () => timers.size,
    };
}

// The browser's WebSocket, reduced to what RemoteScreen touches.
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

    // Test-side: the server accepted the upgrade.
    open() {
        this.onopen?.({});
    }
}

const live = () => MockSocket.sockets.filter((socket) => !socket.closed).length;

function installEnv() {
    const clock = fakeClock();
    MockSocket.sockets = [];
    globalThis.window = {
        location: { protocol: 'http:', host: '127.0.0.1:23712' },
        setTimeout: clock.setTimeout,
        clearTimeout: clock.clearTimeout,
        setInterval: clock.setInterval,
        clearInterval: clock.clearInterval,
    };
    globalThis.WebSocket = MockSocket;
    return clock;
}

function connected() {
    const clock = installEnv();
    const errors = [];
    const screen = new RemoteScreen('test-viewer', { onError: (message) => errors.push(message) });
    screen.connect();
    assert.equal(MockSocket.sockets.length, 1, 'connect opens exactly one socket');
    const first = MockSocket.sockets[0];
    first.open();
    assert.equal(screen.connection, 'open');
    return { clock, screen, first, errors };
}

test('pause closes the socket and stays quiet for a minute', () => {
    const { clock, screen, first } = connected();

    screen.pause();

    assert.equal(first.closed, true, 'the socket is closed');
    assert.equal(screen.paused, true, 'the screen records the pause');
    assert.equal(screen.connection, 'closed');
    assert.equal(clock.pending(), 0, 'no reconnect timer and no watchdog survive pause');

    clock.advance(60_000);
    assert.equal(MockSocket.sockets.length, 1, 'nothing reconnected while hidden');
    assert.equal(live(), 0, 'no socket stays open while hidden');
});

test('resume reconnects once and ignores duplicate signals', () => {
    const { screen } = connected();
    screen.pause();

    screen.resume();
    assert.equal(screen.paused, false);
    assert.equal(MockSocket.sockets.length, 2, 'resume opens a fresh socket');

    // visibilitychange and pageshow both fire on return; the second one must
    // not churn the socket that is already connecting.
    screen.resume();
    screen.resume();
    assert.equal(MockSocket.sockets.length, 2, 'duplicate resumes reuse the socket');
});

test('three hide/show cycles never leave two sockets open', () => {
    const { screen } = connected();

    for (let i = 0; i < 3; i++) {
        screen.pause();
        assert.equal(live(), 0, `cycle ${i}: hidden means no live socket`);
        screen.resume();
        assert.equal(live(), 1, `cycle ${i}: visible means exactly one live socket`);
    }
    assert.equal(MockSocket.sockets.length, 4, 'one socket per visible period');
    assert.equal(live(), 1);
});

test('pause cancels a reconnect that was already scheduled', () => {
    const { clock, screen, first } = connected();

    // The connection drops on its own (host restart, network switch).
    first.close();
    assert.equal(screen.connection, 'closed');
    clock.advance(500);
    assert.equal(MockSocket.sockets.length, 1, 'the retry waits out its delay');

    screen.pause();
    clock.advance(30_000);
    assert.equal(MockSocket.sockets.length, 1, 'a hidden page does not come back on its own');

    screen.resume();
    assert.equal(MockSocket.sockets.length, 2, 'showing the page reconnects immediately');
});

test('the stalled-socket watchdog still works, and stays off while paused', () => {
    const { clock, screen, first, errors } = connected();

    // A live host always sends cursor traffic; silence means half-open.
    screen.lastMessageAt = Date.now() - 60_000;
    clock.advance(3_000);
    assert.equal(MockSocket.sockets.length, 2, 'a quiet socket is replaced');
    assert.equal(errors.length, 1, 'the user is told why the picture froze');
    assert.equal(first.closed, true);

    screen.pause();
    MockSocket.sockets[1].open();
    clock.advance(60_000);
    assert.equal(MockSocket.sockets.length, 2, 'a paused page runs no watchdog');
});

test('a socket from before the pause cannot resurrect it', () => {
    const { clock, screen, first } = connected();
    screen.pause();

    // Events queued before close() landed must be ignored.
    first.onclose?.({});
    first.onmessage?.({ data: JSON.stringify({ type: 'hello' }) });
    clock.advance(5_000);
    assert.equal(MockSocket.sockets.length, 1, 'a late close event does not schedule a retry');
    assert.equal(screen.paused, true);
});

test('connect is a no-op while paused', () => {
    const { screen } = connected();
    screen.pause();

    screen.connect();

    assert.equal(MockSocket.sockets.length, 1, 'no socket is opened for a hidden page');
    assert.equal(screen.paused, true);
});

test('dispose after pause leaves nothing behind', () => {
    const { clock, screen } = connected();
    screen.pause();
    screen.dispose();

    clock.advance(60_000);
    assert.equal(MockSocket.sockets.length, 1);
    assert.equal(clock.pending(), 0, 'no timers survive dispose');
});
