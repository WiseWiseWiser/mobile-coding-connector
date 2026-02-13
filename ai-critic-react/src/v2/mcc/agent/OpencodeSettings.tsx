import { useState, useEffect } from 'react';
import {
    fetchOpencodeAuthStatus,
    fetchOpencodeConfig,
    fetchOpencodeProviders,
    fetchOpencodeSettings,
    fetchOpencodeWebStatus,
    updateAgentConfig,
    updateOpencodeSettings,
    controlOpencodeWebServerStreaming,
    unmapOpencodeDomain,
    mapOpencodeDomainStreaming,
} from '../../../api/agents';
import type { OpencodeAuthStatus, AgentSessionInfo, OpencodeSettings, OpencodeWebStatus } from '../../../api/agents';
import { fetchProviders } from '../../../api/ports';
import { AgentChatHeader } from './AgentChatHeader';
import { AgentPathSettingsSection } from './AgentPathSettingsSection';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { useReconnectingStreamingAction } from '../../../hooks/useReconnectingStreamingAction';
import { LogViewer } from '../../LogViewer';
import { ModelSelector, type ModelOption } from '../components/ModelSelector';

export interface OpencodeSettingsProps {
    agentId: string;
    session: AgentSessionInfo | null;
    projectName: string | null;
    onBack: () => void;
    onRefreshAgents?: () => void;
}

