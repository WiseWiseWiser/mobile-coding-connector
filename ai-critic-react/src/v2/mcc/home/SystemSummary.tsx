import { formatBytes, type ProcessSnapshot } from '../../../api/processes';

export function SystemSummary({ data }: { data: ProcessSnapshot }) {
    const cpu = data.cpu.used_percent;
    const memPct = data.memory.used_percent;
    const swapPct = data.memory.swap_used_percent ?? 0;
    const pressure = memPct >= 90 || swapPct >= 70 || (data.memory.compressor_bytes ?? 0) > 1 << 30;
    return (
        <>
            <div className="mcc-system-tiles">
                <div className="mcc-system-tile">
                    <div className="mcc-system-tile-lab">CPU</div>
                    <div className={`mcc-system-tile-val ${cpu >= 80 ? 'hot' : cpu >= 50 ? 'warn' : ''}`}>
                        {cpu.toFixed(0)}%
                    </div>
                    <div className="mcc-system-tile-meta">
                        load {data.load.m1.toFixed(1)} / {data.load.m5.toFixed(1)} / {data.load.m15.toFixed(1)} · {data.cpu.num_cpu} cores
                    </div>
                </div>
                <div className="mcc-system-tile">
                    <div className="mcc-system-tile-lab">Memory</div>
                    <div className={`mcc-system-tile-val ${memPct >= 90 ? 'hot' : memPct >= 75 ? 'warn' : ''}`}>
                        {formatBytes(data.memory.used)}
                    </div>
                    <div className="mcc-system-bar"><i style={{ width: `${Math.min(100, memPct)}%` }} /></div>
                    <div className="mcc-system-tile-meta">
                        of {formatBytes(data.memory.total)}
                        {data.memory.compressor_bytes ? ` · compressor ${formatBytes(data.memory.compressor_bytes)}` : ''}
                    </div>
                </div>
            </div>
            <div className="mcc-system-kpis">
                <span>Procs <b>{data.counts.processes}</b></span>
                {data.memory.swap_total ? (
                    <span>Swap <b>{formatBytes(data.memory.swap_used ?? 0)} / {formatBytes(data.memory.swap_total)}</b></span>
                ) : null}
            </div>
            {pressure && (
                <div className="mcc-system-alert">
                    Memory pressure is high
                    {data.memory.compressor_bytes ? ` (compressor ${formatBytes(data.memory.compressor_bytes)})` : ''}
                    {data.memory.swap_total ? ` · swap ${swapPct.toFixed(0)}%` : ''}.
                    Killing browsers or stale processes usually helps.
                </div>
            )}
        </>
    );
}
