const API_BASE = '';

export interface MemoryStatus {
    total: number;
    used: number;
    free: number;
    used_percent: number;
}

export interface DiskStatus {
    filesystem: string;
    size: number;
    used: number;
    available: number;
    use_percent: number;
    mount_point: string;
}

export interface CPUStatus {
    num_cpu: number;
    used_percent: number;
}

export interface OSInfo {
    os: string;
    arch: string;
    kernel: string;
    version: string;
}

export interface ProcessStatus {
    pid: number;
    name: string;
    cpu: string;
    mem: string;
    command: string;
}

export interface ServerStatus {
    memory: MemoryStatus;
    disk: DiskStatus[];
    cpu: CPUStatus;
    os_info: OSInfo;
    top_cpu: ProcessStatus[];
    top_mem: ProcessStatus[];
}

export async function getServerStatus(): Promise<ServerStatus> {
    const res = await fetch(`${API_BASE}/api/server/status`);
    if (!res.ok) throw new Error(`server status failed: ${res.status}`);
    const data = await res.json();
    // Go encodes nil slices as JSON null — normalize array fields for safe render.
    return {
        ...data,
        disk: Array.isArray(data?.disk) ? data.disk : [],
        top_cpu: Array.isArray(data?.top_cpu) ? data.top_cpu : [],
        top_mem: Array.isArray(data?.top_mem) ? data.top_mem : [],
        memory: data?.memory ?? { total: 0, used: 0, free: 0, used_percent: 0 },
        cpu: data?.cpu ?? { num_cpu: 0, used_percent: 0 },
        os_info: data?.os_info ?? { os: 'unknown', arch: 'unknown', kernel: 'unknown', version: '' },
    };
}
