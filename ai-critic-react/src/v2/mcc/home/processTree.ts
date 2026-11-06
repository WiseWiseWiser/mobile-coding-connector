import type { HostProcess } from '../../../api/processes';

export function processByPid(list: HostProcess[], pid: number): HostProcess | undefined {
    return list.find(p => p.pid === pid);
}

export function childrenOf(list: HostProcess[], pid: number): HostProcess[] {
    return list.filter(p => p.ppid === pid).sort((a, b) => b.cpu_percent - a.cpu_percent);
}

export function descendantsOf(list: HostProcess[], pid: number): HostProcess[] {
    const out: HostProcess[] = [];
    const walk = (id: number) => {
        for (const child of childrenOf(list, id)) {
            out.push(child);
            walk(child.pid);
        }
    };
    walk(pid);
    return out;
}

export function treeSummary(list: HostProcess[], proc: HostProcess): { count: number; cpu: number; rss: number } {
    const desc = descendantsOf(list, proc.pid);
    return {
        count: 1 + desc.length,
        cpu: desc.reduce((s, p) => s + (p.cpu_percent || 0), proc.cpu_percent || 0),
        rss: desc.reduce((s, p) => s + (p.rss_bytes || 0), proc.rss_bytes || 0),
    };
}

export function processTitle(proc: HostProcess): string {
    const cmd = proc.cmd || '';
    if (cmd.includes('--type=renderer')) return 'Renderer';
    if (cmd.includes('--type=gpu')) return 'GPU';
    if (cmd.includes('NetworkService')) return 'Network';
    return proc.name || `pid ${proc.pid}`;
}
