const API_BASE = '';

export type AnalysisKind = 'cache' | 'media' | 'doc' | 'other' | string;

export type AttemptOutcome = 'completed' | 'cancelled' | 'failed' | 'interrupted' | string;

export interface AnalysisCategory {
    id: string;
    label: string;
    kind: AnalysisKind;
    bytes: number;
    reclaimable: number;
    paths?: string[];
}

export interface AnalysisEntry {
    category: string;
    path: string;
    bytes: number;
    reclaimable: boolean;
}

export interface AnalysisReport {
    id: string;
    schema: number;
    scanned_at: string;
    scan_ms: number;
    complete: boolean;
    truncated: boolean;
    used_bytes: number;
    total_bytes: number;
    scanned_bytes: number;
    files: number;
    no_access_dirs: number;
    roots?: string[];
    categories: AnalysisCategory[];
    entries?: AnalysisEntry[];
    entries_dropped?: boolean;
}

export interface AnalysisReportSummary extends Omit<AnalysisReport, 'entries' | 'entries_dropped' | 'roots'> {
    reclaimable: number;
    has_entries: boolean;
    entries_dropped?: boolean;
}

export interface AnalysisJob {
    id: string;
    started_at: string;
    phase: string;
    walked_bytes: number;
    files: number;
    no_access_dirs: number;
    percent: number;
    state: string;
}

export interface AnalysisAttempt {
    at: string;
    outcome: AttemptOutcome;
    job_id?: string;
    error?: string;
}

export interface AnalysisStatus {
    running: boolean;
    job?: AnalysisJob;
    last_completed_at?: string;
    last_attempt?: AnalysisAttempt;
    report_count: number;
    latest?: AnalysisReportSummary;
}

export interface AnalysisReportsResponse {
    reports: AnalysisReportSummary[];
    count: number;
    last_completed_at?: string;
    last_attempt?: AnalysisAttempt;
}

export interface StartResult {
    job: AnalysisJob;
    /** True when the server was already scanning and the page should attach. */
    attached: boolean;
}

function emptyStatus(): AnalysisStatus {
    return { running: false, report_count: 0 };
}

export async function fetchAnalysisStatus(): Promise<AnalysisStatus> {
    const resp = await fetch(`${API_BASE}/api/server/disk/analysis/status`, { cache: 'no-store' });
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    const data = await resp.json();
    return {
        ...emptyStatus(),
        ...data,
        latest: data?.latest ?? undefined,
        job: data?.job ?? undefined,
        last_completed_at: data?.last_completed_at || undefined,
        last_attempt: data?.last_attempt ?? undefined,
    };
}

export async function startAnalysis(): Promise<StartResult> {
    const resp = await fetch(`${API_BASE}/api/server/disk/analysis/start`, { method: 'POST' });
    const data = await resp.json().catch(() => null);
    if (resp.status === 409 && data?.job) {
        return { job: data.job as AnalysisJob, attached: true };
    }
    if (!resp.ok || !data?.job) {
        throw new Error(data?.error || `HTTP ${resp.status}`);
    }
    return { job: data.job as AnalysisJob, attached: false };
}

export async function cancelAnalysis(): Promise<boolean> {
    const resp = await fetch(`${API_BASE}/api/server/disk/analysis/cancel`, { method: 'POST' });
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    const data = await resp.json().catch(() => null);
    return Boolean(data?.cancelled);
}

export async function fetchAnalysisReports(limit = 50): Promise<AnalysisReportsResponse> {
    const resp = await fetch(`${API_BASE}/api/server/disk/analysis/reports?limit=${limit}`, {
        cache: 'no-store',
    });
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    const data = await resp.json();
    return {
        reports: Array.isArray(data?.reports) ? data.reports : [],
        count: Number(data?.count) || 0,
        last_completed_at: data?.last_completed_at || undefined,
        last_attempt: data?.last_attempt ?? undefined,
    };
}

export async function fetchAnalysisReport(id: string): Promise<AnalysisReport> {
    const resp = await fetch(`${API_BASE}/api/server/disk/analysis/reports/${encodeURIComponent(id)}`, {
        cache: 'no-store',
    });
    if (resp.status === 404) {
        throw new Error('that report is no longer on disk');
    }
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    const data = await resp.json();
    return {
        ...data,
        categories: Array.isArray(data?.categories) ? data.categories : [],
        entries: Array.isArray(data?.entries) ? data.entries : [],
    };
}
