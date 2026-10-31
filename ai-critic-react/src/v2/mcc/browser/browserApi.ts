import type {
    AttachWindowResponse,
    BrowserStatus,
    CloseWindowResponse,
    SessionsResponse,
    TabOpResponse,
} from './browserWire';
import type { BookmarkDocument, BookmarkNode } from './browserBookmarks';

/** Tab ops the server accepts on POST /api/browser/tab. */
export type TabOp = 'activate' | 'close' | 'create';

/**
 * One JSON response, with the server's own words when it refuses.
 *
 * Every browser endpoint answers `{ok:false,error:"…"}` on refusal, and those
 * sentences are what the UI shows — "cannot close the last tab: open a new tab
 * first" beats "request failed (409)".
 */
async function json<T>(response: Response): Promise<T> {
    let body: unknown = null;
    try {
        body = await response.json();
    } catch {
        body = null;
    }
    if (!response.ok) {
        const detail = body && typeof body === 'object' && 'error' in body ? String((body as { error?: unknown }).error ?? '') : '';
        throw new Error(detail || `request failed (${response.status})`);
    }
    return body as T;
}

function post(path: string, body: unknown): Promise<Response> {
    return fetch(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

/** The rail: live sessions in a stable id order, plus explicitly opened ones. */
export function fetchSessions(): Promise<SessionsResponse> {
    return fetch('/api/browser/sessions').then((response) => json<SessionsResponse>(response));
}

/** An explicit user action; passive viewers do not launch Chrome. */
export function startBrowser(): Promise<BrowserStatus> {
    return post('/api/browser/ensure', {}).then((response) => json<BrowserStatus>(response));
}

/** Makes `id` the one session every viewer shows, reopening its window if needed. */
export function switchSession(id: string): Promise<BrowserStatus> {
    return post('/api/browser/session', { session_id: id }).then((response) => json<BrowserStatus>(response));
}

/** Navigate exactly the selected content tab, even if another viewer switches. */
export function navigateTarget(sessionId: string, tabId: number, url: string): Promise<BrowserStatus> {
    return post('/api/browser/navigate', { session_id: sessionId, tab_id: tabId, url }).then((response) => json<BrowserStatus>(response));
}

/** Mints a fresh session id and opens its page in a new Chrome window. */
export function openNewSession(): Promise<BrowserStatus> {
    return post('/api/browser/session', { new: true }).then((response) => json<BrowserStatus>(response));
}

/** Activates, closes or creates one tab inside a session's window. */
export function tabOp(session: string, op: TabOp, tabId = 0, url = ''): Promise<TabOpResponse> {
    return post('/api/browser/tab', { session_id: session, op, tab_id: tabId, url }).then((response) =>
        json<TabOpResponse>(response),
    );
}

export function resolveTabClose(session: string, operationId: string, accept: boolean): Promise<TabOpResponse> {
    return post('/api/browser/tab', { session_id: session, op: 'confirm_close', operation_id: operationId, accept }).then((response) => json<TabOpResponse>(response));
}

/**
 * Removes one whole Chrome window. Exactly one selector: a live session id, or
 * the numeric window id for a discovered window that has no session yet.
 */
export function closeWindow(target: { window_id: number } | { session_id: string }): Promise<CloseWindowResponse> {
    return post('/api/browser/window/close', target).then((response) => json<CloseWindowResponse>(response));
}

/** A bookmark tree: Chrome is read-only, local is app-owned. */
export function fetchBookmarks(source: 'chrome' | 'local' = 'local'): Promise<BookmarkDocument> {
    const query = source === 'chrome' ? '?source=chrome' : '';
    return fetch(`/api/bookmarks${query}`).then((response) => json<BookmarkDocument>(response));
}

/** Saves one URL under a folder, `root` for the top level. */
export function createBookmark(name: string, url: string, parentId = 'root'): Promise<BookmarkNode> {
    return post('/api/bookmarks', { parent_id: parentId, type: 'url', name, url }).then((response) =>
        json<BookmarkNode>(response),
    );
}

/**
 * Adopts a Chrome window that has no agent session yet by opening a /go page
 * in that exact window. The window becomes a normal live session; `sessionId`
 * is the synthetic id the rail already shows, so the row does not move.
 */
export function attachWindow(windowId: number, sessionId?: string): Promise<AttachWindowResponse> {
    return post('/api/browser/window/attach', { window_id: windowId, session_id: sessionId ?? '' }).then((response) =>
        json<AttachWindowResponse>(response),
    );
}
