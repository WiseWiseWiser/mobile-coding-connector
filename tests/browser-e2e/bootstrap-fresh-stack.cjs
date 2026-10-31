// Real built server + /service UI with fake, isolated cloudflared and DNS recovery.
// Nothing is sent to Cloudflare; the only connector is a test-owned child process.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const net = require('node:net');
const https = require('node:https');
const {spawn, spawnSync, execFileSync} = require('node:child_process');
const puppeteer = require('puppeteer');
const binary = process.env.AI_CRITIC_BIN, chrome = process.env.CHROME_PATH;
if (!binary || !chrome) throw Error('AI_CRITIC_BIN and CHROME_PATH are required');
const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'ai-critic-bootstrap-stack-')));
const tmp = fs.realpathSync(os.tmpdir());
if (!root.startsWith(tmp + path.sep + 'ai-critic-bootstrap-stack-')) throw Error('unsafe test root');
const timeout = async (p, ms, name) => { let t; try { return await Promise.race([p,new Promise((_,reject)=>{ t=setTimeout(()=>reject(Error(name+' timed out')),ms); })]); } finally { clearTimeout(t); } };
const pause = ms => new Promise(resolve=>setTimeout(resolve,ms));
const freePort = () => new Promise((resolve,reject)=>{const s=net.createServer();s.on('error',reject);s.listen(0,'127.0.0.1',()=>{const p=s.address().port;s.close(()=>resolve(p));});});
async function waitFor(fn, name) { let last; const until=Date.now()+60000; while(Date.now()<until) { try { if(await fn())return; } catch(e){last=e;} await pause(150); } throw Error(name+': '+(last?.message||'timeout')); }
let app,browser,page,probe,log='',port,domain,managedDir;
(async()=>{try{
    const home=path.join(root,'home'), data=path.join(root,'data'), bin=path.join(root,'bin');
    for(const d of [home,data,bin,path.join(home,'.cloudflared')])fs.mkdirSync(d,{recursive:true});
    const cert=path.join(root,'cert.pem'), key=path.join(root,'key.pem');
    const generated=spawnSync('openssl',['req','-x509','-newkey','rsa:2048','-keyout',key,'-out',cert,'-days','1','-nodes','-subj','/CN=127.0.0.1','-addext','subjectAltName=IP:127.0.0.1'],{stdio:'ignore'});
    if(generated.status!==0)throw Error('could not create isolated test certificate');
    probe=https.createServer({key:fs.readFileSync(key),cert:fs.readFileSync(cert)},(_req,res)=>res.end('pong'));
    await new Promise(resolve=>probe.listen(0,'127.0.0.1',resolve));domain=`127.0.0.1:${probe.address().port}`;
    const id='11111111-1111-1111-1111-111111111111', token='isolated-bootstrap-test-token';
    managedDir=path.join(home,'.cloudflared','managed-tunnels','isolated-bootstrap');
    fs.writeFileSync(path.join(home,'.cloudflared','cert.pem'),'test');fs.writeFileSync(path.join(home,'.cloudflared',id+'.json'),'test');
    fs.writeFileSync(path.join(data,'server-domains.json'),JSON.stringify({domains:[{domain,provider:'cloudflare'}],tunnel_name:'isolated-bootstrap'}));
    const credentials=path.join(root,'credentials');fs.writeFileSync(credentials,token+'\n');
    const ready=path.join(root,'network-ready'), commands=path.join(root,'commands.log'), helper=path.join(root,'cloudflared-fake.cjs');
    fs.writeFileSync(helper,'setInterval(() => {}, 1000);\n');
    const fake=`#!/bin/sh
printf '%s\\n' "$*" >> "$TEST_COMMANDS"
case "$*" in
 'tunnel list --output json') if [ ! -f "$TEST_NETWORK_READY" ]; then echo 'lookup api.cloudflare.com: no such host' >&2; exit 1; fi; echo '[{"id":"${id}","name":"isolated-bootstrap"}]' ;;
 'tunnel info isolated-bootstrap') if [ ! -f "$TEST_NETWORK_READY" ]; then echo 'lookup api.cloudflare.com: no such host' >&2; exit 1; fi; echo 'ID: ${id}' ;;
 'tunnel create isolated-bootstrap') echo 'lookup api.cloudflare.com: no such host' >&2; exit 1 ;;
 'tunnel route dns '*) echo 'Added CNAME' ;;
 'tunnel --config '*" run isolated-bootstrap") exec "$TEST_NODE" "$TEST_HELPER" "$@" ;;
 *) echo "unexpected fake cloudflared command: $*" >&2; exit 1 ;;
esac
`;
    fs.writeFileSync(path.join(bin,'cloudflared'),fake,{mode:0o700});
    port=await freePort();const base=`http://127.0.0.1:${port}`;
    const env={...process.env,HOME:home,AI_CRITIC_HOME:data,AI_CRITIC_NO_OPEN_BROWSER:'1',BROWSER_AGENT_NO_OPEN:'1',PATH:bin+path.delimiter+process.env.PATH,SSL_CERT_FILE:cert,HTTP_PROXY:'',HTTPS_PROXY:'',ALL_PROXY:'',NO_PROXY:'*',TEST_COMMANDS:commands,TEST_NETWORK_READY:ready,TEST_NODE:process.execPath,TEST_HELPER:helper};
    app=spawn(binary,['--port',String(port),'--no-event-bus-publish','--credentials-file',credentials,'--dir',root],{cwd:root,env,stdio:['ignore','pipe','pipe']});
    app.stdout.on('data',s=>{log=(log+s).slice(-30000)});app.stderr.on('data',s=>{log=(log+s).slice(-30000)});
    await waitFor(async()=>{if(app.exitCode!==null)throw Error('test server exited');return (await fetch(base+'/service')).ok},'isolated server');
    browser=await puppeteer.launch({headless:true,executablePath:chrome,userDataDir:path.join(root,'browser-profile'),args:['--no-sandbox']});
    page=await browser.newPage();await page.setViewport({width:390,height:844,isMobile:true,hasTouch:true});await page.setCookie({name:'ai-critic-token',value:token,url:base});
    const requests=[];page.on('request',req=>{if(req.url().includes('/api/domains/tunnel/stop')||req.url().includes('/api/ports?port='))requests.push({url:req.url(),method:req.method()});});
    await page.goto(base+'/service',{waitUntil:'domcontentloaded',timeout:60000});
    await page.waitForFunction(d=>[...document.querySelectorAll('.mcc-port-card')].some(x=>x.textContent.includes(d)&&x.textContent.includes('could not determine tunnel ID')), {timeout:60000},domain);
    fs.writeFileSync(ready,'ready');
    // macOS Go uses the system trust store, so the test's self-signed server
    // must remain untrusted; never install a test CA into the user's keychain.
    await page.waitForFunction(d=>[...document.querySelectorAll('.mcc-port-card')].some(x=>x.textContent.includes(d)&&x.textContent.includes('public health check failed')),{timeout:60000},domain);
    const stateFile=path.join(managedDir,'state.json');
    const state=JSON.parse(fs.readFileSync(stateFile,'utf8'));assert.ok(state.hosts[domain]);assert.ok(state.connector_pid>0);
    const cmd=execFileSync('/bin/ps',['-p',String(state.connector_pid),'-o','command='],{encoding:'utf8'});assert.ok(cmd.includes(helper),'connector must be the isolated test helper');
    const commandOutput=fs.readFileSync(commands,'utf8');assert.match(commandOutput,/tunnel route dns --overwrite-dns isolated-bootstrap/);assert.doesNotMatch(commandOutput,/mcc-/);
    const stoppedResponse = page.waitForResponse(res=>res.request().method()==='POST' && res.url().endsWith('/api/domains/tunnel/stop'),{timeout:30000});
    await page.evaluate(d=>[...document.querySelectorAll('.mcc-port-card')].find(x=>x.querySelector('.mcc-port-label')?.textContent===d).querySelector('.mcc-port-stop').click(),domain);
    assert.equal((await stoppedResponse).status(),200);
    await page.waitForFunction(d=>[...document.querySelectorAll('.mcc-port-card')].some(x=>x.querySelector('.mcc-port-label')?.textContent===d&&x.querySelector('.mcc-port-status')?.textContent==='🔴'&&x.querySelector('.mcc-port-stop')?.disabled&&!x.querySelector('.mcc-port-url-error')),{timeout:60000},domain);
    assert.ok(requests.some(x=>x.method==='POST'&&x.url.endsWith('/api/domains/tunnel/stop')));assert.ok(!requests.some(x=>x.method==='DELETE'));
    assert.equal(JSON.parse(fs.readFileSync(stateFile,'utf8')).hosts[domain],undefined);
    console.log('PASS isolated built /service: offline bootstrap route retries; live UI reflects untrusted test TLS; Stop targets only its domain');
}catch(error){console.error(error);if(page)console.error('UI:',(await page.evaluate(()=>document.body.innerText).catch(()=>'' )).slice(0,1600));console.error('TEST SERVER LOG:',log);process.exitCode=1;}
finally{
    if(browser){try{await timeout(browser.close(),5000,'browser cleanup')}catch{browser.process()?.kill('SIGKILL')}}
    if(app && app.exitCode===null && app.signalCode===null){app.kill('SIGTERM');await timeout(new Promise(resolve=>app.once('exit',resolve)),5000,'server cleanup').catch(()=>app.kill('SIGKILL'));}
    if(managedDir){try{const st=JSON.parse(fs.readFileSync(path.join(managedDir,'state.json'),'utf8'));const pid=st.connector_pid;if(pid>0){const cmd=execFileSync('/bin/ps',['-p',String(pid),'-o','command='],{encoding:'utf8'});if(cmd.includes(path.join(root,'cloudflared-fake.cjs')))process.kill(pid,'SIGTERM');else console.error('Not stopping unverified connector',pid)}}catch{}}
    if(probe){probe.closeAllConnections();await new Promise(resolve=>probe.close(resolve))}
    fs.rmSync(root,{recursive:true,force:true});
}})().then(()=>process.exit(process.exitCode||0),e=>{console.error(e);process.exit(1)});
