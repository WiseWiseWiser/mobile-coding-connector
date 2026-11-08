import { useEffect, useRef, useState } from 'react';
import {
    fetchCleanupPlan,
    measureCleanup,
    runCleanup,
    type CleanupPlan,
    type RunCleanupResult,
    type SudoPolicy,
} from '../../../api/diskCleanup';
import { defaultSelection, type CleanupRunRow, type CleanupRowStatus } from './spaceCleanupTone';

const POLL_MS = 1500;

function errText(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
}

export interface CleanupState {
    plan: CleanupPlan | null;
    error: string;
    notice: string;
    pending: boolean;
    /** Ids being sized right now, on demand. */
    sizing: string[];
    selected: string[];
    toggle: (id: string) => void;
    selectSafe: () => void;
    selectNone: () => void;
    open: boolean;
    setOpen: (open: boolean) => void;
    sudo: SudoPolicy;
    setSudo: (policy: SudoPolicy) => void;
    running: boolean;
    rows: CleanupRunRow[];
    result: RunCleanupResult | null;
    reload: () => void;
    run: () => Promise<void>;
    /** Ask the server to size classes that the background pass skips. */
    measure: (ids: string[]) => void;
}

/**
 * Owns the cleanup plan: the server measures in the background, so this keeps
 * polling while `pending` is set and reloads once a run invalidates the sizes.
 */
export function useCleanupPlan(): CleanupState {
    const [plan, setPlan] = useState<CleanupPlan | null>(null);
    const [error, setError] = useState('');
    const [notice, setNotice] = useState('');
    const [selected, setSelected] = useState<string[]>([]);
    const [open, setOpen] = useState(false);
    const [sudo, setSudo] = useState<SudoPolicy>('ask');
    const [running, setRunning] = useState(false);
    const [rows, setRows] = useState<CleanupRunRow[]>([]);
    const [result, setResult] = useState<RunCleanupResult | null>(null);
    const [nonce, setNonce] = useState(0);
    const [sizing, setSizing] = useState<string[]>([]);

    const touched = useRef(false);
    const planRef = useRef<CleanupPlan | null>(null);
    planRef.current = plan;

    useEffect(() => {
        let alive = true;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchCleanupPlan(false);
                if (!alive) return;
                setPlan(next);
                setError('');
                if (!touched.current) setSelected(defaultSelection(next));
                if (next.pending) timer = window.setTimeout(tick, POLL_MS);
            } catch (err) {
                if (alive) setError(errText(err));
            }
        };
        void tick();
        return () => {
            alive = false;
            if (timer) window.clearTimeout(timer);
        };
    }, [nonce]);

    const toggle = (id: string) => {
        touched.current = true;
        setSelected(prev => (prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]));
        // System classes are never sized by the background pass — they can shell
        // out to xcrun, so they are only measured once the user asks for them.
        const target = planRef.current?.classes.find(c => c.id === id);
        if (target && !target.measured && !target.readOnly) measure([id]);
    };

    const selectSafe = () => {
        touched.current = true;
        setSelected(defaultSelection(planRef.current));
    };

    const selectNone = () => {
        touched.current = true;
        setSelected([]);
    };

    const reload = () => {
        touched.current = false;
        setNonce(n => n + 1);
    };

    // Sizing an admin class can block on a password dialog, so this never
    // gates rendering: the row shows "measuring" until the probe answers.
    const measure = (ids: string[]) => {
        if (ids.length === 0) return;
        setSizing(prev => [...prev, ...ids.filter(id => !prev.includes(id))]);
        void measureCleanup(ids)
            .then(() => setNonce(n => n + 1))
            .catch(err => setError(errText(err)))
            .finally(() => setSizing(prev => prev.filter(id => !ids.includes(id))));
    };

    const run = async () => {
        if (running || selected.length === 0) return;
        setRunning(true);
        setRows([]);
        setResult(null);
        setNotice('');
        setError('');
        const upsert = (row: CleanupRunRow) => {
            setRows(prev => {
                const at = prev.findIndex(r => r.id === row.id);
                if (at === -1) return [...prev, row];
                const next = prev.slice();
                next[at] = { ...next[at], ...row };
                return next;
            });
        };
        try {
            const done = await runCleanup(selected, sudo, frame => {
                if (frame.type === 'progress' && frame.id) {
                    upsert({
                        id: String(frame.id),
                        name: String(frame.name ?? frame.id),
                        status: (frame.status ?? 'done') as CleanupRowStatus,
                        detail: String(frame.detail ?? ''),
                    });
                }
                if (frame.type === 'log' && frame.message) {
                    setNotice(String(frame.message));
                }
            });
            setResult(done);
        } catch (err) {
            setError(errText(err));
        } finally {
            setRunning(false);
            reload();
        }
    };

    return {
        plan,
        error,
        notice,
        pending: Boolean(plan?.pending),
        sizing,
        selected,
        toggle,
        selectSafe,
        selectNone,
        open,
        setOpen,
        sudo,
        setSudo,
        running,
        rows,
        result,
        reload,
        run,
        measure,
    };
}
