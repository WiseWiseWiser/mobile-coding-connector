import { useRef, useState } from 'react';
import type { BrowserSessions } from './useBrowserSessions';

export interface PendingTabClose {
    session: string;
    tabId: number;
    operationId: string;
    title: string;
    opener: HTMLElement | null;
    uncertain?: string;
}

/** Owns one in-flight close; a failed decision is never automatically retried. */
export function useBrowserTabClose(
    sessions: BrowserSessions,
    onClosed: (session: string, tabId: number) => void,
    onError: (message: string) => void,
) {
    const [pending, setPending] = useState<PendingTabClose | null>(null);
    const [deciding, setDeciding] = useState(false);
    const inFlight = useRef(false);

    const close = async (session: string, tabId: number): Promise<boolean> => {
        if (inFlight.current || pending) return false;
        inFlight.current = true;
        const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        const tab = sessions.sessions.find((row) => row.id === session)?.tabs?.find((entry) => entry.tab_id === tabId);
        try {
            const result = await sessions.close(session, tabId);
            if (result.error) {
                if (/outcome is unknown|failed to fetch|networkerror|load failed/i.test(result.error)) {
                    setPending({ session, tabId, operationId: '', title: tab?.title?.trim() || 'this tab', opener,
                        uncertain: `${result.error} Check Chrome and refresh the tab list before retrying.` });
                } else onError(result.error);
                return false;
            }
            if (result.closeConfirmation) {
                setPending({ session, tabId, operationId: result.closeConfirmation.operation_id, title: tab?.title?.trim() || 'this tab', opener });
                return false;
            }
            onClosed(session, tabId);
            return true;
        } finally { inFlight.current = false; }
    };

    const choose = async (accept: boolean) => {
        if (!pending || pending.uncertain || inFlight.current) return;
        inFlight.current = true;
        setDeciding(true);
        try {
            const result = await sessions.resolveTabClose(pending.session, pending.operationId, accept);
            if (result.error || !result.closeOutcome) {
                setPending({ ...pending, uncertain: result.error || 'The close result is unknown. Check Chrome and refresh the tab list.' });
            } else {
                if (result.closeOutcome === 'closed') onClosed(pending.session, pending.tabId);
                setPending(null);
            }
        } finally { setDeciding(false); inFlight.current = false; }
    };

    return { pending, deciding, close, choose, dismiss: () => setPending(null) };
}
