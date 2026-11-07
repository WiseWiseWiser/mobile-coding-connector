import { useState } from 'react';
import type { SweepWindowsResponse } from './browserWire';
import type { SweepResult } from './useBrowserSessions';
import { appliedCounts, sweepSummary } from './sweep';

/** What the dialog and the button need to know about one sweep in progress. */
export interface SweepFlow {
    /** The dry-run request is in flight; the button shows that. */
    planning: boolean;
    /** The real request is in flight; the dialog shows that. */
    applying: boolean;
    /** The plan the dialog renders, or null when no dialog is open. */
    plan: SweepWindowsResponse | null;
    /** Where focus goes when the dialog closes. */
    trigger: HTMLButtonElement | null;
    error: string;
    open: boolean;
    start: (trigger: HTMLButtonElement) => void;
    cancel: () => void;
    confirm: () => void;
}

/**
 * The sweep's two calls and its one dialog, shared by the phone sheet and the
 * desktop rail so both surfaces count, confirm and report the same way.
 *
 * Planning first is the point: the dialog's numbers are the server's own plan,
 * and the confirm call recomputes it, so a stale dialog can never overreach.
 */
export function useSweepFlow(opts: {
    /** Identifies the surface the dialog belongs to; a change closes it. */
    scope: string;
    /** Another action owns the browser right now. */
    blocked: boolean;
    onSweep: (dryRun: boolean) => Promise<SweepResult>;
    onToast: (message: string, warn?: boolean) => void;
    /** The sweep landed: the caller closes or refreshes whatever hosted it. */
    onApplied?: () => void;
}): SweepFlow {
    const [selection, setSelection] = useState<{
        scope: string;
        trigger: HTMLButtonElement;
        plan: SweepWindowsResponse;
    } | null>(null);
    const [planning, setPlanning] = useState(false);
    const [applying, setApplying] = useState(false);
    const [error, setError] = useState('');

    const start = (trigger: HTMLButtonElement) => {
        if (planning || applying || opts.blocked) return;
        setPlanning(true);
        setError('');
        void opts.onSweep(true).then((result) => {
            setPlanning(false);
            if (!result.ok || !result.plan) {
                // A failed plan has no card to live in yet, so it becomes a toast
                // and nothing opens.
                opts.onToast(result.error ?? 'cannot plan the sweep', true);
                return;
            }
            setSelection({ scope: opts.scope, trigger, plan: result.plan });
        });
    };

    const confirm = () => {
        if (applying || planning || !selection) return;
        setApplying(true);
        setError('');
        void opts.onSweep(false).then((result) => {
            setApplying(false);
            const counts = appliedCounts(result.plan);
            if (!result.ok) {
                // Abort-and-report: name what did close, keep the failure in the
                // dialog so a retry does not mean hunting for a toast.
                if (counts.windows || counts.tabs) opts.onToast(sweepSummary(counts.windows, counts.tabs));
                setError(result.error ?? 'cannot close the other windows');
                return;
            }
            setSelection(null);
            opts.onToast(sweepSummary(counts.windows, counts.tabs));
            opts.onApplied?.();
        });
    };

    const cancel = () => {
        if (applying) return;
        setSelection(null);
        setError('');
    };

    const open = Boolean(selection && selection.scope === opts.scope);
    return {
        planning,
        applying,
        plan: open ? selection?.plan ?? null : null,
        trigger: selection?.trigger ?? null,
        error,
        open,
        start,
        cancel,
        confirm,
    };
}
