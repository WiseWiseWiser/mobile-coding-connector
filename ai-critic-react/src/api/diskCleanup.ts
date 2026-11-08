const API_BASE = '';

/** How a class is cleared: an in-process `rm -rf`, a tool's own command, or root. */
export type CleanupActionKind = 'rm-rf' | 'official' | 'sudo' | 'none' | string;

export interface CleanupAction {
    kind: CleanupActionKind;
    text: string;
}

/** One clearable class — derived from clear-tmp's step registry on the server. */
export interface CleanupClass {
    id: string;
    group: string;
    label: string;
    sizeBytes?: number;
    measured: boolean;
    selected: boolean;
    needsAdmin: boolean;
    readOnly: boolean;
    actions: CleanupAction[];
    command: string;
    paths?: string[];
    notes?: string[];
    details?: string[];
}

export interface CleanupGroup {
    id: string;
    label: string;
    note: string;
}

export interface CleanupVolume {
    path: string;
    totalBytes: number;
    freeBytes: number;
}

export interface CleanupTotals {
    selectedBytes: number;
    aggressiveBytes: number;
    systemBytes: number;
    inventoryBytes: number;
    unmeasured: number;
}

export interface CleanupPlan {
    scannedAt?: string;
    scanMillis: number;
    /** True while the server is still measuring in the background. */
    pending?: boolean;
    home: string;
    volume?: CleanupVolume;
    groups: CleanupGroup[];
    classes: CleanupClass[];
    totals: CleanupTotals;
}

export type SudoPolicy = 'ask' | 'never';

/** One SSE frame from the run endpoint. */
export interface CleanupFrame {
    type: string;
    layer?: string;
    name?: string;
    status?: string;
    detail?: string;
    hint?: string;
    message?: string;
    [key: string]: unknown;
}

function emptyTotals(): CleanupTotals {
    return { selectedBytes: 0, aggressiveBytes: 0, systemBytes: 0, inventoryBytes: 0, unmeasured: 0 };
}

function normalizePlan(data: CleanupPlan | null | undefined): CleanupPlan {
    return {
        scannedAt: data?.scannedAt || undefined,
        scanMillis: Number(data?.scanMillis) || 0,
        pending: Boolean(data?.pending),
        home: data?.home ?? '',
        volume: data?.volume,
        groups: Array.isArray(data?.groups) ? data.groups : [],
        classes: Array.isArray(data?.classes)
            ? data.classes.map(c => ({ ...c, notes: c.notes ?? [], details: c.details ?? [] }))
            : [],
        totals: { ...emptyTotals(), ...(data?.totals ?? {}) },
    };
}

export async function fetchCleanupPlan(refresh = false): Promise<CleanupPlan> {
    const query = refresh ? '?refresh=1' : '';
    const resp = await fetch(`${API_BASE}/api/server/disk/cleanup/plan${query}`, { cache: 'no-store' });
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    return normalizePlan(await resp.json());
}

/** Ask the server to measure just these classes; resolves when they are stored. */
export async function measureCleanup(ids: string[]): Promise<boolean> {
    const resp = await fetch(`${API_BASE}/api/server/disk/cleanup/measure`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids }),
    });
    const data = await resp.json().catch(() => null);
    if (!resp.ok) {
        throw new Error(data?.error || (await resp.text()) || `HTTP ${resp.status}`);
    }
    return Boolean(data?.pending);
}

export interface RunCleanupResult {
    freedBytes: number;
    freed: string;
    skipped: unknown[];
}

function frameFrom(raw: string): CleanupFrame | null {
    const trimmed = raw.trim();
    if (!trimmed || trimmed.startsWith(':')) return null;
    const payload = trimmed.startsWith('data:') ? trimmed.slice(5).trim() : trimmed;
    if (!payload || payload === '[DONE]') return null;
    try {
        const parsed = JSON.parse(payload);
        if (!parsed || typeof parsed !== 'object') return null;
        return parsed as CleanupFrame;
    } catch {
        return null;
    }
}

/**
 * Stream a cleanup run. The server sends `section`/`progress`/`log`/`done`
 * frames; `onFrame` sees each one as it lands, and the resolved value is the
 * `done` payload when the run finished.
 */
export async function runCleanup(
    ids: string[],
    sudo: SudoPolicy,
    onFrame: (frame: CleanupFrame) => void
): Promise<RunCleanupResult> {
    const resp = await fetch(`${API_BASE}/api/server/disk/cleanup/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
        body: JSON.stringify({ ids, sudo }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        let message = text;
        try {
            message = JSON.parse(text)?.error || text;
        } catch {
            // keep the raw body
        }
        throw new Error(message || `HTTP ${resp.status}`);
    }
    if (!resp.body) {
        throw new Error('the server did not stream a response');
    }

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let done: RunCleanupResult = { freedBytes: 0, freed: '', skipped: [] };

    const consume = (frame: CleanupFrame | null) => {
        if (!frame) return;
        onFrame(frame);
        if (frame.type === 'done') {
            done = {
                freedBytes: Number(frame.freedBytes) || 0,
                freed: String(frame.freed ?? ''),
                skipped: Array.isArray(frame.skipped) ? frame.skipped : [],
            };
        }
        if (frame.type === 'error') {
            throw new Error(frame.message || 'cleanup failed');
        }
    };

    for (;;) {
        const { value, done: finished } = await reader.read();
        if (finished) break;
        buffer += decoder.decode(value, { stream: true });
        let cut = buffer.indexOf('\n\n');
        while (cut !== -1) {
            const chunk = buffer.slice(0, cut);
            buffer = buffer.slice(cut + 2);
            chunk.split('\n').forEach(line => consume(frameFrom(line)));
            cut = buffer.indexOf('\n\n');
        }
    }
    buffer += decoder.decode();
    if (buffer.trim()) {
        buffer.split('\n').forEach(line => consume(frameFrom(line)));
    }
    return done;
}
