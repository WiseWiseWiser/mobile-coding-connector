import { useState } from 'react';
import { bookmarkFallbackName, checkAddress, hostLabel, isHttpURL } from './browserAddress';
import type { OpenMode } from './BrowserOpenPalette';

interface ComposerSessions {
    busy: boolean;
    create: (session: string, url?: string) => Promise<{ error?: string }>;
}

interface ComposerBookmarks {
    has: (url?: string) => boolean;
    add: (name: string, url: string) => Promise<string | null>;
}

interface BrowserComposerArgs {
    sessions: ComposerSessions;
    bookmarks: ComposerBookmarks;
    sessionId: string;
    readOnly: boolean;
    currentUrl?: string;
    /** Active tab title, then the window title, for a saved bookmark's name. */
    pageTitle?: string;
    notify: (message: string, warn?: boolean) => void;
    navigate: (url: string) => void;
    closeSheet: () => void;
    resetZoom: () => void;
    refresh: () => void;
}

/**
 * Tab creation, address navigation and bookmark saving, shared by the desktop
 * palette and the phone sheet. No tab is created until a composer submits.
 */
export function useBrowserComposer(args: BrowserComposerArgs) {
    const [composer, setComposer] = useState<OpenMode | null>(null);
    const [seed, setSeed] = useState('');
    const { sessions, bookmarks, sessionId, readOnly, currentUrl, pageTitle, notify } = args;

    const createTab = async (url = '') => {
        if (!sessionId) {
            notify('Attach a Chrome window before opening a tab.', true);
            return false;
        }
        if (readOnly) {
            notify('This browser-agent cannot open tabs — it needs 1.0.20.', true);
            return false;
        }
        const result = await sessions.create(sessionId, url);
        if (result.error) {
            notify(result.error, true);
            return false;
        }
        args.resetZoom();
        args.refresh();
        return true;
    };

    const createFromComposer = async (url: string) => {
        const ok = await createTab(url);
        if (ok) notify(url ? `Opened ${hostLabel(url)} in a new tab.` : 'New blank tab opened.');
        return ok;
    };

    const openComposer = (mode: OpenMode, initial = '') => {
        if (!sessionId) {
            notify('Attach a Chrome window before opening a page.', true);
            return;
        }
        if (readOnly) {
            notify('This browser-agent cannot open tabs — it needs 1.0.20.', true);
            return;
        }
        setSeed(initial);
        setComposer(mode);
    };

    const saveCurrentPage = async () => {
        if (!currentUrl || !isHttpURL(currentUrl)) {
            notify('Open an http(s) page before saving a bookmark.', true);
            return false;
        }
        const name = pageTitle?.trim() || bookmarkFallbackName(currentUrl);
        const failure = await bookmarks.add(name, currentUrl);
        if (failure) {
            notify(failure, true);
            return false;
        }
        notify('Saved to local bookmarks.');
        return true;
    };

    /** `Go`, Enter and the phone address sheet navigate the tab already in view. */
    const openAddress = (raw: string) => {
        const check = checkAddress(raw);
        if (!check.ok) {
            notify(check.error, true);
            return;
        }
        args.closeSheet();
        setComposer(null);
        args.navigate(check.target);
        notify(`Opening ${hostLabel(check.target)}…`);
    };

    return {
        composer,
        seed,
        closeComposer: () => setComposer(null),
        createTab,
        createFromComposer,
        openComposer,
        saveCurrentPage,
        openAddress,
        bookmarked: bookmarks.has(currentUrl),
    };
}
