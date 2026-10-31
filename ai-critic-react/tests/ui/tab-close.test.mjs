// Run from ai-critic-react: CHROME_PATH=... node --test tests/ui/tab-close.test.mjs
import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import puppeteer from 'puppeteer';
const root = fileURLToPath(new URL('../../', import.meta.url));
let server, browser, base;
before(async () => {
    server = await createServer({ root, server: { host: '127.0.0.1', port: 0, strictPort: false }, logLevel: 'error' });
    await server.listen();
    base = `http://127.0.0.1:${server.httpServer.address().port}/tests/ui/tab-close.html`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });
async function setup(query = '', width = 390) {
    const page = await browser.newPage();
    await page.setViewport({ width, height: 844 });
    await page.goto(base + query, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('.bwp-tclose');
    return page;
}
const calls = (page) => page.evaluate(() => window.fixtureCalls);

test('dirty tab presents safe modal at mobile and desktop widths', async () => {
    for (const width of [320, 390, 1024]) {
        const page = await setup('', width);
        await page.click('.bwp-tclose');
        await page.waitForSelector('.bw-tabclose-card');
        const info = await page.evaluate(() => {
            const card = document.querySelector('.bw-tabclose-card'), rect = card.getBoundingClientRect();
            return { text: card.textContent, left: rect.left, right: rect.right, focus: document.activeElement?.textContent, inert: document.querySelector('main')?.inert };
        });
        assert.match(info.text, /Editor with unsaved changes/); assert.match(info.text, /may discard unsaved changes/);
        assert.ok(info.left >= 0 && info.right <= width); assert.equal(info.focus, 'Keep tab'); assert.equal(info.inert, true);
        await page.click('.bw-tabclose-actions button');
        await page.waitForFunction(() => !document.querySelector('.bw-tabclose-card'));
        assert.equal(await page.evaluate(() => document.activeElement?.className), 'bwp-tclose');
        assert.deepEqual(await calls(page), [{ session: 'session-1', tabId: 7 }, { session: 'session-1', operationId: 'operation-1', accept: false }]);
        assert.equal(await page.$$eval('.bwp-trow', (rows) => rows.length), 2);
        await page.close();
    }
});

test('discard sends explicit true and removes only the selected tab', async () => {
    const page = await setup(); await page.click('.bwp-tclose'); await page.waitForSelector('.bw-tabclose-card');
    await page.click('.bw-tabclose-actions .is-danger'); await page.waitForFunction(() => !document.querySelector('.bw-tabclose-card'));
    assert.deepEqual(await calls(page), [{ session: 'session-1', tabId: 7 }, { session: 'session-1', operationId: 'operation-1', accept: true }]);
    assert.deepEqual(await page.$$eval('.bwp-tname', (els) => els.map((el) => el.textContent)), ['Other tab']);
    await page.close();
});

test('Escape chooses keep, never discard', async () => {
    const page = await setup(); await page.click('.bwp-tclose'); await page.waitForSelector('.bw-tabclose-card');
    await page.keyboard.press('Escape'); await page.waitForFunction(() => !document.querySelector('.bw-tabclose-card'));
    assert.equal((await calls(page))[1].accept, false); await page.close();
});

test('ambiguous failure does not offer a second destructive choice', async () => {
    const page = await setup('?failure'); await page.click('.bwp-tclose'); await page.waitForSelector('.bw-tabclose-card');
    await page.click('.bw-tabclose-actions .is-danger');
    await page.waitForFunction(() => document.querySelector('.bw-tabclose-card')?.textContent.includes('outcome is unknown'));
    assert.equal(await page.$('.bw-tabclose-actions .is-danger'), null);
    assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Dismiss');
    await page.click('.bw-tabclose-actions button'); assert.equal((await calls(page)).length, 2); await page.close();
});

test('old extension timeout reports unknown without retrying', async () => {
    const page = await setup('?unknown'); await page.click('.bwp-tclose');
    await page.waitForSelector('.bw-tabclose-card');
    assert.match(await page.$eval('.bw-tabclose-card', (el) => el.textContent), /outcome is unknown/);
    assert.equal(await page.$('.bw-tabclose-actions .is-danger'), null); assert.equal((await calls(page)).length, 1);
    await page.close();
});

test('network failure prompts a check rather than retrying close', async () => {
    const page = await setup('?network'); await page.click('.bwp-tclose');
    await page.waitForSelector('.bw-tabclose-card');
    assert.match(await page.$eval('.bw-tabclose-card', (el) => el.textContent), /Check Chrome and refresh/);
    assert.equal(await page.$('.bw-tabclose-actions .is-danger'), null); assert.equal((await calls(page)).length, 1);
    await page.close();
});

test('clean tab closes without showing confirmation', async () => {
    const page = await setup('?clean'); await page.click('.bwp-tclose');
    await page.waitForFunction(() => document.querySelectorAll('.bwp-trow').length === 1);
    assert.equal(await page.$('.bw-tabclose-card'), null); assert.equal((await calls(page)).length, 1); await page.close();
});
