import { useState, useEffect } from 'react';
import { fetchAgentPathConfig, updateAgentPathConfig, fetchAgents, fetchAgentEffectivePath } from '../../../api/agents';
import type { AgentDef, AgentEffectivePath } from '../../../api/agents';
import { CursorApiKeyInput } from './CursorApiKeyInput';

export interface AgentPathSettingsProps {
    agentId: string;
    onBack: () => void;
    onRefreshAgents: () => void;
}

export function AgentPathSettings({ agentId, onBack, onRefreshAgents }: AgentPathSettingsProps) {
    const [agent, setAgent] = useState<AgentDef | null>(null);
    const [binaryPath, setBinaryPath] = useState('');
    const [effectivePath, setEffectivePath] = useState<AgentEffectivePath | null>(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');
    
    const isCursorAgent = agentId === 'cursor-agent';

    const loadEffectivePath = async () => {
        try {
            const pathInfo = await fetchAgentEffectivePath(agentId);
            setEffectivePath(pathInfo);
        } catch {
            // Ignore errors fetching effective path
        }
    };

    useEffect(() => {
        Promise.all([
            fetchAgents(),
            fetchAgentPathConfig(agentId),
            fetchAgentEffectivePath(agentId),
        ]).then(([agents, config, pathInfo]) => {
            const foundAgent = agents.find(a => a.id === agentId);
            setAgent(foundAgent || null);
            setBinaryPath(config.binary_path || '');
            setEffectivePath(pathInfo);
            setLoading(false);
        }).catch(() => setLoading(false));
    }, [agentId]);

    const handleSave = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateAgentPathConfig(agentId, binaryPath);
            setSuccess('Binary path saved. The agent list will refresh.');
            onRefreshAgents();
            await loadEffectivePath();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    const handleClear = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateAgentPathConfig(agentId, '');
            setBinaryPath('');
            setSuccess('Binary path cleared. The agent will use the default command.');
            onRefreshAgents();
            await loadEffectivePath();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to clear settings');
        } finally {
            setSaving(false);
        }
    };

    const handleApiKeySuccess = (message: string) => {
        setError('');
        setSuccess(message);
    };

    const handleApiKeyError = (message: string) => {
        setSuccess('');
        setError(message);
    };

    return (
        <div className="mcc-agent-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2>{agent?.name || agentId} Settings</h2>
            </div>

            {loading ? (
                <div className="mcc-agent-loading">Loading settings...</div>
            ) : (
                <div className="mcc-agent-settings-form">
                    {/* Effective Binary Path Display */}
                    <div className="mcc-agent-settings-field" style={{ marginBottom: 16 }}>
                        <label className="mcc-agent-settings-label">
                            Effective Binary Path
                        </label>
                        <div style={{
                            padding: '10px 12px',
                            background: effectivePath?.found ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                            border: `1px solid ${effectivePath?.found ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                            borderRadius: 8,
                            fontFamily: 'monospace',
                            fontSize: '13px',
                            color: effectivePath?.found ? '#86efac' : '#fca5a5',
                            wordBreak: 'break-all',
                        }}>
                            {effectivePath?.found ? (
                                effectivePath.effective_path
                            ) : (
                                <span>Not found{effectivePath?.error ? `: ${effectivePath.error}` : ''}</span>
                            )}
                        </div>
                    </div>

                    <div className="mcc-agent-settings-field">
                        <label className="mcc-agent-settings-label">
                            Custom Binary Path
                        </label>
                        <div className="mcc-agent-settings-hint">
                            Specify a custom path to the {agent?.name || agentId} binary.
                            Leave empty to use the default command ({agent?.command || 'unknown'}).
                        </div>
                        <input
                            type="text"
                            className="mcc-agent-settings-input"
                            value={binaryPath}
                            onChange={e => setBinaryPath(e.target.value)}
                            placeholder={`e.g. /usr/local/bin/${agent?.command || 'agent'}`}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                background: '#1e293b',
                                border: '1px solid #334155',
                                borderRadius: 8,
                                color: '#e2e8f0',
                                fontSize: '14px',
                            }}
                        />
                    </div>

                    <div className="mcc-agent-settings-field" style={{ marginTop: 12 }}>
                        <div className="mcc-agent-settings-hint">
                            <strong>Status:</strong> {agent?.installed ? (
                                <span style={{ color: '#86efac' }}>Installed</span>
                            ) : (
                                <span style={{ color: '#fca5a5' }}>Not installed</span>
                            )}
                        </div>
                        {!agent?.installed && (
                            <div className="mcc-agent-settings-hint" style={{ marginTop: 8 }}>
                                The agent binary was not found in PATH. Configure a custom binary path above
                                to specify where the agent is installed.
                            </div>
                        )}
                    </div>

                    {/* API Key Section (cursor-agent only) */}
                    {isCursorAgent && (
                        <div style={{ marginTop: 20, paddingTop: 20, borderTop: '1px solid #334155' }}>
                            <CursorApiKeyInput
                                onSuccess={handleApiKeySuccess}
                                onError={handleApiKeyError}
                            />
                        </div>
                    )}

                    <div className="mcc-agent-settings-actions" style={{ display: 'flex', gap: 10, marginTop: 16 }}>
                        <button
                            className="mcc-agent-settings-save-btn"
                            onClick={handleSave}
                            disabled={saving}
                            style={{
                                flex: 1,
                                padding: '10px 16px',
                                background: '#3b82f6',
                                color: '#fff',
                                border: 'none',
                                borderRadius: 8,
                                fontSize: '14px',
                                fontWeight: 600,
                                cursor: 'pointer',
                            }}
                        >
                            {saving ? 'Saving...' : 'Save'}
                        </button>
                        {binaryPath && (
                            <button
                                className="mcc-agent-settings-clear-btn"
                                onClick={handleClear}
                                disabled={saving}
                                style={{
                                    padding: '10px 16px',
                                    background: '#1e293b',
                                    color: '#f87171',
                                    border: '1px solid #334155',
                                    borderRadius: 8,
                                    fontSize: '14px',
                                    cursor: 'pointer',
                                }}
                            >
                                Clear
                            </button>
                        )}
                    </div>

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
