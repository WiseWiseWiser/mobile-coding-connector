import { useEffect, useRef, useState } from 'react';
import { closeWindow as closeWindowRequest, fetchSessions, openNewSession, resolveTabClose as resolveTabCloseRequest, switchSession, tabOp } from './browserApi';
import type { TabOp } from './browserApi';
import type { BrowserTab, DaemonStatus, SessionInfo } from './browserWire';

/** The tab strip refreshes on this beat while nothing else is happening. */
const POLL_MS = 8000;

export interface TabResult {
    tabs: BrowserTab[];
    error?: string;
    closeConfirmation?: { operation_id: string; tab_id: number };
    closeOutcome?: 'closed' | 'kept';
    /** Present when the operation targeted the session this page is driving. */
    status?: unknown;
}

export interface WindowCloseResult {
    ok: boolean;
    error?: string;
    /** The session the server made current after closing the current window. */
    current?: string;
}

export interface BrowserSessions {
    sessions: SessionInfo[];
    current: string;
    daemon: DaemonStatus | null;
    windowsSupported: boolean;
    /** Safe to offer whole-window close in this build of the extension. */
    windowCloseSupported: boolean;
    readOnly: boolean;
    loading: boolean;
    busy: boolean;
    error: string | null;
    newWindowError: string | null;
    clearError: () => void;
    refresh: () => Promise<void>;
    switchTo: (id: string) => Promise<void>;
    addSession: () => Promise<boolean>;
    closeWindow: (target: { window_id: number } | { session_id: string }) => Promise<WindowCloseResult>;
    activate: (session: string, tabId: number) => Promise<TabResult>;
    close: (session: string, tabId: number) => Promise<TabResult>;
    resolveTabClose: (session: string, operationId: string, accept: boolean) => Promise<TabResult>;
    create: (session: string, url?: string) => Promise<TabResult>;
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
    const [windowsSupported, setWindowsSupported] = useState(false);
    const [windowCloseSupported, setWindowCloseSupported] = useState(false);
    const [readOnly, setReadOnly] = useState(false);
    const [loading, setLoading] = useState(true);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [newWindowError, setNewWindowError] = useState<string | null>(null);
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
            setWindowsSupported(Boolean(response.windows_supported));
            setWindowCloseSupported(Boolean(response.window_close_supported));
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
    }, [active]);

    const switchTo = async (id: string) => {
        // A live current session is already selected; a stale one is not. The
        // Reopen action on a dropped session must still reach the server.
        const row = sessions.find((session) => session.id === id);
        if (id === current && row?.attached) return;
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

    const addSession = async (): Promise<boolean> => {
        setNewWindowError(null);
        setBusy(true);
        try {
            const response = await openNewSession();
            if (!response.ok) {
                const detail = response.error ?? 'cannot open a new window';
                setNewWindowError(detail);
                setError(detail);
                return false;
            }
            setCurrent(response.session.id);
            await load();
            return true;
        } catch (err) {
            setNewWindowError(message(err));
            setError(message(err));
            return false;
        } finally {
            setBusy(false);
        }
    };

    /** Closing the current window is a server-side switch; refresh, never guess. */
    const closeWindow = async (
        target: { window_id: number } | { session_id: string },
    ): Promise<WindowCloseResult> => {
        setBusy(true);
        try {
            const response = await closeWindowRequest(target);
            if (!response.ok) return { ok: false, error: response.error ?? 'cannot close that window' };
            if (response.current) setCurrent(response.current);
            await load();
            return { ok: true, current: response.current };
        } catch (err) {
            return { ok: false, error: message(err) };
        } finally {
            setBusy(false);
        }
    };

    const runTab = async (session: string, op: TabOp, tabId = 0, url = ''): Promise<TabResult> => {
        setBusy(true);
        try {
            const response = await tabOp(session, op, tabId, url);
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
            return { tabs: response.tabs ?? [], closeConfirmation: response.close_confirmation, closeOutcome: response.close_outcome };
        } catch (err) {
            if (op === 'close') void load();
            return { tabs: [], error: message(err) };
        } finally {
            setBusy(false);
        }
    };

    const resolveTabClose = async (session: string, operationId: string, accept: boolean): Promise<TabResult> => {
        setBusy(true);
        try {
            const response = await resolveTabCloseRequest(session, operationId, accept);
            if (!response.ok) return { tabs: [], error: response.error ?? 'cannot resolve tab close' };
            setSessions((previous) => previous.map((row) => row.id === session
                ? { ...row, tabs: response.tabs, active_tab_id: response.active_tab_id ?? row.active_tab_id }
                : row));
            return { tabs: response.tabs ?? [], closeOutcome: response.close_outcome };
        } catch (err) {
            void load();
            return { tabs: [], error: message(err) };
        } finally {
            setBusy(false);
        }
    };

    return {
        sessions,
        current,
        daemon,
        windowsSupported,
        windowCloseSupported,
        readOnly,
        loading,
        busy,
        error,
        newWindowError,
        clearError: () => setError(null),
        refresh: load,
        switchTo,
        addSession,
        closeWindow,
        activate: (session, tabId) => runTab(session, 'activate', tabId),
        close: (session, tabId) => runTab(session, 'close', tabId),
        resolveTabClose,
        create: (session, url = '') => runTab(session, 'create', 0, url),
    };
}
