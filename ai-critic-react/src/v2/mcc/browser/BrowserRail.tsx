import { useEffect, useState } from 'react';
import type { BrowserTab, DaemonStatus, SessionInfo } from './browserWire';
import { contentTabs } from './browserView';

export interface BrowserRailProps {
    sessions: SessionInfo[];
    current: string;
    daemon: DaemonStatus | null;
    loading: boolean;
    busy: boolean;
    open: boolean;
    onSelect: (session: string) => void;
    onOpenTab: (session: string, tabId: number) => void;
    onCloseTab: (session: string, tabId: number) => void;
    onNew: () => void;
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
    // Stale registry rows stay out of the rail until something opens them: the
    // current session, or a row the user reopened earlier in this visit.
    const [opened, setOpened] = useState<string[]>([]);

    // Following the current session keeps the rail informative after a switch.
    useEffect(() => {
        if (!current) return;
        setExpanded(current);
        setOpened((was) => (was.includes(current) ? was : [...was, current]));
    }, [current]);

    const visible = sessions.filter(
        (session) => session.attached || session.id === current || opened.includes(session.id),
    );

    const toggle = (id: string) => setExpanded((was) => (was === id ? '' : id));

    return (
        <aside className={open ? 'bw-rail is-open' : 'bw-rail'} aria-label="Browser sessions">
            <header className="bw-rail-head">
                <span className="bw-rail-title">SESSIONS</span>
                <span className="bw-rail-count">{visible.length}</span>
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
                    const tabs = contentTabs(session.tabs);
                    const isOpen = expanded === session.id;
                    const live = session.attached;
                    return (
                        <div
                            key={session.id}
                            className={`bw-srow${session.id === current ? ' is-current' : ''}${live ? '' : ' is-stale'}`}
                            data-session={session.id}
                        >
                            <div className="bw-srow-line">
                                <button
                                    type="button"
                                    className="bw-caret"
                                    onClick={() => toggle(session.id)}
                                    aria-expanded={isOpen}
                                    aria-label={isOpen ? `Collapse ${session.id}` : `Expand ${session.id}`}
                                    disabled={tabs.length === 0}
                                >
                                    ›
                                </button>
                                <span className={`bw-dot ${live ? 'is-ok' : 'is-warn'}`} />
                                <button
                                    type="button"
                                    className="bw-sname"
                                    onClick={() => props.onSelect(session.id)}
                                    disabled={busy && !live}
                                    title={live ? `Switch to ${session.id}` : `Reopen ${session.id}`}
                                >
                                    {session.id}
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
                            {live && isOpen && (
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
                                                disabled={busy}
                                            >
                                                <span className="bw-fav">{initialOf(tab)}</span>
                                                <span className="bw-tname">{tab.title || hostOf(tab) || 'Untitled'}</span>
                                            </button>
                                            <button
                                                type="button"
                                                className="bw-tx"
                                                onClick={() => props.onCloseTab(session.id, tab.tab_id)}
                                                title="Close tab"
                                                aria-label={`Close ${tab.title ?? 'tab'}`}
                                                disabled={busy}
                                            >
                                                ×
                                            </button>
                                        </li>
                                    ))}
                                    {tabs.length === 0 && <li className="bw-trow bw-trow-empty">No tabs</li>}
                                </ul>
                            )}
                        </div>
                    );
                })}
            </div>
            <button type="button" className="bw-newwin" onClick={props.onNew} disabled={busy}>
                + New window
            </button>
            <footer className="bw-rail-foot">
                browser-agent {daemon?.version ?? '—'} · :{daemon?.port ?? '—'}
            </footer>
        </aside>
    );
}
