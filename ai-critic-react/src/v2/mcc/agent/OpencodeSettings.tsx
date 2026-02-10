import { useState, useEffect } from 'react';
import { fetchOpencodeAuthStatus, fetchOpencodeConfig, fetchOpencodeProviders, fetchOpencodeSettings, updateAgentConfig } from '../../../api/agents';
import type { OpencodeAuthStatus, AgentSessionInfo, OpencodeModelInfo } from '../../../api/agents';
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
    const [savedModel, setSavedModel] = useState<string>('');
    const [selectedModel, setSelectedModel] = useState<string>('');
    const [models, setModels] = useState<Record<string, OpencodeModelInfo>>({});
    const [defaultModel, setDefaultModel] = useState<string>('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');

    const hasChanges = selectedModel !== savedModel;

    useEffect(() => {
        // If no session, only load auth status (login/model require a session)
        if (!session) {
            fetchOpencodeAuthStatus().then(auth => {
                setAuthStatus(auth);
                setLoading(false);
            }).catch(() => setLoading(false));
            return;
        }

        Promise.all([
            fetchOpencodeAuthStatus(),
            fetchOpencodeConfig(session.id),
            fetchOpencodeProviders(session.id),
            fetchOpencodeSettings(),
        ]).then(([auth, config, providers, settings]) => {
            setAuthStatus(auth);
            
            // Use our saved settings as the source of truth for the model
            // Fall back to the opencode server's current config if no saved model
            const savedModelFromSettings = settings.model || '';
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
            setLoading(false);
        }).catch(() => setLoading(false));
    }, [session?.id]);

    const handleSave = async () => {
        if (!session) return;
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateAgentConfig(session.id, { model: { modelID: selectedModel } });
            setSavedModel(selectedModel);
            setSuccess('Model updated successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update model');
        } finally {
            setSaving(false);
        }
    };

    const handleCancel = () => {
        setSelectedModel(savedModel);
        setError('');
        setSuccess('');
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

                    {/* Preferred Model (only when session exists) */}
                    {session && Object.keys(models).length > 0 && (
                        <>
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
                                    disabled={saving}
                                    style={{
                                        width: '100%',
                                        padding: '10px 12px',
                                        background: '#1e293b',
                                        border: hasChanges ? '1px solid #3b82f6' : '1px solid #334155',
                                        borderRadius: 8,
                                        color: '#e2e8f0',
                                        fontSize: '14px',
                                        cursor: 'pointer',
                                    }}
                                >
                                    {Object.entries(models).map(([id, model]) => (
                                        <option key={id} value={id}>
                                            {model.name || id}
                                            {id === defaultModel ? ' (default)' : ''}
                                            {id === savedModel ? ' (saved)' : ''}
                                        </option>
                                    ))}
                                </select>
                                {savedModel && savedModel !== defaultModel && (
                                    <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                        Saved: <strong style={{ color: '#e2e8f0' }}>{models[savedModel]?.name || savedModel}</strong>
                                    </div>
                                )}
                            </div>

                            {/* Save/Cancel Buttons */}
                            {hasChanges && (
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
                                        onClick={handleSave}
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
                        </>
                    )}

                    {!session && (
                        <div className="mcc-agent-settings-hint" style={{ marginTop: 12, fontStyle: 'italic', color: '#64748b' }}>
                            Start a chat session to configure the preferred model.
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
