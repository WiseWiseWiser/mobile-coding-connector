import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import puppeteer from 'puppeteer';
const root = fileURLToPath(new URL('../../', import.meta.url));
let server, browser, base;
before(async () => {
    server = await createServer({ root, server: { host: '127.0.0.1', port: 0, strictPort: false }, logLevel: 'error' }); await server.listen();
    base = `http://127.0.0.1:${server.httpServer.address().port}/tests/ui/browser-offline.html`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });
test('offline screen does not launch until tapped and rejects double clicks', async () => {
    const page = await browser.newPage(); await page.setViewport({ width: 390, height: 844 }); let calls = 0;
    await page.setRequestInterception(true);
    page.on('request', (req) => {
        if (!req.url().endsWith('/api/browser/ensure')) { void req.continue(); return; }
        calls++; setTimeout(() => void req.respond({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) }), 300);
    });
    await page.goto(base, { waitUntil: 'networkidle0' }); assert.equal(calls, 0);
    assert.match(await page.$eval('.bw-offline', (el) => el.textContent), /No connected Chrome window/);
    await page.evaluate(() => { const b = document.querySelector('.bw-offline-start'); b.click(); b.click(); });
    await page.waitForFunction(() => document.querySelector('.bw-offline-start')?.disabled);
    await page.waitForFunction(() => document.querySelector('.bw-stage')?.getAttribute('data-refreshes') === '1');
    assert.equal(calls, 1); await page.close();
});
test('offline action is visible at phone and desktop widths', async () => {
    for (const width of [320, 390, 1024]) {
        const page = await browser.newPage(); await page.setViewport({ width, height: 844 });
        await page.goto(base, { waitUntil: 'networkidle0' });
        const r = await page.$eval('.bw-offline-start', (el) => { const r = el.getBoundingClientRect(); return { width:r.width, height:r.height, right:r.right, left:r.left }; });
        assert.ok(r.width >= 44 && r.height >= 44 && r.left >= 0 && r.right <= width);
        await page.close();
    }
});
test('a failed startup exposes the error and offers explicit retry', async () => {
    const page = await browser.newPage(); let calls = 0;
    await page.setRequestInterception(true);
    page.on('request', (req) => {
        if (!req.url().endsWith('/api/browser/ensure')) { void req.continue(); return; }
        calls++; void req.respond({ status: 500, contentType: 'application/json', body: JSON.stringify({ ok: false, error: 'Chrome could not start' }) });
    });
    await page.goto(base, { waitUntil: 'networkidle0' }); await page.click('.bw-offline-start');
    await page.waitForSelector('.bw-offline-error');
    assert.match(await page.$eval('.bw-offline-error', (el) => el.textContent), /Chrome could not start/);
    assert.equal(await page.$eval('.bw-offline-start', (el) => el.textContent), 'Retry'); assert.equal(calls, 1);
    await page.close();
});
