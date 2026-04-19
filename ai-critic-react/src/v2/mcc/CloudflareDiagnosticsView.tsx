import { useV2Context } from '../V2Context';

export function CloudflareDiagnosticsView({ onBack }: { onBack: () => void }) {
    const { diagnostics: data, diagnosticsLoading: loading, refreshDiagnostics } = useV2Context();

    const statusColors: Record<string, string> = {
        ok: '#22c55e',
        warning: '#eab308',
        error: '#ef4444',
    };

    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Cloudflare Diagnostics</h2>
            </div>
            {loading ? (
                <div className="mcc-diag-loading">Running diagnostics...</div>
            ) : !data ? (
                <div className="mcc-ports-error">Failed to load diagnostics</div>
            ) : (
                <>
                    <div className={`mcc-diag-overall mcc-diag-overall-${data.overall}`}>
                        <span className="mcc-diag-overall-icon">
                            {data.overall === 'ok' ? '✅' : data.overall === 'warning' ? '⚠️' : '❌'}
                        </span>
                        <span>
                            {data.overall === 'ok' ? 'All checks passed' :
                             data.overall === 'warning' ? 'Some warnings' :
                             'Issues found'}
                        </span>
                    </div>
                    <div className="mcc-diag-checks">
                        {data.checks.map(check => (
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
                            </div>
                        ))}
                    </div>
                    <button className="mcc-diag-refresh" onClick={refreshDiagnostics}>
                        Refresh
                    </button>
                </>
            )}
        </div>
    );
}
