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
    return res.json();
}
