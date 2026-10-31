import type { BrowserTab } from './browserWire';
import type { WindowRow } from './windowClose';

/**
 * Open/close intent for the browser sheet's window rail (the left pane).
 *
 * The rail is collapsed by default on phones — tabs are the everyday surface —
 * and expanded by default where there is room for both panes (768px and up).
 * Whatever the person last chose wins and survives sheet restarts.
 */
const RAIL_KEY = 'bw.railOpen';

/** The intent this device stored, or null when it has none. */
export function readRailStored(): boolean | null {
    try {
        const raw = window.localStorage.getItem(RAIL_KEY);
        if (raw === '1') return true;
        if (raw === '0') return false;
        return null;
    } catch {
        // No window (tests) or storage denied: fall back to the layout default.
        return null;
    }
}

export function storeRailOpen(open: boolean): void {
    try {
        window.localStorage.setItem(RAIL_KEY, open ? '1' : '0');
    } catch {
        // Private mode: the intent just lives for this visit.
    }
}

/** Does this host have room to show both panes at once? */
export function railWide(): boolean {
    try {
        return window.matchMedia('(min-width: 768px)').matches;
    } catch {
        return false;
    }
}

/** First paint: stored intent wins; otherwise the layout decides. */
export function railInitialOpen(stored: boolean | null, wide: boolean): boolean {
    if (stored !== null) return stored;
    return wide;
}

export function nextRailOpen(open: boolean): boolean {
    return !open;
}

/** The TABS header names the window whose tabs are listed. */
export function paneTitleFor(row: Pick<WindowRow, 'title' | 'sessionId'>): string {
    return row.title.trim() || row.sessionId || 'Chrome window';
}

/**
 * The right pane lists the selected window's tabs. The connected window uses
 * the live poll result; every other window reads the tabs the sessions poll
 * already carries, and a window without any report shows an empty pane.
 */
export function paneTabsFor(
    session: { tabs?: BrowserTab[] } | undefined,
    currentTabs: BrowserTab[],
    isCurrent: boolean,
): BrowserTab[] {
    if (isCurrent || !session) return currentTabs;
    return session.tabs ?? [];
}
