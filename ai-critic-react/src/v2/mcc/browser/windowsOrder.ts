import type { BrowserTab, SessionInfo } from './browserWire';

/**
 * Chrome never reports when a window was created, but every window id comes
 * from a counter in the browser process that only moves forward. Two windows
 * therefore keep a stable order by id, and the older window always carries the
 * smaller one. The counter restarts with the browser, which is why callers
 * prune to the live window set instead of trusting ids across restarts.
 */
export function windowOrderRank(session: Pick<SessionInfo, 'window_id'>): number {
    return typeof session.window_id === 'number' && session.window_id > 0
        ? session.window_id
        : Number.MAX_SAFE_INTEGER;
}

/**
 * Windows oldest first. Rows with no live window id (a remembered session
 * whose Chrome window is gone) keep their given order at the end, where they
 * read as "not a window right now" instead of pretending to be young.
 */
export function orderByCreation<T extends Pick<SessionInfo, 'window_id'>>(sessions: readonly T[]): T[] {
    return sessions
        .map((session, at) => ({ session, at, rank: windowOrderRank(session) }))
        .sort((a, b) => a.rank - b.rank || a.at - b.at)
        .map((entry) => entry.session);
}

/** Tabs left to right. `index` is Chrome's own position; ids only break ties. */
export function orderTabs<T extends Pick<BrowserTab, 'index' | 'tab_id'>>(tabs: readonly T[]): T[] {
    return tabs
        .map((tab, at) => ({
            tab,
            at,
            rank: typeof tab.index === 'number' ? tab.index : Number.MAX_SAFE_INTEGER,
        }))
        .sort((a, b) => a.rank - b.rank || a.at - b.at)
        .map((entry) => entry.tab);
}
