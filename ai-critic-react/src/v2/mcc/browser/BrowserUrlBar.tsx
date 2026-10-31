import { useState } from 'react';
import { checkAddress } from './browserAddress';

interface BrowserUrlBarProps {
    url: string | undefined;
    disabled?: boolean;
    /** The current page is already in the app-local bookmark tree. */
    bookmarked?: boolean;
    onNavigate: (url: string) => void;
    /** Opens the new-tab composer, prefilled with the dirty draft when valid. */
    onNew: (draft: string) => void;
    onBookmarks: () => void;
}

/**
 * Address bar for the cast window.
 *
 * `Go` navigates the current tab; `New` and the strip's `+` open the composer.
 * The draft follows status only when the field is not focused, so clicking Go
 * (which blurs first) can never replace the typed URL with the old one.
 */
export function BrowserUrlBar({ url, disabled, bookmarked, onNavigate, onNew, onBookmarks }: BrowserUrlBarProps) {
    const [draft, setDraft] = useState(url ?? '');
    const [lastUrl, setLastUrl] = useState(url);
    const [focused, setFocused] = useState(false);
    const [error, setError] = useState('');

    if (url !== lastUrl) {
        setLastUrl(url);
        if (!focused) setDraft(url ?? '');
    }

    const submit = (event: React.FormEvent) => {
        event.preventDefault();
        if (disabled) return;
        const check = checkAddress(draft);
        if (!check.ok) {
            setError(check.error);
            return;
        }
        setError('');
        onNavigate(check.target);
    };

    const newTab = () => {
        if (disabled) return;
        if (!draft.trim()) {
            setError('');
            onNew('');
            return;
        }
        const check = checkAddress(draft);
        if (!check.ok) {
            setError(check.error);
            return;
        }
        setError('');
        const dirty = check.target !== (url ?? '').trim();
        onNew(dirty ? check.target : '');
    };

    return (
        <form className={`bw-url${error ? ' has-error' : ''}`} onSubmit={submit}>
            <button
                type="button"
                className={`bw-urlmark${bookmarked ? ' is-on' : ''}`}
                onClick={onBookmarks}
                disabled={disabled}
                title={bookmarked ? 'Bookmarks — this page is saved' : 'Bookmarks'}
                aria-label="Open bookmarks"
            >
                {bookmarked ? '★' : '☆'}
            </button>
            <input
                className="bw-url-input"
                value={draft}
                onChange={(event) => {
                    setDraft(event.target.value);
                    if (error) setError('');
                }}
                onFocus={() => setFocused(true)}
                onBlur={() => setFocused(false)}
                onKeyDown={(event) => {
                    if (event.key !== 'Escape') return;
                    setDraft(url ?? '');
                    setError('');
                    event.currentTarget.blur();
                }}
                placeholder={disabled ? 'Attach a window to open a page' : 'example.com'}
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                inputMode="url"
                disabled={disabled}
                aria-invalid={!!error}
            />
            <button type="submit" className="bw-btn" disabled={disabled}>
                Go
            </button>
            <button type="button" className="bw-btn bw-url-new" onClick={newTab} disabled={disabled}>
                New
            </button>
            {error && (
                <span className="bw-urlerr" role="alert">
                    {error}
                </span>
            )}
        </form>
    );
}
