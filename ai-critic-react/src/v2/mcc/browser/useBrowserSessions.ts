import { useEffect, useRef, useState } from 'react';
import { fetchSessions, openNewSession, switchSession, tabOp } from './browserApi';
import type { TabOp } from './browserApi';
import type { BrowserTab, DaemonStatus, SessionInfo } from './browserWire';

/** The tab strip refreshes on this beat while nothing else is happening. */
const POLL_MS = 8000;

export interface TabResult {
    tabs: BrowserTab[];
    error?: string;
    /** Present when the operation targeted the session this page is driving. */
    status?: unknown;
}

export interface BrowserSessions {
    sessions: SessionInfo[];
    current: string;
    daemon: DaemonStatus | null;
    readOnly: boolean;
    loading: boolean;
    busy: boolean;
    error: string | null;
    clearError: () => void;
    refresh: () => Promise<void>;
    switchTo: (id: string) => Promise<void>;
    addSession: () => Promise<void>;
    activate: (session: string, tabId: number) => Promise<TabResult>;
    close: (session: string, tabId: number) => Promise<TabResult>;
    create: (session: string) => Promise<TabResult>;
}

function message(err: unknown): string {
    if (err instanceof Error) return err.message;
    return String(err);
}

/**
 * The session rail's data source: the registry, the tab list for every session,
 * and the operations that change them.
 *
 * Tab operations answer with the fresh strip so the strip updates without
 * waiting for the next poll; the poll then keeps it honest when another viewer
 * (or Chrome itself) opens or closes something.
 */
export function useBrowserSessions(active: boolean): BrowserSessions {
    const [sessions, setSessions] = useState<SessionInfo[]>([]);
    const [current, setCurrent] = useState('');
    const [daemon, setDaemon] = useState<DaemonStatus | null>(null);
    const [readOnly, setReadOnly] = useState(false);
    const [loading, setLoading] = useState(true);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const inFlight = useRef(false);

    const load = async () => {
        if (inFlight.current) return;
        inFlight.current = true;
        try {
            const response = await fetchSessions();
            if (!response.ok) {
                setError(response.error ?? 'cannot list sessions');
                return;
            }
            setSessions(response.sessions ?? []);
            setCurrent(response.current ?? '');
            setDaemon(response.daemon);
            setReadOnly(Boolean(response.read_only));
            setError(null);
        } catch (err) {
            setError(message(err));
        } finally {
            inFlight.current = false;
            setLoading(false);
        }
    };

    useEffect(() => {
        if (!active) return;
        void load();
        const timer = setInterval(() => {
            if (document.visibilityState === 'visible') void load();
        }, POLL_MS);
        const onVisible = () => {
            if (document.visibilityState === 'visible') void load();
        };
        document.addEventListener('visibilitychange', onVisible);
        return () => {
            clearInterval(timer);
            document.removeEventListener('visibilitychange', onVisible);
        };
        // load() only reads refs and setState; the first closure stays valid.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [active]);

    const switchTo = async (id: string) => {
        if (id === current) return;
        setBusy(true);
        try {
            const response = await switchSession(id);
            if (!response.ok) {
                setError(response.error ?? 'cannot switch session');
                return;
            }
            setCurrent(response.session?.id || id);
            await load();
        } catch (err) {
            setError(message(err));
        } finally {
            setBusy(false);
        }
    };

    const addSession = async () => {
        setBusy(true);
        try {
            const response = await openNewSession();
            if (!response.ok) {
                setError(response.error ?? 'cannot open a new window');
                return;
            }
            setCurrent(response.session.id);
            await load();
        } catch (err) {
            setError(message(err));
        } finally {
            setBusy(false);
        }
    };

    const runTab = async (session: string, op: TabOp, tabId = 0): Promise<TabResult> => {
        setBusy(true);
        try {
            const response = await tabOp(session, op, tabId);
            if (!response.ok) return { tabs: [], error: response.error ?? 'tab operation failed' };
            setSessions((previous) =>
                previous.map((row) =>
                    row.id === session
                        ? {
                              ...row,
                              tabs: response.tabs,
                              active_tab_id: response.active_tab_id ?? row.active_tab_id,
                          }
                        : row,
                ),
            );
            return { tabs: response.tabs ?? [] };
        } catch (err) {
            return { tabs: [], error: message(err) };
        } finally {
            setBusy(false);
        }
    };

    return {
        sessions,
        current,
        daemon,
        readOnly,
        loading,
        busy,
        error,
        clearError: () => setError(null),
        refresh: load,
        switchTo,
        addSession,
        activate: (session, tabId) => runTab(session, 'activate', tabId),
        close: (session, tabId) => runTab(session, 'close', tabId),
        create: (session) => runTab(session, 'create'),
    };
}
