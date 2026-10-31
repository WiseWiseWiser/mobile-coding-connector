// Run from ai-critic-react: node --test tests/ui/window-drawer.test.mjs
// Set CHROME_PATH if Puppeteer's bundled Chrome is unavailable.
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
    base = `http://127.0.0.1:${server.httpServer.address().port}/tests/ui/window-drawer.html`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });

async function setup(query = '', width = 390) {
    const page = await browser.newPage();
    await page.setViewport({ width, height: 844, isMobile: true, hasTouch: true });
    await page.goto(base + query, { waitUntil: 'networkidle0' });
    await page.waitForSelector('.bwp-winbar');
    await page.click('.bwp-winbar');
    await page.waitForSelector('.bwp-wdrawer');
    return page;
}
const closeAt = (n) => `.bwp-wrowgroup:nth-child(${n}) .bwp-wclose`;
const calls = (page) => page.evaluate(() => window.fixtureCalls);

test('compact controls and 44px per-row targets fit phone widths', async () => {
    for (const width of [320, 390, 414]) {
        const page = await setup('', width);
        const info = await page.evaluate(() => ({
            count: document.querySelectorAll('.bwp-wclose').length,
            plus: document.querySelector('.bwp-wdhead .bwp-mini')?.getAttribute('aria-label'),
            footer: !!document.querySelector('.bwp-wdfoot'),
            overflow: document.documentElement.scrollWidth > innerWidth,
            controls: Array.from(document.querySelectorAll('.bwp-wclose')).map((el) => {
                const r = el.getBoundingClientRect(); return { width: r.width, height: r.height, right: r.right };
            }),
            currentCheck: !!document.querySelector('.bwp-wrow.is-current .bwp-wcheck'),
        }));
        assert.equal(info.count, 4); assert.equal(info.plus, 'New window'); assert.equal(info.footer, false);
        assert.equal(info.overflow, false); assert.equal(info.currentCheck, true);
        for (const r of info.controls) { assert.ok(r.width >= 44 && r.height >= 44); assert.ok(r.right <= width); }
        await page.close();
    }
});

test('window list scrolls while the new button and footer remain visible', async () => {
    const page = await setup('?many', 320);
    const result = await page.$eval('.bwp-wlist', (el) => {
        el.scrollTop = el.scrollHeight;
        return { top: el.scrollTop, height: el.clientHeight, total: el.scrollHeight };
    });
    assert.ok(result.total > result.height && result.top > 0);
    assert.ok(await page.$('.bwp-wdhead .bwp-mini')); assert.ok(await page.$('.bwp-foot'));
    await page.close();
});

test('background close confirms the selected row without switching; cancel restores focus', async () => {
    const page = await setup();
    // Safari taps do not necessarily focus buttons before dispatching click.
    await page.evaluate((selector) => { document.activeElement?.blur(); document.querySelector(selector).click(); }, closeAt(2));
    await page.waitForSelector('.bwp-ccard');
    const text = await page.$eval('.bwp-ccard', (el) => el.textContent);
    assert.match(text, /Background window/); assert.match(text, /2 tabs/); assert.doesNotMatch(text, /next live window becomes current/);
    assert.deepEqual(await calls(page), []);
    await page.click('.bwp-ccancel');
    await page.waitForFunction(() => !document.querySelector('.bwp-ccard'));
    assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('aria-label')), 'Close window Background window with a very long name to test truncation');
    await page.click(closeAt(2)); await page.click('.bwp-cdanger');
    await page.waitForFunction(() => document.querySelectorAll('.bwp-wrow').length === 3);
    assert.deepEqual(await calls(page), [{ session_id: 'background' }]);
    assert.ok(await page.$('.bwp-sheetwrap.is-open .bwp-wdrawer'));
    assert.match(await page.$eval('.bwp-wrow.is-current', (el) => el.textContent), /Current window/);
    await page.close();
});

