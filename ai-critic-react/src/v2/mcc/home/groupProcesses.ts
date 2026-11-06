import type { HostProcess } from '../../../api/processes';

export type ProcessFilter = 'all' | 'mine' | 'stale';
export type ProcessSort = 'cpu' | 'mem' | 'name' | 'pid';

export interface ProcessGroup {
    name: string;
    cpu: number;
    rss: number;
    procs: HostProcess[];
}

export function filterProcesses(list: HostProcess[], query: string, filter: ProcessFilter): HostProcess[] {
    const q = query.trim().toLowerCase();
    return list.filter(p => {
        if (filter === 'mine' && !(p.tags ?? []).includes('mine')) return false;
        if (filter === 'stale' && !(p.tags ?? []).some(t => t === 'stale' || t === 'tun')) return false;
        if (!q) return true;
        return (
            p.name.toLowerCase().includes(q) ||
            p.cmd.toLowerCase().includes(q) ||
            String(p.pid).includes(q) ||
            p.user.toLowerCase().includes(q)
        );
    });
}

export function groupProcesses(list: HostProcess[], sort: ProcessSort): ProcessGroup[] {
    const map = new Map<string, HostProcess[]>();
    for (const p of list) {
        const key = p.name || '?';
        const arr = map.get(key);
        if (arr) arr.push(p);
        else map.set(key, [p]);
    }
    const groups: ProcessGroup[] = [];
    for (const [name, procs] of map) {
        procs.sort(compareProcess(sort));
        groups.push({
            name,
            cpu: procs.reduce((s, p) => s + (p.cpu_percent || 0), 0),
            rss: procs.reduce((s, p) => s + (p.rss_bytes || 0), 0),
            procs,
        });
    }
    groups.sort((a, b) => {
        if (sort === 'name') return a.name.localeCompare(b.name);
        if (sort === 'pid') return (a.procs[0]?.pid ?? 0) - (b.procs[0]?.pid ?? 0);
        if (sort === 'mem') return b.rss - a.rss;
        return b.cpu - a.cpu;
    });
    return groups;
}

function compareProcess(sort: ProcessSort) {
    return (a: HostProcess, b: HostProcess) => {
        if (sort === 'name') return a.name.localeCompare(b.name) || a.pid - b.pid;
        if (sort === 'pid') return a.pid - b.pid;
        if (sort === 'mem') return b.rss_bytes - a.rss_bytes;
        return b.cpu_percent - a.cpu_percent;
    };
}
