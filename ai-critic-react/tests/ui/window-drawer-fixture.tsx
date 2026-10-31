import { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserSheet } from '../../src/v2/mcc/browser/BrowserSheet';
import type { SheetView } from '../../src/v2/mcc/browser/BrowserSheet';
import type { SessionInfo } from '../../src/v2/mcc/browser/browserWire';
import '../../src/v2/mcc/browser/browser.css';

const params = new URLSearchParams(location.search);
const tab = (id: number, title: string) => ({ tab_id: id, title, active: false, role: 'user' });
const initial: SessionInfo[] = [
    { id: 'current', title: 'Current window', current: true, attached: true, attachable: true, window_id: 11, tabs: [tab(1, 'A')] },
    { id: 'background', title: 'Background window with a very long name to test truncation', current: false, attached: true, attachable: true, window_id: 22, tabs: [tab(2, 'B'), tab(3, 'C')] },
    { id: 'sess-window-33', title: 'Unattached window', current: false, attached: false, attachable: false, sessionless: true, adoptable: true, window_id: 33, tabs: [tab(4, 'D')] },
    { id: 'missing', title: 'Missing window ID', current: false, attached: false, attachable: false, sessionless: true, adoptable: false },
];
if (params.has('offline')) initial.splice(0, initial.length, { id: 'current', current: true, attached: false, attachable: false });
if (params.has('many') || params.has('currentlast') || params.has('currentmiddle') || params.has('late')) {
    for (let n = 0; n < 12; n++) initial.push({ id: `extra-${n}`, title: `Extra ${n}`, current: false, attached: true, attachable: true });
}
if (params.has('currentlast') || params.has('currentmiddle') || params.has('late')) {
    const row = initial.shift()!;
    initial.splice(params.has('currentmiddle') ? 8 : initial.length, 0, row);
}
const fixtureTabs = params.has('offline') ? [] : params.has('manytabs') ? Array.from({length: 24}, (_, n) => tab(n + 1, `Tab ${n + 1}`)) : [tab(1, 'A')];
const noop = () => {};
const calls: unknown[] = [];
Object.assign(window, { fixtureCalls: calls });

// This standalone test entry renders directly; it is not an HMR component module.
// eslint-disable-next-line react-refresh/only-export-components
function Fixture() {
    const [rows, setRows] = useState(params.has('one') ? initial.slice(0, 1) : params.has('late') ? initial.filter(row => row.id !== 'current') : initial);
    const [current, setCurrent] = useState('current');
    const [open, setOpen] = useState(!params.has('closed'));
    const [tabsReady, setTabsReady] = useState(!params.has('latetabs'));
    const [view, setView] = useState<SheetView>(params.has('url') ? 'url' : params.has('bookmarks') ? 'bookmarks' : 'main');
    const [newWindowError, setNewWindowError] = useState<string | null>(null);
    useEffect(() => {
        const onKey = (event: KeyboardEvent) => { if (event.key === 'Escape') setOpen(false); };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, []);
    return <><button id="fixture-open-address" onClick={() => { setView('url'); setOpen(true); }}>Open address</button><button id="fixture-open-main" onClick={() => { setView('main'); setOpen(true); }}>Open main</button><button id="fixture-load-current" onClick={() => setRows(initial)}>Load current</button><button id="fixture-poll" onClick={() => setRows([...rows])}>Poll</button><button id="fixture-load-tabs" onClick={() => setTabsReady(true)}>Load tabs</button><BrowserSheet
        open={open} view={view} sessions={rows} current={current} tabs={tabsReady ? fixtureTabs : []}
        activeTabId={params.has('manytabs') ? 24 : 1} currentUrl="https://example.test/path?q=1#part" busy={params.has('busy')} readOnly={params.has('readonly')}
        windowCloseSupported={!params.has('unsupported')} helpOpen={false} adopting={null} attachFailure={null} newWindowError={newWindowError}
        preset="mac" viewport={null} presets={[]} mode="shot" onClose={() => setOpen(false)} onView={setView}
        onSession={(id) => { calls.push({ switch: id }); setCurrent(id); }}
        onNewWindow={async () => { calls.push({ newWindow: true }); if (params.has('newfailure')) { setNewWindowError('Could not open new window'); return false; } return true; }}
        onCloseWindow={async (target) => {
            calls.push(target);
            if (params.has('failure')) return 'Server rejected close';
            const selected = rows.find((row) => 'session_id' in target ? row.id === target.session_id : row.window_id === target.window_id);
            setRows(rows.filter((row) => row !== selected));
            if (selected?.id === current) setCurrent('background');
            return null;
        }}
        onAttachWindow={noop} onTab={noop} onCloseTab={noop} onCreateTab={async () => true}
        bookmarked={false} bookmarks={[]} bookmarkDoc={null} bookmarksLoading={false} bookmarksError={null}
        bookmarkSource="local" bookmarkProfile="" localBookmarkCount={0} onSaveBookmark={async () => true}
        onRefreshBookmarks={noop} onShowLocalBookmarks={noop} onShowChromeBookmarks={noop}
        onPreset={noop} onMode={noop} onReload={noop} onNavigateTarget={async () => null} onSelectAddressTab={async () => null} onHelp={noop}
    /></>;
}

createRoot(document.getElementById('root')!).render(<Fixture />);
