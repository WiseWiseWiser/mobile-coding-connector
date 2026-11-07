const API_BASE = '';

export interface DiskVolume {
    mount: string;
    device: string;
    fs_type: string;
    total_bytes: number;
    used_bytes: number;
    free_bytes: number;
    used_percent: number;
    read_only: boolean;
    primary: boolean;
}

export interface DiskMergedMount {
    mount: string;
    device: string;
    fs_type: string;
    reason: string;
}

export interface DiskIO {
    device: string;
    read_bps: number;
    write_bps: number;
    read_iops: number;
    write_iops: number;
    interval_sec: number;
    sampled_at: string;
}

export interface DiskSnapshot {
    sampled_at: string;
    primary_mount: string;
    volumes: DiskVolume[];
    merged?: DiskMergedMount[];
    io?: DiskIO;
    warning?: string;
}

export async function fetchDisk(): Promise<DiskSnapshot> {
    const resp = await fetch(`${API_BASE}/api/server/disk`);
    if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`);
    }
    const data = await resp.json();
    return {
        sampled_at: '',
        primary_mount: '',
        ...data,
        volumes: Array.isArray(data?.volumes) ? data.volumes : [],
        merged: Array.isArray(data?.merged) ? data.merged : [],
        io: data?.io ?? undefined,
        warning: data?.warning || undefined,
    };
}
