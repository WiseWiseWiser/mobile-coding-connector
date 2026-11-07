import type { BrowserTab } from './browserWire';

/** One window the search may look through, already in creation order. */
export interface SearchWindow {
    key: string;
    title: string;
    tabs: readonly BrowserTab[];
}

export interface SearchHit {
    tabId: number;
    index: number;
    title: string;
    url: string;
}

export interface SearchGroup {
    key: string;
    title: string;
    hits: SearchHit[];
}

export interface Span {
    start: number;
    end: number;
}

/**
 * The agent transport page is infrastructure, not a tab the person browses,
 * so a search never offers it back to them.
 */
const HIDDEN_ROLE = 'session_page';

/** Lowercased words, whitespace-split. An empty query yields no tokens. */
export function searchTokens(query: string): string[] {
    return query
        .toLowerCase()
        .split(/\s+/)
        .map((token) => token.trim())
        .filter(Boolean);
}

/** Every token must appear somewhere in the tab's title or URL. */
export function tabMatches(tab: BrowserTab, tokens: readonly string[]): boolean {
    if (tokens.length === 0 || tab.role === HIDDEN_ROLE) return false;
    const haystack = `${tab.title ?? ''} ${tab.url ?? ''}`.toLowerCase();
    return tokens.every((token) => haystack.includes(token));
}

/**
 * Groups the matching tabs by window, keeping the caller's window order (the
 * rail's creation order) and each window's tab order. An empty query matches
 * nothing: the caller shows its normal panes instead of results.
 */
export function searchWindows(windows: readonly SearchWindow[], query: string): SearchGroup[] {
    const tokens = searchTokens(query);
    if (tokens.length === 0) return [];
    const groups: SearchGroup[] = [];
    for (const window of windows) {
        const hits: SearchHit[] = [];
        for (const tab of window.tabs) {
            if (!tabMatches(tab, tokens)) continue;
            hits.push({
                tabId: tab.tab_id,
                index: typeof tab.index === 'number' ? tab.index : Number.MAX_SAFE_INTEGER,
                title: tab.title?.trim() || '',
                url: tab.url?.trim() || '',
            });
        }
        if (hits.length === 0) continue;
        hits.sort((a, b) => a.index - b.index);
        groups.push({ key: window.key, title: window.title, hits });
    }
    return groups;
}

export function matchCount(groups: readonly SearchGroup[]): number {
    return groups.reduce((total, group) => total + group.hits.length, 0);
}

/**
 * Where to paint `<mark>`: every token occurrence, overlapping occurrences
 * merged so nested marks never nest in the DOM. Indices are into `text`.
 */
export function highlightSpans(text: string, tokens: readonly string[]): Span[] {
    if (!text || tokens.length === 0) return [];
    const haystack = text.toLowerCase();
    const found: Span[] = [];
    for (const token of tokens) {
        if (!token) continue;
        let at = haystack.indexOf(token);
        while (at !== -1) {
            found.push({ start: at, end: at + token.length });
            at = haystack.indexOf(token, at + token.length);
        }
    }
    found.sort((a, b) => a.start - b.start || b.end - a.end);
    const spans: Span[] = [];
    for (const span of found) {
        const last = spans[spans.length - 1];
        if (last && span.start < last.end) {
            last.end = Math.max(last.end, span.end);
            continue;
        }
        spans.push({ ...span });
    }
    return spans;
}
