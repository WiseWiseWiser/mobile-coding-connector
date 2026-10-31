export interface WindowRow {
    /** Stable React key and lookup id: the session id, or the synthetic window id. */
    key: string;
    sessionId?: string;
    windowId?: number;
    title: string;
    tabCount: number;
    current: boolean;
    /** False for the remembered session when no live Chrome window is known. */
    attached?: boolean;
    sessionless: boolean;
}

export type CloseTarget = { window_id: number } | { session_id: string };

export interface CloseAvailability {
    supported: boolean;
    readOnly: boolean;
    busy: boolean;
    windowCount: number;
}

/**
 * Why whole-window close is unavailable, or '' when it can be offered.
 * The server enforces the same rules; this only explains a disabled button.
 */
export function closeWindowBlocker({ supported, readOnly, busy, windowCount }: CloseAvailability): string {
    if (!supported) return 'Reload browser-agent 1.0.22+ to close windows';
    if (readOnly) return 'Read-only: this daemon cannot control windows';
    if (busy) return 'Wait for the current action';
    if (windowCount <= 1) return 'The last Chrome window cannot be closed';
    return '';
}

/**
 * Sessionless windows are addressed by Chrome window id; live sessions by id.
 * A sessionless row without a window id has no addressable target.
 */
export function closeTargetFor(row: WindowRow): CloseTarget | null {
    if (row.sessionless) return row.windowId ? { window_id: row.windowId } : null;
    return row.sessionId ? { session_id: row.sessionId } : null;
}
