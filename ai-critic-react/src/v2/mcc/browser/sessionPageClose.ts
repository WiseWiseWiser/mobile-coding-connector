import type { BrowserTab } from './browserWire';

/** The copy a session-page close confirmation shows. Pure so tests can read it. */
export interface SessionPageCloseCopy {
    title: string;
    /** The page that is about to go away; empty when nothing is registered. */
    anchor: string;
    body: string;
    note: string;
    confirm: string;
}

/**
 * The /go page is the tab that registers a Chrome window with the agent, and it
 * is a tab like any other: it closes, but never blind. Callers show this copy
 * first, then send the close.
 */
export function isSessionPageTab(tab: Pick<BrowserTab, 'role' | 'url'> | undefined): boolean {
    if (!tab) return false;
    if (tab.role === 'session_page') return true;
    return /\/go\?session=/.test(tab.url ?? '');
}

/**
 * tabCount counts every tab in the window, the session page included. One means
 * closing it takes Chrome's window with it, which is the case that needs the
 * loudest wording and the honest button label.
 */
export function sessionPageCloseCopy(input: { sessionId: string; tabCount: number; current: boolean }): SessionPageCloseCopy {
    const others = input.tabCount - 1;
    if (others <= 0) {
        return {
            title: 'Close the session page?',
            anchor: '',
            body: 'This is the only tab in its Chrome window. Closing it closes the window too, and the mirror moves to the next live window.',
            note: `The session ${input.sessionId} ends. The last Chrome window is never closed.`,
            confirm: 'Close window',
        };
    }
    return {
        title: 'Close the session page?',
        anchor: `/go?session=${input.sessionId}`,
        body: `This page registers the Chrome window with the agent. Closing it disconnects the mirror here; the ${others === 1 ? 'other tab keeps' : `${others} other tabs keep`} running.`,
        note: input.current
            ? 'Reopen the session page from the rail ("+") when you need the mirror back.'
            : 'This window keeps running in Chrome without an agent session.',
        confirm: 'Close page',
    };
}
