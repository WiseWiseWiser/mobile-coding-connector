import { useEffect } from 'react';
import { BookmarkSaveCard } from './BookmarkSaveCard';
import { BookmarkSourceMeta } from './BookmarkSourceMeta';
import { BrowserBookmarkBrowser } from './BrowserBookmarkBrowser';
import {
    bookmarkSourceFooter,
    bookmarkSourceKicker,
    localBookmarksLink,
} from './browserBookmarks';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';

export type OpenMode = 'newtab' | 'bookmarks';

interface BrowserOpenPaletteProps {
    mode: OpenMode;
    initialUrl: string;
    currentUrl?: string;
    bookmarked: boolean;
    bookmarks: BookmarkItem[];
    bookmarkDoc: BookmarkDocument | null;
    bookmarksLoading: boolean;
    bookmarksError: string | null;
    bookmarkSource: BookmarkSource;
    bookmarkProfile: string;
    localBookmarkCount: number | null;
    disabled: boolean;
    busy: boolean;
    onClose: () => void;
    onCreate: (url: string) => Promise<boolean>;
    onSaveCurrent: () => Promise<boolean>;
    onRefreshBookmarks: () => void;
    onShowLocalBookmarks: () => void;
    onShowChromeBookmarks: () => void;
    onShowBookmarks: () => void;
}

/**
 * One composer for `+`, `New` and the bookmark button. The same field filters
 * the folder tree and opens addresses (or a blank tab) from the same submit;
 * nothing is created until the text is submitted.
 */
export function BrowserOpenPalette({
    mode,
    initialUrl,
    currentUrl,
    bookmarked,
    bookmarks,
    bookmarkDoc,
    bookmarksLoading,
    bookmarksError,
    bookmarkSource,
    bookmarkProfile,
    localBookmarkCount,
    disabled,
    busy,
    onClose,
    onCreate,
    onSaveCurrent,
    onRefreshBookmarks,
    onShowLocalBookmarks,
    onShowChromeBookmarks,
    onShowBookmarks,
}: BrowserOpenPaletteProps) {
    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [onClose]);

    const local = bookmarkSource === 'local';
    const fallback = localBookmarksLink(localBookmarkCount);
    const openLocalBookmarks = () => {
        onShowLocalBookmarks();
        onShowBookmarks();
    };
    const open = async (item: BookmarkItem) => {
        if (disabled || busy) return;
        const ok = await onCreate(item.url);
        if (ok) onClose();
    };
    const create = async (value: string) => {
        if (disabled || busy) return;
        const ok = await onCreate(value);
        if (ok) onClose();
    };
    const emptyText =
        mode === 'newtab'
            ? local
                ? 'Open a saved page, or type an address above.'
                : 'No Chrome bookmarks. Type an address above.'
            : local
              ? 'No bookmarks yet.'
              : 'No Chrome bookmarks.';

    return (
        <div className="bw-palette-layer" onMouseDown={onClose}>
            <div
                className={`bw-palette is-${mode}`}
                role="dialog"
                aria-modal="true"
                aria-label={mode === 'newtab' ? 'Open a new tab' : 'Bookmarks'}
                onMouseDown={(event) => event.stopPropagation()}
            >
                <div className="bw-pal-head">
                    <div>
                        <span className="bw-pal-kicker">
                            {mode === 'newtab' ? 'NEW TAB' : bookmarkSourceKicker(bookmarkSource)}
                        </span>
                        <h3>{mode === 'newtab' ? 'Where should it go?' : 'Open a saved page'}</h3>
                    </div>
                    <button type="button" className="bw-pal-close" onClick={onClose} aria-label="Close">
                        ×
                    </button>
                </div>

                {mode === 'bookmarks' && (
                    <BookmarkSourceMeta
                        source={bookmarkSource}
                        profile={bookmarkProfile}
                        count={bookmarks.length}
                        disabled={disabled || busy}
                        onShowChrome={local ? onShowChromeBookmarks : undefined}
                    />
                )}

                {mode === 'bookmarks' && local && (
                    <BookmarkSaveCard
                        currentUrl={currentUrl}
                        bookmarked={bookmarked}
                        disabled={disabled || busy}
                        onSave={onSaveCurrent}
                    />
                )}

                <BrowserBookmarkBrowser
                    doc={bookmarkDoc}
                    loading={bookmarksLoading}
                    error={bookmarksError}
                    disabled={disabled || busy}
                    addressMode
                    seed={mode === 'newtab' ? initialUrl : ''}
                    autoFocus
                    emptyText={emptyText}
                    emptyHint={!local ? 'This profile has none.' : undefined}
                    emptyAction={
                        !local
                            ? {
                                  label: fallback,
                                  onClick: mode === 'newtab' ? openLocalBookmarks : onShowLocalBookmarks,
                              }
                            : undefined
                    }
                    onCreate={(value) => void create(value)}
                    onOpen={(item) => void open(item)}
                />

                <div className="bw-pal-foot">
                    {mode === 'newtab' ? (
                        <button
                            type="button"
                            className="bw-pal-blank"
                            onClick={() => void onCreate('').then((ok) => ok && onClose())}
                            disabled={disabled || busy}
                        >
                            Open a blank tab
                        </button>
                    ) : (
                        <button type="button" className="bw-pal-blank" onClick={onRefreshBookmarks}>
                            Refresh bookmarks
                        </button>
                    )}
                    <span className="bw-spacer" />
                    <span className="bw-pal-path">{bookmarkSourceFooter(bookmarkSource)}</span>
                </div>
            </div>
        </div>
    );
}
