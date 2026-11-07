import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    baselineSeen,
    emptySeen,
    isTabSeen,
    isWindowSeen,
    parseSeen,
    pruneSeen,
    serializeSeen,
    tabSeenKey,
    unreadCount,
    withTabSeen,
    withWindowSeen,
} from '../src/v2/mcc/browser/seen.ts';

test('an unseeded ledger knows nothing and badges nothing', () => {
    const state = emptySeen();
    assert.equal(state.seeded, false);
    assert.equal(isWindowSeen(state, 'sess-a'), false);
    assert.equal(unreadCount(state, ['sess-a', 'sess-b']), 0);
});

test('the first run baselines the live windows and tabs', () => {
    const state = baselineSeen(['sess-a', 'sess-b'], ['sess-a:11', 'sess-b:22']);
    assert.equal(state.seeded, true);
    assert.equal(isWindowSeen(state, 'sess-a'), true);
    assert.equal(isTabSeen(state, 'sess-a:11'), true);
    assert.equal(unreadCount(state, ['sess-a', 'sess-b']), 0);
});

test('a window that appears after the baseline is unread', () => {
    const state = baselineSeen(['sess-a'], ['sess-a:11']);
    assert.equal(isWindowSeen(state, 'sess-c'), false);
    assert.equal(unreadCount(state, ['sess-a', 'sess-c']), 1);
});

test('marking a window read leaves its tabs read state alone', () => {
    const state = withWindowSeen(baselineSeen(['sess-a'], []), 'sess-b');
    assert.equal(isWindowSeen(state, 'sess-b'), true);
    assert.equal(isTabSeen(state, 'sess-b:5'), false);
});

test('marking the same window twice keeps one entry', () => {
    const once = withWindowSeen(baselineSeen([], []), 'sess-b');
    const twice = withWindowSeen(once, 'sess-b');
    assert.equal(twice, once);
    assert.deepEqual(twice.windows, ['sess-b']);
});

test('tab keys are window scoped', () => {
    assert.equal(tabSeenKey('sess-a', 12), 'sess-a:12');
    assert.notEqual(tabSeenKey('sess-a', 12), tabSeenKey('sess-b', 12));
    const state = withTabSeen(baselineSeen([], []), tabSeenKey('sess-a', 12));
    assert.equal(isTabSeen(state, 'sess-a:12'), true);
    assert.equal(isTabSeen(state, 'sess-b:12'), false);
});

test('pruning drops marks for windows and tabs that are gone', () => {
    const state = { ...baselineSeen(['sess-a', 'sess-b'], ['sess-a:1', 'sess-b:2']) };
    const pruned = pruneSeen(state, ['sess-a'], ['sess-a:1']);
    assert.deepEqual(pruned.windows, ['sess-a']);
    assert.deepEqual(pruned.tabs, ['sess-a:1']);
});

test('a pruned window that reopens under a new id is unread again', () => {
    const state = pruneSeen(baselineSeen(['sess-old'], []), ['sess-new'], []);
    assert.equal(isWindowSeen(state, 'sess-new'), false);
    assert.equal(unreadCount(state, ['sess-new']), 1);
});

test('pruning without changes returns the same object', () => {
    const state = baselineSeen(['sess-a'], ['sess-a:1']);
    assert.equal(pruneSeen(state, ['sess-a'], ['sess-a:1']), state);
});

test('pruning never touches an unseeded ledger', () => {
    const state = emptySeen();
    assert.equal(pruneSeen(state, [], []), state);
});

test('an empty enumeration does not unread anything', () => {
    // The poll answers after the first paint; before it does the target list is
    // empty and must not be mistaken for "the browser has no windows".
    const state = baselineSeen(['sess-a'], ['sess-a:1']);
    assert.equal(pruneSeen(state, [], []), state);
    assert.equal(unreadCount(state, ['sess-a']), 0);
});

test('a round trip through JSON preserves the ledger', () => {
    const state = withTabSeen(withWindowSeen(baselineSeen(['sess-a'], []), 'sess-b'), 'sess-b:7');
    const back = parseSeen(serializeSeen(state));
    assert.deepEqual(back, state);
});

test('a corrupt or foreign blob reads as nothing seen', () => {
    assert.deepEqual(parseSeen('{oops'), emptySeen());
    assert.deepEqual(parseSeen('null'), emptySeen());
    assert.deepEqual(parseSeen('{"v":2,"seeded":true,"windows":["sess-a"],"tabs":[]}'), emptySeen());
    assert.deepEqual(parseSeen(null), emptySeen());
});

test('non-string members are dropped instead of trusted', () => {
    const state = parseSeen('{"v":1,"seeded":true,"windows":["sess-a",7,null],"tabs":["sess-a:1"]}');
    assert.deepEqual(state.windows, ['sess-a']);
    assert.deepEqual(state.tabs, ['sess-a:1']);
});
