import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchProcesses, type ProcessSnapshot } from '../../../api/processes';
import './SystemView.css';

export function SystemStrip() {
    const navigate = useNavigate();
    const [data, setData] = useState<ProcessSnapshot | null>(null);

    useEffect(() => {
        let cancelled = false;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchProcesses(true);
                if (!cancelled) setData(next);
            } catch {
                // keep last snapshot
            }
            if (!cancelled) timer = window.setTimeout(tick, 5000);
        };
        tick();
        return () => {
            cancelled = true;
            if (timer) window.clearTimeout(timer);
        };
    }, []);

    const cpu = data?.cpu.used_percent ?? 0;
    const memPct = data?.memory.used_percent ?? 0;
    const load = data?.load.m1 ?? 0;
    const procs = data?.counts.processes ?? 0;
    const cpuClass = cpu >= 80 ? 'hot' : cpu >= 50 ? 'warn' : '';
    const memClass = memPct >= 90 ? 'hot' : memPct >= 75 ? 'warn' : '';

    return (
        <button className="mcc-system-strip" type="button" onClick={() => navigate('system')}>
            <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                    <span style={{ fontSize: 15, fontWeight: 600, color: '#e2e8f0' }}>System</span>
                    <span className="mcc-system-live">{data ? 'live' : '…'}</span>
                </div>
                <div className="mcc-system-kpis" style={{ padding: 0, margin: 0 }}>
                    <span className={cpuClass}>CPU <b>{cpu.toFixed(0)}%</b></span>
                    <span className={memClass}>Mem <b>{memPct.toFixed(0)}%</b></span>
                    <span>Load <b>{load.toFixed(1)}</b></span>
                    <span>{procs ? <><b>{procs}</b> procs</> : 'host resources'}</span>
                </div>
            </div>
            <span className="mcc-system-strip-go">Processes →</span>
        </button>
    );
}
