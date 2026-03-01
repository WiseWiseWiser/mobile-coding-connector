import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    fetchOpencodeSettings,
    OpencodeWebTargetPreferences,
    updateOpencodeSettings,
} from '../../../../api/agents';
import type {
    OpencodeSettings,
    OpencodeWebTargetPreference,
} from '../../../../api/agents';
import { BeakerIcon } from '../../../../pure-view/icons/BeakerIcon';

function normalizeTargetPreference(preference?: OpencodeWebTargetPreference): OpencodeWebTargetPreference {
    if (preference === OpencodeWebTargetPreferences.Localhost) {
        return preference;
    }
    return OpencodeWebTargetPreferences.Domain;
}

export function OpencodeWebSettingsView() {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');
    const [savedSettings, setSavedSettings] = useState<OpencodeSettings | null>(null);
    const [webPort, setWebPort] = useState(4096);
    const [defaultDomain, setDefaultDomain] = useState('');
    const [targetPreference, setTargetPreference] = useState<OpencodeWebTargetPreference>(OpencodeWebTargetPreferences.Domain);

    const loadSettings = async () => {
        setLoading(true);
        setError('');
        try {
            const settings = await fetchOpencodeSettings();
            setSavedSettings(settings);
            setWebPort(settings.web_server?.port || 4096);
            setDefaultDomain(settings.default_domain || '');
            setTargetPreference(normalizeTargetPreference(settings.web_server?.target_preference));
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load OpenCode settings');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        void loadSettings();
    }, []);

    const handlePortChange = (value: string) => {
        const parsed = Number.parseInt(value, 10);
        if (Number.isNaN(parsed)) {
            setWebPort(4096);
            return;
        }
        setWebPort(parsed);
    };

    const handleSave = async () => {
        if (!savedSettings) {
            return;
        }
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            const currentWebServer = savedSettings.web_server || { enabled: false, port: 4096 };
            const nextSettings: OpencodeSettings = {
                ...savedSettings,
                default_domain: defaultDomain.trim(),
                web_server: {
                    ...currentWebServer,
                    port: webPort,
                    target_preference: targetPreference,
                },
            };
            await updateOpencodeSettings(nextSettings);
            setSavedSettings(nextSettings);
            setSuccess('OpenCode settings saved');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save OpenCode settings');
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="codex-web-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../opencode-web')}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>OpenCode Settings</h2>
            </div>

            <div className="codex-web-content" style={{ padding: 12, gap: 12 }}>
                {loading && <div className="codex-web-action-status">Loading settings...</div>}
                {error && <div className="codex-web-action-error">{error}</div>}
                {success && <div className="codex-web-action-status">{success}</div>}

                {!loading && (
                    <div style={{ display: 'grid', gap: 12 }}>
                        <div style={{ display: 'grid', gap: 6 }}>
                            <label style={{ fontSize: 13, color: '#cbd5e1' }}>Web Server Port</label>
                            <input
                                type="number"
                                min={1024}
                                max={65535}
                                value={webPort}
                                disabled={saving}
                                onChange={(e) => handlePortChange(e.target.value)}
                                style={{
                                    width: '100%',
                                    padding: '10px 12px',
                                    background: '#1e293b',
                                    border: '1px solid #334155',
                                    borderRadius: 8,
                                    color: '#e2e8f0',
                                    fontSize: 14,
                                }}
                            />
                        </div>

                        <div style={{ display: 'grid', gap: 6 }}>
                            <label style={{ fontSize: 13, color: '#cbd5e1' }}>Domain Config</label>
                            <input
                                type="text"
                                value={defaultDomain}
                                disabled={saving}
                                onChange={(e) => setDefaultDomain(e.target.value)}
                                placeholder="e.g. your-domain.com"
                                style={{
                                    width: '100%',
                                    padding: '10px 12px',
                                    background: '#1e293b',
                                    border: '1px solid #334155',
                                    borderRadius: 8,
                                    color: '#e2e8f0',
                                    fontSize: 14,
                                }}
                            />
                        </div>

                        <div style={{ display: 'grid', gap: 8 }}>
                            <label style={{ fontSize: 13, color: '#cbd5e1' }}>Preferred Target URL</label>
                            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 14 }}>
                                <input
                                    type="radio"
                                    name="opencode-web-target-preference"
                                    checked={targetPreference === OpencodeWebTargetPreferences.Domain}
                                    onChange={() => setTargetPreference(OpencodeWebTargetPreferences.Domain)}
                                    disabled={saving}
                                />
                                Use mapped domain when available
                            </label>
                            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 14 }}>
                                <input
                                    type="radio"
                                    name="opencode-web-target-preference"
                                    checked={targetPreference === OpencodeWebTargetPreferences.Localhost}
                                    onChange={() => setTargetPreference(OpencodeWebTargetPreferences.Localhost)}
                                    disabled={saving}
                                />
                                Always use localhost
                            </label>
                        </div>

                        <div>
                            <button className="mcc-btn-primary" onClick={handleSave} disabled={saving}>
                                {saving ? 'Saving...' : 'Save Settings'}
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
