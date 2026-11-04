import { useCallback, useEffect, useState } from 'react';
import {
    fetchHttpProxyConnections,
    fetchHttpProxyStatus,
    rotateHttpProxyPassword,
    startHttpProxy,
    stopHttpProxy,
    type HttpProxyConnections,
    type HttpProxyStatus,
    type ProxyConnInfo,
} from '../../api/proxyServices';

const POLL_MS = 2000;

function formatBytes(n: number): string {
    if (n < 1024) return `${n}B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`;
    if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)}MB`;
    return `${(n / 1024 / 1024 / 1024).toFixed(2)}GB`;
}

function formatDuration(secs: number): string {
    if (secs < 60) return `${secs}s`;
    if (secs < 3600) return `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, '0')}`;
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    return `${h}:${String(m).padStart(2, '0')}:${String(secs % 60).padStart(2, '0')}`;
}

/** HTTPProxyPage is the sub-page detail for the HTTP Proxy system service. */
export function HTTPProxyPage() {
    const [status, setStatus] = useState<HttpProxyStatus | null>(null);
    const [conns, setConns] = useState<HttpProxyConnections | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [revealed, setRevealed] = useState(false);
    const [password, setPassword] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        try {
            setStatus(await fetchHttpProxyStatus());
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    }, []);

    useEffect(() => {
        void refresh();
        const timer = window.setInterval(async () => {
            try { setConns(await fetchHttpProxyConnections()); } catch { /* transient */ }
        }, POLL_MS);
        void fetchHttpProxyConnections().then(setConns).catch(() => undefined);
        return () => window.clearInterval(timer);
    }, [refresh]);

    const act = async (fn: () => Promise<unknown>) => {
        try { await fn(); await refresh(); } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const rotate = async () => {
        try {
            const { password: pw } = await rotateHttpProxyPassword();
            setPassword(pw); setRevealed(true); await refresh();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const items: ProxyConnInfo[] = conns?.items ?? [];

    return (
        <div className="mcc-ws-page">
            <div className="mcc-crumb"><span>Services</span><span className="mcc-crumb-sep">/</span><span>HTTP Proxy</span></div>
            <div className="mcc-headerbar">
                <h2>HTTP Proxy</h2>
                <span className={`mcc-pill ${status?.running ? 'run' : 'stop'}`}>
                    {status?.running ? 'running' : 'stopped'}{status?.uptime_secs ? ` · ${formatDuration(status.uptime_secs)}` : ''}
                </span>
            </div>
            {error && <div className="mcc-svc-detail-error">{error}</div>}

            <div className="mcc-grid2">
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>Status</div>
                    <div className="mcc-kv"><span className="k">Detail</span><span>{status?.detail ?? '…'}</span></div>
                    <div className="mcc-kv"><span className="k">Listen</span><span>:{status?.port ?? '-'} (0.0.0.0)</span></div>
                    <div className="mcc-kv"><span className="k">Auth user</span><span>{status?.auth_user ?? '-'}</span></div>
                    <div className="mcc-kv"><span className="k">Domain</span><span>{status?.domain || <em>LAN/direct only</em>}</span></div>
                    {status?.publish_error && (
                        <div className="mcc-kv"><span className="k">Publish</span><span className="mcc-conn-err">{status.publish_error}</span></div>
                    )}
                </div>
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>Credentials</div>
                    <div className="mcc-kv">
                        <span className="k">Password</span>
                        <span className="mcc-mono">{revealed && password ? password : '••••••••'}</span>
                    </div>
                    <div className="mcc-svc-detail-actions">
                        <button className="mcc-btn" onClick={() => setRevealed((v) => !v)}>{revealed ? 'Hide' : 'Show'}</button>
                        <button className="mcc-btn" onClick={rotate}>Rotate</button>
                        {status?.running
                            ? <button className="mcc-btn mcc-btn-danger" onClick={() => act(stopHttpProxy)}>Stop</button>
                            : <button className="mcc-btn mcc-btn-primary" onClick={() => act(startHttpProxy)}>Start</button>}
                    </div>
                    <div className="mcc-svc-detail-hint">
                        iPhone Wi-Fi 代理:服务器填映射域名或本机 IP,端口 {status?.port ?? '-'};CONNECT 走直连/灰云,CF 橙色云用 WS Proxy。
                    </div>
                </div>
            </div>

            <div className="mcc-card" style={{ marginTop: 12 }}>
                <div className="mcc-sect" style={{ marginTop: 0 }}>
                    Connections
                    <span className="mcc-conn-summary">
                        {conns ? `${conns.active} active · ${conns.connect_active} CONNECT · ${conns.failures} auth failures` : 'loading…'}
                    </span>
                </div>
                {!items.length && <div className="mcc-ports-empty">No connections yet.</div>}
                {items.length > 0 && (
                    <table className="mcc-conn-table">
                        <thead>
                            <tr><th>Client</th><th>Target</th><th>Method</th><th>↑/↓</th><th>Duration</th><th>State</th></tr>
                        </thead>
                        <tbody>
                            {items.map((c) => (
                                <tr key={`${c.id}-${c.client}`}>
                                    <td>{c.client}</td>
                                    <td>{c.target}</td>
                                    <td className={c.method === 'CONNECT' ? 'mcc-conn-method-connect' : 'mcc-conn-method-plain'}>{c.method}</td>
                                    <td>{formatBytes(c.up)} / {formatBytes(c.down)}</td>
                                    <td>{formatDuration(c.secs)}</td>
                                    <td>{c.state}{c.err ? ` ⚠ ${c.err}` : ''}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
}
