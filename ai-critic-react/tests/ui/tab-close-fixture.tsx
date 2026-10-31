import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserTabList } from '../../src/v2/mcc/browser/BrowserTabList';
import { BrowserTabCloseDialog } from '../../src/v2/mcc/browser/BrowserTabCloseDialog';
import { useBrowserTabClose } from '../../src/v2/mcc/browser/useBrowserTabClose';
import type { BrowserSessions } from '../../src/v2/mcc/browser/useBrowserSessions';
import type { BrowserTab } from '../../src/v2/mcc/browser/browserWire';
import '../../src/v2/mcc/browser/browser.css';

const calls: unknown[] = [];
Object.assign(window, { fixtureCalls: calls });
const params = new URLSearchParams(location.search);
const initial: BrowserTab[] = [
    { tab_id: 7, title: 'Editor with unsaved changes', active: true, role: 'user' },
    { tab_id: 8, title: 'Other tab', active: false, role: 'user' },
];
const noop = () => {};
// Standalone test entry; it is not an HMR component module.
// eslint-disable-next-line react-refresh/only-export-components
function Fixture() {
    const [tabs, setTabs] = useState(initial);
    const [notice, setNotice] = useState('');
    const sessions = {
        sessions: [{ id: 'session-1', tabs }],
        close: async (session: string, tabId: number) => {
            calls.push({ session, tabId });
            if (params.has('network')) return { tabs, error: 'Failed to fetch' };
            if (params.has('unknown')) return { tabs, error: 'tab close outcome is unknown; check Chrome for a dialog and refresh the tab list before retrying' };
            if (params.has('clean')) return { tabs: tabs.slice(1) };
            return { tabs, closeConfirmation: { operation_id: 'operation-1', tab_id: tabId } };
        },
        resolveTabClose: async (session: string, operationId: string, accept: boolean) => {
            calls.push({ session, operationId, accept });
            if (params.has('failure')) return { tabs, error: 'The close outcome is unknown; check Chrome.' };
            return { tabs: accept ? tabs.slice(1) : tabs, closeOutcome: accept ? 'closed' : 'kept' };
        },
    } as unknown as BrowserSessions;
    const action = useBrowserTabClose(sessions, (_session, tabId) => { setTabs((prev) => prev.filter((tab) => tab.tab_id !== tabId)); setNotice('closed'); }, setNotice);
    return <div className="bw-page">
        <main><BrowserTabList tabs={tabs} activeTabId={7} busy={false} readOnly={false} tabsLocked={false} locked={false}
            onView={noop} onTab={noop} onCloseTab={(id) => void action.close('session-1', id)} /><div id="notice">{notice}</div></main>
        {action.pending && <BrowserTabCloseDialog pending={action.pending} busy={action.deciding}
            onKeep={() => void action.choose(false)} onDiscard={() => void action.choose(true)} onDismiss={action.dismiss} />}
    </div>;
}
createRoot(document.getElementById('root')!).render(<Fixture />);
