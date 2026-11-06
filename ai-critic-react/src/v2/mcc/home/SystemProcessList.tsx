import { useState } from 'react';
import { formatBytes, formatElapsed, type HostProcess } from '../../../api/processes';
import { groupProcesses, type ProcessSort } from './groupProcesses';

export function SystemProcessList({
    processes,
    sort,
    onKill,
}: {
    processes: HostProcess[];
    sort: ProcessSort;
    onKill: (proc: HostProcess) => void;
}) {
    const groups = groupProcesses(processes, sort);
    const [open, setOpen] = useState<Record<string, boolean>>({});

    if (groups.length === 0) {
        return <div className="mcc-system-empty">No processes match.</div>;
    }

    return (
        <div className="mcc-system-list">
            {groups.map(group => {
                const expanded = open[group.name] ?? group.procs.length <= 2;
                return (
                    <div className="mcc-system-group" key={group.name}>
                        <button
                            type="button"
                            className="mcc-system-ghead"
                            onClick={() => setOpen(cur => ({ ...cur, [group.name]: !expanded }))}
                        >
                            <span>{expanded ? '▾' : '▸'}</span>
                            <div className="mcc-system-who">
                                <div className="mcc-system-n">
                                    {group.name}
                                    {group.procs.length > 1 ? ` · ${group.procs.length}` : ''}
                                </div>
                            </div>
                            <span className={`mcc-system-cpu ${cpuClass(group.cpu)}`}>{group.cpu.toFixed(1)}%</span>
                            <span className="mcc-system-mem">{formatBytes(group.rss)}</span>
                            <span />
                        </button>
                        {expanded && group.procs.map(proc => (
                            <ProcessRow key={proc.pid} proc={proc} onKill={onKill} />
                        ))}
                    </div>
                );
            })}
        </div>
    );
}

function ProcessRow({ proc, onKill }: { proc: HostProcess; onKill: (proc: HostProcess) => void }) {
    const tags = proc.tags ?? [];
    return (
        <div className="mcc-system-row">
            <span />
            <div className="mcc-system-who">
                <div className="mcc-system-n">
                    pid {proc.pid}
                    {tags.includes('stale') ? <span className="mcc-system-tag stale">stale</span> : null}
                    {tags.includes('tun') ? <span className="mcc-system-tag tun">tunnel</span> : null}
                    {proc.protected ? <span className="mcc-system-tag lock">protected</span> : null}
                </div>
                <div className="mcc-system-s">{proc.user} · {formatElapsed(proc.elapsed_sec)} · {proc.cmd}</div>
            </div>
            <span className={`mcc-system-cpu ${cpuClass(proc.cpu_percent)}`}>{proc.cpu_percent.toFixed(1)}%</span>
            <span className="mcc-system-mem">{formatBytes(proc.rss_bytes)}</span>
            <button
                type="button"
                className="mcc-system-kill"
                title={proc.protected || 'Kill'}
                onClick={() => onKill(proc)}
            >
                {proc.protected ? '🔒' : '⏻'}
            </button>
        </div>
    );
}

function cpuClass(n: number): string {
    if (n >= 80) return 'hot';
    if (n >= 20) return 'warn';
    return '';
}
