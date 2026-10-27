import { useEffect, useState } from 'react';

interface BrowserUrlBarProps {
    url: string | undefined;
    onNavigate: (url: string) => void;
}

/**
 * Address bar for the cast window. Edits are local until Go; the server
 * normalizes the URL (a bare host becomes https://…) and reports back through
 * status, so the input follows the status value when it changes.
 */
export function BrowserUrlBar({ url, onNavigate }: BrowserUrlBarProps) {
    const [draft, setDraft] = useState(url ?? '');
    const [editing, setEditing] = useState(false);

    useEffect(() => {
        if (!editing) setDraft(url ?? '');
    }, [url, editing]);

    const submit = (event: React.FormEvent) => {
        event.preventDefault();
        const target = draft.trim();
        if (!target) return;
        setEditing(false);
        onNavigate(target);
    };

    return (
        <form className="bw-url" onSubmit={submit}>
            <input
                className="bw-url-input"
                value={draft}
                onChange={(event) => {
                    setEditing(true);
                    setDraft(event.target.value);
                }}
                onBlur={() => setEditing(false)}
                placeholder="example.com"
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                inputMode="url"
            />
            <button type="submit" className="bw-btn">
                Go
            </button>
        </form>
    );
}
