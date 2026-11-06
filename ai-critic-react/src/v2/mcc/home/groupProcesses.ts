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

export function applyLockedOrder(list: HostProcess[], lockedPids: number[] | null | undefined): HostProcess[] {
    if (!lockedPids || lockedPids.length === 0) return list;
    const idx = new Map(lockedPids.map((id, i) => [id, i]));
    const known = list.filter(p => idx.has(p.pid)).sort((a, b) => (idx.get(a.pid) ?? 0) - (idx.get(b.pid) ?? 0));
    const unknown = list.filter(p => !idx.has(p.pid));
    return [...known, ...unknown];
}

export function orderedPids(list: HostProcess[], sort: ProcessSort): number[] {
    return groupProcesses(list, sort).flatMap(g => g.procs.map(p => p.pid));
}

export function groupProcesses(list: HostProcess[], sort: ProcessSort, lockedPids?: number[] | null): ProcessGroup[] {
    if (lockedPids && lockedPids.length > 0) {
        const ordered = applyLockedOrder(list, lockedPids);
        const map = new Map<string, HostProcess[]>();
        const groupOrder: string[] = [];
        for (const p of ordered) {
            const key = p.name || '?';
            const arr = map.get(key);
            if (arr) {
                arr.push(p);
            } else {
                map.set(key, [p]);
                groupOrder.push(key);
            }
        }
        return groupOrder.map(name => {
            const procs = map.get(name) ?? [];
            return {
                name,
                cpu: procs.reduce((s, p) => s + (p.cpu_percent || 0), 0),
                rss: procs.reduce((s, p) => s + (p.rss_bytes || 0), 0),
                procs,
            };
        });
    }
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
