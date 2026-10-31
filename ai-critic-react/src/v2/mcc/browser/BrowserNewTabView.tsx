import { BrowserBookmarkBrowser } from './BrowserBookmarkBrowser';
import { localBookmarksLink } from './browserBookmarks';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';

interface BrowserNewTabViewProps {
    source: BookmarkSource;
    localCount: number | null;
    doc: BookmarkDocument | null;
    loading: boolean;
    error: string | null;
    disabled: boolean;
    onBack: () => void;
    onClose: () => void;
    onCreate: (value: string) => Promise<boolean>;
    onOpenBookmark: (item: BookmarkItem) => Promise<void>;
    onOpenLocal: () => void;
}

/** Phone new tab: one field filters Chrome and opens addresses from the same text. */
export function BrowserNewTabView({
    source,
    localCount,
    doc,
    loading,
    error,
    disabled,
    onBack,
    onClose,
    onCreate,
    onOpenBookmark,
    onOpenLocal,
}: BrowserNewTabViewProps) {
    const local = source === 'local';
    const create = (value: string) => {
        void onCreate(value).then((ok) => ok && onClose());
    };
    const open = (item: BookmarkItem) => {
        void onOpenBookmark(item);
    };

    return (
        <div className="bwp-urlview bwp-compose">
            <div className="bwp-shead">
                <button
                    type="button"
                    className="bwp-icon2"
                    onClick={onBack}
                    aria-label="Back to sessions and tabs"
                >
                    ‹
                </button>
                <span className="bwp-kicker">New tab</span>
                <button type="button" className="bwp-x" onClick={onClose} aria-label="Close">
                    ×
                </button>
            </div>
            <BrowserBookmarkBrowser
                doc={doc}
                loading={loading}
                error={error}
                disabled={disabled}
                addressMode
                autoFocus
                emptyText={local ? 'No bookmarks yet.' : 'No Chrome bookmarks.'}
                emptyHint={local ? undefined : 'This profile has none.'}
                emptyAction={
                    local
                        ? undefined
                        : {
                              label: localBookmarksLink(localCount),
                              onClick: onOpenLocal,
                          }
                }
                onCreate={create}
                onOpen={open}
            />
            <div className="bwp-newfoot">
                <button
                    type="button"
                    className="bwp-btn2"
                    onClick={() => void onCreate('').then((ok) => ok && onClose())}
                    disabled={disabled}
                >
                    Open a blank tab
                </button>
            </div>
        </div>
    );
}
