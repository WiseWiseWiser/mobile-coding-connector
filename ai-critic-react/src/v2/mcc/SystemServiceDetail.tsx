import { useCallback, useEffect, useRef, useState } from 'react';
import type { ServiceStatus } from '../../api/services';
import {
    fetchHttpProxyConfig,
    fetchHttpProxyConnections,
    fetchHttpProxyStatus,
    fetchWsProxyConnections,
    fetchWsProxyQR,
    fetchWsProxyVMessLink,
    rotateHttpProxyPassword,
    startHttpProxy,
    stopHttpProxy,
    type HttpProxyConnections,
    type HttpProxyStatus,
    type ProxyConnInfo,
    type WsProxyConnections,
} from '../../api/proxyServices';

export interface SystemServiceDetailProps {
    service: ServiceStatus;
    onClose: () => void;
}

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

function MethodBadge({ method }: { method: string }) {
    return (
        <span className={method === 'CONNECT' ? 'mcc-conn-method-connect' : 'mcc-conn-method-plain'}>
            {method}
        </span>
    );
}

function ConnTable({ items, emptyText }: { items: ProxyConnInfo[]; emptyText: string }) {
    if (!items.length) {
        return <div className="mcc-ports-empty">{emptyText}</div>;
    }
    return (
        <table className="mcc-conn-table">
            <thead>
                <tr>
                    <th>Client</th>
                    <th>Target</th>
                    <th>Method</th>
                    <th>↑/↓</th>
                    <th>Duration</th>
                    <th>State</th>
                </tr>
            </thead>
            <tbody>
                {items.map((c) => (
                    <tr key={`${c.id}-${c.client}-${c.target}`}>
                        <td>{c.client}</td>
                        <td className="mcc-conn-target">{c.target}</td>
                        <td><MethodBadge method={c.method} /></td>
                        <td>{formatBytes(c.up)} / {formatBytes(c.down)}</td>
                        <td>{formatDuration(c.secs)}</td>
                        <td>
                            <span className={`mcc-conn-state mcc-conn-state--${c.state}`}>
                                {c.state === 'established' || c.state === 'active' ? '●' : '○'} {c.state}
                            </span>
                            {c.err && <span className="mcc-conn-err" title={c.err}> ⚠</span>}
                        </td>
                    </tr>
                ))}
            </tbody>
        </table>
    );
}

