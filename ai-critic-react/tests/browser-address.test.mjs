import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    addressLabel,
    bookmarkFallbackName,
    checkAddress,
    hostLabel,
    isHttpURL,
    looksLikeAddress,
} from '../src/v2/mcc/browser/browserAddress.ts';

const errorOf = (raw) => {
    const result = checkAddress(raw);
    assert.equal(result.ok, false, `${raw} should be refused`);
    return result.error;
};

test('the address field refuses empty input and control characters', () => {
    errorOf('');
    errorOf('   ');
    errorOf('example.com\nfoo');
});

test('schemes outside the mirror are refused before a request', () => {
    for (const raw of [
        'chrome://version',
        'file:///tmp/x',
        'javascript:alert(1)',
        'mailto:a@b.c',
        'about:blank',
        'view-source:https://example.com',
    ]) {
        const error = errorOf(raw);
        assert.match(error, /http and https|http:\/\/|not valid/);
    }
});

test('http and https are accepted and sent through unchanged', () => {
    for (const raw of ['http://localhost:3000', 'https://example.com/a?b=1', 'HTTP://example.com']) {
        assert.deepEqual(checkAddress(raw), { ok: true, target: raw });
    }
});

test('bare host:port and search phrases are left for the server', () => {
    assert.deepEqual(checkAddress('localhost:3000'), { ok: true, target: 'localhost:3000' });
    assert.deepEqual(checkAddress('127.0.0.1:8080/x'), { ok: true, target: '127.0.0.1:8080/x' });
    assert.deepEqual(checkAddress('example.com'), { ok: true, target: 'example.com' });
    assert.deepEqual(checkAddress('how to cook rice'), { ok: true, target: 'how to cook rice' });
    assert.deepEqual(checkAddress('//example.com'), { ok: true, target: 'https://example.com' });
});

test('the primary action follows the shape of the text', () => {
    for (const raw of [
        'https://example.com',
        '//example.com',
        'example.com',
        'localhost:3000',
        '127.0.0.1:8080/x',
        'example.com/search?q=1',
    ]) {
        assert.equal(looksLikeAddress(raw), true, `${raw} should offer Open`);
    }
    for (const raw of ['', '   ', 'how to cook rice', 'work docs', 'calendar']) {
        assert.equal(looksLikeAddress(raw), false, `${raw} should offer Search`);
    }
});

test('a scheme without slashes is treated as a scheme, not a host', () => {
    const error = errorOf('http:example.com');
    assert.match(error, /http:\/\/ or https:\/\//);
});

test('http(s) labels drop the scheme and trailing root slash', () => {
    assert.equal(addressLabel('https://example.com/'), 'example.com');
    assert.equal(addressLabel('https://example.com/a/b?q=1'), 'example.com/a/b?q=1');
    assert.equal(addressLabel('not a url'), 'not a url');
    assert.equal(addressLabel(null), '');
    assert.equal(hostLabel('https://example.com/a'), 'example.com');
    assert.equal(hostLabel('localhost:3000'), 'localhost:3000');
});

test('only absolute http(s) URLs can be mirrored or bookmarked', () => {
    assert.equal(isHttpURL('https://example.com'), true);
    assert.equal(isHttpURL('http://127.0.0.1:3000'), true);
    assert.equal(isHttpURL('file:///tmp/x'), false);
    assert.equal(isHttpURL('javascript:alert(1)'), false);
    assert.equal(isHttpURL(''), false);
    assert.equal(isHttpURL(undefined), false);
});

test('an untitled page still gets a readable fallback name', () => {
    assert.equal(bookmarkFallbackName('https://example.com/docs/start'), 'example.com/docs/start');
    assert.equal(bookmarkFallbackName('https://example.com'), 'example.com');
    assert.equal(bookmarkFallbackName('nonsense'), 'nonsense');
});
