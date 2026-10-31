// Direct-window variant of new-window-stack.cjs: the fake browser-agent
// advertises direct_supported and has NO live sessions — only plain Chrome
// windows reachable through the instance rail. The mobile /browser UI must
// list those windows (sess-window-<id> rows), and switching to one must drive
// it with window_id-addressed jobs, creating zero sessions and zero /go pages.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const net = require('node:net');
const http = require('node:http');
const {spawn} = require('node:child_process');
const puppeteer = require('puppeteer');
const binary = process.env.AI_CRITIC_BIN, chrome = process.env.CHROME_PATH;
if (!binary || !chrome) throw Error('AI_CRITIC_BIN and CHROME_PATH are required');
const root=fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(),'ai-critic-direct-window-stack-'))),tmp=fs.realpathSync(os.tmpdir());
if(!root.startsWith(tmp+path.sep+'ai-critic-direct-window-stack-'))throw Error('unsafe test root');
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
const freePort=()=>new Promise((resolve,reject)=>{const s=net.createServer();s.on('error',reject);s.listen(0,'127.0.0.1',()=>{const p=s.address().port;s.close(()=>resolve(p));});});
async function waitFor(fn,label){const end=Date.now()+45000;let last;while(Date.now()<end){try{if(await fn())return;}catch(e){last=e}await sleep(100)}throw Error(`${label}: ${last?.message||'timeout'}`)}
let fake,app,browser,page,log='', directJobs=[], sessionCreates=0, goNavigations=[];
// Two plain windows, no sessions at all. Window 11 is focused with a real
// page; window 12 has only chrome:// (driving it must create a content tab).
const WINDOWS=[
 {window_id:11,type:'normal',focused:true,width:1280,height:900,state:'normal',title:'Fixture Docs',session_ids:[],tabs:[
   {tab_id:111,index:0,title:'Fixture Docs',url:'https://docs.example.test/start',active:true,capturable:true,role:'user'}]},
 {window_id:12,type:'normal',focused:false,width:900,height:700,state:'normal',title:'Settings',session_ids:[],tabs:[
   {tab_id:121,index:0,title:'Settings',url:'chrome://settings',active:true,capturable:false,role:'user'}]},
];
const windowTabs=new Map([ [11,JSON.parse(JSON.stringify(WINDOWS[0].tabs))], [12,JSON.parse(JSON.stringify(WINDOWS[1].tabs))] ]);
fake=http.createServer(async(req,res)=>{try{let raw='';for await(const p of req)raw+=p;const u=new URL(req.url,'http://127.0.0.1'),body=raw?JSON.parse(raw):{};let out={ok:true};
 if(u.pathname==='/v1/health')out={ok:true,daemon_version:'1.0.24',direct_supported:true};
 else if(u.pathname==='/v1/sessions'&&req.method==='GET')out=[];
 else if(u.pathname==='/v1/sessions'&&req.method==='POST'){sessionCreates++;res.statusCode=409;out={error:'session already exists'}}
 else if(u.pathname==='/v1/session'){res.statusCode=404;out={error:'unknown session'}}
 else if(u.pathname==='/v1/windows'){out={type:'windows',version:'1.0.24',features:['all-windows','window-close'],windows:WINDOWS.map(w=>({...w,tabs:windowTabs.get(w.window_id)}))}}
 else if(u.pathname==='/v1/jobs'){
   if(body.session_id){out={ok:false,error:'direct fixture: no sessions exist'};}
   else if(!body.window_id){out={ok:false,error:'direct fixture: window_id required'};}
   else{const win=body.window_id,tabs=windowTabs.get(win);directJobs.push({window_id:win,type:body.type,tab_id:body.tab_id||null});let data={};
     if(body.type==='info')data={type:'info',window_id:win,direct:true,tabs:(tabs||[]).map(t=>({id:t.tab_id,index:t.index+1,title:t.title,url:t.url,active:t.active,role:'user'}))};
     else if(body.type==='window'){if(body.params?.op==='resize')data={viewport:{width:1200,height:800,dpr:1}};else data={window_id:win,window:{left:0,top:0,width:1280,height:1000}};}
     else if(body.type==='eval')data={value:'{"w":1200,"h":800,"dpr":1}'};
     else if(body.type==='create_tab'){const id=win*100+1;windowTabs.set(win,[...(tabs||[]),{tab_id:id,index:(tabs||[]).length,title:'ai-critic',url:'data:text/html,fixture',active:true,capturable:true,role:'user'}]);data={tab_id:id,window_id:win,url:'data:text/html,fixture'};}
     else if(body.type==='screenshot')data={base64:'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZ0AAAAASUVORK5CYII='};
     else if(body.type==='cdp'&&body.params?.method==='Page.navigate'){const url=body.params.params.url;const t=windowTabs.get(win)||[];const act=t.find(x=>x.active);if(act){act.url=url}data={result:{}};}
     out={ok:true,data};}
 }
 res.setHeader('Content-Type','application/json');res.end(JSON.stringify(out));}catch(e){res.statusCode=500;res.end(JSON.stringify({error:String(e)}))}});
