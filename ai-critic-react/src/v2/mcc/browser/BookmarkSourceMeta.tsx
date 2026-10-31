import { bookmarkCountLabel, bookmarkSourceNote } from './browserBookmarks';
import type { BookmarkSource } from './browserBookmarks';

interface BookmarkSourceMetaProps {
    source: BookmarkSource;
    profile: string;
    count: number;
    disabled?: boolean;
    onShowChrome?: () => void;
}

/** Profile and ownership line shared by the palette and the phone sheet. */
export function BookmarkSourceMeta({
    source,
    profile,
    count,
    disabled,
    onShowChrome,
}: BookmarkSourceMetaProps) {
    return (
        <div className="bw-bmsrc">
            <span className="bw-bmsrc-note">{bookmarkSourceNote(source, profile)}</span>
            <span className="bw-bmcount">{bookmarkCountLabel(count)}</span>
            {source === 'local' && onShowChrome && (
                <button
                    type="button"
                    className="bw-bmsrc-switch"
                    onClick={onShowChrome}
                    disabled={disabled}
                >
                    Chrome bookmarks →
                </button>
            )}
        </div>
    );
}
