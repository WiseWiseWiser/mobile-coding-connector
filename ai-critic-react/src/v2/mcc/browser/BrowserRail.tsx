import { useRef, useState } from 'react';
import type { KeyboardEvent as ReactKeyboardEvent } from 'react';
import type { BrowserTab, DaemonStatus, SessionInfo } from './browserWire';
import { windowTabs } from './browserView';
import { closeTargetFor, closeWindowBlocker } from './windowClose';
import type { CloseTarget } from './windowClose';

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

    const visible = sessions.filter(
        (session) =>
            session.attached ||
            session.sessionless ||
            session.id === current ||
            opened.includes(session.id),
    );

    const toggle = (id: string) => setExpanded((was) => (was === id ? '' : id));

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
    const closeBlocked =
        closeWindowBlocker({
            supported: props.windowCloseSupported,
            readOnly: props.readOnly,
            busy,
            windowCount: liveWindows,
        }) || (closeTarget ? '' : 'Select a window first');

    const triggerRef = useRef<HTMLButtonElement>(null);

    const dismissClose = () => {
        if (closing) return;
        setCloseFor('');
        setCloseError('');
        triggerRef.current?.focus();
    };

    // The prompt traps Tab between its two buttons and Escape behaves like Cancel.
    const closePromptKeys = (event: ReactKeyboardEvent<HTMLDivElement>) => {
        if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            dismissClose();
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
            <div className="bw-rail-body">
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
                                    onClick={() => props.onSelect(session.id)}
                                    disabled={busy && !live}
                                    title={live ? `Switch to ${session.id}` : `Reopen ${session.id}`}
                                >
                                    <span className="bw-sname">{label}</span>
                                    {secondary && <span className="bw-sid">{secondary}</span>}
                                </button>
                                <span className="bw-schip">{tabs.length}</span>
                            </div>
                            {!live && (
                                <button
                                    type="button"
                                    className="bw-reopen"
                                    onClick={() => props.onSelect(session.id)}
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
                                                onClick={() => props.onOpenTab(session.id, tab.tab_id)}
                                                title={`${tab.title ?? ''} — ${hostOf(tab)}`}
                                                disabled={busy || !live}
                                            >
                                                <span className="bw-fav">{initialOf(tab)}</span>
                                                <span className="bw-tname">{tab.title || hostOf(tab) || 'Untitled'}</span>
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
            </div>
            {closingNow && (
                <div
                    className="bw-closeconfirm"
                    role="alertdialog"
                    aria-label="Close this Chrome window"
                    onKeyDown={closePromptKeys}
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
            </div>
            <footer className="bw-rail-foot">
                browser-agent {daemon?.version ?? '—'} · :{daemon?.port ?? '—'}
            </footer>
        </aside>
    );
}
