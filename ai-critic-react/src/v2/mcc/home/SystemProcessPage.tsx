import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import { fetchProcesses, formatBytes, formatElapsed, type HostProcess, type ProcessSnapshot } from '../../../api/processes';
import { childrenOf, processByPid, processTitle, treeSummary } from './processTree';
import { SystemKillModal } from './SystemKillModal';
import './SystemView.css';

export function SystemProcessPage() {
    const { pid: pidParam } = useParams();
    const pid = Number(pidParam);
    const navigate = useNavigate();
    const [data, setData] = useState<ProcessSnapshot | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [lastProc, setLastProc] = useState<HostProcess | null>(null);
    const [killTarget, setKillTarget] = useState<HostProcess | null>(null);
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        let cancelled = false;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchProcesses();
                if (!cancelled) {
                    setData(next);
                    setError(null);
                    const found = processByPid(next.processes, pid);
                    if (found) setLastProc(found);
                }
            } catch (err) {
                if (!cancelled) setError(err instanceof Error ? err.message : String(err));
            } finally {
                if (!cancelled) setLoading(false);
            }
            if (!cancelled) timer = window.setTimeout(tick, 2000);
        };
        tick();
        return () => {
            cancelled = true;
            if (timer) window.clearTimeout(timer);
        };
    }, [pid]);

    const live = data ? processByPid(data.processes, pid) : undefined;
    const proc = live ?? (lastProc?.pid === pid ? lastProc : null);
    const gone = !live && proc != null;
    const list = data?.processes ?? [];
    const parent = proc && proc.ppid > 0 ? processByPid(list, proc.ppid) : undefined;
    const kids = proc ? childrenOf(list, proc.pid) : [];
    const tree = proc ? treeSummary(list, proc) : null;

    return (
        <div className="mcc-system">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button className="mcc-back-btn" onClick={() => navigate('../..', { relative: 'path' })} style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}>
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {proc ? processTitle(proc) : `pid ${pidParam}`}
                </h2>
                {proc && (
                    <button type="button" className="mcc-system-kill" title={proc.protected || 'Kill'} onClick={() => setKillTarget(proc)}>
                        {proc.protected ? '🔒' : '⏻'}
                    </button>
                )}
            </div>
            <div className="mcc-system-sub">System / pid {pidParam}</div>
            {loading && data == null && (
                <div className="mcc-system-empty" style={{ margin: '0 16px' }}>Loading process…</div>
            )}
            {error && <div className="mcc-system-alert">{error}</div>}
            {gone && <div className="mcc-system-alert">This process is no longer running. Showing last snapshot.</div>}
            {data != null && proc == null && (
                <div className="mcc-system-empty" style={{ margin: '0 16px' }}>Process {pidParam} was not found.</div>
            )}
            {proc != null && tree && (
                <div className="mcc-system-pid">
                    <div className="mcc-system-hero">
                        <div className="mcc-system-hero-name">
                            {proc.name}
                            {proc.protected ? <span className="mcc-system-tag lock">protected</span> : null}
                            {(proc.tags ?? []).includes('tun') ? <span className="mcc-system-tag tun">tunnel</span> : null}
                            {(proc.tags ?? []).includes('stale') ? <span className="mcc-system-tag stale">stale</span> : null}
                        </div>
                        <div className="mcc-system-hero-meta">
                            pid {proc.pid} · ppid {proc.ppid} · {proc.user} · state {proc.state} · {formatElapsed(proc.elapsed_sec)}
                        </div>
                    </div>
                    <div className="mcc-system-tiles">
                        <div className="mcc-system-tile">
                            <div className="mcc-system-tile-lab">CPU</div>
                            <div className={`mcc-system-tile-val ${cpuClass(proc.cpu_percent)}`}>{proc.cpu_percent.toFixed(1)}%</div>
                            <div className="mcc-system-tile-meta">this process</div>
                        </div>
                        <div className="mcc-system-tile">
                            <div className="mcc-system-tile-lab">RSS</div>
                            <div className="mcc-system-tile-val">{formatBytes(proc.rss_bytes)}</div>
                            <div className="mcc-system-tile-meta">this process</div>
                        </div>
                        <div className="mcc-system-tile">
                            <div className="mcc-system-tile-lab">Tree CPU</div>
                            <div className={`mcc-system-tile-val ${cpuClass(tree.cpu)}`}>{tree.cpu.toFixed(1)}%</div>
                            <div className="mcc-system-tile-meta">{tree.count} procs · self + descendants</div>
                        </div>
                        <div className="mcc-system-tile">
                            <div className="mcc-system-tile-lab">Tree RSS</div>
                            <div className="mcc-system-tile-val">{formatBytes(tree.rss)}</div>
                            <div className="mcc-system-tile-meta">{kids.length} direct children</div>
                        </div>
                    </div>
                    <div className="mcc-system-pid-acts">
                        <button
                            type="button"
                            className="mcc-system-btn"
                            onClick={() => {
                                navigator.clipboard.writeText(proc.cmd).then(() => {
                                    setCopied(true);
                                    window.setTimeout(() => setCopied(false), 1200);
                                }).catch(() => {});
                            }}
                        >
                            {copied ? 'Copied' : 'Copy command'}
                        </button>
                        <button
                            type="button"
                            className="mcc-system-btn danger"
                            onClick={() => setKillTarget(proc)}
                            disabled={!!proc.protected}
                        >
                            {proc.protected ? 'Cannot Kill' : 'Kill Process'}
                        </button>
                    </div>
                    <div className="mcc-system-sec">Command <span>full argv</span></div>
                    <div className="mcc-system-cmd"><pre>{proc.cmd}</pre></div>
                    <div className="mcc-system-sec">Identity</div>
                    <div className="mcc-system-card mcc-system-kv">
                        <div><span>PID</span><b>{proc.pid}</b></div>
                        <div><span>PPID</span><b>{proc.ppid}</b></div>
                        <div><span>User</span><b>{proc.user}</b></div>
                        <div><span>State</span><b>{proc.state}</b></div>
                        <div><span>Elapsed</span><b>{formatElapsed(proc.elapsed_sec)}</b></div>
                        {proc.protected ? <div><span>Protected</span><b>{proc.protected}</b></div> : null}
                    </div>
                    <div className="mcc-system-sec">Parent <span>open pid page</span></div>
                    {parent ? (
                        <button type="button" className="mcc-system-card mcc-system-link" onClick={() => navigate(`../${parent.pid}`, { relative: 'path' })}>
                            <div>
                                <div className="mcc-system-n">{processTitle(parent)} · {parent.pid}</div>
                                <div className="mcc-system-s">{parent.user} · {parent.cpu_percent.toFixed(1)}% · {formatBytes(parent.rss_bytes)}</div>
                            </div>
                            <span className="mcc-system-go">Open →</span>
                        </button>
                    ) : (
                        <div className="mcc-system-card"><div className="mcc-system-muted">No parent</div></div>
                    )}
                    <div className="mcc-system-sec">Children <span>{kids.length}</span></div>
                    {kids.length === 0 ? (
                        <div className="mcc-system-card"><div className="mcc-system-muted">No child processes</div></div>
                    ) : (
                        <div className="mcc-system-group" style={{ margin: 0 }}>
                            {kids.map(child => (
                                <button
                                    type="button"
                                    className="mcc-system-child"
                                    key={child.pid}
                                    onClick={() => navigate(`../${child.pid}`, { relative: 'path' })}
                                >
                                    <div className="mcc-system-who">
                                        <div className="mcc-system-n">{processTitle(child)}</div>
                                        <div className="mcc-system-s">pid {child.pid} · {child.user} · {formatElapsed(child.elapsed_sec)}</div>
                                    </div>
                                    <span className={`mcc-system-cpu ${cpuClass(child.cpu_percent)}`}>{child.cpu_percent.toFixed(1)}%</span>
                                    <span className="mcc-system-mem">{formatBytes(child.rss_bytes)}</span>
                                    <span className="mcc-system-go">›</span>
                                </button>
                            ))}
                        </div>
                    )}
                </div>
            )}
            {killTarget && (
                <SystemKillModal
                    proc={killTarget}
                    onClose={() => setKillTarget(null)}
                    onKilled={() => {
                        setKillTarget(null);
                        fetchProcesses().then(setData).catch(() => {});
                    }}
                />
            )}
        </div>
    );
}

function cpuClass(n: number): string {
    if (n >= 80) return 'hot';
    if (n >= 20) return 'warn';
    return '';
}
