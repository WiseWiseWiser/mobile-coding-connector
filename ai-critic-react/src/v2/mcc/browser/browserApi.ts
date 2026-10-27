import type { BrowserStatus, SessionsResponse, TabOpResponse } from './browserWire';

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

/** The rail: live sessions plus explicitly opened ones, current first. */
export function fetchSessions(): Promise<SessionsResponse> {
    return fetch('/api/browser/sessions').then((response) => json<SessionsResponse>(response));
}

/** Makes `id` the one session every viewer shows, reopening its window if needed. */
export function switchSession(id: string): Promise<BrowserStatus> {
    return post('/api/browser/session', { session_id: id }).then((response) => json<BrowserStatus>(response));
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
