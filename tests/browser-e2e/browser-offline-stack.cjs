// Isolated full-page integration: fake daemon, isolated ai-critic, temporary headless Chrome.
// BROWSER_AGENT_NO_OPEN prevents opening the user's normal Chrome profile.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const http = require('node:http');
const net = require('node:net');
const { spawn } = require('node:child_process');
const puppeteer = require('puppeteer');
const binary = process.env.AI_CRITIC_BIN, chrome = process.env.CHROME_PATH;
if (!binary || !chrome) throw Error('AI_CRITIC_BIN and CHROME_PATH are required');
const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'ai-critic-offline-stack-')));
const temp = fs.realpathSync(os.tmpdir());
if (!root.startsWith(temp + path.sep + 'ai-critic-offline-stack-')) throw Error('unsafe test root');
const timeout = async (p, ms, name) => { let timer; try { return await Promise.race([p, new Promise((_, reject) => { timer = setTimeout(() => reject(Error(name + ' timed out')), ms); })]); } finally { clearTimeout(timer); } };
const freePort = () => new Promise((resolve, reject) => { const server = net.createServer(); server.on('error', reject); server.listen(0, '127.0.0.1', () => { const p = server.address().port; server.close(() => resolve(p)); }); });
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function waitFor(fn, name) { const end = Date.now() + 60000; let last; while (Date.now() < end) { try { if (await fn()) return; } catch(e) { last = e; } await sleep(200); } throw Error(`${name}: ${last?.message || 'timeout'}`); }
let connected = false, browser, child, page, log = '', ensureRequests = 0;
const session = () => ({ session_id: 'ai-critic', phase: connected ? 'extension_connected' : 'waiting_extension', extension: { connected }, session_page_count: 1 });
const fake = http.createServer(async (req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1'); let data = { ok: true };
    if (url.pathname === '/v1/health') data = { ok: true, daemon_version: '1.0.21' };
    else if (url.pathname === '/v1/sessions' && req.method === 'GET') data = [session()];
    else if (url.pathname === '/v1/session') data = session();
    else if (url.pathname === '/v1/jobs') {
        let body = ''; for await (const part of req) body += part; const request = JSON.parse(body);
        if (!connected) data = { ok: false, error: 'extension disconnected' };
        else if (request.type === 'info') data = { ok:true, data:{window_id:900, features:['browser-agent'], tabs:[{id:1,url:`http://127.0.0.1:${fake.address().port}/go?session=ai-critic`,role:'session_page'},{id:2,url:'data:text/html,temporary',title:'Test page',role:'user',active:true}]} };
        else if (request.type === 'window') data = { ok:true, data:{window_id:900,window:{left:0,top:0,width:1200,height:900},viewport:{width:1200,height:900,dpr:1}} };
        else if (request.type === 'eval') data = { ok:true, data:{value:'{"w":1200,"h":900,"dpr":1}'} };
        else if (request.type === 'screenshot') data = { ok:true, data:{base64:'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZ0AAAAASUVORK5CYII='} };
        else data = { ok:true, data:{} };
    }
    res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(data));
});
(async () => {
 try {
    await new Promise((resolve) => fake.listen(0, '127.0.0.1', resolve)); const port = await freePort();
    const credentials = path.join(root, 'credentials'); fs.writeFileSync(credentials, 'isolated-test-token\n'); const home = path.join(root, 'home'); fs.mkdirSync(home);
    child = spawn(binary, ['--port',String(port),'--no-event-bus-publish','--dir',root,'--credentials-file',credentials], { env: {...process.env, AI_CRITIC_HOME:home, BROWSER_AGENT_PORT:String(fake.address().port), BROWSER_AGENT_NO_OPEN:'1'}, stdio:['ignore','pipe','pipe'] });
    child.stdout.on('data', (s) => { log = (log + s).slice(-20000); }); child.stderr.on('data', (s) => { log = (log + s).slice(-20000); });
    const base = `http://127.0.0.1:${port}`; await waitFor(async () => (await fetch(base + '/browser')).ok, 'app startup');
    browser = await puppeteer.launch({ headless:true, executablePath:chrome, userDataDir:path.join(root,'chrome-profile'), args:['--no-sandbox'] });
    page = await browser.newPage(); await page.setViewport({width:390,height:844,isMobile:true,hasTouch:true});
    await page.setCookie({name:'ai-critic-token',value:'isolated-test-token',url:base});
    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith('/api/browser/ensure')) { ensureRequests++; setTimeout(() => { connected = true; }, 600); } });
    await page.goto(base + '/browser', {waitUntil:'domcontentloaded',timeout:60000});
    await page.waitForSelector('.bw-offline-start', {timeout:60000});
    assert.equal(ensureRequests, 0); assert.equal(connected, false);
    assert.match(await page.$eval('.bw-offline', (el) => el.textContent), /No connected Chrome window/);
    await page.click('.bw-offline-start');
    await page.waitForFunction(() => !document.querySelector('.bw-offline'), {timeout:60000});
    assert.equal(ensureRequests, 1); assert.equal(connected, true);
    console.log('PASS actual /browser: stale session stays offline without starting, one click attaches simulated extension');
 } catch(error) {
    console.error(error); if(page) console.error('UI:', (await page.evaluate(() => document.body.innerText).catch(() => '')).slice(0, 1500));
    console.error('SERVER LOG:', log); process.exitCode=1;
 } finally {
    if(browser) { try { await timeout(browser.close(),5000,'browser cleanup'); } catch { browser.process()?.kill('SIGKILL'); } }
    if(child && child.exitCode === null && child.signalCode === null) { child.kill('SIGTERM'); await timeout(new Promise((resolve)=>child.once('exit',resolve)),3000,'app cleanup').catch(()=>child.kill('SIGKILL')); }
    fake.closeAllConnections(); fake.close(); fs.rmSync(root,{recursive:true,force:true});
 }
})().then(()=>process.exit(process.exitCode||0), (error)=>{console.error(error);process.exit(1)});
