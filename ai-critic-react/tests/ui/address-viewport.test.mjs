// Simulates VisualViewport changes; real iOS keyboard behavior still needs device verification.
import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import net from 'node:net';
import { createServer } from 'vite';
import puppeteer from 'puppeteer';

const root = fileURLToPath(new URL('../../', import.meta.url));
let server, browser, base;
before(async () => {
    const port = await new Promise((resolve, reject) => {
        const probe = net.createServer().on('error', reject);
        probe.listen(0, '127.0.0.1', () => { const p = probe.address().port; probe.close(() => resolve(p)); });
    });
    server = await createServer({ root, server: { host: '127.0.0.1', port, strictPort: true }, logLevel: 'error' });
    await server.listen();
    base = `http://127.0.0.1:${port}/tests/ui/window-drawer.html?url`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });
async function setup(noViewport = false) {
    const page = await browser.newPage();
    await page.setViewport({ width: 390, height: 844, isMobile: true, hasTouch: true });
    await page.evaluateOnNewDocument((missing) => {
        if (missing) { Object.defineProperty(window, 'visualViewport', { value: undefined, configurable: true }); return; }
        const viewport = new EventTarget();
        Object.assign(viewport, { width: 390, height: 844, offsetTop: 0, offsetLeft: 0 });
        Object.defineProperty(window, 'visualViewport', { value: viewport, configurable: true });
        window.fixtureViewport = viewport;
    }, noViewport);
    await page.goto(base, { waitUntil: 'networkidle0' });
    await page.waitForSelector('.bwp-urlinput');
    return page;
}
async function changeViewport(page, values, event = 'resize') {
    await page.evaluate(({ values, event }) => { Object.assign(window.fixtureViewport, values); window.fixtureViewport.dispatchEvent(new Event(event)); }, { values, event });
}

test('keyboard resize and pan keep address controls visible without losing draft', async () => {
    const page = await setup();
    assert.equal(await page.$eval('.bwp-urlinput', el => el === document.activeElement), true);
    await page.click('.bwp-urlinput', { clickCount: 3 });
    await page.type('.bwp-urlinput', 'https://draft.example/path#part');
    await changeViewport(page, { height: 420, offsetTop: 24 });
    const geometry = await page.evaluate(() => {
        const rect = (selector) => document.querySelector(selector).getBoundingClientRect().toJSON();
        return { wrap: rect('.bwp-sheetwrap'), input: rect('.bwp-urlinput'), go: rect('.bwp-go'), close: rect('.bwp-address-editor .bwp-x'), draft: document.querySelector('.bwp-urlinput').value };
    });
    assert.equal(geometry.wrap.top, 24); assert.equal(geometry.wrap.height, 420);
    for (const control of [geometry.input, geometry.go, geometry.close]) { assert.ok(control.top >= 24, JSON.stringify(control)); assert.ok(control.bottom <= 444, JSON.stringify(control)); }
    assert.equal(geometry.draft, 'https://draft.example/path#part');
    await changeViewport(page, { offsetTop: 46 }, 'scroll');
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getBoundingClientRect().top), 46);
    await changeViewport(page, { height: 844, offsetTop: 0 });
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getBoundingClientRect().height), 844);
    assert.equal(await page.$eval('.bwp-urlinput', el => el.value), 'https://draft.example/path#part');
    await page.close();
});

test('leaving and reopening address view removes stale geometry and listeners', async () => {
    const page = await setup();
    await changeViewport(page, { height: 430, offsetTop: 30 });
    await page.click('.bwp-address-editor .bwp-icon2');
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getAttribute('style') || ''), '');
    await changeViewport(page, { height: 350, offsetTop: 60 });
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getAttribute('style') || ''), '');
    await page.evaluate(() => document.querySelector('#fixture-open-address').click());
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getBoundingClientRect().top), 60);
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getBoundingClientRect().height), 350);
    await page.click('.bwp-address-editor .bwp-x');
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getAttribute('style') || ''), '');
    await page.close();
});

test('standalone bookmark view keeps its own search field', async () => {
    const page = await browser.newPage();
    await page.setViewport({ width: 390, height: 844, isMobile: true, hasTouch: true });
    await page.goto(base.replace('?url', '?bookmarks'), { waitUntil: 'networkidle0' });
    assert.ok(await page.$('.bw-omni-input'));
    assert.equal(await page.$('.bwp-urlinput'), null);
    await page.close();
});

test('works without VisualViewport support', async () => {
    const page = await setup(true);
    assert.ok(await page.$('.bwp-urlinput'));
    assert.equal(await page.$eval('.bwp-sheetwrap', el => el.getAttribute('style') || ''), '');
    await page.close();
});
