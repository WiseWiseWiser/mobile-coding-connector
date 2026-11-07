import assert from 'node:assert/strict';
import { test } from 'node:test';
import { orderByCreation, orderTabs, windowOrderRank } from '../src/v2/mcc/browser/windowsOrder.ts';

const win = (window_id, id = `sess-${window_id}`) => ({ window_id, id });
const tab = (tab_id, index) => ({ tab_id, index });

test('windows sort oldest first by ascending window id', () => {
    const rows = [win(41), win(12), win(30)];
    assert.deepEqual(orderByCreation(rows).map((row) => row.window_id), [12, 30, 41]);
});

test('a window with no live id sorts last without disturbing the rest', () => {
    const rows = [win(undefined, 'sess-gone'), win(9), win(3)];
    assert.deepEqual(orderByCreation(rows).map((row) => row.window_id), [3, 9, undefined]);
});

test('no id and id zero rank the same way', () => {
    assert.equal(windowOrderRank({ window_id: 0 }), windowOrderRank({ window_id: undefined }));
    assert.ok(windowOrderRank({ window_id: 7 }) < windowOrderRank({ window_id: undefined }));
});

test('equal ranks keep the order they arrived in', () => {
    const rows = [win(undefined, 'sess-a'), win(undefined, 'sess-b'), win(undefined, 'sess-c')];
    assert.deepEqual(orderByCreation(rows).map((row) => row.id), ['sess-a', 'sess-b', 'sess-c']);
});

test('sorting is not in place and returns a new array', () => {
    const rows = [win(2), win(1)];
    const sorted = orderByCreation(rows);
    assert.notEqual(sorted, rows);
    assert.deepEqual(rows.map((row) => row.window_id), [2, 1]);
});

test('tabs sort by Chrome index, not by id', () => {
    const tabs = [tab(90, 2), tab(12, 0), tab(70, 1)];
    assert.deepEqual(orderTabs(tabs).map((item) => item.tab_id), [12, 70, 90]);
});

test('a tab with no index sorts last', () => {
    const tabs = [tab(5, undefined), tab(6, 1), tab(7, 0)];
    assert.deepEqual(orderTabs(tabs).map((item) => item.tab_id), [7, 6, 5]);
});
