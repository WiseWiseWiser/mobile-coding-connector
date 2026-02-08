import { useState, useEffect } from 'react';
import { fetchAgentSettings, updateAgentSettings, fetchAgentTemplates } from '../../../api/agents';
import type { AgentSessionInfo, AgentSettings, AgentTemplate } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';

export interface CursorAgentSettingsProps {
    session: AgentSessionInfo;
    projectName: string | null;
    onBack: () => void;
}

export function CursorAgentSettings({ session, projectName, onBack }: CursorAgentSettingsProps) {
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
        Promise.all([
            fetchAgentSettings(session.id),
            fetchAgentTemplates(session.id),
        ]).then(([s, t]) => {
            setSettings(s);
            setTemplates(t);
            setLoading(false);
        }).catch(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id]);

    const handleSave = async () => {
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

    return (
        <div className="mcc-agent-view">
            <AgentChatHeader agentName={session.agent_name} projectName={projectName} onBack={onBack} />
            <div className="mcc-agent-header" style={{ paddingTop: 4 }}>
                <h2>Settings</h2>
            </div>

            {loading ? (
                <div className="mcc-agent-loading">Loading settings...</div>
            ) : (
                <div className="mcc-agent-settings-form">
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
