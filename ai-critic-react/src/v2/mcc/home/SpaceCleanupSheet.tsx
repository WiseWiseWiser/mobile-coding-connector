import { formatBytes } from '../../../api/processes';
import type { CleanupPlan, RunCleanupResult, SudoPolicy } from '../../../api/diskCleanup';
import SpaceCleanupRow from './SpaceCleanupRow';
import SpaceCleanupRunList from './SpaceCleanupRunList';
import { classSize, sumSelected, type CleanupRunRow } from './spaceCleanupTone';

interface Props {
    plan: CleanupPlan;
    selected: string[];
    sudo: SudoPolicy;
    onSudo: (policy: SudoPolicy) => void;
    running: boolean;
    rows: CleanupRunRow[];
    result: RunCleanupResult | null;
    notice: string;
    error: string;
    pending: boolean;
    sizing: string[];
    onClose: () => void;
    onRun: () => void;
}

/** Dry-run preview, then live progress, for the classes the user picked. */
export default function SpaceCleanupSheet(props: Props) {
    const { plan, selected, sudo, onSudo, running, rows, result, notice, error, pending, sizing, onClose, onRun } =
        props;
    const ids = new Set(selected);
    const picked = plan.classes.filter(c => ids.has(c.id));
    const untouched = plan.classes.filter(c => !ids.has(c.id) && c.readOnly);
    const total = sumSelected(plan, selected);
    const started = running || rows.length > 0 || result !== null;
    const canRun = !running && selected.length > 0;

    return (
        <div className="mcc-sheet-scrim" onClick={running ? undefined : onClose}>
            <div className="mcc-clean-sheet" onClick={event => event.stopPropagation()}>
                <div className="mcc-sheet-head">
                    <div>
                        <h3>{running ? 'Clearing…' : result ? 'Cleared' : 'Clear caches'}</h3>
                        <div className="mcc-sheet-meta">
                            {running
                                ? 'streaming each class as it finishes'
                                : 'dry-run preview · the exact command each class runs'}
                        </div>
                    </div>
                    <button type="button" className="mcc-sheet-x" onClick={onClose} disabled={running}>
                        ✕
                    </button>
                </div>

                <div className="mcc-sheet-scroll">
                    {error && <div className="mcc-cl-alert">{error}</div>}

                    {!started && (
                        <>
                            <div className="mcc-cl-warn ok">
                                Nothing deleted yet. Every line below is the command this class runs — derived
                                from the step's op list, not typed by hand.
                            </div>
                            {pending && (
                                <div className="mcc-cl-pending">
                                    some sizes are still being measured; the run uses what is measured now
                                </div>
                            )}
                            <div className="mcc-cl-group">
                                {picked.map(item => (
                                    <SpaceCleanupRow
                                        key={item.id}
                                        item={item}
                                        checked
                                        preview
                                        measuring={pending || sizing.includes(item.id)}
                                        onToggle={() => undefined}
                                    />
                                ))}
                                {picked.length === 0 && (
                                    <div className="mcc-cl-empty">nothing selected — pick a class first</div>
                                )}
                            </div>
                            {untouched.length > 0 && (
                                <div className="mcc-cl-note">
                                    Also listed, never touched:{' '}
                                    {untouched
                                        .map(c => `${c.label} (${classSize(c.sizeBytes)})`)
                                        .join(' · ')}
                                </div>
                            )}
                            <div className="mcc-cl-note">
                                <b>rm -rf</b> = in-process delete, same effect, no shell ·{' '}
                                <b>official</b> = the tool's own command, verbatim · <b>sudo</b> = needs root.
                                A missing tool skips that one class and reports it; the rest still run.
                            </div>
                        </>
                    )}

                    {started && (
                        <>
                            <SpaceCleanupRunList rows={rows} />
                            {notice && <div className="mcc-cl-log">{notice}</div>}
                            {result && (
                                <div className="mcc-cl-warn ok">
                                    Freed {result.freed || formatBytes(result.freedBytes)}
                                    {result.skipped.length > 0
                                        ? ` · ${result.skipped.length} skipped`
                                        : ' · nothing skipped'}
                                </div>
                            )}
                        </>
                    )}
                </div>

                <div className="mcc-sheet-foot">
                    <div className="mcc-sheet-line">
                        <span>
                            {selected.length} {selected.length === 1 ? 'class' : 'classes'} selected
                        </span>
                        <b>{formatBytes(total)}</b>
                    </div>
                    {!running && !result && (
                        <div className="mcc-sheet-sudo">
                            <span>admin classes</span>
                            <button
                                type="button"
                                className={`mcc-sheet-opt${sudo === 'ask' ? ' on' : ''}`}
                                onClick={() => onSudo('ask')}
                            >
                                ask for password
                            </button>
                            <button
                                type="button"
                                className={`mcc-sheet-opt${sudo === 'never' ? ' on' : ''}`}
                                onClick={() => onSudo('never')}
                            >
                                skip them
                            </button>
                        </div>
                    )}
                    {result ? (
                        <button type="button" className="mcc-sheet-run" onClick={onClose}>
                            Done
                        </button>
                    ) : (
                        <button type="button" className="mcc-sheet-run" onClick={onRun} disabled={!canRun}>
                            {running ? 'Clearing…' : `Clear ${formatBytes(total)}`}
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
}
