import { useState, useEffect } from 'react';
import { fetchAgentSettings, updateAgentSettings, fetchAgentTemplates } from '../../../api/agents';
import type { AgentSessionInfo, AgentSettings, AgentTemplate } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { CursorApiKeyInput } from './CursorApiKeyInput';
import { AgentPathSettingsSection } from './AgentPathSettingsSection';

export interface CursorAgentSettingsProps {
    agentId: string;
    session: AgentSessionInfo | null;
    projectName: string | null;
    onBack: () => void;
    onRefreshAgents?: () => void;
}

export function CursorAgentSettings({ agentId, session, projectName, onBack, onRefreshAgents }: CursorAgentSettingsProps) {
    const [settings, setSettings] = useState<AgentSettings>({
        prompt_append_message: '',
        followup_append_message: '',
    });
    const [templates, setTemplates] = useState<AgentTemplate[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        if (!session) {
            setLoading(false);
            return;
        }
        Promise.all([
            fetchAgentSettings(session.id),
            fetchAgentTemplates(session.id),
        ]).then(([s, t]) => {
            setSettings(s);
            setTemplates(t);
            setLoading(false);
        }).catch(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session?.id]);

    const handleSave = async () => {
        if (!session) return;
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            const updated = await updateAgentSettings(session.id, settings);
            setSettings(updated);
            setSuccess('Settings saved');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
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
            {session ? (
                <AgentChatHeader agentName={session.agent_name} projectName={projectName} onBack={onBack} />
            ) : (
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                    <h2>Cursor Settings</h2>
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

                    {/* API Key Section */}
                    <div style={{ marginBottom: 20 }}>
                        <CursorApiKeyInput
                            onSuccess={handleApiKeySuccess}
                            onError={handleApiKeyError}
                        />
                    </div>

                    {/* Session-specific settings */}
                    {session ? (
                        <>
                            <div className="mcc-agent-settings-field">
                                <label className="mcc-agent-settings-label">
                                    Prompt Append Message
                                </label>
                                <div className="mcc-agent-settings-hint">
                                    This text will be appended to every message you type in the chat box before sending.
                                </div>
                                <textarea
                                    className="mcc-agent-settings-textarea"
                                    value={settings.prompt_append_message}
                                    onChange={e => setSettings(prev => ({ ...prev, prompt_append_message: e.target.value }))}
                                    rows={5}
                                    placeholder="e.g. Always explain your reasoning step by step."
                                />
                                {templates.length > 0 && (
                                    <div className="mcc-agent-settings-templates">
                                        {templates.map(t => (
                                            <button
                                                key={t.id}
                                                className="mcc-agent-settings-template-btn"
                                                onClick={() => setSettings(prev => ({
                                                    ...prev,
                                                    prompt_append_message: prev.prompt_append_message
                                                        ? prev.prompt_append_message + '\n' + t.content
                                                        : t.content,
                                                }))}
                                                title={`Append "${t.name}" template`}
                                            >
                                                + {t.name}
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </div>

                            <div className="mcc-agent-settings-field">
                                <label className="mcc-agent-settings-label">
                                    Followup Append Message
                                </label>
                                <div className="mcc-agent-settings-hint">
                                    This text will be appended to messages consumed via a followup callback.
                                </div>
                                <textarea
                                    className="mcc-agent-settings-textarea"
                                    value={settings.followup_append_message}
                                    onChange={e => setSettings(prev => ({ ...prev, followup_append_message: e.target.value }))}
                                    rows={5}
                                    placeholder="e.g. After completing the task, summarize what was done."
                                />
                                {templates.length > 0 && (
                                    <div className="mcc-agent-settings-templates">
                                        {templates.map(t => (
                                            <button
                                                key={t.id}
                                                className="mcc-agent-settings-template-btn"
                                                onClick={() => setSettings(prev => ({
                                                    ...prev,
                                                    followup_append_message: prev.followup_append_message
                                                        ? prev.followup_append_message + '\n' + t.content
                                                        : t.content,
                                                }))}
                                                title={`Append "${t.name}" template`}
                                            >
                                                + {t.name}
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </div>

                            <div className="mcc-agent-settings-actions">
                                <button
                                    className="mcc-agent-settings-save-btn"
                                    onClick={handleSave}
                                    disabled={saving}
                                >
                                    {saving ? 'Saving...' : 'Save Settings'}
                                </button>
                            </div>
                        </>
                    ) : (
                        <div className="mcc-agent-settings-hint" style={{ marginTop: 12, fontStyle: 'italic', color: '#64748b' }}>
                            Start a chat session to configure prompt and followup messages.
                        </div>
                    )}

                    {error && (
                        <div className="mcc-agent-settings-message mcc-agent-settings-error">
                            {error}
                        </div>
                    )}
                    {success && (
                        <div className="mcc-agent-settings-message mcc-agent-settings-success">
                            {success}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
