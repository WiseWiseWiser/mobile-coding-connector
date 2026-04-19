import { useState, useEffect } from 'react';
import { fetchTunnelGroups, restartDNS } from '../../api/ports';
import type { TunnelGroupInfo } from '../../api/ports';

export function TunnelGroupsSection() {
    const [groups, setGroups] = useState<TunnelGroupInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [restartingDNS, setRestartingDNS] = useState<Record<string, boolean>>({});
    const [dnsResults, setDnsResults] = useState<Record<string, { ok: boolean; message: string }>>({});

    useEffect(() => {
        setLoading(true);
        fetchTunnelGroups()
            .then(data => { setGroups(data); setError(null); })
            .catch(err => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    const handleRestartDNS = (hostname: string, groupName: string) => {
        const key = `${groupName}:${hostname}`;
        setRestartingDNS(prev => ({ ...prev, [key]: true }));
        setDnsResults(prev => { const next = { ...prev }; delete next[key]; return next; });

        restartDNS(hostname, groupName)
            .then(result => setDnsResults(prev => ({ ...prev, [key]: { ok: true, message: result.message } })))
            .catch(err => setDnsResults(prev => ({ ...prev, [key]: { ok: false, message: err.message } })))
            .finally(() => setRestartingDNS(prev => ({ ...prev, [key]: false })));
    };

    const refreshGroups = () => {
        setLoading(true);
        fetchTunnelGroups()
            .then(data => { setGroups(data); setError(null); })
            .catch(err => setError(err.message))
            .finally(() => setLoading(false));
    };

    if (loading) {
        return (
            <div className="mcc-tunnel-groups">
                <div className="mcc-ports-subtitle">Tunnel Groups</div>
                <div className="mcc-diag-loading">Loading tunnel groups...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="mcc-tunnel-groups">
                <div className="mcc-ports-subtitle">Tunnel Groups</div>
                <div className="mcc-ports-error">Error: {error}</div>
            </div>
        );
    }

    if (groups.length === 0) {
        return null;
    }

    return (
        <div className="mcc-tunnel-groups">
            <div className="mcc-ports-subtitle" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span>Tunnel Groups</span>
                <button className="mcc-diag-refresh" style={{ margin: 0, padding: '4px 10px', fontSize: '12px' }} onClick={refreshGroups}>
                    Refresh
                </button>
            </div>
            {groups.map(group => (
                <div key={group.name} className="mcc-tunnel-group-card">
                    <div className="mcc-tunnel-group-header">
                        <span className="mcc-port-status">{group.running ? '🟢' : '🔴'}</span>
                        <span className="mcc-tunnel-group-name">{group.name}</span>
                        <span className={`mcc-port-provider-badge ${group.running ? '' : 'mcc-badge-stopped'}`}>
                            {group.running ? 'Running' : 'Stopped'}
                        </span>
                        {group.config && (
                            <span className="mcc-tunnel-group-tunnel-name" title={`Tunnel ID: ${group.config.tunnel_id}`}>
                                {group.config.tunnel_name || group.config.tunnel_id}
                            </span>
                        )}
                    </div>
                    {group.mappings.length === 0 ? (
                        <div className="mcc-ports-empty">No mappings configured.</div>
                    ) : (
                        <div className="mcc-tunnel-mappings">
                            {group.mappings.map(mapping => {
                                const key = `${group.name}:${mapping.hostname}`;
                                const isRestarting = restartingDNS[key];
                                const result = dnsResults[key];

                                return (
                                    <div key={mapping.id} className="mcc-tunnel-mapping">
                                        <div className="mcc-tunnel-mapping-info">
                                            <a
                                                href={`https://${mapping.hostname}`}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="mcc-tunnel-mapping-hostname"
                                            >
                                                {mapping.hostname}
                                            </a>
                                            <span className="mcc-port-arrow">&rarr;</span>
                                            <span className="mcc-tunnel-mapping-service">{mapping.service}</span>
                                        </div>
                                        <div className="mcc-tunnel-mapping-actions">
                                            <button
                                                className="mcc-port-action-btn"
                                                onClick={() => handleRestartDNS(mapping.hostname, group.name)}
                                                disabled={isRestarting}
                                            >
                                                {isRestarting ? 'Restarting...' : 'Restart DNS'}
                                            </button>
                                        </div>
                                        {result && (
                                            <div className={`mcc-tunnel-dns-result ${result.ok ? 'mcc-tunnel-dns-ok' : 'mcc-tunnel-dns-error'}`}>
                                                {result.message}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            ))}
        </div>
    );
}
