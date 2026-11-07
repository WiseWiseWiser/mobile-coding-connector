import { useEffect, useRef, useState } from 'react';
import type { KeyboardEvent as ReactKeyboardEvent } from 'react';
import type { BrowserTab, DaemonStatus, SessionInfo } from './browserWire';
import { windowTabs } from './browserView';
import { BrowserRailResults } from './BrowserRailResults';
import { BrowserSearchRow } from './BrowserSearchRow';
import { searchWindows } from './tabSearch';
import type { SearchGroup } from './tabSearch';
import { useSeenMarks } from './useSeenMarks';
import { orderByCreation } from './windowsOrder';
import { closeTargetFor, closeWindowBlocker } from './windowClose';
import type { CloseTarget } from './windowClose';
import { ledgerFor, sweepAction, sweepBlocker, sweepBody, sweepWindowCount } from './sweep';
import { useSweepFlow } from './useSweepFlow';
import type { SweepResult } from './useBrowserSessions';

export interface BrowserRailProps {
    sessions: SessionInfo[];
    current: string;
    daemon: DaemonStatus | null;
    loading: boolean;
    busy: boolean;
    readOnly: boolean;
    windowCloseSupported: boolean;
    open: boolean;
    onSelect: (session: string) => void;
    onOpenTab: (session: string, tabId: number) => void;
    onCloseTab: (session: string, tabId: number) => void;
    onNew: () => void;
    onCloseWindow: (target: CloseTarget) => Promise<string | null>;
    onSweep: (dryRun: boolean) => Promise<SweepResult>;
    onToast: (message: string, warn?: boolean) => void;
    onRefresh: () => void;
}

