import { useRevealCurrent } from './useRevealCurrent';
import { sweepBlocker, sweepWindowCount } from './sweep';
import { closeTargetFor, closeWindowBlocker } from './windowClose';
import type { WindowRow } from './windowClose';

export interface BrowserWindowsRailProps {
    rows: WindowRow[];
    /** Expanded (cards) vs collapsed (chips). Pure presentation; state lives in the sheet. */
    open: boolean;
    /** The window whose tabs the right pane lists. */
    selectedKey: string;
    busy: boolean;
    readOnly: boolean;
    windowCloseSupported: boolean;
    newBusy: boolean;
    newWindowError: string | null;
    onToggle: () => void;
    /** Preview the window's tabs on the right; never switches by itself. */
    onPreview: (row: WindowRow) => void;
    /** A window the person has not opened since it appeared. */
    windowNew: (key: string) => boolean;
    onNew: () => void;
    onClose: (row: WindowRow, trigger: HTMLButtonElement) => void;
    /** Fetching the plan: the button says so instead of opening a stale dialog. */
    sweepPlanning: boolean;
    /** The element comes along so the dialog can hand focus back to it. */
    onSweep: (trigger: HTMLButtonElement) => void;
}

/**
 * The rail handle's glyph: one frame with a left divider — "toggle the left
 * pane". Deliberately unlike the stacked-windows glyph the window rows use,
 * so the handle can never be mistaken for another window chip. The left pane
 * cell fills while the rail is open, folding the on/off state into the icon.
 */
function RailGlyph() {
    return (
        <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <rect className="rail-pane" x="2.6" y="3.2" width="10.8" height="9.6" rx="1.8" />
            <rect x="2.6" y="3.2" width="10.8" height="9.6" rx="1.8" stroke="currentColor" strokeWidth="1.25" />
            <path d="M6.4 3.4v9.2" stroke="currentColor" strokeWidth="1.25" />
        </svg>
    );
}

/**
 * A window, drawn as two stacked frames so it cannot be confused with the
 * handle's single divided frame. The front frame carries the row's background
 * (--glyph-bg) to occlude the back one, like a favicon tile.
 */
function WindowStackGlyph() {
    return (
        <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <rect x="5.4" y="2.6" width="8" height="6.8" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
            <rect
                x="2.6"
                y="6.6"
                width="8"
                height="6.8"
                rx="1.5"
                stroke="currentColor"
                strokeWidth="1.2"
                fill="var(--glyph-bg, #1a2230)"
            />
        </svg>
    );
}

/**
 * The sweep glyph: a bare red cross. The other closes in this pane (a window,
 * a tab) are lone gray glyphs too, so the shape reads as "close" and only the
 * color says "this one closes many".
 */
function SweepGlyph() {
    return (
        <svg viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M4.6 4.6 11.4 11.4" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
            <path d="M11.4 4.6 4.6 11.4" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
        </svg>
    );
}

/**
 * The left pane: every window, collapsed to chips or expanded to cards.
 *
 * Tapping a window only previews its tabs on the right — switching happens
 * when a tab is tapped — so the rail carries no "switch" button. New-window
 * and close stay in the expanded cards where mis-taps cannot happen.
 */
