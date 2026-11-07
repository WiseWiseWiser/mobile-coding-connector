import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    highlightSpans,
    matchCount,
    searchTokens,
    searchWindows,
    tabMatches,
} from '../src/v2/mcc/browser/tabSearch.ts';

const tab = (tab_id, index, title, url, role = 'tab') => ({ tab_id, index, title, url, role, active: false });

const windows = [
    {
        key: 'sess-a',
        title: 'ai-critic-react',
        tabs: [
            tab(11, 0, 'Pull requests · ai-critic', 'https://github.com/xhd2015/ai-critic/pulls'),
            tab(12, 1, 'Release notes', 'https://github.com/xhd2015/ai-critic/releases'),
            tab(13, 2, 'Agent transport', 'http://127.0.0.1:23712/go?session=sess-a', 'session_page'),
        ],
    },
    {
        key: 'sess-b',
        title: 'spring-ai',
        tabs: [tab(21, 0, 'Spring AI reference', 'https://docs.spring.io/spring-ai/reference/')],
    },
];

test('tokens are lowercased and whitespace split', () => {
    assert.deepEqual(searchTokens('  GitHub   PULLS '), ['github', 'pulls']);
    assert.deepEqual(searchTokens('   '), []);
});

test('an empty query matches nothing', () => {
    assert.deepEqual(searchWindows(windows, ''), []);
    assert.deepEqual(searchWindows(windows, '   '), []);
});

test('title matches, case-insensitively, grouped by window', () => {
    const groups = searchWindows(windows, 'release');
    assert.equal(groups.length, 1);
    assert.equal(groups[0].key, 'sess-a');
    assert.deepEqual(groups[0].hits.map((hit) => hit.tabId), [12]);
});

test('url matches count as well as titles', () => {
    const groups = searchWindows(windows, 'docs.spring.io');
    assert.deepEqual(groups.map((group) => group.key), ['sess-b']);
    assert.equal(groups[0].hits[0].title, 'Spring AI reference');
});

test('every token must match, across title and url together', () => {
    assert.equal(matchCount(searchWindows(windows, 'github releases')), 1);
    assert.equal(matchCount(searchWindows(windows, 'github spring')), 0);
});

test('matching is grouped in the order the caller passes windows', () => {
    const groups = searchWindows(windows, 'ai');
    assert.deepEqual(groups.map((group) => group.key), ['sess-a', 'sess-b']);
    // Both ai-critic tabs in the first window match on title or url.
    assert.equal(matchCount(groups), 3);
});

test('the agent transport page is never a search result', () => {
    assert.equal(matchCount(searchWindows(windows, '127.0.0.1')), 0);
    assert.equal(matchCount(searchWindows(windows, 'agent transport')), 0);
    assert.equal(tabMatches(tab(1, 0, 'x', 'x', 'session_page'), ['x']), false);
});

test('hits inside a window keep tab order', () => {
    const unsorted = {
        key: 'sess-z',
        title: 'z',
        tabs: [tab(31, 2, 'match three', 'u'), tab(32, 0, 'match one', 'u'), tab(33, 1, 'match two', 'u')],
    };
    const [group] = searchWindows([unsorted], 'match');
    assert.deepEqual(group.hits.map((hit) => hit.tabId), [32, 33, 31]);
});

test('a tab with no title still matches on its url', () => {
    const [group] = searchWindows([{ key: 'w', title: 'w', tabs: [tab(4, 0, undefined, 'https://example.com/thing')] }], 'thing');
    assert.equal(group.hits[0].tabId, 4);
    assert.equal(group.hits[0].title, '');
});

test('highlight spans cover both tokens and merge overlaps', () => {
    assert.deepEqual(highlightSpans('github.com/git', ['git']), [
        { start: 0, end: 3 },
        { start: 11, end: 14 },
    ]);
    assert.deepEqual(highlightSpans('Github', ['git', 'github']), [{ start: 0, end: 6 }]);
    assert.deepEqual(highlightSpans('nothing here', ['zzz']), []);
    assert.deepEqual(highlightSpans('', ['a']), []);
});

test('an empty query reports zero matches', () => {
    assert.equal(matchCount(searchWindows(windows, '')), 0);
    assert.deepEqual(searchWindows(windows, 'nomatch'), []);
});
