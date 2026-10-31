import { useState } from 'react';
import { addressLabel, isHttpURL } from './browserAddress';

interface BookmarkSaveCardProps {
    currentUrl?: string;
    bookmarked: boolean;
    disabled?: boolean;
    onSave: () => Promise<boolean>;
}

/** "Bookmark this page" row, shared by the desktop palette and phone sheet. */
export function BookmarkSaveCard({ currentUrl, bookmarked, disabled, onSave }: BookmarkSaveCardProps) {
    const [saving, setSaving] = useState(false);
    const canSave = isHttpURL(currentUrl);

    const save = async () => {
        if (!canSave || bookmarked || saving || disabled) return;
        setSaving(true);
        try {
            await onSave();
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className={`bw-bmsave${bookmarked ? ' is-saved' : ''}`}>
            <div className="bw-bmsave-text">
                <strong>{bookmarked ? 'Saved in bookmarks' : 'Bookmark this page'}</strong>
                <span>{canSave ? addressLabel(currentUrl) : 'Open an http(s) page first.'}</span>
            </div>
            <button
                type="button"
                className="bw-bmsave-btn"
                onClick={() => void save()}
                disabled={!canSave || bookmarked || saving || disabled}
            >
                {bookmarked ? 'Saved' : saving ? 'Saving…' : 'Save'}
            </button>
        </div>
    );
}
