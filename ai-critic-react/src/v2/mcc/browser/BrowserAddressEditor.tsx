import { useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { BrowserBookmarkBrowser } from './BrowserBookmarkBrowser';
import { checkAddress, isHttpURL } from './browserAddress';
import { tabLabel } from './browserView';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';
import type { BrowserTab } from './browserWire';

interface BrowserAddressEditorProps {
    session: string;
    tabs: BrowserTab[];
    activeTabId: number;
    currentUrl?: string;
    disabled: boolean;
    bookmarkDoc: BookmarkDocument | null;
    bookmarksLoading: boolean;
    bookmarksError: string | null;
    bookmarkSource: BookmarkSource;
    onShowLocal: () => void;
    onShowChrome: () => void;
    onBack: () => void;
    onClose: () => void;
    onSelectTab: (session: string, tabId: number) => Promise<string | null>;
    onNavigate: (session: string, tabId: number, url: string) => Promise<string | null>;
}

/** The address draft belongs to a specific tab, not whichever tab changes later. */
export function BrowserAddressEditor(props: BrowserAddressEditorProps) {
    const [session] = useState(props.session);
    const [target, setTarget] = useState(props.activeTabId);
    const [draft, setDraft] = useState(isHttpURL(props.currentUrl) ? props.currentUrl! : '');
    // The prefilled tab address is not a bookmark filter until the user edits it.
    const [edited, setEdited] = useState(false);
    const [mode, setMode] = useState<'tabs' | 'bookmarks'>('tabs');
    const [error, setError] = useState('');
    const [pending, setPending] = useState(false);
    const inFlight = useRef(false);
    const inputRef = useRef<HTMLInputElement>(null);
    const stale = props.session !== session;
    const targetTab = props.tabs.find((tab) => tab.tab_id === target);
    const locked = props.disabled || pending || stale || !targetTab;

    const selectTab = async (tab: BrowserTab) => {
        if (inFlight.current || props.disabled || stale) return;
        inFlight.current = true;
        setPending(true);
        const failure = await props.onSelectTab(session, tab.tab_id).catch((e: unknown) => String(e));
        inFlight.current = false;
        setPending(false);
        if (failure) { setError(failure); return; }
        setTarget(tab.tab_id);
        setDraft(isHttpURL(tab.url) ? tab.url! : '');
        setEdited(false);
        setError('');
    };

    const navigate = async (raw: string) => {
        if (inFlight.current || locked) return;
        const check = checkAddress(raw);
        if (!check.ok) { setError(check.error); return; }
        inFlight.current = true;
        setPending(true);
        const failure = await props.onNavigate(session, target, check.target).catch((e: unknown) => String(e));
        inFlight.current = false;
        setPending(false);
        if (failure) { setError(failure); return; }
        props.onClose();
    };
    const submit = (event: FormEvent) => { event.preventDefault(); void navigate(draft); };
    const openBookmark = (item: BookmarkItem) => { void navigate(item.url); };
    return (
        <div className="bwp-urlview bwp-address-editor">
            <div className="bwp-shead">
                <button type="button" className="bwp-icon2" onClick={props.onBack} aria-label="Back to sessions and tabs">‹</button>
                <span className="bwp-kicker">Open address</span><span className="bwp-gap" />
                <button type="button" className="bwp-x" onClick={props.onClose} aria-label="Close">×</button>
            </div>
            <p className="bwp-urlhint">Target: <strong>{targetTab ? tabLabel(targetTab) : 'Unavailable'}</strong> · current window</p>
            <form className={`bwp-urlform${error ? ' has-error' : ''}`} onSubmit={submit}>
                <div className="bwp-address-inputwrap">
                    <input ref={inputRef} className="bwp-urlinput" autoFocus type="text" inputMode="url" enterKeyHint="go" autoCapitalize="off" autoCorrect="off" spellCheck={false} placeholder="Address, search, or bookmarks" aria-label="Address, search, or bookmarks" value={draft} onChange={(event) => { setDraft(event.target.value); setEdited(true); setError(''); }} />
                    {draft && <button type="button" className="bwp-address-clear" aria-label="Clear address or search" onClick={() => { setDraft(''); setEdited(true); setError(''); inputRef.current?.focus(); }}>×</button>}
                </div>
                <button type="submit" className="bwp-go" disabled={locked}>{pending ? '…' : 'Go'}</button>
            </form>
            <p className="bwp-address-hint">{mode === 'bookmarks' ? 'Type to filter bookmarks · Go opens an address or searches the web' : 'Go opens an address or searches the web'}</p>
            {(error || stale || !targetTab) && <p className="bwp-urlerr" role="alert">{stale ? 'The current window changed. Reopen the address editor.' : !targetTab ? 'The selected tab is no longer available.' : error}</p>}
            <div className="bwp-address-modes" role="tablist" aria-label="Address destinations">
                <button type="button" role="tab" aria-selected={mode === 'tabs'} onClick={() => setMode('tabs')}>Window tabs</button>
                <button type="button" role="tab" aria-selected={mode === 'bookmarks'} onClick={() => setMode('bookmarks')}>Bookmarks</button>
            </div>
            {mode === 'tabs' ? (
                <div className="bwp-address-list" aria-label="Choose target tab">
                    {props.tabs.map((tab) => (
                        <button type="button" className={`bwp-address-row${target === tab.tab_id ? ' is-selected' : ''}`} key={tab.tab_id} onClick={() => void selectTab(tab)} disabled={props.disabled || pending || stale}>
                            <span className="bwp-address-copy"><strong>{tabLabel(tab)}</strong><small>{isHttpURL(tab.url) ? tab.url : 'New tab'}</small></span>
                            {target === tab.tab_id && <span aria-label="Selected">✓</span>}
                        </button>
                    ))}
                </div>
            ) : (
                <div className="bwp-address-bookmarks">
                    <div className="bwp-address-source"><span>{props.bookmarkSource === 'chrome' ? 'Chrome bookmarks' : 'Local bookmarks'}</span><button type="button" onClick={props.bookmarkSource === 'chrome' ? props.onShowLocal : props.onShowChrome}>{props.bookmarkSource === 'chrome' ? 'Local' : 'Chrome'}</button></div>
                    <BrowserBookmarkBrowser doc={props.bookmarkDoc} loading={props.bookmarksLoading} error={props.bookmarksError} disabled={locked} filterQuery={edited ? draft : ''} onOpen={openBookmark} />
                </div>
            )}
        </div>
    );
}