function initialOf(tab: BrowserTab): string {
    const title = (tab.title ?? '').trim();
    if (title) return title.slice(0, 1).toUpperCase();
    const host = (tab.url ?? '').replace(/^[a-z]+:\/\//i, '').split('/')[0];
    return (host.slice(0, 1) || '·').toUpperCase();
}

function hostOf(tab: BrowserTab): string {
    return (tab.url ?? '').replace(/^[a-z]+:\/\//i, '').split('/')[0] || tab.url || '';
}

/**
 * The session rail: every Chrome window the agent can drive, and the tabs inside
 * each one.
 *
 * The daemon registry keeps hundreds of ids from earlier runs. Only sessions with
 * a live extension connection make the cut; a session that drops out stays visible
 * only if it was current (or reopened) in this visit, then dimmed with a Reopen
 * action so it is still one click away from coming back.
 */
export function BrowserRail(props: BrowserRailProps) {
    const { sessions, current, daemon, loading, busy, open } = props;
    const [expanded, setExpanded] = useState<string>(current);
    const [closeFor, setCloseFor] = useState('');
    const [closing, setClosing] = useState(false);
    const [closeError, setCloseError] = useState('');
    // Stale registry rows stay out of the rail until something opens them: the
    // current session, or a row the user reopened earlier in this visit.
    const [opened, setOpened] = useState<string[]>([]);
    const [seenCurrent, setSeenCurrent] = useState(current);
    const [query, setQuery] = useState('');
    const searchRef = useRef<HTMLInputElement>(null);

    // Following the current session keeps the rail informative after a switch.
    // Synced during render, not in an effect, so the rail never paints the old
    // session as expanded for a frame.
    if (current !== seenCurrent) {
        setSeenCurrent(current);
        if (current) {
            setExpanded(current);
            setOpened((was) => (was.includes(current) ? was : [...was, current]));
        }
    }

    // Earliest window first: `window_id` is the only creation-order signal the
    // extension exposes, and rows without one sink to the bottom.
    const visible = orderByCreation(
        sessions.filter(
            (session) =>
                session.attached ||
                session.sessionless ||
                session.id === current ||
                opened.includes(session.id),
        ),
    );

    // The same ledger the phone sheet writes, so "not opened yet" means one
    // thing across both surfaces. It lives per client, next to the rail state.
    const railTargets = visible.map((session) => ({
        key: session.id,
        title: session.title?.trim() || session.id,
        tabs: windowTabs(session.tabs),
    }));
    const marks = useSeenMarks(railTargets);
    const unread = marks.unread(
        visible.filter((session) => session.attached || session.sessionless).map((session) => session.id),
    );
    const searching = query.trim().length > 0;
    const groups = searching ? searchWindows(railTargets, query) : [];

    const toggle = (id: string) => setExpanded((was) => (was === id ? '' : id));

    const pickResult = (group: SearchGroup, tabId: number) => {
        marks.markWindow(group.key);
        marks.markTab(group.key, tabId);
        setQuery('');
        props.onOpenTab(group.key, tabId);
    };

    // A click is what clears a badge, never the fact that a row was painted.
    const selectSession = (id: string) => {
        marks.markWindow(id);
        props.onSelect(id);
    };

    const openTab = (session: string, tabId: number) => {
        marks.markWindow(session);
        marks.markTab(session, tabId);
        props.onOpenTab(session, tabId);
    };

    // The close prompt belongs to the current window only. Any switch makes it
    // stale, and the mismatch hides it without an effect.
    const closeRow = sessions.find((session) => session.id === current);
    const closingNow = closeFor === current && Boolean(closeRow);
    const closeTarget = closeRow
        ? closeTargetFor({
              key: closeRow.id,
              sessionId: closeRow.id,
              windowId: closeRow.window_id,
              title: closeRow.title ?? '',
              tabCount: windowTabs(closeRow.tabs).length,
              current: true,
              sessionless: Boolean(closeRow.sessionless),
          })
        : null;
    const closeTabCount = windowTabs(closeRow?.tabs).length;
    const liveWindows = sessions.filter((session) => session.attached || session.sessionless).length;
    // Windows, not sessions: a window's own agent page and the session that
    // opened it can be two rows, and a sweep closes windows.
    const sweepWindows = sweepWindowCount(
        sessions.map((session) => ({
            key: session.id,
            attached: session.attached || Boolean(session.sessionless),
            windowId: session.window_id,
        })),
    );
    // The same plan-and-confirm flow the phone sheet uses, rendered inline:
    // the desktop rail has room, so the prompt never covers the tabs.
    const sweep = useSweepFlow({
        scope: 'rail',
        blocked: busy,
        onSweep: props.onSweep,
        onToast: props.onToast,
    });
    const sweepBlocked = sweepBlocker({
        supported: props.windowCloseSupported,
        readOnly: props.readOnly,
        busy: (busy && !sweep.planning) || sweep.applying,
        windowCount: sweepWindows,
    });
    const sweepCards = sweep.open && sweep.plan ? sweep.plan : null;
    const sweepLedger = sweepCards ? ledgerFor(sweepCards) : null;
    // Cancel is Escape or the Cancel button; either way the rail keeps focus on
    // the button that opened the prompt, so the keyboard never lands on body.
    const dismissSweep = () => {
        const back = sweep.trigger;
        sweep.cancel();
        back?.focus();
    };
    const closeBlocked =
        closeWindowBlocker({
            supported: props.windowCloseSupported,
            readOnly: props.readOnly,
            busy,
            windowCount: liveWindows,
        }) || (closeTarget ? '' : 'Select a window first');

    const triggerRef = useRef<HTMLButtonElement>(null);

    // ⌘K reaches the field from anywhere on the page. The rail is the desktop
    // surface and is always mounted, so visibility needs no extra guard.
    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 'k') return;
            event.preventDefault();
            searchRef.current?.focus();
            searchRef.current?.select();
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, []);

    const dismissClose = () => {
        if (closing) return;
        setCloseFor('');
        setCloseError('');
        triggerRef.current?.focus();
    };

    // The prompt traps Tab between its two buttons and Escape behaves like Cancel.
    const promptKeys = (
        event: ReactKeyboardEvent<HTMLDivElement>,
        cancel: () => void,
        locked: boolean,
    ) => {
        if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            if (!locked) cancel();
            return;
        }
        if (event.key !== 'Tab') return;
        const buttons = Array.from(
            event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not([disabled])'),
        );
        const first = buttons[0];
        const last = buttons[buttons.length - 1];
        if (!first || !last) return;
        if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
    };

    const closeWindow = async () => {
        if (!closeTarget) return;
        setClosing(true);
        setCloseError('');
        const failure = await props.onCloseWindow(closeTarget);
        setClosing(false);
        if (failure) {
            setCloseError(failure);
            return;
        }
        setCloseFor('');
    };

    return (
        <aside className={open ? 'bw-rail is-open' : 'bw-rail'} aria-label="Browser sessions">
            <header className="bw-rail-head">
                <span className="bw-rail-title">WINDOWS</span>
                <span className="bw-rail-count">{liveWindows}</span>
                {unread > 0 && (
                    <span className="bw-rail-new" title="Windows you have not opened yet">
                        {unread} NEW
                    </span>
                )}
                <button
                    type="button"
                    className="bw-btn bw-btn-ghost bw-rail-refresh"
                    onClick={props.onRefresh}
                    disabled={busy}
                    title="Refresh sessions"
                    aria-label="Refresh sessions"
                >
                    ↻
                </button>
            </header>
            <BrowserSearchRow
                surface="rail"
                value={query}
                onChange={setQuery}
                onClear={() => setQuery('')}
                onEnter={() => {
                    const first = groups[0]?.hits[0];
                    if (first) pickResult(groups[0], first.tabId);
                }}
                placeholder="Search tabs and URLs"
                shortcut="⌘K"
                disabled={busy}
                inputRef={(node) => {
                    searchRef.current = node;
                }}
            />
            <div className="bw-rail-body">
                {searching ? (
                    <BrowserRailResults
                        query={query}
                        groups={groups}
                        windowCount={visible.length}
                        onPick={pickResult}
                        disabled={busy}
                    />
                ) : (
                    <>
                    {visible.length === 0 && (
                        <p className="bw-rail-none">{loading ? 'Looking for Chrome…' : 'No sessions yet.'}</p>
                    )}
                    {visible.map((session) => {
                        const tabs = windowTabs(session.tabs);
                        const isOpen = expanded === session.id;
                        const live = session.attached;
                        const sessionless = Boolean(session.sessionless);
                        const label = session.title?.trim() || session.id;
                        const secondary = session.title?.trim() ? session.id : '';
                        const rowError = session.error || '';
                        return (
                            <div
                                key={session.id}
                                className={`bw-srow${session.id === current ? ' is-current' : ''}${
                                    live ? '' : sessionless ? '' : ' is-stale'
                                }`}
                                data-session={session.id}
                            >
                                <div className="bw-srow-line">
                                    <button
                                        type="button"
                                        className="bw-caret"
                                        onClick={() => toggle(session.id)}
                                        aria-expanded={isOpen}
                                        aria-label={isOpen ? `Collapse ${label}` : `Expand ${label}`}
                                        disabled={tabs.length === 0}
                                    >
                                        ›
                                    </button>
                                    <span className={`bw-dot ${live ? 'is-ok' : 'is-warn'}`} />
                                    <button
                                        type="button"
                                        className="bw-slabel bw-sopen"
                                        onClick={() => selectSession(session.id)}
                                        disabled={busy && !live}
                                        title={live ? `Switch to ${session.id}` : `Reopen ${session.id}`}
                                    >
                                        <span className="bw-snameline">
                                            <span className="bw-sname">{label}</span>
                                            {marks.windowNew(session.id) && (
                                                <span className="new" title="Window you have not opened yet">
                                                    New
                                                </span>
                                            )}
                                        </span>
                                        {secondary && <span className="bw-sid">{secondary}</span>}
                                    </button>
                                    <span className="bw-schip">{tabs.length}</span>
                                </div>
                                {!live && (
                                    <button
                                        type="button"
                                        className="bw-reopen"
                                        onClick={() => selectSession(session.id)}
                                        disabled={busy}
                                    >
                                        Reopen
                                    </button>
                                )}
                                {rowError && <p className="bw-serr">{rowError}</p>}
                                {isOpen && (
                                    <ul className="bw-trows">
                                        {tabs.map((tab) => (
                                            <li
                                                key={tab.tab_id}
                                                className={`bw-trow${
                                                    tab.tab_id === session.active_tab_id ? ' is-active' : ''
                                                }`}
                                                data-tab={tab.tab_id}
                                            >
                                                <button
                                                    type="button"
                                                    className="bw-topen"
                                                    onClick={() => openTab(session.id, tab.tab_id)}
                                                    title={`${tab.title ?? ''} — ${hostOf(tab)}`}
                                                    disabled={busy || !live}
                                                >
                                                    <span className="bw-fav">{initialOf(tab)}</span>
                                                    <span className="bw-tname">{tab.title || hostOf(tab) || 'Untitled'}</span>
                                                    {marks.tabNew(session.id, tab.tab_id) && (
                                                        <span className="new is-tab" title="Tab you have not followed yet">
                                                            New
                                                        </span>
                                                    )}
                                                </button>
                                                {live && (
                                                    <button
                                                        type="button"
                                                        className="bw-tx"
                                                        onClick={(event) => { event.currentTarget.focus({ preventScroll: true }); props.onCloseTab(session.id, tab.tab_id); }}
                                                        title="Close tab"
                                                        aria-label={`Close ${tab.title ?? 'tab'}`}
                                                        disabled={busy}
                                                    >
                                                        ×
                                                    </button>
                                                )}
                                            </li>
                                        ))}
                                        {tabs.length === 0 && <li className="bw-trow bw-trow-empty">No tabs</li>}
                                    </ul>
                                )}
                            </div>
                        );
                    })}
                    </>
                )}
            </div>
            {closingNow && (
                <div
                    className="bw-closeconfirm"
                    role="alertdialog"
                    aria-label="Close this Chrome window"
                    onKeyDown={(event) => promptKeys(event, dismissClose, closing)}
                >
                    <p>
                        Close <strong>{closeRow?.title?.trim() || current}</strong> and its{' '}
                        {closeTabCount} {closeTabCount === 1 ? 'tab' : 'tabs'}?
                    </p>
                    <p className="bw-closehint">Cannot be undone. The last Chrome window stays open.</p>
                    {closeError && <p className="bw-closeerr" role="alert">{closeError}</p>}
                    <div className="bw-closeacts">
                        <button
                            type="button"
                            className="bw-closecancel"
                            autoFocus
                            onClick={dismissClose}
                            disabled={closing}
                        >
                            Cancel
                        </button>
                        <button type="button" className="bw-closego" onClick={() => void closeWindow()} disabled={closing}>
                            {closing ? 'Closing…' : 'Close window'}
                        </button>
                    </div>
                </div>
            )}
            {sweepCards && (
                <div
                    className="bw-closeconfirm"
                    role="alertdialog"
                    aria-label="Close all other windows"
                    onKeyDown={(event) => promptKeys(event, dismissSweep, sweep.applying)}
                >
                    <p>
                        <strong>{sweepBody(sweepCards).closing}</strong> {sweepBody(sweepCards).keeping}
                    </p>
                    <p className="bw-closehint">
                        Windows {sweepLedger?.windows} · Tabs {sweepLedger?.tabs} · Keeps {sweepLedger?.kept}
                    </p>
                    {sweep.error && <p className="bw-closeerr" role="alert">{sweep.error}</p>}
                    <div className="bw-closeacts">
                        <button
                            type="button"
                            className="bw-closecancel"
                            autoFocus
                            onClick={dismissSweep}
                            disabled={sweep.applying}
                        >
                            Cancel
                        </button>
                        <button
                            type="button"
                            className="bw-closego"
                            onClick={sweep.confirm}
                            disabled={sweep.applying}
                        >
                            {sweep.applying ? 'Closing…' : sweepAction(sweepCards)}
                        </button>
                    </div>
                </div>
            )}
            <div className="bw-rail-actions">
                <button type="button" className="bw-newwin" onClick={props.onNew} disabled={busy}>
                    + New window
                </button>
                <button
                    type="button"
                    className="bw-closewin"
                    ref={triggerRef}
                    onClick={() => {
                        setCloseError('');
                        setCloseFor(current);
                    }}
                    disabled={Boolean(closeBlocked) || closing}
                    title={closeBlocked || 'Close the current Chrome window'}
                >
                    Close window
                </button>
                <button
                    type="button"
                    className="bw-sweepwin"
                    onClick={(event) => sweep.start(event.currentTarget)}
                    disabled={Boolean(sweepBlocked) || sweep.planning}
                    title={sweepBlocked || 'Close every Chrome window except this one'}
                >
                    <span className="bw-sweepx" aria-hidden="true">×</span>
                    <span className="bw-sweepname">{sweep.planning ? 'Planning…' : 'Close other windows'}</span>
                    {!sweep.planning && sweepWindows > 1 && (
                        <span className="bw-sweepnum">{sweepWindows} → 1</span>
                    )}
                </button>
            </div>
            <footer className="bw-rail-foot">
                browser-agent {daemon?.version ?? '—'} · :{daemon?.port ?? '—'}
            </footer>
        </aside>
    );
}
