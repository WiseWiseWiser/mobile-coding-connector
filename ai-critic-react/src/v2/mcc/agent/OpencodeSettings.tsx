import { useState, useEffect } from 'react';
import { 
    fetchOpencodeAuthStatus, 
    fetchOpencodeConfig, 
    fetchOpencodeProviders, 
    fetchOpencodeSettings, 
    fetchOpencodeWebStatus,
    updateAgentConfig,
    updateOpencodeSettings 
} from '../../../api/agents';
import type { OpencodeAuthStatus, AgentSessionInfo, OpencodeModelInfo, OpencodeSettings, OpencodeWebStatus } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { AgentPathSettingsSection } from './AgentPathSettingsSection';

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

    // Session model selection state
    const [savedModel, setSavedModel] = useState<string>('');
    const [selectedModel, setSelectedModel] = useState<string>('');
    const [models, setModels] = useState<Record<string, OpencodeModelInfo>>({});
    const [defaultModel, setDefaultModel] = useState<string>('');

    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');

    const hasChanges = selectedModel !== savedModel;
    const hasSettingsChanges = defaultDomain !== (savedSettings.default_domain || '');

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
            setWebStatus(webStat);
            setAuthStatus(auth);

            // Initialize model selection from saved settings
            const savedModelFromSettings = settings.model || '';
            setSavedModel(savedModelFromSettings);
            setSelectedModel(savedModelFromSettings);

            // Load session-specific data if session exists
            if (session) {
                const [config, providers] = await Promise.all([
                    fetchOpencodeConfig(session.id),
                    fetchOpencodeProviders(session.id),
                ]);

                // Use server config if no saved model
                const currentModelFromServer = config.model?.modelID || '';
                const modelToUse = savedModelFromSettings || currentModelFromServer;

                setSavedModel(modelToUse);
                setSelectedModel(modelToUse);

                // Combine all models from all providers
                const allModels: Record<string, OpencodeModelInfo> = {};
                let defModel = '';
                for (const provider of providers.providers) {
                    for (const [id, model] of Object.entries(provider.models)) {
                        allModels[id] = model;
                    }
                    if (providers.default[provider.id]) {
                        defModel = providers.default[provider.id];
                    }
                }
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
        if (!session) return;
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateAgentConfig(session.id, { model: { modelID: selectedModel } });
            setSavedModel(selectedModel);
            // Also update savedSettings to reflect the new preferred model
            setSavedSettings({
                ...savedSettings,
                model: selectedModel,
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
            await updateOpencodeSettings({
                ...savedSettings,
                default_domain: defaultDomain,
            });
            setSavedSettings({
                ...savedSettings,
                default_domain: defaultDomain,
            });
            setSuccess('Settings saved successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    const handleCancel = () => {
        setSelectedModel(savedModel);
        setDefaultDomain(savedSettings.default_domain || '');
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
                            }}>
                                {webStatus?.running ? '✓ Running' : '✗ Not running'}
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
                        <select
                            value={selectedModel || defaultModel}
                            onChange={(e) => setSelectedModel(e.target.value)}
                            disabled={saving || Object.keys(models).length === 0}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                background: '#1e293b',
                                border: hasChanges ? '1px solid #3b82f6' : '1px solid #334155',
                                borderRadius: 8,
                                color: '#e2e8f0',
                                fontSize: '14px',
                                cursor: Object.keys(models).length === 0 ? 'not-allowed' : 'pointer',
                                opacity: Object.keys(models).length === 0 ? 0.6 : 1,
                            }}
                        >
                            {Object.keys(models).length === 0 ? (
                                <option value="">Start a session to see available models</option>
                            ) : (
                                Object.entries(models).map(([id, model]) => (
                                    <option key={id} value={id}>
                                        {model.name || id}
                                        {id === defaultModel ? ' (default)' : ''}
                                        {id === savedModel ? ' (saved)' : ''}
                                    </option>
                                ))
                            )}
                        </select>
                        {savedModel && savedModel !== defaultModel && Object.keys(models).length > 0 && (
                            <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                Saved: <strong style={{ color: '#e2e8f0' }}>{models[savedModel]?.name || savedModel}</strong>
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
