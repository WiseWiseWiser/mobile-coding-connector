import { useEffect, useRef } from 'react';

export interface BrowserCloseWindowDialogProps {
    title: string;
    sessionId?: string;
    sessionless: boolean;
    current: boolean;
    returnFocus: HTMLElement | null;
    tabCount: number;
    busy: boolean;
    error: string;
    onCancel: () => void;
    onConfirm: () => void;
}

/**
 * Destructive confirmation for a whole Chrome window. Cancel owns the initial
 * focus; Escape and a backdrop tap both take the safe path. The caller marks the
 * sheet body inert while this is mounted.
 */
export function BrowserCloseWindowDialog(props: BrowserCloseWindowDialogProps) {
    const { busy, error, onCancel, returnFocus } = props;
    const dialogRef = useRef<HTMLDivElement>(null);
    const cancelRef = useRef<HTMLButtonElement>(null);

    useEffect(() => {
        const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        cancelRef.current?.focus();
        return () => (returnFocus?.isConnected ? returnFocus : previous)?.focus({ preventScroll: true });
    }, [returnFocus]);

    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                event.preventDefault();
                event.stopPropagation();
                if (!busy) onCancel();
                return;
            }
            if (event.key !== 'Tab') return;
            const focusable = dialogRef.current?.querySelectorAll<HTMLElement>('button:not([disabled])');
            if (!focusable || focusable.length === 0) return;
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };
        document.addEventListener('keydown', onKey);
        return () => document.removeEventListener('keydown', onKey);
    }, [busy, onCancel]);

    const tabs = `${props.tabCount} ${props.tabCount === 1 ? 'tab' : 'tabs'}`;
    const title = props.title.trim() || 'this Chrome window';
    return (
        <div
            className="bwp-clayer"
            onClick={(event) => {
                if (event.target === event.currentTarget && !busy) props.onCancel();
            }}
        >
            <div
                ref={dialogRef}
                className="bwp-ccard"
                role="dialog"
                aria-modal="true"
                aria-labelledby="bwp-ctitle"
                aria-describedby="bwp-cdesc"
                aria-busy={busy}
            >
                <div className="bwp-ckicker">
                    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <path d="M8 2.3 14 13H2L8 2.3Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
                        <path d="M8 6.2v3.1M8 11.4h.01" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                    </svg>
                    Close Chrome window
                </div>
                <h3 id="bwp-ctitle">Close this window?</h3>
                <p id="bwp-cdesc">
                    <strong>“{title}”</strong> and its {tabs} will close.
                    {props.sessionless
                        ? ' This window has no agent session; Chrome closes all of its tabs.'
                        : ` The agent session ${props.sessionId} ends.`}
                    {props.current && ' The next live window becomes current.'}
                </p>
                <div className="bwp-cnote">
                    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <circle cx="8" cy="8" r="5.6" stroke="currentColor" strokeWidth="1.2" />
                        <path d="M8 7.2v3.1M8 5.1h.01" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
                    </svg>
                    This cannot be undone. The last Chrome window is never closed.
                </div>
                {error && (
                    <p className="bwp-cerr" role="alert">
                        {error}
                    </p>
                )}
                <div className="bwp-cactions">
                    <button ref={cancelRef} type="button" className="bwp-ccancel" onClick={props.onCancel} disabled={busy}>
                        Cancel
                    </button>
                    <button type="button" className="bwp-cdanger" onClick={props.onConfirm} disabled={busy} aria-busy={busy}>
                        {busy ? 'Closing…' : 'Close window'}
                    </button>
                </div>
            </div>
        </div>
    );
}
