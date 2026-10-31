import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import puppeteer from 'puppeteer';

const root = fileURLToPath(new URL('../../', import.meta.url));
let server, browser, base;
before(async () => {
    server = await createServer({ root, server: { host: '127.0.0.1', port: 0 }, logLevel: 'error' });
    await server.listen();
    base = `http://127.0.0.1:${server.httpServer.address().port}/tests/ui/bootstrap-port.html`;
    browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH || undefined, args: ['--no-sandbox'] });
});
after(async () => { await browser?.close(); await server?.close(); });

test('bootstrap stop targets its domain and leaves normal port removal alone', async () => {
    const page = await browser.newPage();
    const calls = [];
    await page.setRequestInterception(true);
    page.on('request', req => {
        if (!req.url().endsWith('/api/domains/tunnel/stop')) { void req.continue(); return; }
        calls.push({ method:req.method(), body:JSON.parse(req.postData()) });
        setTimeout(() => void req.respond({ status:200, contentType:'application/json', body:'{"status":"stopped"}' }), 250);
    });
    await page.goto(base, { waitUntil:'networkidle0' });
    assert.match(await page.$eval('[data-testid="bootstrap"]', el => el.textContent), /DNS unavailable/);
    await page.evaluate(() => { const b = document.querySelector('[data-testid="bootstrap"] .mcc-port-stop'); b.click(); b.click(); });
    await page.waitForFunction(() => document.querySelector('[data-testid="bootstrap"] .mcc-port-stop')?.disabled);
    await page.waitForFunction(() => !document.querySelector('[data-testid="bootstrap"] .mcc-port-stop')?.disabled);
    assert.deepEqual(calls, [{ method:'POST', body:{ domain:'bootstrap.example' } }]);
    assert.equal(await page.$eval('[data-testid="removed"]', el => el.textContent), '0');
    await page.click('[data-testid="normal"] .mcc-port-stop');
    assert.equal(await page.$eval('[data-testid="removed"]', el => el.textContent), '1');
    await page.close();
});

test('bootstrap stop exposes backend errors', async () => {
    const page = await browser.newPage();
    await page.setRequestInterception(true);
    page.on('request', req => {
        if (!req.url().endsWith('/api/domains/tunnel/stop')) { void req.continue(); return; }
        void req.respond({ status:500, contentType:'application/json', body:'{"error":"cannot stop selected tunnel"}' });
    });
    await page.goto(base, { waitUntil:'networkidle0' });
    await page.click('[data-testid="bootstrap"] .mcc-port-stop');
    await page.waitForSelector('[data-testid="bootstrap"] [role="alert"]');
    assert.match(await page.$eval('[data-testid="bootstrap"] [role="alert"]', el => el.textContent), /cannot stop selected tunnel/);
    await page.close();
});
