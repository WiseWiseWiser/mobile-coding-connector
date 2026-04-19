import { useState } from 'react';
import { ensureTunnel } from '../../api/ports';
import { useV2Context } from '../V2Context';

export function CloudflareInlineDiagnostics() {
    const { diagnostics: data, diagnosticsLoading: loading, refreshDiagnostics } = useV2Context();
    const [expanded, setExpanded] = useState(false);
    const [actionLoading, setActionLoading] = useState<Record<string, boolean>>({});
    const [actionResults, setActionResults] = useState<Record<string, { ok: boolean; message: string }>>({});

    const statusIcon = !data ? '⏳' : data.overall === 'ok' ? '✅' : data.overall === 'warning' ? '⚠️' : '❌';
    const statusText = !data ? 'Checking...' : data.overall === 'ok' ? 'Cloudflare: Ready' : data.overall === 'warning' ? 'Cloudflare: Warning' : 'Cloudflare: Issues Found';

    const statusColors: Record<string, string> = {
        ok: '#22c55e',
        warning: '#eab308',
        error: '#ef4444',
    };

    const hasIssues = data && data.overall !== 'ok';

    const handleCreateTunnel = (check: { id: string; label: string; description: string }) => {
        const match = check.label.match(/Tunnel '([^']+)'/);
        if (!match) return;
        const tunnelName = match[1];

        setActionLoading(prev => ({ ...prev, [check.id]: true }));
        setActionResults(prev => { const next = { ...prev }; delete next[check.id]; return next; });

        ensureTunnel(tunnelName)
            .then(result => {
                setActionResults(prev => ({ ...prev, [check.id]: { ok: true, message: `Tunnel created: ${result.tunnel_id}` } }));
                refreshDiagnostics();
            })
            .catch(err => setActionResults(prev => ({ ...prev, [check.id]: { ok: false, message: err.message } })))
            .finally(() => setActionLoading(prev => ({ ...prev, [check.id]: false })));
    };

    return (
        <div className="mcc-cf-inline-diag">
            <button
                className={`mcc-cf-status-banner mcc-cf-status-${data?.overall ?? 'loading'}`}
                onClick={() => setExpanded(!expanded)}
            >
                <span className="mcc-cf-status-icon">{statusIcon}</span>
                <span className="mcc-cf-status-text">{statusText}</span>
                <span className={`mcc-cf-status-chevron ${expanded ? 'expanded' : ''}`}>›</span>
            </button>
            {(expanded || hasIssues) && data && (
                <div className="mcc-diag-checks">
                    {data.checks.filter(c => expanded || c.status !== 'ok').map(check => (
                        <div key={check.id} className="mcc-diag-check">
                            <div className="mcc-diag-check-header">
                                <span
                                    className="mcc-diag-check-dot"
                                    style={{ background: statusColors[check.status] ?? '#64748b' }}
                                />
                                <span className="mcc-diag-check-label">{check.label}</span>
                                <span className={`mcc-diag-check-status mcc-diag-check-status-${check.status}`}>
                                    {check.status.toUpperCase()}
                                </span>
                            </div>
                            <div className="mcc-diag-check-desc">{check.description}</div>
                            {check.id === 'tunnel_exists' && check.status === 'error' && (
                                <div className="mcc-diag-check-action">
                                    <button
                                        className="mcc-port-action-btn"
                                        onClick={() => handleCreateTunnel(check)}
                                        disabled={actionLoading[check.id]}
                                    >
                                        {actionLoading[check.id] ? 'Creating...' : 'Create Tunnel'}
                                    </button>
                                    {actionResults[check.id] && (
                                        <span className={actionResults[check.id].ok ? 'mcc-tunnel-dns-ok' : 'mcc-tunnel-dns-error'}
                                            style={{ display: 'inline-block', padding: '4px 8px', borderRadius: '4px', fontSize: '12px', marginLeft: '8px' }}>
                                            {actionResults[check.id].message}
                                        </span>
                                    )}
                                </div>
                            )}
                        </div>
                    ))}
                    <button className="mcc-diag-refresh" onClick={refreshDiagnostics} disabled={loading}>
                        {loading ? 'Refreshing...' : 'Refresh'}
                    </button>
                </div>
            )}
        </div>
    );
}
