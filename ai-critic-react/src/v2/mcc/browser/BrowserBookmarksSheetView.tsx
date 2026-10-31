import { BookmarkSaveCard } from './BookmarkSaveCard';
import { BookmarkSourceMeta } from './BookmarkSourceMeta';
import { BrowserBookmarkBrowser } from './BrowserBookmarkBrowser';
import { bookmarkSourceKicker, localBookmarksLink } from './browserBookmarks';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';

interface BrowserBookmarksSheetViewProps {
    source: BookmarkSource;
    profile: string;
    localCount: number | null;
    items: BookmarkItem[];
    doc: BookmarkDocument | null;
    loading: boolean;
    error: string | null;
    currentUrl?: string;
    bookmarked: boolean;
    disabled: boolean;
    onBack: () => void;
    onClose: () => void;
    onRefresh: () => void;
    onSave: () => Promise<boolean>;
    onOpen: (item: BookmarkItem) => void;
    onCreate: (value: string) => Promise<boolean>;
    onShowLocal: () => void;
    onShowChrome: () => void;
}

/** The phone sheet's bookmark list, with Chrome as the read-only default. */
export function BrowserBookmarksSheetView({
    source,
    profile,
    localCount,
    items,
    doc,
    loading,
    error,
    currentUrl,
    bookmarked,
    disabled,
    onBack,
    onClose,
    onRefresh,
    onSave,
    onOpen,
    onCreate,
    onShowLocal,
    onShowChrome,
}: BrowserBookmarksSheetViewProps) {
    const local = source === 'local';
    const localLink = localBookmarksLink(localCount);
    const create = (value: string) => {
        void onCreate(value).then((ok) => ok && onClose());
    };

    return (
        <div className="bwp-urlview bwp-bmview">
            <div className="bwp-shead">
                <button
                    type="button"
                    className="bwp-icon2"
                    onClick={onBack}
                    aria-label="Back to sessions and tabs"
                >
                    ‹
                </button>
                <span className="bwp-kicker">{bookmarkSourceKicker(source)}</span>
                <span className="bwp-gap" />
                <button
                    type="button"
                    className="bwp-icon2"
                    onClick={onRefresh}
                    aria-label="Refresh bookmarks"
                >
                    ↻
                </button>
                <button type="button" className="bwp-x" onClick={onClose} aria-label="Close">
                    ×
                </button>
            </div>
            <BookmarkSourceMeta
                source={source}
                profile={profile}
                count={items.length}
                disabled={disabled}
                onShowChrome={local ? onShowChrome : undefined}
            />
            {local && (
                <BookmarkSaveCard
                    currentUrl={currentUrl}
                    bookmarked={bookmarked}
                    disabled={disabled}
                    onSave={onSave}
                />
            )}
            <BrowserBookmarkBrowser
                doc={doc}
                loading={loading}
                error={error}
                disabled={disabled}
                addressMode
                emptyText={local ? 'No bookmarks yet.' : 'No Chrome bookmarks.'}
                emptyHint={local ? undefined : 'This profile has none.'}
                emptyAction={!local ? { label: localLink, onClick: onShowLocal } : undefined}
                onCreate={create}
                onOpen={onOpen}
            />
        </div>
    );
}