(async()=>{try{
 await new Promise(r=>fake.listen(0,'127.0.0.1',r));const port=await freePort(),base=`http://127.0.0.1:${port}`,home=path.join(root,'home'),data=path.join(root,'data');fs.mkdirSync(home);fs.mkdirSync(data);
 fs.writeFileSync(path.join(data,'browser-config.json'),JSON.stringify({preset:'mac',mode:'shot'}));const credentials=path.join(root,'credentials');fs.writeFileSync(credentials,'test-direct-token\n');
 app=spawn(binary,['--port',String(port),'--no-event-bus-publish','--dir',root,'--credentials-file',credentials],{cwd:root,env:{...process.env,HOME:home,AI_CRITIC_HOME:data,AI_CRITIC_NO_OPEN_BROWSER:'1',BROWSER_AGENT_NO_OPEN:'1',BROWSER_AGENT_PORT:String(fake.address().port)},stdio:['ignore','pipe','pipe']});app.stdout.on('data',x=>log=(log+x).slice(-20000));app.stderr.on('data',x=>log=(log+x).slice(-20000));
 await waitFor(async()=>{if(app.exitCode!==null)throw Error('app exited');return (await fetch(base+'/browser')).ok},'server startup');
 browser=await puppeteer.launch({headless:true,executablePath:chrome,userDataDir:path.join(root,'chrome-profile'),args:['--no-sandbox']});page=await browser.newPage();await page.setViewport({width:390,height:844,isMobile:true,hasTouch:true});await page.setCookie({name:'ai-critic-token',value:'test-direct-token',url:base});
 page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/api/browser/session')){sessionCreates++}});
 await page.goto(base+'/browser',{waitUntil:'domcontentloaded',timeout:60000});
 // The rail comes up straight on window 11 (focused): no /go, no session.
 await page.waitForFunction(()=>document.querySelector('.bwp-sessname')?.textContent.includes('sess-window-11')||document.querySelector('.bwp-title')?.textContent.includes('Fixture Docs'),{timeout:45000});
 const status0=await (await fetch(base+'/api/browser/status',{headers:{cookie:'ai-critic-token=test-direct-token'}})).json();
 assert.equal(status0.session.window_id,11,'focused window 11 must be driven');
 assert.ok(status0.session.content_tab_id===111,'active tab 111 must be the content tab');
 assert.equal(status0.session.url,'https://docs.example.test/start');
 assert.equal(sessionCreates,0,'no session may be created in direct mode');
 assert.equal(goNavigations.length,0);
 // Drawer lists both windows as sessionless rows.
 await page.click('.bwp-title');await page.waitForSelector('.bwp-winbar');await page.evaluate(()=>document.querySelector('.bwp-winbar').click());await page.waitForSelector('.bwp-wdrawer');
 await page.waitForFunction(()=>document.querySelector('.bwp-wdrawer')?.textContent.includes('Settings'),{timeout:45000});
 const drawerText=await page.evaluate(()=>document.querySelector('.bwp-wdrawer')?.innerText||'');
 assert.ok(drawerText.includes('Fixture Docs'),'drawer should show window 11 title');
 // Attach window 12 through the direct switch (chrome:// only → placeholder tab).
 const attachResponse=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/api/browser/window/attach'),{timeout:60000});
 await page.evaluate(()=>{const rows=[...document.querySelectorAll('.bwp-wdrawer .bwp-wrow')];const row=rows.find(x=>x.textContent.includes('Settings'));if(!row)throw Error('window 12 row not found: '+rows.map(x=>x.textContent).join('|'));const main=row.querySelector('.bwp-wrowmain');if(!main)throw Error('row main button missing');main.click();});
 const attach=await attachResponse;assert.equal(attach.status(),200);
 await page.waitForFunction(()=>document.querySelector('.bwp-winbar')?.textContent.includes('sess-window-12'),{timeout:45000});
 const status12=await (await fetch(base+'/api/browser/status',{headers:{cookie:'ai-critic-token=test-direct-token'}})).json();
 assert.equal(status12.session.window_id,12,'window 12 must be driven after attach');
 assert.equal(sessionCreates,0,'attach must not register a session in direct mode');
 assert.ok(directJobs.some(j=>j.type==='create_tab'&&j.window_id===12),'placeholder tab must be created in window 12');
 // Navigate the window-12 content tab; the job must be window-addressed.
 await page.evaluate(()=>document.querySelector('.bwp-shead .bwp-x')?.click());await page.waitForFunction(()=>!document.querySelector('.bwp-sheetwrap.is-open'));
 await page.evaluate(()=>document.querySelector('.bwp-urlcap').click());await page.waitForSelector('.bwp-address-editor');await page.type('.bwp-urlinput','https://example.test/direct');
 const navResponse=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/api/browser/navigate'),{timeout:30000});await page.click('.bwp-go');assert.equal((await navResponse).status(),200);
 await waitFor(()=>directJobs.some(j=>j.type==='cdp'&&j.window_id===12),'navigate job window-addressed');
 assert.ok(directJobs.every(j=>!j.session),'every direct job must be window-addressed, never session-addressed');
 console.log('PASS direct rail: windows listed sessionless, attach+drive window 12 with zero sessions and zero /go pages');
}catch(e){console.error(e);if(page)console.error('SESSIONS:',await page.evaluate(async()=>{const r=await fetch('/api/browser/sessions');return await r.text()}).catch(e2=>'fetch fail: '+String(e2)));if(page)console.error('UI:',(await page.evaluate(()=>document.body.innerText).catch(()=>'' )).slice(0,3000));console.error('APP LOG:',log);console.error('directJobs:',JSON.stringify(directJobs));process.exitCode=1}
 finally{if(browser)await browser.close().catch(()=>{});if(app&&app.exitCode===null&&app.signalCode===null){app.kill('SIGTERM');await Promise.race([new Promise(r=>app.once('exit',r)),sleep(4000)]);if(app.exitCode===null&&app.signalCode===null)app.kill('SIGKILL')}fake.closeAllConnections();await new Promise(r=>fake.close(r));fs.rmSync(root,{recursive:true,force:true})}
})().then(()=>process.exit(process.exitCode||0),e=>{console.error(e);process.exit(1)});