function usePolling<T>(fn: () => Promise<T>, enabled: boolean, deps: unknown[] = []) {
    const [data, setData] = useState<T | null>(null);
    const [error, setError] = useState<string | null>(null);
    const fnRef = useRef(fn);
    fnRef.current = fn;
    useEffect(() => {
        if (!enabled) return;
        let alive = true;
        const tick = async () => {
            try {
                const next = await fnRef.current();
                if (alive) {
                    setData(next);
                    setError(null);
                }
            } catch (err) {
                if (alive) setError(err instanceof Error ? err.message : String(err));
            }
        };
        tick();
        const timer = window.setInterval(tick, POLL_MS);
        return () => {
            alive = false;
            window.clearInterval(timer);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [enabled, ...deps]);
    return { data, error };
}

/**
 * SystemServiceDetail is the click-through detail view for the proxy system
 * services: live connections for both, QR + vmess link for the WS proxy, and
 * credential management for the HTTP proxy.
 */
export function SystemServiceDetail({ service, onClose }: SystemServiceDetailProps) {
    const isHttp = service.id === 'sys-http-proxy';
    const isWs = service.id === 'sys-ws-proxy';

    const [httpStatus, setHttpStatus] = useState<HttpProxyStatus | null>(null);
    const [revealed, setRevealed] = useState(false);
    const [config, setConfig] = useState<{ username: string; password: string; domain?: string } | null>(null);
    const [actionErr, setActionErr] = useState<string | null>(null);
    const [qrUrl, setQrUrl] = useState<string | null>(null);
    const [vmessLink, setVmessLink] = useState<string | null>(null);

    const loadHttp = useCallback(async () => {
        const st = await fetchHttpProxyStatus();
        setHttpStatus(st);
        return st;
    }, []);
    const httpConns = usePolling<HttpProxyConnections>(
        fetchHttpProxyConnections, isHttp && service.status === 'running',
    );
    const wsConns = usePolling<WsProxyConnections>(fetchWsProxyConnections, isWs);

    useEffect(() => {
        if (!isHttp) return;
        loadHttp().catch((err) => setActionErr(err instanceof Error ? err.message : String(err)));
    }, [isHttp, loadHttp]);

    useEffect(() => {
        if (!isWs) return;
        fetchWsProxyQR()
            .then((url) => {
                setQrUrl((prev) => {
                    if (prev) URL.revokeObjectURL(prev);
                    return url;
                });
            })
            .catch(() => setQrUrl(null));
        fetchWsProxyVMessLink().then((r) => setVmessLink(r.vmess_link)).catch(() => setVmessLink(null));
    }, [isWs]);

    const doAction = async (fn: () => Promise<unknown>) => {
        setActionErr(null);
        try {
            await fn();
            await loadHttp();
        } catch (err) {
            setActionErr(err instanceof Error ? err.message : String(err));
        }
    };

    const rotate = async () => {
        setActionErr(null);
        try {
            const { password } = await rotateHttpProxyPassword();
            setRevealed(true);
            setConfig((prev) => (prev ? { ...prev, password } : prev));
            await loadHttp();
        } catch (err) {
            setActionErr(err instanceof Error ? err.message : String(err));
        }
    };

    const revealPassword = async () => {
        if (revealed && config) {
            setRevealed(false);
            return;
        }
        try {
            const cfg = await fetchHttpProxyConfig();
            setConfig({ username: cfg.username, password: cfg.password, domain: cfg.domain });
            setRevealed(true);
        } catch (err) {
            setActionErr(err instanceof Error ? err.message : String(err));
        }
    };

    return (
        <div className="mcc-svc-detail-overlay" onClick={onClose}>
            <div className="mcc-svc-detail" onClick={(e) => e.stopPropagation()}>
                <div className="mcc-svc-detail-header">
                    <h3>{service.name}</h3>
                    <span className={`mcc-service-status-pill mcc-service-status--${service.status}`}>
                        {service.status}
                    </span>
                    <button className="mcc-svc-detail-close" onClick={onClose} aria-label="Close">×</button>
                </div>

                {actionErr && <div className="mcc-svc-detail-error">{actionErr}</div>}

                {isHttp && (
                    <>
                        <div className="mcc-svc-detail-grid">
                            <div className="mcc-svc-detail-card">
                                <h4>Status</h4>
                                <div className="mcc-kv"><span>Detail</span><span>{httpStatus?.detail ?? '…'}</span></div>
                                <div className="mcc-kv"><span>Listen</span><span>:{httpStatus?.port ?? '-'}</span></div>
                                <div className="mcc-kv"><span>Auth user</span><span>{httpStatus?.auth_user ?? '-'}</span></div>
                                <div className="mcc-kv">
                                    <span>Domain</span>
                                    <span>{httpStatus?.domain || <em>LAN/direct only</em>}</span>
                                </div>
                                {httpStatus?.publish_error && (
                                    <div className="mcc-kv"><span>Publish</span><span className="mcc-conn-err">{httpStatus.publish_error}</span></div>
                                )}
                            </div>
                            <div className="mcc-svc-detail-card">
                                <h4>Credentials</h4>
                                <div className="mcc-kv">
                                    <span>Password</span>
                                    <span className="mcc-mono">
                                        {revealed && config ? config.password : '••••••••'}
                                    </span>
                                </div>
                                <div className="mcc-svc-detail-actions">
                                    <button className="mcc-btn" onClick={revealPassword}>
                                        {revealed ? 'Hide' : 'Show'}
                                    </button>
                                    <button className="mcc-btn" onClick={rotate}>Rotate</button>
                                    {httpStatus?.running ? (
                                        <button className="mcc-btn mcc-btn-danger" onClick={() => doAction(stopHttpProxy)}>Stop</button>
                                    ) : (
                                        <button className="mcc-btn mcc-btn-primary" onClick={() => doAction(startHttpProxy)}>Start</button>
                                    )}
                                </div>
                                <div className="mcc-svc-detail-hint">
                                    iPhone Wi-Fi 代理:服务器填映射域名或本机 IP,端口 {httpStatus?.port ?? '-'},
                                    认证 user / 上方密码。
                                </div>
                            </div>
                        </div>
                        <div className="mcc-svc-detail-card">
                            <h4>
                                Connections
                                <span className="mcc-conn-summary">
                                    {httpConns.data
                                        ? `${httpConns.data.active} active · ${httpConns.data.connect_active} CONNECT · ${httpConns.data.failures} auth failures`
                                        : httpConns.error ?? 'loading…'}
                                </span>
                            </h4>
                            <ConnTable
                                items={httpConns.data?.items ?? []}
                                emptyText={httpConns.error ? `live stats unavailable: ${httpConns.error}` : 'No connections yet.'}
                            />
                        </div>
                    </>
                )}

                {isWs && (
                    <>
                        <div className="mcc-svc-detail-grid">
                            <div className="mcc-svc-detail-card mcc-svc-detail-card--center">
                                <h4>Scan to import (Shadowrocket / Stash)</h4>
                                {qrUrl ? (
                                    <img className="mcc-qr-img" src={qrUrl} alt="vmess QR" />
                                ) : (
                                    <div className="mcc-ports-empty">QR unavailable (proxy not running?)</div>
                                )}
                                {vmessLink && (
                                    <div className="mcc-mono mcc-vmess-link" title={vmessLink}>
                                        {vmessLink.slice(0, 40)}…
                                        <button
                                            className="mcc-btn"
                                            onClick={() => navigator.clipboard?.writeText(vmessLink)}
                                        >Copy</button>
                                    </div>
                                )}
                            </div>
                            <div className="mcc-svc-detail-card">
                                <h4>Sessions</h4>
                                <div className="mcc-kv">
                                    <span>Active streams</span>
                                    <span>{wsConns.data ? `${wsConns.data.sessions}` : wsConns.error ?? '…'}</span>
                                </div>
                                {(wsConns.data?.items ?? []).map((s, i) => (
                                    <div className="mcc-kv" key={i}>
                                        <span className="mcc-mono">{s.peer}</span>
                                        <span className="mcc-mono">→ {s.local}</span>
                                    </div>
                                ))}
                                <div className="mcc-svc-detail-hint">
                                    Per-device attribution needs the Xray stats API; streams are counted on the
                                    xray listen port. {wsConns.data?.port ? `(:${wsConns.data.port})` : ''}
                                </div>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
