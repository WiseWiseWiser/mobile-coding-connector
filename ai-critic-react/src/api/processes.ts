const API_BASE = '';

export interface ProcessCPU {
    num_cpu: number;
    used_percent: number;
}

export interface ProcessLoad {
    m1: number;
    m5: number;
    m15: number;
}

export interface ProcessMemory {
    total: number;
    used: number;
    free: number;
    used_percent: number;
    compressor_bytes?: number;
    swap_total?: number;
    swap_used?: number;
    swap_free?: number;
    swap_used_percent?: number;
}

export interface HostProcess {
    pid: number;
    ppid: number;
    name: string;
    cmd: string;
    user: string;
    state: string;
    cpu_percent: number;
    rss_bytes: number;
    elapsed_sec: number;
    protected?: string;
    tags?: string[];
}

export interface ProcessSnapshot {
    sampled_at: string;
    interval_ms: number;
    cpu: ProcessCPU;
    load: ProcessLoad;
    memory: ProcessMemory;
    counts: { processes: number };
    self_pid: number;
    processes: HostProcess[];
}

function emptySnapshot(): ProcessSnapshot {
    return {
        sampled_at: '',
        interval_ms: 0,
        cpu: { num_cpu: 0, used_percent: 0 },
        load: { m1: 0, m5: 0, m15: 0 },
        memory: { total: 0, used: 0, free: 0, used_percent: 0 },
        counts: { processes: 0 },
        self_pid: 0,
        processes: [],
    };
}

export async function fetchProcesses(summary = false): Promise<ProcessSnapshot> {
    const url = summary ? `${API_BASE}/api/server/processes?summary=1` : `${API_BASE}/api/server/processes`;
    const resp = await fetch(url);
    if (!resp.ok) {
        throw new Error(await resp.text() || `HTTP ${resp.status}`);
    }
    const data = await resp.json();
    const base = emptySnapshot();
    return {
        ...base,
        ...data,
        cpu: data?.cpu ?? base.cpu,
        load: data?.load ?? base.load,
        memory: data?.memory ?? base.memory,
        counts: data?.counts ?? base.counts,
        processes: Array.isArray(data?.processes) ? data.processes : [],
    };
}

export async function killHostProcess(pid: number, signal: 'TERM' | 'KILL' = 'TERM'): Promise<void> {
    const resp = await fetch(`${API_BASE}/api/server/processes/kill?pid=${pid}&signal=${signal}`, {
        method: 'POST',
    });
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
}

export function formatBytes(bytes: number): string {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let n = bytes;
    let i = 0;
    while (n >= 1024 && i < units.length - 1) {
        n /= 1024;
        i += 1;
    }
    const digits = n >= 10 || i === 0 ? 0 : 1;
    return `${n.toFixed(digits)} ${units[i]}`;
}

export function formatElapsed(sec: number): string {
    if (sec < 60) return `${sec}s`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m`;
    if (sec < 86400) {
        const h = Math.floor(sec / 3600);
        const m = Math.floor((sec % 3600) / 60);
        return m ? `${h}h${m}m` : `${h}h`;
    }
    const d = Math.floor(sec / 86400);
    const h = Math.floor((sec % 86400) / 3600);
    return h ? `${d}d${h}h` : `${d}d`;
}