export function BrowserWindowsRail(props: BrowserWindowsRailProps) {
    const { rows, open, selectedKey, busy, readOnly, windowCloseSupported } = props;
    const listRef = useRevealCurrent<HTMLDivElement>(open, '.bwp-wrowgroup');
    const liveCount = rows.filter((row) => row.attached !== false).length;
    // Windows, not rows: the sweep closes windows, and two sessions can share one.
    const sweepWindows = sweepWindowCount(rows);
    const blocker = closeWindowBlocker({
        supported: windowCloseSupported,
        readOnly,
        busy: busy || props.newBusy,
        windowCount: liveCount,
    });
    // The sweep answers to its own rules: it needs a second window, which is a
    // different question from "can this one window be closed".
    const sweepReason = sweepBlocker({
        supported: windowCloseSupported,
        readOnly,
        busy: (busy && !props.sweepPlanning) || props.newBusy,
        windowCount: sweepWindows,
    });
    return (
        <aside className={`bwp-rail${open ? ' is-open' : ''}`}>
            <div className="bwp-railhead">
                <button
                    type="button"
                    className="bwp-railtoggle"
                    onClick={props.onToggle}
                    aria-expanded={open}
                    aria-controls="bwp-railbody"
                    aria-label={open ? 'Collapse the window list' : 'Show the window list'}
                    title={open ? 'Collapse windows' : 'Show windows'}
                >
                    <RailGlyph />
                    {!open && <span className="bwp-railcount">{liveCount}</span>}
                </button>
                {open && (
                    <>
                        <span className="bwp-kicker">Windows</span>
                        <span className="bwp-gap" />
                        <button
                            type="button"
                            className="bwp-mini"
                            onClick={props.onNew}
                            disabled={busy || props.newBusy || readOnly}
                            aria-busy={props.newBusy}
                            aria-label={props.newBusy ? 'Opening new window' : 'New window'}
                            title="New window"
                        >
                            {props.newBusy ? '…' : '+'}
                        </button>
                    </>
                )}
            </div>
            {open && props.newWindowError && <p className="bwp-wrowerr bwp-railerr" role="alert">{props.newWindowError}</p>}
            <div id="bwp-railbody" ref={listRef} className="bwp-railbody">
                {open ? (
                    <>
                        {rows.length === 0 && <p className="bwp-empty">No Chrome window yet.</p>}
                        {rows.map((row) => {
                            const label = row.title.trim() || row.sessionId || 'Chrome window';
                            const closeReason = row.attached === false
                                ? 'No connected Chrome window'
                                : blocker || (closeTargetFor(row) ? '' : 'Window cannot be addressed');
                            const kind = row.windowType && row.windowType !== 'normal' ? ` · ${row.windowType}` : '';
                            const meta = row.attached === false
                                ? `${row.sessionId ?? 'session'} · disconnected`
                                : `${row.sessionId ?? 'window'}${kind} · ${row.current ? 'focused' : 'background'}`;
                            return (
                                <div key={row.key} className={`bwp-wrowgroup${row.key === selectedKey ? ' is-sel' : ''}`}>
                                    <div className="bwp-wrow">
                                        <button
                                            type="button"
                                            className="bwp-wrowmain"
                                            onClick={() => props.onPreview(row)}
                                            aria-current={row.key === selectedKey ? 'true' : undefined}
                                            title={`Preview ${label}'s tabs`}
                                        >
                                            <span className="bwp-wmark" aria-hidden="true">
                                                <WindowStackGlyph />
                                            </span>
                                            <span className="bwp-wcopy">
                                                <span className="bwp-wline">
                                                    <span className="bwp-wtitle">{label}</span>
                                                    {props.windowNew(row.key) && (
                                                        <span className="new" title="Window you have not opened yet">
                                                            New
                                                        </span>
                                                    )}
                                                </span>
                                                <span className="bwp-wmeta">{meta}</span>
                                            </span>
                                            <span className="bwp-wtabs">
                                                {row.tabCount} {row.tabCount === 1 ? 'tab' : 'tabs'}
                                            </span>
                                            {row.current && row.attached !== false && (
                                                <span className="bwp-chipdot" aria-hidden="true" />
                                            )}
                                        </button>
                                        <button
                                            type="button"
                                            className="bwp-tclose bwp-wclose"
                                            onClick={(event) => props.onClose(row, event.currentTarget)}
                                            disabled={Boolean(closeReason)}
                                            aria-label={`Close window ${label}${closeReason ? `: ${closeReason}` : ''}`}
                                            title={closeReason || `Close window ${label}`}
                                        >
                                            ×
                                        </button>
                                    </div>
                                </div>
                            );
                        })}
                    </>
                ) : (
                    rows.map((row) => (
                        <button
                            key={row.key}
                            type="button"
                            className={`bwp-chip${row.key === selectedKey ? ' is-sel' : ''}`}
                            onClick={() => props.onPreview(row)}
                            aria-label={`Preview ${row.title.trim() || row.sessionId || 'Chrome window'}`}
                            title={row.title.trim() || row.sessionId || 'Chrome window'}
                        >
                            <span className="bwp-chipmark" aria-hidden="true">
                                <WindowStackGlyph />
                            </span>
                            <span className={`bwp-chipn${props.windowNew(row.key) ? ' is-new' : ''}`}>
                                {row.tabCount}
                            </span>
                            {row.current && row.attached !== false && <span className="bwp-chipdot" aria-hidden="true" />}
                        </button>
                    ))
                )}
            </div>
            {/*
              Pinned below the list, in both states: the sweep is the one
              window-level action that has no per-window home, and hiding it
              behind the expander would make it undiscoverable on a phone.
            */}
            <div className="bwp-railfoot">
                <button
                    type="button"
                    className={`bwp-sweep${sweepReason ? ' is-off' : ''}`}
                    onClick={(event) => props.onSweep(event.currentTarget)}
                    disabled={Boolean(sweepReason) || props.sweepPlanning}
                    aria-busy={props.sweepPlanning}
                    aria-label={`Close all other windows${sweepReason ? `: ${sweepReason}` : ''}`}
                    title={sweepReason || 'Close all other windows'}
                >
                    <SweepGlyph />
                    <span className="bwp-sweeplbl">
                        {props.sweepPlanning ? 'Planning…' : 'Close other windows'}
                    </span>
                    {/* The same arithmetic the dialog will show, from live rows. */}
                    {!props.sweepPlanning && sweepWindows > 1 && (
                        <span className="bwp-sweepn">{sweepWindows} → 1</span>
                    )}
                </button>
            </div>
        </aside>
    );
}