export function OpencodeSettings({ agentId, session, projectName, onBack, onRefreshAgents }: OpencodeSettingsProps) {
    const [authStatus, setAuthStatus] = useState<OpencodeAuthStatus | null>(null);
    const [webStatus, setWebStatus] = useState<OpencodeWebStatus | null>(null);

    // Settings state
    const [savedSettings, setSavedSettings] = useState<OpencodeSettings>({});
    const [defaultDomain, setDefaultDomain] = useState<string>('');
    const [password, setPassword] = useState<string>('');

    // Session model selection state
    const [savedModel, setSavedModel] = useState<{ modelID: string; providerID: string } | null>(null);
    const [selectedModel, setSelectedModel] = useState<{ modelID: string; providerID: string } | null>(null);
    const [models, setModels] = useState<ModelOption[]>([]);
    const [defaultModel, setDefaultModel] = useState<string>('');

    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');

    // Web server streaming control
    const [webServerActionState, webServerActionControls] = useStreamingAction((result) => {
        // Refresh web status when streaming completes
        if (!result.ok) {
            setError(result.message);
        } else {
            setSuccess(result.message);
        }
        setTimeout(() => {
            refreshWebStatus();
        }, 1000);
    });

    // Domain mapping state with streaming support
    const [domainMapped, setDomainMapped] = useState(false);
    const [mappedUrl, setMappedUrl] = useState<string>('');
    const [availableProviders, setAvailableProviders] = useState<Array<{ id: string; name: string; available: boolean }>>([]);

    // Domain mapping streaming action with reconnection support
    const [domainMappingState, domainMappingControls] = useReconnectingStreamingAction((result) => {
        if (result.ok) {
            setDomainMapped(true);
            if (result.publicUrl) {
                setMappedUrl(result.publicUrl);
            }
            setSuccess(result.message);
        } else {
            setError(result.message);
        }
        // Refresh web status to get updated port_mapped status
        setTimeout(() => {
            refreshWebStatus();
        }, 1000);
    }, {
        maxReconnects: 20,
        reconnectDelayMs: 2000,
    });

    const hasChanges = selectedModel?.modelID !== savedModel?.modelID || selectedModel?.providerID !== savedModel?.providerID;
    const hasSettingsChanges = defaultDomain !== (savedSettings.default_domain || '') || password !== (savedSettings.web_server?.password || '');

    useEffect(() => {
        loadAllData();
    }, [session?.id]);

    const loadAllData = async () => {
        setLoading(true);
        try {
            // Load settings and web status (always needed)
            const [settings, webStat, auth] = await Promise.all([
                fetchOpencodeSettings(),
                fetchOpencodeWebStatus(),
                fetchOpencodeAuthStatus(),
            ]);
            
            setSavedSettings(settings);
            setDefaultDomain(settings.default_domain || '');
            setPassword(settings.web_server?.password || '');
            setWebStatus(webStat);
            setAuthStatus(auth);

            // Initialize model selection from saved settings
            let savedModelKey: { modelID: string; providerID: string } | null = null;
            if (settings.model) {
                // Parse model format: "providerID/modelID" or just modelID
                const parts = settings.model.split('/');
                if (parts.length >= 2) {
                    savedModelKey = { providerID: parts[0], modelID: parts[1] };
                } else {
                    savedModelKey = { providerID: '', modelID: settings.model };
                }
            }
            setSavedModel(savedModelKey);
            setSelectedModel(savedModelKey);

            // Load available providers for domain mapping
            try {
                const providers = await fetchProviders();
                setAvailableProviders(providers.filter(p => p.available && (p.id === 'cloudflare_owned' || p.id === 'cloudflare_tunnel')));
            } catch (e) {
                // Ignore provider fetch errors
            }

            // Load session-specific data if session exists
            if (session) {
                const [config, providers] = await Promise.all([
                    fetchOpencodeConfig(session.id),
                    fetchOpencodeProviders(session.id),
                ]);

                // Use server config if no saved model
                const currentModelFromServer: { modelID: string; providerID: string } | null = config.model?.modelID
                    ? { modelID: config.model.modelID, providerID: config.model.providerID || '' }
                    : null;
                const modelToUse = savedModelKey || currentModelFromServer;

                setSavedModel(modelToUse);
                setSelectedModel(modelToUse);

                // Build ModelOption array from all providers
                const allModels: ModelOption[] = [];
                let defModel = '';
                for (const provider of providers.providers) {
                    for (const [id, model] of Object.entries(provider.models)) {
                        allModels.push({
                            id,
                            name: model.name || id,
                            providerId: provider.id,
                            providerName: provider.name || provider.id,
                            is_default: providers.default?.[provider.id] === id,
                        });
                    }
                    if (providers.default?.[provider.id]) {
                        defModel = providers.default[provider.id];
                    }
                }
                // Sort models by provider name, then by model name
                allModels.sort((a, b) => {
                    const providerCompare = a.providerName.localeCompare(b.providerName);
                    if (providerCompare !== 0) return providerCompare;
                    return a.name.localeCompare(b.name);
                });
                setModels(allModels);
                setDefaultModel(defModel);
            }
        } catch (err) {
            console.error('Failed to load settings:', err);
        } finally {
            setLoading(false);
        }
    };

    const handleSaveSessionModel = async () => {
        if (!session || !selectedModel) return;
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateAgentConfig(session.id, { model: { modelID: selectedModel.modelID } });
            setSavedModel(selectedModel);
            // Also update savedSettings to reflect the new preferred model
            setSavedSettings({
                ...savedSettings,
                model: `${selectedModel.providerID}/${selectedModel.modelID}`,
            });
            setSuccess('Model updated successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update model');
        } finally {
            setSaving(false);
        }
    };

    const handleSaveSettings = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            const currentConfig = savedSettings.web_server || { enabled: false, port: 4096 };
            await updateOpencodeSettings({
                ...savedSettings,
                default_domain: defaultDomain,
                web_server: {
                    enabled: currentConfig.enabled,
                    port: currentConfig.port,
                    exposed_domain: currentConfig.exposed_domain,
                    password: password,
                },
            });
            setSavedSettings({
                ...savedSettings,
                default_domain: defaultDomain,
                web_server: {
                    enabled: currentConfig.enabled,
                    port: currentConfig.port,
                    exposed_domain: currentConfig.exposed_domain,
                    password: password,
                },
            });
            setSuccess('Settings saved successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    const handleCancel = () => {
        setSelectedModel(savedModel ? { ...savedModel } : null);
        setDefaultDomain(savedSettings.default_domain || '');
        setPassword(savedSettings.web_server?.password || '');
        setError('');
        setSuccess('');
    };

    const refreshWebStatus = async () => {
        try {
            const status = await fetchOpencodeWebStatus();
            setWebStatus(status);
        } catch (err) {
            console.error('Failed to refresh web status:', err);
        }
    };

    const handleWebServerControl = async (action: 'start' | 'stop') => {
        setError('');
        setSuccess('');
        await webServerActionControls.run(() => controlOpencodeWebServerStreaming(action));
    };

    const handleMapDomain = async () => {
        setError('');
        setSuccess('');
        // Use streaming with automatic reconnection
        await domainMappingControls.run((sessionId, logIndex) => mapOpencodeDomainStreaming(undefined, sessionId, logIndex));
    };

    const handleUnmapDomain = async () => {
        setError('');
        setSuccess('');
        try {
            const resp = await unmapOpencodeDomain();
            if (resp.success) {
                setDomainMapped(false);
                setMappedUrl('');
                setSuccess(resp.message);
            } else {
                setError(resp.message);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to unmap domain');
        }
    };

    return (
        <div className="mcc-agent-view">
            {session ? (
                <AgentChatHeader agentName={session.agent_name} projectName={projectName} onBack={onBack} />
            ) : (
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                    <h2>OpenCode Settings</h2>
                </div>
            )}
            <div className="mcc-agent-header" style={{ paddingTop: 4 }}>
                <h2>Settings</h2>
            </div>

            {loading ? (
                <div className="mcc-agent-loading">Loading settings...</div>
            ) : (
                <div className="mcc-agent-settings-form">
                    {/* Binary Path Settings (always shown) */}
                    <div style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
                        <AgentPathSettingsSection agentId={agentId} onRefreshAgents={onRefreshAgents} />
                    </div>

                    {/* Login Status (always shown) */}
                    <div className="mcc-agent-settings-field" style={{ marginBottom: 20 }}>
                        <label className="mcc-agent-settings-label">
                            Login Status
                        </label>
                        <div style={{
                            padding: '12px 14px',
                            background: authStatus?.authenticated ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                            border: `1px solid ${authStatus?.authenticated ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                            borderRadius: 8,
                        }}>
                            <div style={{
                                color: authStatus?.authenticated ? '#86efac' : '#fca5a5',
                                fontWeight: 600,
                                marginBottom: authStatus?.providers?.length ? 8 : 0,
                            }}>
                                {authStatus?.authenticated ? '✓ Authenticated' : '✗ Not authenticated'}
                            </div>
                            {authStatus?.providers && authStatus.providers.length > 0 && (
                                <div style={{ fontSize: '13px', color: '#94a3b8' }}>
                                    <strong>Providers:</strong>
                                    <ul style={{ margin: '8px 0 0 0', paddingLeft: 20 }}>
                                        {authStatus.providers.map(p => (
                                            <li key={p.name} style={{ marginBottom: 4 }}>
                                                {p.name}
                                                {p.has_api_key && <span style={{ color: '#86efac', marginLeft: 8 }}>(configured)</span>}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}
                            {!authStatus?.authenticated && (
                                <div style={{ fontSize: '13px', color: '#94a3b8', marginTop: 8 }}>
                                    Run <code style={{ background: '#1e293b', padding: '2px 6px', borderRadius: 4 }}>opencode auth login</code> to authenticate.
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Web Server Status (always shown) */}
                    <div className="mcc-agent-settings-field" style={{ marginBottom: 20 }}>
                        <label className="mcc-agent-settings-label">
                            Web Server Status
                            <button 
                                onClick={refreshWebStatus}
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
                                {/* Start/Stop Control Buttons */}
                                <div style={{ display: 'flex', gap: 8 }}>
                                    {!webStatus?.running ? (
                                        <button
                                            onClick={() => handleWebServerControl('start')}
                                            disabled={webServerActionState.running}
                                            style={{
                                                padding: '4px 12px',
                                                fontSize: '12px',
                                                background: '#22c55e',
                                                border: 'none',
                                                borderRadius: 4,
                                                color: '#fff',
                                                fontWeight: 500,
                                                cursor: webServerActionState.running ? 'not-allowed' : 'pointer',
                                                opacity: webServerActionState.running ? 0.6 : 1,
                                            }}
                                        >
                                            {webServerActionState.running ? 'Starting...' : 'Start'}
                                        </button>
                                    ) : (
                                        <button
                                            onClick={() => handleWebServerControl('stop')}
                                            disabled={webServerActionState.running}
                                            style={{
                                                padding: '4px 12px',
                                                fontSize: '12px',
                                                background: '#ef4444',
                                                border: 'none',
                                                borderRadius: 4,
                                                color: '#fff',
                                                fontWeight: 500,
                                                cursor: webServerActionState.running ? 'not-allowed' : 'pointer',
                                                opacity: webServerActionState.running ? 0.6 : 1,
                                            }}
                                        >
                                            {webServerActionState.running ? 'Stopping...' : 'Stop'}
                                        </button>
                                    )}
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
                            
                            {/* Streaming Logs */}
                            {webServerActionState.showLogs && (
                                <div style={{ marginTop: 12 }}>
                                    <LogViewer 
                                        lines={webServerActionState.logs} 
                                        maxHeight={200}
                                    />
                                    {webServerActionState.result && (
                                        <div style={{ 
                                            marginTop: 8,
                                            padding: '8px 12px',
                                            borderRadius: 4,
                                            fontSize: '13px',
                                            background: webServerActionState.result.ok ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                                            border: `1px solid ${webServerActionState.result.ok ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                                            color: webServerActionState.result.ok ? '#86efac' : '#fca5a5',
                                        }}>
                                            {webServerActionState.result.ok ? '✓ ' : '✗ '}{webServerActionState.result.message}
                                        </div>
                                    )}
                                </div>
                            )}

                            {/* Server Password Input */}
                            <div style={{ marginTop: 16 }}>
                                <label className="mcc-agent-settings-label">
                                    Server Password (Optional)
                                </label>
                                <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                                    Password to protect the OpenCode web server with HTTP basic auth
                                </div>
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    placeholder="Enter password..."
                                    disabled={saving}
                                    style={{
                                        width: '100%',
                                        padding: '10px 12px',
                                        background: '#1e293b',
                                        border: password !== (savedSettings.web_server?.password || '') ? '1px solid #3b82f6' : '1px solid #334155',
                                        borderRadius: 8,
                                        color: '#e2e8f0',
                                        fontSize: '14px',
                                    }}
                                />
                                {savedSettings.web_server?.password && savedSettings.web_server.password !== password && (
                                    <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                        Password is saved (hidden for security)
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Configuration Settings (always shown) */}
                    <div style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
                        <h3 style={{ margin: '0 0 16px 0', color: '#e2e8f0', fontSize: '16px' }}>Configuration</h3>

                        {/* Default Domain Input */}
                        <div className="mcc-agent-settings-field" style={{ marginBottom: 16 }}>
                            <label className="mcc-agent-settings-label">
                                Default Domain For Web
                            </label>
                            <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                                Domain to map the OpenCode web server port (e.g., "your-domain.com")
                            </div>
                            <input
                                type="text"
                                value={defaultDomain}
                                onChange={(e) => setDefaultDomain(e.target.value)}
                                placeholder="Enter domain..."
                                disabled={saving}
                                style={{
                                    width: '100%',
                                    padding: '10px 12px',
                                    background: '#1e293b',
                                    border: defaultDomain !== (savedSettings.default_domain || '') ? '1px solid #3b82f6' : '1px solid #334155',
                                    borderRadius: 8,
                                    color: '#e2e8f0',
                                    fontSize: '14px',
                                }}
                            />
                            {savedSettings.default_domain && savedSettings.default_domain !== defaultDomain && (
                                <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                    Saved: <strong style={{ color: '#e2e8f0' }}>{savedSettings.default_domain}</strong>
                                </div>
                            )}

                            {/* Domain Mapping Section - Only show when domain is configured and matches owned domains */}
                            {savedSettings.default_domain && availableProviders.length > 0 && (
                                <div style={{ marginTop: 16, padding: '12px', background: 'rgba(59, 130, 246, 0.05)', borderRadius: 8, border: '1px solid rgba(59, 130, 246, 0.2)' }}>
                                    <div style={{ fontSize: '13px', color: '#94a3b8', marginBottom: 8 }}>
                                        <strong style={{ color: '#60a5fa' }}>Domain Mapping Available</strong>
                                        <div style={{ marginTop: 4 }}>
                                            Your domain <strong style={{ color: '#e2e8f0' }}>{savedSettings.default_domain}</strong> can be mapped via Cloudflare.
                                        </div>
                                    </div>

                                    {domainMapped || webStatus?.port_mapped ? (
                                        <div>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                                                <span style={{ color: '#86efac', fontSize: '13px' }}>✓ Domain mapped</span>
                                                {mappedUrl && (
                                                    <a
                                                        href={mappedUrl}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        style={{
                                                            fontSize: '13px',
                                                            color: '#60a5fa',
                                                            textDecoration: 'underline',
                                                        }}
                                                    >
                                                        {mappedUrl}
                                                    </a>
                                                )}
                                            </div>
                                            <button
                                                onClick={handleUnmapDomain}
                                                disabled={domainMappingState.running}
                                                style={{
                                                    padding: '6px 12px',
                                                    fontSize: '12px',
                                                    background: 'transparent',
                                                    border: '1px solid #ef4444',
                                                    borderRadius: 4,
                                                    color: '#ef4444',
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
                                                onClick={handleMapDomain}
                                                disabled={domainMappingState.running || !webStatus?.running}
                                                style={{
                                                    padding: '6px 12px',
                                                    fontSize: '12px',
                                                    background: webStatus?.running ? '#3b82f6' : '#475569',
                                                    border: 'none',
                                                    borderRadius: 4,
                                                    color: '#fff',
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

                                            {/* Streaming Logs for Domain Mapping */}
                                            {domainMappingState.showLogs && domainMappingState.logs.length > 0 && (
                                                <div style={{ marginTop: 12 }}>
                                                    <LogViewer
                                                        lines={domainMappingState.logs}
                                                        maxHeight={200}
                                                    />
                                                    {domainMappingState.result && (
                                                        <div style={{
                                                            marginTop: 8,
                                                            padding: '8px 12px',
                                                            borderRadius: 4,
                                                            fontSize: '13px',
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

                        {/* Save/Cancel Buttons for Settings */}
                        {hasSettingsChanges && (
                            <div style={{ 
                                marginTop: 16, 
                                display: 'flex', 
                                gap: 12,
                                padding: '12px',
                                background: 'rgba(59, 130, 246, 0.1)',
                                borderRadius: 8,
                                border: '1px solid rgba(59, 130, 246, 0.3)',
                            }}>
                                <button
                                    onClick={handleSaveSettings}
                                    disabled={saving}
                                    style={{
                                        flex: 1,
                                        padding: '10px 16px',
                                        background: '#3b82f6',
                                        opacity: saving ? 0.7 : 1,
                                        border: 'none',
                                        borderRadius: 6,
                                        color: '#fff',
                                        fontSize: '14px',
                                        fontWeight: 500,
                                        cursor: saving ? 'not-allowed' : 'pointer',
                                    }}
                                >
                                    {saving ? 'Saving...' : 'Save Settings'}
                                </button>
                                <button
                                    onClick={handleCancel}
                                    disabled={saving}
                                    style={{
                                        flex: 1,
                                        padding: '10px 16px',
                                        background: 'transparent',
                                        border: '1px solid #475569',
                                        borderRadius: 6,
                                        color: '#94a3b8',
                                        fontSize: '14px',
                                        fontWeight: 500,
                                        cursor: saving ? 'not-allowed' : 'pointer',
                                    }}
                                >
                                    Cancel
                                </button>
                            </div>
                        )}
                    </div>

                    {/* Preferred Model Selection (always shown) */}
                    <div className="mcc-agent-settings-field">
                        <label className="mcc-agent-settings-label">
                            Preferred Model
                        </label>
                        <div className="mcc-agent-settings-hint" style={{ marginBottom: 8 }}>
                            Select the AI model to use for this session.
                        </div>
                        <div style={{
                            border: hasChanges ? '1px solid #3b82f6' : '1px solid #334155',
                            borderRadius: 8,
                            padding: '2px',
                        }}>
                            <ModelSelector
                                models={models}
                                currentModel={selectedModel || undefined}
                                onSelect={(model) => setSelectedModel(model)}
                                placeholder={models.length === 0 ? "Start a session to see available models" : "Select a model..."}
                                disabled={saving || models.length === 0}
                            />
                        </div>
                        {savedModel && models.length > 0 && (
                            <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                Saved: <strong style={{ color: '#e2e8f0' }}>
                                    {models.find(m => m.id === savedModel.modelID && m.providerId === savedModel.providerID)?.name || savedModel.modelID}
                                </strong>
                                {savedModel.modelID === defaultModel && <span style={{ color: '#64748b', marginLeft: 4 }}>(default)</span>}
                            </div>
                        )}
                    </div>

                    {/* Save/Cancel Buttons for Preferred Model */}
                    {hasChanges && session && (
                        <div style={{ 
                            marginTop: 16, 
                            display: 'flex', 
                            gap: 12,
                            padding: '12px',
                            background: 'rgba(59, 130, 246, 0.1)',
                            borderRadius: 8,
                            border: '1px solid rgba(59, 130, 246, 0.3)',
                        }}>
                            <button
                                onClick={handleSaveSessionModel}
                                disabled={saving}
                                style={{
                                    flex: 1,
                                    padding: '10px 16px',
                                    background: '#3b82f6',
                                    opacity: saving ? 0.7 : 1,
                                    border: 'none',
                                    borderRadius: 6,
                                    color: '#fff',
                                    fontSize: '14px',
                                    fontWeight: 500,
                                    cursor: saving ? 'not-allowed' : 'pointer',
                                }}
                            >
                                {saving ? 'Saving...' : 'Save'}
                            </button>
                            <button
                                onClick={handleCancel}
                                disabled={saving}
                                style={{
                                    flex: 1,
                                    padding: '10px 16px',
                                    background: 'transparent',
                                    border: '1px solid #475569',
                                    borderRadius: 6,
                                    color: '#94a3b8',
                                    fontSize: '14px',
                                    fontWeight: 500,
                                    cursor: saving ? 'not-allowed' : 'pointer',
                                }}
                            >
                                Cancel
                            </button>
                        </div>
                    )}

                    {error && (
                        <div className="mcc-agent-settings-message mcc-agent-settings-error" style={{
                            marginTop: 12,
                            padding: '10px 14px',
                            background: 'rgba(239, 68, 68, 0.1)',
                            border: '1px solid rgba(239, 68, 68, 0.3)',
                            borderRadius: 8,
                            color: '#fca5a5',
                            fontSize: '13px',
                        }}>
                            {error}
                        </div>
                    )}
                    {success && (
                        <div className="mcc-agent-settings-message mcc-agent-settings-success" style={{
                            marginTop: 12,
                            padding: '10px 14px',
                            background: 'rgba(34, 197, 94, 0.1)',
                            border: '1px solid rgba(34, 197, 94, 0.3)',
                            borderRadius: 8,
                            color: '#86efac',
                            fontSize: '13px',
                        }}>
                            {success}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
