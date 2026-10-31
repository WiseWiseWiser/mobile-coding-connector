import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserAddressEditor } from '../../src/v2/mcc/browser/BrowserAddressEditor';
import { BrowserPhoneTop } from '../../src/v2/mcc/browser/BrowserPhoneTop';
import type { BrowserTab } from '../../src/v2/mcc/browser/browserWire';
import '../../src/v2/mcc/browser/browser.css';

const calls: unknown[] = [];
Object.assign(window, { fixtureCalls: calls });
const tabs: BrowserTab[] = [
    { tab_id: 10, title: 'First', url: 'https://first.example/a?q=1#section', active: true, role: 'user' },
    { tab_id: 20, title: 'Second', url: 'https://second.example/page#anchor', active: false, role: 'user' },
];
const doc = {version:1, roots:[{type:'folder',id:'root',name:'Bookmarks',children:[{type:'url',id:'b1',name:'Docs',url:'https://docs.example/start'},{type:'folder',id:'folder',name:'Projects',children:[{type:'url',id:'b2',name:'GitHub',url:'https://github.com/'}]}]}]};
const localDoc = {version:1, roots:[{type:'folder',id:'local',name:'Local',children:[{type:'url',id:'l1',name:'Local Docs',url:'https://local.example/docs'}]}]};
function Fixture() {
    const [open,setOpen] = useState(false);
    const [active,setActive] = useState(10);
    const [session,setSession] = useState('session-a');
    const [showTabs,setShowTabs] = useState(true);
    const [source,setSource] = useState<'chrome'|'local'>('chrome');
    const current = tabs.find(t => t.tab_id === active)!;
    return <div className="bw-page bw-phone"><div className="bw-main">
        <BrowserPhoneTop session={session} sessionless={false} title={current.title ?? ''} mode="shot" url={current.url} readOnly={false} open={open} onBack={() => {}} onSheet={() => {}} onAddress={() => setOpen(true)} />
        <button id="change-session" onClick={() => setSession('session-b')}>Change session</button>
        <button id="remove-tab" onClick={() => setShowTabs(false)}>Remove tab</button>
        {open && <div className="bwp-sheetwrap is-open"><section className="bwp-sheet"><BrowserAddressEditor session={session} tabs={showTabs ? tabs : []} activeTabId={active} currentUrl={current.url} disabled={false} bookmarkDoc={source === 'chrome' ? doc : localDoc} bookmarksLoading={false} bookmarksError={null} bookmarkSource={source} onShowLocal={() => setSource('local')} onShowChrome={() => setSource('chrome')} onBack={() => setOpen(false)} onClose={() => setOpen(false)} onSelectTab={async (s,t) => { calls.push({select:s,tab:t}); setActive(t); return null; }} onNavigate={async (s,t,url) => { calls.push({navigate:s,tab:t,url}); return url === 'https://fail.example' ? 'Navigation rejected' : null; }} /></section></div>}
    </div></div>;
}
createRoot(document.getElementById('root')!).render(<Fixture />);
