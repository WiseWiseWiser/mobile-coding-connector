import { useEffect, useState } from 'react';
import type { BrowserTab } from './browserWire';
import {
    baselineSeen,
    isTabSeen,
    isWindowSeen,
    parseSeen,
    pruneSeen,
    SEEN_KEY,
    serializeSeen,
    tabSeenKey,
    unreadCount,
    withTabSeen,
    withWindowSeen,
} from './seen';
import type { SeenState } from './seen';

export interface SeenTarget {
    /** Stable window key: the session id or the `sess-window-<id>` anchor. */
    key: string;
    tabs: readonly BrowserTab[];
}

export interface SeenMarks {
    /** False until the first enumeration has been baselined. */
    ready: boolean;
    windowNew: (key: string) => boolean;
    tabNew: (windowKey: string, tabId: number) => boolean;
    markWindow: (key: string) => void;
    markTab: (windowKey: string, tabId: number) => void;
    unread: (windowKeys: readonly string[]) => number;
}

function readStored(): string | null {
    try {
        return localStorage.getItem(SEEN_KEY);
    } catch {
        return null;
    }
}

function store(state: SeenState) {
    try {
        localStorage.setItem(SEEN_KEY, serializeSeen(state));
    } catch {
        // Storage can be full or forbidden; the badges just reset next load.
    }
}

/**
 * Tracks unread windows and tabs for one host.
 *
 * The pure ledger lives in `seen.ts`; this hook adds the two things that need
 * a component: reading/writing localStorage and keeping the marks in step with
 * the polled window list. Badges only ever clear through `markWindow`/`markTab`
 * — rendering a row is not reading it — and the first poll baselines instead of
 * badging, so an existing browser opens quiet.
 */
export function useSeenMarks(targets: readonly SeenTarget[]): SeenMarks {
    const [state, setState] = useState<SeenState>(() => parseSeen(readStored()));
    // One string dep: the poll rebuilds these arrays every tick, and a fresh
    // array identity would re-run the effect on every tick for nothing.
    const signature = JSON.stringify([
        targets.map((target) => target.key),
        targets.map((target) => target.tabs.map((tab) => tabSeenKey(target.key, tab.tab_id))),
    ]);

    useEffect(() => {
        const [windowKeys, tabKeys] = JSON.parse(signature) as [string[], string[][]];
        const flatTabs = tabKeys.flat();
        setState((was) => {
            if (!was.seeded) {
                return windowKeys.length === 0 ? was : baselineSeen(windowKeys, flatTabs);
            }
            const next = pruneSeen(was, windowKeys, flatTabs);
            return next === was ? was : next;
        });
    }, [signature]);

    useEffect(() => {
        if (state.seeded) store(state);
    }, [state]);

    // Two /browser tabs share one ledger. Adopting the other tab's write keeps a
    // badge cleared there from reappearing here on the next poll; adopting an
    // unchanged value is a no-op, and storage events never echo to the writer, so
    // this cannot ping-pong.
    useEffect(() => {
        const onStorage = (event: StorageEvent) => {
            if (event.key !== null && event.key !== SEEN_KEY) return;
            const incoming = parseSeen(event.newValue);
            if (!incoming.seeded) return;
            setState((was) => (serializeSeen(was) === serializeSeen(incoming) ? was : incoming));
        };
        window.addEventListener('storage', onStorage);
        return () => window.removeEventListener('storage', onStorage);
    }, []);

    return {
        ready: state.seeded,
        windowNew: (key) => state.seeded && !isWindowSeen(state, key),
        tabNew: (windowKey, tabId) => state.seeded && !isTabSeen(state, tabSeenKey(windowKey, tabId)),
        markWindow: (key) => setState((was) => withWindowSeen(was, key)),
        markTab: (windowKey, tabId) => setState((was) => withTabSeen(was, tabSeenKey(windowKey, tabId))),
        unread: (windowKeys) => unreadCount(state, windowKeys),
    };
}
