import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    appliedCounts,
    sweepWindowCount,
    ledgerFor,
    sweepAction,
    sweepBlocker,
    sweepBody,
    sweepSummary,
} from '../src/v2/mcc/browser/sweep.ts';

const plan = (over) => ({
    ok: true,
    dry_run: true,
    kept_window_id: 900,
    kept_session_id: 'sess-c0zyta',
    keep_tabs: 1,
    close_tabs: 5,
    close_windows: 2,
    ...over,
});

test('the sweep is offered only when there is a second window to close', () => {
    assert.equal(sweepBlocker({ supported: true, readOnly: false, busy: false, windowCount: 3 }), '');
    assert.equal(sweepBlocker({ supported: true, readOnly: false, busy: false, windowCount: 1 }), 'Only one window is open');
});

test('the preview counts Chrome windows, not sessions sharing one', () => {
    assert.equal(
        sweepWindowCount([
            { key: 'a', attached: true, windowId: 900 },
            { key: 'window-900', attached: true, windowId: 900 },
            { key: 'b', attached: true, windowId: 910 },
        ]),
        2,
    );
    // A row that never learned its window id still counts as one window, and a
    // detached row is not a window at all.
    assert.equal(
        sweepWindowCount([
            { key: 'a', attached: true, windowId: 900 },
            { key: 'b', attached: true },
            { key: 'c', attached: false, windowId: 920 },
        ]),
        2,
    );
});

test('the blocker answers in the order the person can act on', () => {
    const one = { supported: true, readOnly: false, busy: false, windowCount: 1 };
    assert.match(sweepBlocker({ ...one, supported: false }), /1\.0\.22\+/);
    assert.match(sweepBlocker({ ...one, readOnly: true }), /Read-only/);
    assert.match(sweepBlocker({ ...one, busy: true }), /Wait/);
});

test('the ledger counts one surviving window and the blank page it opens', () => {
    const ledger = ledgerFor(plan({}));
    assert.equal(ledger.windows, '3 → 1');
    assert.equal(ledger.tabsDelta, '−5  +1');
    assert.equal(ledger.tabs, '6 → 2');
    assert.equal(ledger.kept, 'sess-c0zyta · agent page');
});

test('several agent pages in the keeper are counted, not named', () => {
    assert.equal(ledgerFor(plan({ keep_tabs: 2 })).kept, '2 agent pages');
});

test('a plan without a session id still says what is kept', () => {
    assert.equal(ledgerFor(plan({ kept_session_id: '' })).kept, 'Registry page · agent page');
});

test('the button names windows, in the singular when there is one', () => {
    assert.equal(sweepAction(plan({ close_windows: 1 })), 'Close 1 window');
    assert.equal(sweepAction(plan({ close_windows: 4 })), 'Close 4 windows');
});

test('the body pairs the closing counts with the promise to keep one page', () => {
    assert.deepEqual(sweepBody(plan({ close_windows: 2, close_tabs: 5 })), {
        closing: '2 Chrome windows and 5 tabs close.',
        keeping: 'This window keeps its agent page, then opens one blank page.',
    });
    assert.equal(sweepBody(plan({ close_windows: 1, close_tabs: 1 })).closing, '1 Chrome window and 1 tab close.');
});

test('the toast reports what closed, singular or plural', () => {
    assert.equal(sweepSummary(2, 5), 'Closed 2 windows and 5 tabs.');
    assert.equal(sweepSummary(1, 1), 'Closed 1 window and 1 tab.');
});

test('applied counts come from the result, not the plan, so a partial run is honest', () => {
    assert.deepEqual(appliedCounts(plan({ closed_windows: [910], closed_tabs: 2 })), { windows: 1, tabs: 2 });
    // A dry-run plan has closed nothing yet: the planned counts must not leak in.
    assert.deepEqual(appliedCounts(plan({})), { windows: 0, tabs: 0 });
    assert.deepEqual(appliedCounts(null), { windows: 0, tabs: 0 });
    assert.deepEqual(appliedCounts(undefined), { windows: 0, tabs: 0 });
});
