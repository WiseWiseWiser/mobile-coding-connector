import { useEffect, useState } from 'react';
import { createBookmark, fetchBookmarks } from './browserApi';
import { flattenBookmarks, hasBookmark } from './browserBookmarks';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';

export interface BrowserBookmarks {
    items: BookmarkItem[];
    /** The active tree, kept whole so the UI can navigate folders. */
    doc: BookmarkDocument | null;
    loading: boolean;
    error: string | null;
    source: BookmarkSource;
    /** Chrome's display profile name, blank for the local store. */
    profile: string;
    /** Local saves, but only after that tree has been read. */
    localCount: number | null;
    refresh: () => Promise<void>;
    /** Saves under the local root; returns an error sentence, or null. */
    add: (name: string, url: string) => Promise<string | null>;
    has: (url: string | null | undefined) => boolean;
    showLocal: () => void;
    showChrome: () => void;
}

interface TreeState {
    doc: BookmarkDocument | null;
    loading: boolean;
    error: string | null;
}

function message(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
}

/**
 * Chrome's bookmark tree, read through the server on every refresh. The
 * local tree stays as a fallback and as the only writable store.
 */
export function useBrowserBookmarks(active: boolean): BrowserBookmarks {
    const [source, setSource] = useState<BookmarkSource>('chrome');
    const [chrome, setChrome] = useState<TreeState>({ doc: null, loading: active, error: null });
    const [local, setLocal] = useState<TreeState>({ doc: null, loading: false, error: null });

    const refreshChrome = async () => {
        setChrome((previous) => ({ ...previous, loading: true, error: null }));
        try {
            const doc = await fetchBookmarks('chrome');
            if (!doc || !Array.isArray(doc.roots)) throw new Error('the Chrome bookmark tree is not readable');
            setChrome({ doc, loading: false, error: null });
        } catch (err) {
            setChrome({ doc: null, loading: false, error: message(err) });
        }
    };

    const refreshLocal = async () => {
        setLocal((previous) => ({ ...previous, loading: true, error: null }));
        try {
            const doc = await fetchBookmarks('local');
            if (!doc || !Array.isArray(doc.roots)) throw new Error('the local bookmark tree is not readable');
            setLocal({ doc, loading: false, error: null });
        } catch (err) {
            setLocal({ doc: null, loading: false, error: message(err) });
        }
    };

    useEffect(() => {
        if (!active) return;
        void refreshChrome();
        // One read when the browser view becomes active; refresh is manual.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [active]);

    const chromeItems = flattenBookmarks(chrome.doc);
    const localItems = flattenBookmarks(local.doc);
    const localCount = local.doc ? localItems.length : null;

    // Load the fallback only when Chrome is empty, so the local store can
    // still be reached (and a first bookmark saved) without merging trees.
    useEffect(() => {
        if (!active || source !== 'chrome') return;
        if (chrome.loading || chrome.error || !chrome.doc) return;
        if (chromeItems.length > 0) return;
        if (local.doc || local.loading || local.error) return;
        void refreshLocal();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [active, source, chrome, local]);

    const add = async (name: string, url: string): Promise<string | null> => {
        try {
            const node = await createBookmark(name, url);
            setLocal((previous) => {
                if (!previous.doc) return previous;
                if (hasBookmark(flattenBookmarks(previous.doc), url)) return previous;
                const roots = previous.doc.roots.map((root) => {
                    if (root.id !== 'root') return root;
                    return {
                        ...root,
                        children: [
                            ...(root.children ?? []),
                            {
                                type: 'url',
                                id: node.id || url,
                                name: node.name?.trim() || name,
                                url: node.url || url,
                            },
                        ],
                    };
                });
                return { ...previous, doc: { ...previous.doc, roots }, error: null };
            });
            // A save can race the first local read; pick the new row up then.
            if (!local.doc) void refreshLocal();
            return null;
        } catch (err) {
            return message(err);
        }
    };

    const showLocal = () => {
        setSource('local');
        if (!local.doc && !local.loading && !local.error) void refreshLocal();
    };

    const showChrome = () => {
        setSource('chrome');
        if (!chrome.doc && !chrome.loading && !chrome.error) void refreshChrome();
    };

    const activeTree = source === 'chrome' ? chrome : local;
    const items = source === 'chrome' ? chromeItems : localItems;

    return {
        items,
        doc: activeTree.doc,
        loading: activeTree.loading,
        error: activeTree.error,
        source,
        profile: source === 'chrome' ? chrome.doc?.profile?.trim() || '' : '',
        localCount,
        refresh: source === 'chrome' ? refreshChrome : refreshLocal,
        add,
        has: (url) => hasBookmark(items, url),
        showLocal,
        showChrome,
    };
}
