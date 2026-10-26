import assert from 'node:assert/strict';
import { test } from 'node:test';
import { TerminalConnection } from '../src/hooks/terminalConnection.ts';
import { fetchTerminalSessions } from '../src/api/terminal.ts';

class Socket {
    readyState = 0;
    sent = [];
    onopen = null;
    onmessage = null;
    onerror = null;
    onclose = null;
    open() { this.readyState = 1; this.onopen?.({}); }
    message(msg) { this.onmessage?.({ data: typeof msg === 'object' ? JSON.stringify(msg) : msg }); }
    send(data) { this.sent.push(data); }
    close() { this.readyState = 3; this.onclose?.({ code: 1000 }); }
}

function setup(t, overrides = {}) {
    t.mock.timers.enable({ apis: ['setTimeout'] });
    const sockets = [], states = [], output = [], ids = [];
    let resets = 0;
    const conn = new TerminalConnection({
        url: id => `/api/terminal?attach_mode=web${id ? '&session_id=' + id : ''}`,
        onState: state => states.push(state), onOutput: data => output.push(data),
        onSessionId: id => ids.push(id), onReplayStart: () => resets++,
        createSocket: url => { const socket = new Socket(); socket.url = url; sockets.push(socket); return socket; },
        ...overrides,
    });
    t.after(() => conn.dispose());
    conn.start();
    function ready(socket, id = 'session-5') {
        socket.open();
        socket.message({ type: 'session_id', session_id: id });
        socket.message({ type: 'attach_role', attach_role: 'attacher', cols: 43, rows: 23 });
        socket.message({ type: 'session_state', status: 'running', session_id: id });
    }
    return { conn, sockets, states, output, ids, ready, resets: () => resets };
}

test('network failure retries same assigned session without rerunning initial command', t => {
    const h = setup(t, { initialCommand: 'serve --port 8422' });
    h.ready(h.sockets[0]);
    assert.deepEqual(h.sockets[0].sent, ['serve --port 8422\n']);
    h.sockets[0].onerror({});
    assert.equal(h.states.at(-1), 'reconnecting');
    t.mock.timers.tick(500);
    assert.match(h.sockets[1].url, /session_id=session-5/);
    h.ready(h.sockets[1]);
    assert.deepEqual(h.sockets[1].sent, []);
    assert.equal(h.states.at(-1), 'connected');
    assert.equal(h.resets(), 2);
    assert.deepEqual(h.output, []);
});

test('OPEN is not connected and control frames never appear as terminal text', t => {
    const h = setup(t);
    h.sockets[0].open();
    assert.equal(h.states.at(-1), 'connecting');
    h.sockets[0].message({ type: 'attach_role', attach_role: 'writer' });
    h.sockets[0].message({ type: 'future_control' });
    h.sockets[0].message('shell text');
    assert.deepEqual(h.output, ['shell text']);
});

test('resume replaces stale OPEN socket; late old callbacks cannot corrupt state', t => {
    const h = setup(t, { sessionId: 'session-5', initialCommand: 'DO_NOT_RUN' });
    h.ready(h.sockets[0]);
    const lateMessage = h.sockets[0].onmessage, lateClose = h.sockets[0].onclose;
    h.conn.reconnect();
    h.ready(h.sockets[1]);
    lateMessage({ data: '{"type":"session_state","status":"exited"}' });
    lateClose({});
    assert.equal(h.states.at(-1), 'connected');
    t.mock.timers.tick(1000);
    assert.equal(h.sockets.length, 2);
    assert.deepEqual(h.sockets.flatMap(s => s.sent), []);
});

test('missing heartbeat pong recovers a half-open socket automatically', t => {
    const h = setup(t);
    h.ready(h.sockets[0]);
    t.mock.timers.tick(25000);
    assert.deepEqual(h.sockets[0].sent, ['{"type":"ping"}']);
    t.mock.timers.tick(5000);
    assert.equal(h.states.at(-1), 'reconnecting');
    t.mock.timers.tick(500);
    assert.equal(h.sockets.length, 2);
});

test('pong keeps an idle connection healthy', t => {
    const h = setup(t);
    h.ready(h.sockets[0]);
    t.mock.timers.tick(25000);
    h.sockets[0].message({ type: 'pong' });
    t.mock.timers.tick(5000);
    assert.equal(h.states.at(-1), 'connected');
    assert.equal(h.sockets.length, 1);
});

for (const [status, expected] of [['exited', 'exited'], ['missing', 'unavailable']]) {
    test(`${status} is terminal, not a reconnect loop or replacement shell`, t => {
        const h = setup(t, { sessionId: 'session-5' });
        h.sockets[0].open();
        h.sockets[0].message({ type: 'session_state', status });
        h.conn.reconnect();
        t.mock.timers.tick(60000);
        assert.equal(h.states.at(-1), expected);
        assert.equal(h.sockets.length, 1);
        h.conn.send('echo SHOULD_NOT_RUN\n');
        assert.deepEqual(h.sockets[0].sent, []);
    });
}

test('cleanup cancels retries and never sends destructive close codes', t => {
    const h = setup(t);
    h.sockets[0].onclose({ code: 1006 });
    h.conn.dispose();
    t.mock.timers.tick(60000);
    assert.equal(h.sockets.length, 1);
});

test('resume bursts before session handshake cannot spawn duplicate shells', t => {
    const h = setup(t);
    h.conn.reconnect();
    h.sockets[0].open();
    h.conn.reconnect();
    h.conn.reconnect();
    assert.equal(h.sockets.length, 1);
    h.ready(h.sockets[0]);
    h.conn.reconnect();
    h.conn.reconnect();
    assert.equal(h.sockets.length, 2);
});

test('connection handshake timeout retries with capped backoff', t => {
    const h = setup(t);
    h.sockets[0].open();
    t.mock.timers.tick(10000);
    assert.equal(h.states.at(-1), 'reconnecting');
    t.mock.timers.tick(500);
    assert.equal(h.sockets.length, 2);
});

test('session restoration follows pagination, retaining sessions beyond first page', async t => {
    const requests = [];
    t.mock.method(globalThis, 'fetch', async url => {
        requests.push(url);
        const page = new URL(url, 'http://localhost').searchParams.get('page');
        return { ok: true, json: async () => ({ sessions: [{ id: `session-${page}` }], total_pages: 3 }) };
    });
    assert.deepEqual((await fetchTerminalSessions()).map(s => s.id), ['session-1', 'session-2', 'session-3']);
    assert.equal(requests.length, 3);
});

test('failed session listing is an error, never an empty list that creates shells', async t => {
    t.mock.method(globalThis, 'fetch', async () => ({ ok: false, status: 503 }));
    await assert.rejects(fetchTerminalSessions(), /503/);
});
