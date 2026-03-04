import type { ReconnectingActionState } from '../../../../../hooks/useReconnectingStreamingAction';
import type { OpencodeWebStatus } from '../../../../../api/agents';
import { LogViewer } from '../../../../LogViewer';

export interface DomainConfigSectionProps {
    defaultDomain: string;
    savedDefaultDomain: string;
    saving: boolean;
    hasSettingsChanges: boolean;
    webStatus: OpencodeWebStatus | null;
    availableProviders: Array<{ id: string; name: string; available: boolean }>;
    domainMapped: boolean;
    mappedUrl: string;
    domainMappingState: ReconnectingActionState;
    onDomainChange: (domain: string) => void;
    onMapDomain: () => void;
    onUnmapDomain: () => void;
    onSaveSettings: () => void;
    onCancel: () => void;
}

export function DomainConfigSection({
    defaultDomain,
    savedDefaultDomain,
    saving,
    hasSettingsChanges,
    webStatus,
    availableProviders,
    domainMapped,
    mappedUrl,
    domainMappingState,
    onDomainChange,
    onMapDomain,
    onUnmapDomain,
    onSaveSettings,
    onCancel,
}: DomainConfigSectionProps) {
    return (
        <div style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
            <h3 style={{ margin: '0 0 16px 0', color: '#e2e8f0', fontSize: '16px' }}>Configuration</h3>

            <div className="mcc-agent-settings-field" style={{ marginBottom: 16 }}>
                <label className="mcc-agent-settings-label">Default Domain For Web</label>
                <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                    Domain to map the OpenCode web server port (e.g., "your-domain.com")
                </div>
                <input
                    type="text"
                    value={defaultDomain}
                    onChange={(e) => onDomainChange(e.target.value)}
                    placeholder="Enter domain..."
                    disabled={saving}
                    style={{
                        width: '100%',
                        padding: '10px 12px',
                        background: '#1e293b',
                        border: defaultDomain !== savedDefaultDomain ? '1px solid #3b82f6' : '1px solid #334155',
                        borderRadius: 8,
                        color: '#e2e8f0',
                        fontSize: '14px',
                    }}
                />
                {savedDefaultDomain && savedDefaultDomain !== defaultDomain && (
                    <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                        Saved: <strong style={{ color: '#e2e8f0' }}>{savedDefaultDomain}</strong>
                    </div>
                )}

                {savedDefaultDomain && availableProviders.length > 0 && (
                    <div style={{ marginTop: 16, padding: '12px', background: 'rgba(59, 130, 246, 0.05)', borderRadius: 8, border: '1px solid rgba(59, 130, 246, 0.2)' }}>
                        <div style={{ fontSize: '13px', color: '#94a3b8', marginBottom: 8 }}>
                            <strong style={{ color: '#60a5fa' }}>Domain Mapping Available</strong>
                            <div style={{ marginTop: 4 }}>
                                Your domain <strong style={{ color: '#e2e8f0' }}>{savedDefaultDomain}</strong> can be mapped via Cloudflare.
                            </div>
                        </div>

                        {domainMapped || webStatus?.port_mapped ? (
                            <div>
                                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                                    <span style={{ color: '#86efac', fontSize: '13px' }}>✓ Domain mapped</span>
                                    {mappedUrl && (
                                        <a href={mappedUrl} target="_blank" rel="noopener noreferrer"
                                            style={{ fontSize: '13px', color: '#60a5fa', textDecoration: 'underline' }}>
                                            {mappedUrl}
                                        </a>
                                    )}
                                </div>
                                <button
                                    onClick={onUnmapDomain}
                                    disabled={domainMappingState.running}
                                    style={{
                                        padding: '6px 12px', fontSize: '12px', background: 'transparent',
                                        border: '1px solid #ef4444', borderRadius: 4, color: '#ef4444',
                                        cursor: domainMappingState.running ? 'not-allowed' : 'pointer',
                                        opacity: domainMappingState.running ? 0.6 : 1,
                                    }}
                                >
                                    {domainMappingState.running ? 'Removing...' : 'Remove Mapping'}
                                </button>
                            </div>
                        ) : (
                            <div>
                                <button
                                    onClick={onMapDomain}
                                    disabled={domainMappingState.running || !webStatus?.running}
                                    style={{
                                        padding: '6px 12px', fontSize: '12px',
                                        background: webStatus?.running ? '#3b82f6' : '#475569',
                                        border: 'none', borderRadius: 4, color: '#fff',
                                        cursor: (domainMappingState.running || !webStatus?.running) ? 'not-allowed' : 'pointer',
                                        opacity: (domainMappingState.running || !webStatus?.running) ? 0.6 : 1,
                                    }}
                                >
                                    {domainMappingState.running
                                        ? (domainMappingState.reconnecting
                                            ? `Reconnecting... (${domainMappingState.reconnectionCount})`
                                            : 'Mapping...')
                                        : webStatus?.running
                                            ? 'Map Domain via Cloudflare'
                                            : 'Start Web Server to Map Domain'}
                                </button>

                                {domainMappingState.showLogs && domainMappingState.logs.length > 0 && (
                                    <div style={{ marginTop: 12 }}>
                                        <LogViewer lines={domainMappingState.logs} maxHeight={200} />
                                        {domainMappingState.result && (
                                            <div style={{
                                                marginTop: 8, padding: '8px 12px', borderRadius: 4, fontSize: '13px',
                                                background: domainMappingState.result.ok ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                                                border: `1px solid ${domainMappingState.result.ok ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                                                color: domainMappingState.result.ok ? '#86efac' : '#fca5a5',
                                            }}>
                                                {domainMappingState.result.ok ? '✓ ' : '✗ '}{domainMappingState.result.message}
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                )}
            </div>

            {hasSettingsChanges && (
                <div style={{ 
                    marginTop: 16, display: 'flex', gap: 12, padding: '12px',
                    background: 'rgba(59, 130, 246, 0.1)', borderRadius: 8,
                    border: '1px solid rgba(59, 130, 246, 0.3)',
                }}>
                    <button onClick={onSaveSettings} disabled={saving}
                        style={{
                            flex: 1, padding: '10px 16px', background: '#3b82f6',
                            opacity: saving ? 0.7 : 1, border: 'none', borderRadius: 6,
                            color: '#fff', fontSize: '14px', fontWeight: 500,
                            cursor: saving ? 'not-allowed' : 'pointer',
                        }}>
                        {saving ? 'Saving...' : 'Save Settings'}
                    </button>
                    <button onClick={onCancel} disabled={saving}
                        style={{
                            flex: 1, padding: '10px 16px', background: 'transparent',
                            border: '1px solid #475569', borderRadius: 6,
                            color: '#94a3b8', fontSize: '14px', fontWeight: 500,
                            cursor: saving ? 'not-allowed' : 'pointer',
                        }}>
                        Cancel
                    </button>
                </div>
            )}
        </div>
    );
}
