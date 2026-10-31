import assert from 'node:assert/strict';
import { test } from 'node:test';
import { closeTargetFor, closeWindowBlocker } from '../src/v2/mcc/browser/windowClose.ts';

const row = (over) => ({
    key: 'sess-1',
    sessionId: 'sess-1',
    title: 'A window',
    tabCount: 3,
    current: false,
    sessionless: false,
    adoptable: false,
    ...over,
});

test('window close explains the first blocker in priority order', () => {
    assert.match(closeWindowBlocker({ supported: false, readOnly: true, busy: true, windowCount: 1 }), /1\.0\.22/);
    assert.match(closeWindowBlocker({ supported: true, readOnly: true, busy: true, windowCount: 1 }), /Read-only/);
    assert.match(closeWindowBlocker({ supported: true, readOnly: false, busy: true, windowCount: 1 }), /Wait/);
    assert.match(closeWindowBlocker({ supported: true, readOnly: false, busy: false, windowCount: 1 }), /last Chrome window/);
    assert.equal(closeWindowBlocker({ supported: true, readOnly: false, busy: false, windowCount: 2 }), '');
});

test('a live session is closed by session id, even when it reports a window id', () => {
    assert.deepEqual(closeTargetFor(row({ windowId: 42 })), { session_id: 'sess-1' });
});

test('a sessionless window is closed by Chrome window id', () => {
    assert.deepEqual(
        closeTargetFor(row({ sessionId: undefined, sessionless: true, windowId: 42 })),
        { window_id: 42 },
    );
});

test('a sessionless row without a window id has no safe target', () => {
    assert.equal(closeTargetFor(row({ sessionId: undefined, sessionless: true })), null);
});

test('a live row without a session id has no safe target', () => {
    assert.equal(closeTargetFor(row({ sessionId: undefined })), null);
});

