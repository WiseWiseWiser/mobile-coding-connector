import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    bookmarkCountLabel,
    bookmarkFolderTrail,
    bookmarkPath,
    bookmarkSourceFooter,
    bookmarkSourceKicker,
    bookmarkSourceNote,
    buildBookmarkTree,
    filterBookmarks,
    flattenBookmarks,
    flattenTreeLinks,
    hasBookmark,
    localBookmarksLink,
    searchBookmarkTree,
} from '../src/v2/mcc/browser/browserBookmarks.ts';

const doc = {
    version: 1,
    roots: [
        { type: 'url', id: 'a', name: 'Example', url: 'https://example.com/' },
        {
            type: 'folder',
            id: 'f1',
            name: 'Work',
            children: [
                { type: 'url', id: 'b', name: 'Docs', url: 'https://example.com/docs' },
                {
                    type: 'folder',
                    id: 'f2',
                    name: 'Deep',
                    children: [{ type: 'url', id: 'c', name: 'API', url: 'http://localhost:3000/api' }],
                },
                { type: 'url', id: 'bad', name: 'File', url: 'file:///tmp/x' },
                { type: 'url', id: 'd', name: '  ', url: 'https://untitled.test/a' },
            ],
        },
    ],
};

test('flattening keeps tree order, folder paths and only http(s) pages', () => {
    const items = flattenBookmarks(doc);
    assert.deepEqual(
        items.map((item) => item.id),
        ['a', 'b', 'c', 'd'],
    );
    assert.equal(items[0].path, '');
    assert.equal(items[1].path, 'Work');
    assert.equal(items[2].path, 'Work / Deep');
    assert.equal(items[3].name, 'untitled.test/a');
    assert.equal(bookmarkPath(items[0]), 'Bookmarks');
    assert.equal(bookmarkPath(items[1]), 'Work');
});

test('the tree keeps folders, recursive counts and full search paths', () => {
    const tree = buildBookmarkTree(doc);
    assert.equal(tree.total, 4);
    assert.deepEqual(
        tree.roots.map((root) => root.name),
        ['Example', 'Work'],
    );
    assert.deepEqual(
        tree.roots.map((root) => root.count),
        [1, 3],
    );
    const work = tree.roots[1];
    assert.deepEqual(
        work.folders.map((folder) => folder.name),
        ['Deep'],
    );
    assert.deepEqual(
        work.links.map((item) => item.id),
        ['b', 'd'],
    );
    assert.equal(work.folders[0].links[0].path, 'Work / Deep');
    assert.deepEqual(
        bookmarkFolderTrail(work, ['f2']).map((folder) => folder.name),
        ['Work', 'Deep'],
    );
    assert.deepEqual(
        bookmarkFolderTrail(work, ['f2', 'missing']).map((folder) => folder.name),
        ['Work', 'Deep'],
    );
    assert.deepEqual(
        flattenTreeLinks(tree).map((item) => item.id),
        ['a', 'b', 'd', 'c'],
    );
    assert.deepEqual(
        searchBookmarkTree(tree, 'localhost').map((item) => item.id),
        ['c'],
    );
    assert.equal(searchBookmarkTree(tree, 'localhost')[0].path, 'Work / Deep');
});

test('a folder count is recursive while its children stay grouped', () => {
    const counted = {
        version: 1,
        roots: [
            {
                type: 'folder',
                id: 'root',
                name: 'Bar',
                children: [
                    {
                        type: 'folder',
                        id: 'a',
                        name: 'A',
                        children: [{ type: 'url', id: 'deep', name: 'Deep', url: 'https://a.test/deep' }],
                    },
                    { type: 'url', id: 'one', name: 'One', url: 'https://a.test/one' },
                    { type: 'url', id: 'two', name: 'Two', url: 'https://a.test/two' },
                ],
            },
        ],
    };
    const root = buildBookmarkTree(counted).roots[0];
    assert.equal(root.count, 3);
    assert.equal(root.folders.length, 1);
    assert.equal(root.folders[0].count, 1);
    assert.equal(root.links.length, 2);
});

test('a missing or malformed tree flattens to nothing', () => {
    assert.deepEqual(flattenBookmarks(null), []);
    assert.deepEqual(flattenBookmarks({ version: 1 }), []);
    assert.deepEqual(flattenBookmarks({ version: 1, roots: [{ type: 'url', id: 'x' }] }), []);
});

test('search matches title, URL and folder path, and requires every term', () => {
    const items = flattenBookmarks(doc);
    assert.deepEqual(
        filterBookmarks(items, 'docs').map((item) => item.id),
        ['b'],
    );
    assert.deepEqual(
        filterBookmarks(items, 'work localhost').map((item) => item.id),
        ['c'],
    );
    assert.deepEqual(
        filterBookmarks(items, 'deep api').map((item) => item.id),
        ['c'],
    );
    assert.equal(filterBookmarks(items, 'missing').length, 0);
    assert.equal(filterBookmarks(items, '  ').length, items.length);
});

test('the launcher caps the list but search still reaches later entries', () => {
    const many = {
        version: 1,
        roots: Array.from({ length: 130 }, (_, i) => ({
            type: 'url',
            id: `i${i}`,
            name: `Page ${i}`,
            url: `https://example.com/${i}`,
        })),
    };
    const items = flattenBookmarks(many);
    assert.equal(filterBookmarks(items, '').length, 120);
    assert.deepEqual(
        filterBookmarks(items, '129').map((item) => item.id),
        ['i129'],
    );
});

test('the star is on only when the exact page URL is saved', () => {
    const items = flattenBookmarks(doc);
    assert.equal(hasBookmark(items, 'https://example.com/docs'), true);
    assert.equal(hasBookmark(items, 'https://example.com/docs/'), false);
    assert.equal(hasBookmark(items, ''), false);
    assert.equal(hasBookmark(items, undefined), false);
});

test('source labels name Chrome and keep the local store editable', () => {
    assert.equal(bookmarkSourceKicker('chrome'), 'BOOKMARKS · CHROME');
    assert.equal(bookmarkSourceKicker('local'), 'LOCAL BOOKMARKS');
    assert.equal(bookmarkSourceNote('chrome', 'example.com'), 'example.com · read-only');
    assert.equal(bookmarkSourceNote('chrome', '  '), 'Default profile · read-only');
    assert.equal(bookmarkSourceNote('local', 'example.com'), 'On this Mac · editable');
    assert.equal(bookmarkCountLabel(0), '0 saved');
    assert.equal(bookmarkCountLabel(3), '3 saved');
    assert.equal(localBookmarksLink(3), 'Local bookmarks (3) →');
    assert.equal(localBookmarksLink(0), 'Local bookmarks →');
    assert.equal(localBookmarksLink(null), 'Local bookmarks →');
    assert.equal(bookmarkSourceFooter('chrome'), 'Chrome · read-only');
    assert.equal(bookmarkSourceFooter('local'), '~/.ai-critic/bookmarks.json');
});
