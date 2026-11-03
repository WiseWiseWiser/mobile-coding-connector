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
    base = `http://127.0.0.1:${server.httpServer.address().port}/tests/ui/port-target.html`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });

async function pageWithAPI() {
    const page = await browser.newPage();
    const calls = [];
    await page.setRequestInterception(true);
    page.on('request', req => {
        if (!new URL(req.url()).pathname.startsWith('/api/ports')) { void req.continue(); return; }
        const body = req.postData() ? JSON.parse(req.postData()) : undefined;
        calls.push({ method: req.method(), url: new URL(req.url()), body });
        const response = req.url().endsWith('/target-check')
            ? { target: body.target, tcpReachable: false, error: 'connection refused from Pi' }
            : { id: 'map_mac', status: 'active' };
        void req.respond({ status: 200, contentType: 'application/json', body: JSON.stringify(response) });
    });
    await page.goto(base, { waitUntil: 'networkidle0' });
    return { page, calls };
}

test('scoped IPv6 preview and diagnostic use the server target, not local port discovery', async () => {
    const { page, calls } = await pageWithAPI();
    await page.$eval('[data-testid="fields"] input', input => { const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set; setter.call(input, 'fe80::1234%eth0'); input.dispatchEvent(new Event('input', { bubbles: true })); });
    await page.waitForFunction(() => document.querySelector('[data-testid="fields"] code').textContent.includes('[fe80::1234%25eth0]:3080'));
    await page.click('[data-testid="fields"] button');
    await page.waitForFunction(() => document.querySelector('[data-testid="fields"] [role="status"]')?.textContent.includes('connection refused'));
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url.pathname, '/api/ports/target-check');
    assert.deepEqual(calls[0].body.target, { host: 'fe80::1234%eth0', scheme: 'http', port: 3080 });
    await page.close();
});

test('editing and deleting a same-port Mac mapping uses its ID and preserves auth', async () => {
    const { page, calls } = await pageWithAPI();
    await page.click('[data-testid="map_mac"] .mcc-port-actions button');
    await page.waitForSelector('[aria-label="Edit mapping target"]');
    await page.evaluate(() => { const b = [...document.querySelectorAll('[aria-label="Edit mapping target"] button')].find(b => b.textContent === 'Save mapping'); b.click(); });
    await page.waitForFunction(() => !document.querySelector('[aria-label="Edit mapping target"]'));
    const update = calls.find(c => c.method === 'PUT');
    assert.equal(update.url.searchParams.get('id'), 'map_mac');
    assert.equal(update.body.target.host, '192.168.1.1');
    assert.equal(update.body.target.port, 3080);
    assert.equal(Object.hasOwn(update.body, 'auth'), false);
    assert.equal(Object.hasOwn(update.body, 'provider'), false);
    await page.click('[data-testid="map_mac"] .mcc-port-stop');
    await page.waitForNetworkIdle();
    const remove = calls.find(c => c.method === 'DELETE');
    assert.equal(remove.url.searchParams.get('id'), 'map_mac');
    assert.equal(remove.url.searchParams.has('port'), false);
    assert.equal(await page.$eval('[data-testid="map_pi"] code', el => el.textContent.includes('127.0.0.1')), true);
    await page.close();
});
