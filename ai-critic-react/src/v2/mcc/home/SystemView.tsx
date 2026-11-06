import { useEffect, useState } from 'react';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { NavTabs } from '../types';
import { fetchProcesses, type HostProcess, type ProcessSnapshot } from '../../../api/processes';
import { filterProcesses, type ProcessFilter, type ProcessSort } from './groupProcesses';
import { SystemSummary } from './SystemSummary';
import { SystemProcessList } from './SystemProcessList';
import { SystemKillModal } from './SystemKillModal';
import './SystemView.css';

export function SystemView() {
    const { goBack } = useTabHistory(NavTabs.Home);
    const [data, setData] = useState<ProcessSnapshot | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [paused, setPaused] = useState(false);
    const [query, setQuery] = useState('');
    const [sort, setSort] = useState<ProcessSort>('cpu');
    const [filter, setFilter] = useState<ProcessFilter>('all');
    const [target, setTarget] = useState<HostProcess | null>(null);

    useEffect(() => {
        let cancelled = false;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchProcesses();
                if (!cancelled) {
                    setData(next);
                    setError(null);
                }
            } catch (err) {
                if (!cancelled) setError(err instanceof Error ? err.message : String(err));
            } finally {
                if (!cancelled) setLoading(false);
            }
            if (!cancelled && !paused) {
                timer = window.setTimeout(tick, 2000);
            }
        };
        tick();
        return () => {
            cancelled = true;
            if (timer) window.clearTimeout(timer);
        };
    }, [paused]);

    const visible = data ? filterProcesses(data.processes, query, filter) : [];

    return (
        <div className="mcc-system">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button className="mcc-back-btn" onClick={goBack} style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}>
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0 }}>System</h2>
                <span className={`mcc-system-live${paused ? ' paused' : ''}`}>
                    {paused ? 'paused' : data ? `${Math.max(1, Math.round((data.interval_ms || 2000) / 1000))}s` : '…'}
                </span>
                <button
                    type="button"
                    className={`mcc-system-chip${paused ? ' on' : ''}`}
                    onClick={() => setPaused(v => !v)}
                >
                    {paused ? 'Resume' : 'Pause'}
                </button>
            </div>
            {data && (
                <div className="mcc-system-sub">
                    {data.cpu.num_cpu} cores · {data.counts.processes} processes · host of this page
                </div>
            )}
            {loading && data == null && (
                <div className="mcc-system-empty" style={{ margin: '0 16px' }}>Loading processes…</div>
            )}
            {error && (
                <div className="mcc-system-alert">{error}</div>
            )}
            {data != null && (
                <>
                    <SystemSummary data={data} />
                    <div className="mcc-system-controls">
                        <input
                            className="mcc-system-search"
                            type="search"
                            placeholder="Filter by name, pid, command…"
                            value={query}
                            onChange={e => setQuery(e.target.value)}
                        />
                        <div className="mcc-system-chips">
                            {([
                                ['cpu', 'CPU ↓'],
                                ['mem', 'Memory'],
                                ['name', 'Name'],
                                ['pid', 'PID'],
                            ] as const).map(([id, label]) => (
                                <button
                                    key={id}
                                    type="button"
                                    className={`mcc-system-chip${sort === id ? ' on' : ''}`}
                                    onClick={() => setSort(id)}
                                >
                                    {label}
                                </button>
                            ))}
                        </div>
                        <div className="mcc-system-chips">
                            {([
                                ['all', 'All'],
                                ['mine', 'Mine'],
                                ['stale', 'Stale / tunnels'],
                            ] as const).map(([id, label]) => (
                                <button
                                    key={id}
                                    type="button"
                                    className={`mcc-system-chip${filter === id ? ' on' : ''}`}
                                    onClick={() => setFilter(id)}
                                >
                                    {label}
                                </button>
                            ))}
                        </div>
                    </div>
                    <SystemProcessList processes={visible} sort={sort} onKill={setTarget} />
                </>
            )}
            {target && (
                <SystemKillModal
                    proc={target}
                    onClose={() => setTarget(null)}
                    onKilled={() => {
                        setTarget(null);
                        fetchProcesses().then(setData).catch(() => {});
                    }}
                />
            )}
        </div>
    );
}
