import type { SweepWindowsResponse } from './browserWire';
import type { WindowRow } from './windowClose';

/**
 * Sweeping keeps one Chrome window and closes every other drivable window, so
 * the whole feature is presentational glue over one server call: this module
 * holds the wording and the arithmetic so the rail and the dialog cannot
 * disagree with each other or with the server's plan.
 */

export interface SweepAvailability {
    supported: boolean;
    readOnly: boolean;
    busy: boolean;
    /** Live windows in the rail; devtools windows are not rows. */
    windowCount: number;
}

/**
 * How many Chrome windows the rail is showing.
 *
 * The rail is one row per session, and two sessions can share one window (the
 * window's own agent page plus the session that opened it). Closing windows
 * counts windows, so the button preview and its gate have to dedupe by window
 * id; a row that never learned its window id still counts as its own window.
 */
export function sweepWindowCount(rows: Array<Pick<WindowRow, 'key' | 'attached' | 'windowId'>>): number {
    const windows = new Set<string | number>();
    for (const row of rows) {
        if (row.attached === false) continue;
        windows.add(row.windowId ?? row.key);
    }
    return windows.size;
}

/**
 * Why the sweep is unavailable, or '' when it can be offered. The server
 * enforces the same rules; this only explains a disabled control, and it stays
 * quiet while a plan is being fetched so the button can show its own busy state.
 */
export function sweepBlocker({ supported, readOnly, busy, windowCount }: SweepAvailability): string {
    if (!supported) return 'Reload browser-agent 1.0.22+ to close windows';
    if (readOnly) return 'Read-only: this daemon cannot control windows';
    if (busy) return 'Wait for the current action';
    if (windowCount <= 1) return 'Only one window is open';
    return '';
}

export interface SweepLedger {
    /** "3 → 1" */
    windows: string;
    /** "−5 +1": tabs that leave, then the one blank page that opens. */
    tabsDelta: string;
    /** "6 → 2" */
    tabs: string;
    /** "sess-c0zyta · agent page" */
    kept: string;
}

/**
 * The dialog's numbers, derived from the server's plan rather than from the
 * rail: the plan is one window listing taken when the dialog opened, so the copy
 * and the button label always describe the same sweep.
 */
export function ledgerFor(plan: SweepWindowsResponse): SweepLedger {
    const windowsBefore = plan.close_windows + 1;
    const tabsBefore = plan.close_tabs + plan.keep_tabs;
    const tabsAfter = plan.keep_tabs + 1;
    return {
        windows: `${windowsBefore} → 1`,
        tabsDelta: `−${plan.close_tabs}  +1`,
        tabs: `${tabsBefore} → ${tabsAfter}`,
        kept: plan.keep_tabs > 1 ? `${plan.keep_tabs} agent pages` : `${plan.kept_session_id || 'Registry page'} · agent page`,
    };
}

/** The danger button only names windows: tabs are the dialog's job to explain. */
export function sweepAction(plan: SweepWindowsResponse): string {
    return plan.close_windows === 1 ? 'Close 1 window' : `Close ${plan.close_windows} windows`;
}

/** The dialog body: what closes, what stays, and what opens last. */
export function sweepBody(plan: SweepWindowsResponse): { closing: string; keeping: string } {
    const windows = plan.close_windows === 1 ? '1 Chrome window' : `${plan.close_windows} Chrome windows`;
    const tabs = plan.close_tabs === 1 ? '1 tab' : `${plan.close_tabs} tabs`;
    return {
        closing: `${windows} and ${tabs} close.`,
        keeping: 'This window keeps its agent page, then opens one blank page.',
    };
}

/** The toast after a sweep; a partial sweep passes the numbers it managed. */
export function sweepSummary(closedWindows: number, closedTabs: number): string {
    const windows = closedWindows === 1 ? '1 window' : `${closedWindows} windows`;
    const tabs = closedTabs === 1 ? '1 tab' : `${closedTabs} tabs`;
    return `Closed ${windows} and ${tabs}.`;
}

/** What actually closed, for both the success toast and a partial failure. */
export function appliedCounts(result: SweepWindowsResponse | null | undefined): { windows: number; tabs: number } {
    return {
        windows: result?.closed_windows?.length ?? 0,
        tabs: result?.closed_tabs ?? 0,
    };
}
