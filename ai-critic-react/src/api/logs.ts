const API_BASE = '';

export interface LogFile {
    name: string;
    path: string;
}

export async function fetchLogFiles(): Promise<LogFile[]> {
    const res = await fetch(`${API_BASE}/api/logs/files`);
    if (!res.ok) throw new Error(`fetch log files failed: ${res.status}`);
    return res.json();
}

export async function addLogFile(name: string, path: string): Promise<void> {
    const res = await fetch(`${API_BASE}/api/logs/files`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, path }),
    });
    if (!res.ok) throw new Error(`add log file failed: ${res.status}`);
}

export async function removeLogFile(name: string): Promise<void> {
    const res = await fetch(`${API_BASE}/api/logs/files?name=${encodeURIComponent(name)}`, {
        method: 'DELETE',
    });
    if (!res.ok) throw new Error(`remove log file failed: ${res.status}`);
}

export interface StreamLogFileParams {
    file?: string;
    path?: string;
    lines?: number;
}

export function streamLogFile(params: StreamLogFileParams = {}): Promise<Response> {
    const url = new URL(`${API_BASE}/api/logs/stream`);
    if (params.file) {
        url.searchParams.set('file', params.file);
    }
    if (params.path) {
        url.searchParams.set('path', params.path);
    }
    if (params.lines) {
        url.searchParams.set('lines', String(params.lines));
    } else {
        url.searchParams.set('lines', '1000');
    }

    return fetch(url.toString(), {
        headers: { 'Accept': 'text/event-stream' },
    });
}
