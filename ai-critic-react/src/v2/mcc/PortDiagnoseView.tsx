import { useState, useEffect } from 'react';
import type { PortForward } from '../../hooks/usePortForwards';
import { PortStatuses } from '../../hooks/usePortForwards';
import { fetchPortLogs as apiFetchPortLogs } from '../../api/ports';
import { LogViewer } from '../LogViewer';

export function PortDiagnoseView({ port, portData, onBack }: { port: number; portData?: PortForward; onBack: () => void }) {
    const [result, setResult] = useState<{ status: string; body: string } | null>(null);
    const [loading, setLoading] = useState(false);

    const publicUrl = portData?.publicUrl;

    const runCheck = async () => {
        if (!publicUrl) return;
        setLoading(true);
        try {
            const resp = await fetch(publicUrl, { mode: 'no-cors' }).catch(() => null);
            if (!resp) {
                setResult({ status: 'error', body: 'Failed to reach the URL. The tunnel may not be active or the local service is not running.' });
            } else {
                setResult({ status: 'reachable', body: `Got response (opaque due to CORS). Status type: ${resp.type}` });
            }
        } catch {
            setResult({ status: 'error', body: 'Network error when trying to reach the URL.' });
        }
        setLoading(false);
    };

    const [logs, setLogs] = useState<string[]>([]);
    useEffect(() => {
        apiFetchPortLogs(port)
            .then(data => setLogs(data))
            .catch(() => {});
    }, [port]);

    const logsText = logs.join('\n');
    const isViteError = logsText.includes('allowedHosts') || logsText.includes('This host') || logsText.includes('is not allowed');
    const isTunnelError = logsText.includes('failed to start') || logsText.includes('tunnel exited');
    const isTimeout = logsText.includes('timeout');

    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Diagnose Port {port}</h2>
            </div>

            {portData && (
                <div className="mcc-diag-port-info">
                    <div><strong>Status:</strong> {portData.status}</div>
                    <div><strong>Provider:</strong> {portData.provider}</div>
                    {publicUrl && <div><strong>URL:</strong> <a href={publicUrl} target="_blank" rel="noopener noreferrer">{publicUrl}</a></div>}
                    {portData.error && <div className="mcc-port-url-error"><strong>Error:</strong> {portData.error}</div>}
                </div>
            )}

            <div className="mcc-diag-checks">
                {isViteError && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#ef4444' }} />
                            <span className="mcc-diag-check-label">Vite Host Blocked</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-error">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            Vite's dev server is blocking requests from the tunnel hostname. Add the following to your <code>vite.config.ts</code>:
                        </div>
                        <div className="mcc-ports-help-cmd" style={{ margin: '8px 0' }}>
                            <code>{`server: {\n  allowedHosts: true\n}`}</code>
                        </div>
                        <div className="mcc-diag-check-desc">
                            Then restart the Vite dev server.
                        </div>
                    </div>
                )}

                {isTunnelError && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#ef4444' }} />
                            <span className="mcc-diag-check-label">Tunnel Process Error</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-error">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel process failed to start or exited unexpectedly. Check the logs below for details.
                        </div>
                    </div>
                )}

                {isTimeout && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#eab308' }} />
                            <span className="mcc-diag-check-label">Timeout</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-warning">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel took too long to establish. This may be a network issue.
                        </div>
                    </div>
                )}

                {!isViteError && !isTunnelError && !isTimeout && portData?.status === PortStatuses.Active && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#22c55e' }} />
                            <span className="mcc-diag-check-label">No issues detected</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-ok">OK</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel appears to be running normally. If you're having issues, check that the local service on port {port} is running.
                        </div>
                    </div>
                )}
            </div>

            {publicUrl && (
                <button className="mcc-diag-refresh" onClick={runCheck} disabled={loading}>
                    {loading ? 'Checking...' : 'Test Connectivity'}
                </button>
            )}
            {result && (
                <div className={`mcc-diag-port-info ${result.status === 'error' ? 'mcc-diag-port-error' : ''}`}>
                    {result.body}
                </div>
            )}

            {logs.length > 0 && (
                <>
                    <div className="mcc-ports-subtitle" style={{ margin: '16px 16px 8px' }}>Recent Logs</div>
                    <div style={{ margin: '0 16px 16px' }}>
                        <LogViewer lines={logs.map(text => ({ text }))} />
                    </div>
                </>
            )}
        </div>
    );
}