test('current close targets current session and closes sheet after success', async () => {
    const page = await setup();
    await page.click(closeAt(1));
    assert.match(await page.$eval('.bwp-ccard', (el) => el.textContent), /next live window becomes current/);
    await page.click('.bwp-cdanger');
    await page.waitForFunction(() => !document.querySelector('.bwp-sheetwrap.is-open'));
    assert.deepEqual(await calls(page), [{ session_id: 'current' }]);
    await page.close();
});

test('sessionless row closes by window ID; missing ID is disabled', async () => {
    const page = await setup();
    assert.equal(await page.$eval(closeAt(4), (el) => el.disabled), true);
    assert.match(await page.$eval(closeAt(4), (el) => el.title), /cannot be addressed/);
    await page.click(closeAt(3));
    assert.match(await page.$eval('.bwp-ccard', (el) => el.textContent), /no agent session/);
    await page.click('.bwp-cdanger');
    await page.waitForFunction(() => document.querySelectorAll('.bwp-wrow').length === 3);
    assert.deepEqual(await calls(page), [{ window_id: 33 }]);
    assert.ok(await page.$('.bwp-sheetwrap.is-open .bwp-wdrawer'));
    await page.close();
});

test('rejection remains visible and Escape dismisses without closing sheet', async () => {
    const page = await setup('?failure');
    await page.click(closeAt(2)); await page.click('.bwp-cdanger');
    await page.waitForSelector('.bwp-cerr');
    assert.match(await page.$eval('.bwp-cerr', (el) => el.textContent), /Server rejected close/);
    await page.keyboard.press('Escape');
    await page.waitForFunction(() => !document.querySelector('.bwp-ccard'));
    assert.ok(await page.$('.bwp-sheetwrap.is-open .bwp-wdrawer'));
    assert.deepEqual(await calls(page), [{ session_id: 'background' }]);
    await page.close();
});

test('blockers disable all close controls and expose reasons', async () => {
    for (const [query, reason] of [['?one', /last Chrome window/], ['?readonly', /Read-only/], ['?unsupported', /1\.0\.22/], ['?busy', /Wait/]]) {
        const page = await setup(query);
        const controls = await page.$$eval('.bwp-wclose', (els) => els.map((el) => ({ disabled: el.disabled, title: el.title })));
        assert.ok(controls.length > 0); for (const c of controls) { assert.equal(c.disabled, true); assert.match(c.title, reason); }
        await page.close();
    }
});

test('disconnected current session is not counted as an open window', async () => {
    const page = await setup('?offline');
    assert.match(await page.$eval('.bwp-wdhead .bwp-live', (el) => el.textContent), /0 open/);
    assert.match(await page.$eval('.bwp-shead .bwp-live', (el) => el.textContent), /0 windows/);
    assert.match(await page.$eval('.bwp-wrow', (el) => el.textContent), /disconnected/i);
    assert.doesNotMatch(await page.$eval('.bwp-wrow', (el) => el.textContent), /focused/i);
    assert.equal(await page.$('.bwp-wcheck'), null);
    assert.equal(await page.$eval('.bwp-wclose', (el) => el.disabled), true);
    await page.close();
});

test('new window failure remains visible in the open drawer', async () => {
    const page = await setup('?newfailure');
    await page.click('.bwp-wdhead .bwp-mini');
    await page.waitForSelector('.bwp-wdrawer [role="alert"]');
    assert.match(await page.$eval('.bwp-wdrawer [role="alert"]', el => el.textContent), /Could not open new window/);
    assert.ok(await page.$('.bwp-sheetwrap.is-open .bwp-wdrawer'));
    assert.deepEqual(await calls(page), [{ newWindow: true }]);
    await page.close();
});

test('new window plus invokes creation and closes the sheet on success', async () => {
    const page = await setup();
    await page.click('.bwp-wdhead .bwp-mini');
    await page.waitForFunction(() => !document.querySelector('.bwp-sheetwrap.is-open'));
    assert.deepEqual(await calls(page), [{ newWindow: true }]);
    await page.close();
});
