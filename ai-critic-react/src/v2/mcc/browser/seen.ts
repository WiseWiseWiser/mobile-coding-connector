/**
 * Which windows and tabs the person has already looked at.
 *
 * This is a client-local ledger on purpose: "new" is about one pair of eyes,
 * not about the browser. It lives in localStorage so it survives reloads and
 * app restarts, and it is seeded on the first run so an existing browser does
 * not greet anyone with every row badged.
 */
export const SEEN_KEY = 'ai-critic.browser.seen.v1';

export interface SeenState {
    v: 1;
    /** False until the first enumeration has baselined what already exists. */
    seeded: boolean;
    /** Live window keys the person has opened. */
    windows: string[];
    /** `<window key>:<tab id>` for every tab already looked at. */
    tabs: string[];
}

export function emptySeen(): SeenState {
    return { v: 1, seeded: false, windows: [], tabs: [] };
}

function stringList(value: unknown): string[] {
    return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

/** Never throws: a corrupt or foreign blob reads as "nothing seen yet". */
export function parseSeen(raw: string | null | undefined): SeenState {
    if (!raw) return emptySeen();
    let body: unknown;
    try {
        body = JSON.parse(raw);
    } catch {
        return emptySeen();
    }
    if (!body || typeof body !== 'object') return emptySeen();
    const record = body as Partial<SeenState>;
    if (record.v !== 1) return emptySeen();
    return {
        v: 1,
        seeded: Boolean(record.seeded),
        windows: stringList(record.windows),
        tabs: stringList(record.tabs),
    };
}

export function serializeSeen(state: SeenState): string {
    return JSON.stringify({
        v: 1,
        seeded: state.seeded,
        windows: state.windows,
        tabs: state.tabs,
    });
}

/** Tab marks are scoped to their window: the same tab id means nothing elsewhere. */
export function tabSeenKey(windowKey: string, tabId: number | string): string {
    return `${windowKey}:${tabId}`;
}

export function isWindowSeen(state: SeenState, key: string): boolean {
    return state.windows.includes(key);
}

export function isTabSeen(state: SeenState, key: string): boolean {
    return state.tabs.includes(key);
}

/** Marks a window read. Returns `state` unchanged when it already was. */
export function withWindowSeen(state: SeenState, key: string): SeenState {
    if (!key || state.windows.includes(key)) return state;
    return { ...state, windows: [...state.windows, key] };
}

export function withTabSeen(state: SeenState, key: string): SeenState {
    if (!key || state.tabs.includes(key)) return state;
    return { ...state, tabs: [...state.tabs, key] };
}

/** The first run: everything that exists right now counts as already seen. */
export function baselineSeen(windowKeys: readonly string[], tabKeys: readonly string[]): SeenState {
    return {
        v: 1,
        seeded: true,
        windows: [...new Set(windowKeys)],
        tabs: [...new Set(tabKeys)],
    };
}

/**
 * Drops marks for windows and tabs that no longer exist, so a closed window
 * that later reopens with a fresh id is new again while a window that stays
 * open keeps its mark. Returns `state` itself when there is nothing to drop,
 * which keeps the polling hook from re-rendering on every tick.
 */
export function pruneSeen(
    state: SeenState,
    windowKeys: readonly string[],
    tabKeys: readonly string[],
): SeenState {
    if (!state.seeded) return state;
    // No enumeration means "the poll has not answered yet", not "every window
    // closed": pruning against it would silently unread everything and badge the
    // whole rail on the next tick.
    if (windowKeys.length === 0) return state;
    const windows = new Set(windowKeys);
    const tabs = new Set(tabKeys);
    const keptWindows = state.windows.filter((key) => windows.has(key));
    const keptTabs = state.tabs.filter((key) => tabs.has(key));
    if (keptWindows.length === state.windows.length && keptTabs.length === state.tabs.length) {
        return state;
    }
    return { ...state, windows: keptWindows, tabs: keptTabs };
}

/** How many of these windows the person has not opened yet. */
export function unreadCount(state: SeenState, windowKeys: readonly string[]): number {
    if (!state.seeded) return 0;
    return windowKeys.filter((key) => !state.windows.includes(key)).length;
}
