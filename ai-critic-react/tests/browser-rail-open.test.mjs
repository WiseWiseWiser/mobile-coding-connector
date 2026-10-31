import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    nextRailOpen,
    paneTabsFor,
    paneTitleFor,
    railInitialOpen,
    readRailStored,
    storeRailOpen,
} from '../src/v2/mcc/browser/railOpen.ts';

test('rail first paint: stored intent wins over the layout default', () => {
    assert.equal(railInitialOpen(null, false), false, 'phone without a stored intent stays collapsed');
    assert.equal(railInitialOpen(null, true), true, 'wide hosts start expanded');
    assert.equal(railInitialOpen(true, false), true, 'a stored open intent wins on a phone');
    assert.equal(railInitialOpen(false, true), false, 'a stored collapsed intent wins on a wide host');
});

test('rail toggle round-trips', () => {
    assert.equal(nextRailOpen(false), true);
    assert.equal(nextRailOpen(nextRailOpen(true)), true);
});

test('rail storage helpers survive a missing window (node)', () => {
    assert.equal(readRailStored(), null);
    assert.doesNotThrow(() => storeRailOpen(true));
});

test('pane title names the window, trimming and falling back', () => {
    assert.equal(paneTitleFor({ title: '  可观测性 | 凤凰架构 ', sessionId: 'sess-1' }), '可观测性 | 凤凰架构');
    assert.equal(paneTitleFor({ title: '   ', sessionId: 'sess-9' }), 'sess-9');
    assert.equal(paneTitleFor({ title: '' }), 'Chrome window');
});

test('pane tabs: the connected window is live, others read the polled list', () => {
    const live = [{ tab_id: 11, active: true, role: 'user' }];
    const polled = [{ tab_id: 21, active: true, role: 'user' }];
    assert.equal(paneTabsFor(undefined, live, true), live, 'current window without a session entry still shows live tabs');
    assert.deepEqual(paneTabsFor({ tabs: polled }, live, false), polled);
    assert.deepEqual(paneTabsFor({ tabs: polled }, live, true), live, 'current window prefers the live poll');
    assert.deepEqual(paneTabsFor({}, live, false), [], 'a window without a tab report shows an empty pane');
});

