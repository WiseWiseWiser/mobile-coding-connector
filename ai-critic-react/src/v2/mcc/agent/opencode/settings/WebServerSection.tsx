import type { StreamingActionState } from '../../../../../hooks/useStreamingAction';
import type { OpencodeWebStatus } from '../../../../../api/agents';
import { LogViewer } from '../../../../LogViewer';

export interface WebServerSectionProps {
    webStatus: OpencodeWebStatus | null;
    webServerEnabled: boolean;
    webServerPort: number;
    password: string;
    authProxyEnabled: boolean;
    saving: boolean;
    savedWebServer?: { port?: number; password?: string };
    actionState: StreamingActionState;
    onRefresh: () => void;
    onControl: (action: 'start' | 'stop') => void;
    onEnabledChange: (enabled: boolean) => void;
    onPortChange: (port: number) => void;
    onPasswordChange: (password: string) => void;
    onAuthProxyChange: (enabled: boolean) => void;
}

export function WebServerSection({
    webStatus,
    webServerEnabled,
    webServerPort,
    password,
    authProxyEnabled,
    saving,
    savedWebServer,
    actionState,
    onRefresh,
    onControl,
    onEnabledChange,
    onPortChange,
    onPasswordChange,
    onAuthProxyChange,
}: WebServerSectionProps) {
    return (
        <div className="mcc-agent-settings-field" style={{ marginBottom: 20 }}>
            <label className="mcc-agent-settings-label">
                Web Server Status
                <button 
                    onClick={onRefresh}
                    style={{
                        marginLeft: 8,
                        padding: '2px 8px',
                        fontSize: '12px',
                        background: 'transparent',
                        border: '1px solid #475569',
                        borderRadius: 4,
                        color: '#94a3b8',
                        cursor: 'pointer',
                    }}
                >
                    Refresh
                </button>
            </label>
            <div style={{
                padding: '12px 14px',
                background: webStatus?.running ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                border: `1px solid ${webStatus?.running ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                borderRadius: 8,
            }}>
                <div style={{
                    color: webStatus?.running ? '#86efac' : '#fca5a5',
                    fontWeight: 600,
                    marginBottom: 8,
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                }}>
                    <span>{webStatus?.running ? '✓ Running' : '✗ Not running'}</span>
                    <div style={{ display: 'flex', gap: 8 }}>
                        {!webStatus?.running ? (
                            <button
                                onClick={() => onControl('start')}
                                disabled={actionState.running}
                                style={{
                                    padding: '4px 12px',
                                    fontSize: '12px',
                                    background: '#22c55e',
                                    border: 'none',
                                    borderRadius: 4,
                                    color: '#fff',
                                    fontWeight: 500,
                                    cursor: actionState.running ? 'not-allowed' : 'pointer',
                                    opacity: actionState.running ? 0.6 : 1,
                                }}
                            >
                                {actionState.running ? 'Starting...' : 'Start'}
                            </button>
                        ) : (
                            <button
                                onClick={() => onControl('stop')}
                                disabled={actionState.running}
                                style={{
                                    padding: '4px 12px',
                                    fontSize: '12px',
                                    background: '#ef4444',
                                    border: 'none',
                                    borderRadius: 4,
                                    color: '#fff',
                                    fontWeight: 500,
                                    cursor: actionState.running ? 'not-allowed' : 'pointer',
                                    opacity: actionState.running ? 0.6 : 1,
                                }}
                            >
                                {actionState.running ? 'Stopping...' : 'Stop'}
                            </button>
                        )}
                    </div>
                </div>

                <div style={{ marginBottom: 12, marginTop: 8 }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                        <input
                            type="checkbox"
                            checked={webServerEnabled}
                            onChange={(e) => onEnabledChange(e.target.checked)}
                            style={{ width: 18, height: 18 }}
                        />
                        <span style={{ fontWeight: 500 }}>Enable Web Server</span>
                    </label>
                    <div style={{ fontSize: '12px', color: '#94a3b8', marginTop: 4, marginLeft: 26 }}>
                        When enabled, the server will auto-start on boot if configured
                    </div>
                </div>

                <div style={{ fontSize: '13px', color: '#94a3b8' }}>
                    <div>Port: <strong style={{ color: '#e2e8f0' }}>{webStatus?.port || 'N/A'}</strong></div>
                    {webStatus?.domain && (
                        <div style={{ marginTop: 4 }}>
                            Domain: <strong style={{ color: '#e2e8f0' }}>{webStatus.domain}</strong>
                        </div>
                    )}
                    {webStatus?.domain && (
                        <div style={{ marginTop: 4 }}>
                            Port Mapped: 
                            <strong style={{ color: webStatus.port_mapped ? '#86efac' : '#fca5a5', marginLeft: 4 }}>
                                {webStatus.port_mapped ? '✓ Yes' : '✗ No'}
                            </strong>
                        </div>
                    )}
                </div>
                
                {actionState.showLogs && (
                    <div style={{ marginTop: 12 }}>
                        <LogViewer 
                            lines={actionState.logs} 
                            maxHeight={200}
                        />
                        {actionState.result && (
                            <div style={{ 
                                marginTop: 8,
                                padding: '8px 12px',
                                borderRadius: 4,
                                fontSize: '13px',
                                background: actionState.result.ok ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                                border: `1px solid ${actionState.result.ok ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                                color: actionState.result.ok ? '#86efac' : '#fca5a5',
                            }}>
                                {actionState.result.ok ? '✓ ' : '✗ '}{actionState.result.message}
                            </div>
                        )}
                    </div>
                )}

                <div style={{ marginTop: 16 }}>
                    <label className="mcc-agent-settings-label">Web Server Port</label>
                    <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                        Port for the OpenCode web server (default: 4096). Changing this will apply on next server restart.
                    </div>
                    <input
                        type="number"
                        value={webServerPort}
                        onChange={(e) => onPortChange(parseInt(e.target.value, 10) || 4096)}
                        min={1024}
                        max={65535}
                        disabled={saving}
                        style={{
                            width: '100%',
                            padding: '10px 12px',
                            background: '#1e293b',
                            border: webServerPort !== (savedWebServer?.port || 4096) ? '1px solid #3b82f6' : '1px solid #334155',
                            borderRadius: 8,
                            color: '#e2e8f0',
                            fontSize: '14px',
                        }}
                    />
                </div>

                <div style={{ marginTop: 16 }}>
                    <label className="mcc-agent-settings-label">Server Password (Optional)</label>
                    <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                        Password to protect the OpenCode web server with HTTP basic auth
                    </div>
                    <input
                        type="password"
                        value={password}
                        onChange={(e) => onPasswordChange(e.target.value)}
                        placeholder="Enter password..."
                        disabled={saving}
                        style={{
                            width: '100%',
                            padding: '10px 12px',
                            background: '#1e293b',
                            border: password !== (savedWebServer?.password || '') ? '1px solid #3b82f6' : '1px solid #334155',
                            borderRadius: 8,
                            color: '#e2e8f0',
                            fontSize: '14px',
                        }}
                    />
                    {savedWebServer?.password && savedWebServer.password !== password && (
                        <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                            Password is saved (hidden for security)
                        </div>
                    )}
                </div>

                <div style={{ marginTop: 16 }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}>
                        <input
                            type="checkbox"
                            checked={authProxyEnabled}
                            onChange={(e) => onAuthProxyChange(e.target.checked)}
                            disabled={saving}
                            style={{ width: 18, height: 18, cursor: saving ? 'not-allowed' : 'pointer' }}
                        />
                        <div>
                            <div className="mcc-agent-settings-label" style={{ marginBottom: 4 }}>Enable Auth Proxy</div>
                            <div className="mcc-agent-settings-hint" style={{ fontSize: '13px', color: '#94a3b8' }}>
                                Replace browser basic auth popup with a login page. Requires basic-auth-proxy binary in PATH.
                            </div>
                        </div>
                    </label>
                    {webStatus && (
                        <div style={{ marginTop: 8, marginLeft: 30, fontSize: '13px', color: webStatus.auth_proxy_found ? '#86efac' : '#f87171' }}>
                            Binary: {webStatus.auth_proxy_found ? webStatus.auth_proxy_path : 'Not Found'} | 
                            Running: {webStatus.auth_proxy_running ? 'Yes' : 'No'}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
