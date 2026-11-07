import { useEffect, useRef } from 'react';
import { ledgerFor, sweepAction, sweepBody } from './sweep';
import type { SweepWindowsResponse } from './browserWire';

export interface BrowserSweepDialogProps {
    /** The dry-run plan: every number in this dialog comes from it. */
    plan: SweepWindowsResponse;
    returnFocus: HTMLElement | null;
    busy: boolean;
    error: string;
    onCancel: () => void;
    onConfirm: () => void;
}

/**
 * Destructive confirmation for the whole-browser sweep. Cancel owns the initial
 * focus; Escape and a backdrop tap both take the safe path. The caller marks the
 * sheet body inert while this is mounted.
 */
export function BrowserSweepDialog(props: BrowserSweepDialogProps) {
    const { plan, busy, error, onCancel, returnFocus } = props;
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

    const ledger = ledgerFor(plan);
    const body = sweepBody(plan);
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
                role="alertdialog"
                aria-modal="true"
                aria-labelledby="bwp-stitle"
                aria-describedby="bwp-sdesc"
                aria-busy={busy}
            >
                <div className="bwp-ckicker">
                    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <path d="M8 2.3 14 13H2L8 2.3Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
                        <path d="M8 6.2v3.1M8 11.4h.01" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                    </svg>
                    Close windows
                </div>
                <h3 id="bwp-stitle">Close all other windows?</h3>
                <p id="bwp-sdesc">
                    <strong>{body.closing}</strong> {body.keeping}
                </p>
                <div className="bwp-ledger">
                    <div className="bwp-lg">
                        <span>Windows</span>
                        <b>{ledger.windows}</b>
                    </div>
                    <div className="bwp-lg">
                        <span>Tabs</span>
                        <em>{ledger.tabsDelta}</em>
                        <b>{ledger.tabs}</b>
                    </div>
                    <div className="bwp-lg">
                        <span>Kept</span>
                        <b>{ledger.kept}</b>
                    </div>
                </div>
                <div className="bwp-cnote">
                    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <path d="M8 2.4 14.2 13.1H1.8L8 2.4Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
                        <path d="M8 6.3v3.1" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
                        <circle cx="8" cy="11.3" r=".85" fill="currentColor" />
                    </svg>
                    Chrome&apos;s last window is never closed. Closed tabs cannot be reopened.
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
                        {busy ? 'Closing…' : sweepAction(plan)}
                    </button>
                </div>
            </div>
        </div>
    );
}
