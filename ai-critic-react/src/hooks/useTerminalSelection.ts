import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import type { TerminalTab } from './useTerminalTabs';

interface SelectionOptions {
    tabs: TerminalTab[];
    activeTabId: string | null;
    loading: boolean;
    isVisible: boolean;
    setActiveTabId: (id: string) => void;
}

/** URL changes select a loaded session; local selection is then canonicalized
 * using the backend identity, never the transient tab ID or display name. */
export function useTerminalSelection({ tabs, activeTabId, loading, isVisible, setActiveTabId }: SelectionOptions) {
    const [params, setParams] = useSearchParams();
    const search = params.toString();
    const seenSearch = useRef<string | null>(null);
    const [notice, setNotice] = useState<string | null>(null);
    useEffect(() => {
        if (loading || !isVisible || tabs.length === 0) return;
        if (seenSearch.current !== search) {
            seenSearch.current = search;
            const requested = params.get('terminal');
            if (requested) {
                const target = tabs.find(t => t.sessionId === requested);
                // Synchronize with external browser history (including Back/Forward).
                // eslint-disable-next-line react-hooks/set-state-in-effect
                if (target) setNotice(null);
                if (target && target.id !== activeTabId) {
                    setActiveTabId(target.id);
                    return;
                }
                if (!target) setNotice('The selected terminal is no longer available. Showing another terminal.');
            }
        }
        const active = tabs.find(t => t.id === activeTabId);
        if (!active) return;
        const id = active.sessionId;
        if ((params.get('terminal') ?? undefined) === id) return;
        const next = new URLSearchParams(params);
        if (id) next.set('terminal', id);
        else next.delete('terminal');
        seenSearch.current = next.toString();
        setParams(next, { replace: true });
    }, [search, tabs, activeTabId, loading, isVisible, params, setParams, setActiveTabId]);
    return notice;
}
