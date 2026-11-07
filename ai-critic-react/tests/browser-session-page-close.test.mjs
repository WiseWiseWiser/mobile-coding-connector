import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isSessionPageTab, sessionPageCloseCopy } from '../src/v2/mcc/browser/sessionPageClose.ts';

const tab = (over) => ({ tab_id: 1, title: 'sess-uvxz6o', url: '', active: false, role: 'user', ...over });

test('only the session control page asks first', () => {
    assert.equal(isSessionPageTab(tab({ role: 'session_page' })), true);
    assert.equal(isSessionPageTab(tab({ url: 'http://127.0.0.1:43761/go?session=sess-1' })), true);
    assert.equal(isSessionPageTab(tab({ url: 'https://example.com/go?next=1' })), false);
    assert.equal(isSessionPageTab(undefined), false);
});

test('a window with other tabs keeps them and only drops the mirror', () => {
    const copy = sessionPageCloseCopy({ sessionId: 'sess-1', tabCount: 2, current: true });
    assert.equal(copy.anchor, '/go?session=sess-1');
    assert.equal(copy.title, 'Close the session page?');
    assert.match(copy.body, /registers the Chrome window/);
    assert.match(copy.body, /other tab keeps running/);
    assert.match(copy.note, /rail/);
    assert.equal(copy.confirm, 'Close page');
});

test('plural wording follows the number of surviving tabs', () => {
    assert.match(sessionPageCloseCopy({ sessionId: 's', tabCount: 4, current: false }).body, /3 other tabs keep running/);
});

test('the last tab warns that the window goes with it', () => {
    const copy = sessionPageCloseCopy({ sessionId: 'sess-1', tabCount: 1, current: true });
    assert.equal(copy.anchor, '');
    assert.match(copy.body, /only tab/);
    assert.match(copy.note, /window too|last Chrome window/);
    assert.equal(copy.confirm, 'Close window');
});

test('a non-current window notes that it keeps running without a session', () => {
    assert.match(sessionPageCloseCopy({ sessionId: 's', tabCount: 3, current: false }).note, /without an agent session/);
});
