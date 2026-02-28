import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../../../icons';

const API_PREFIX = '/api/agent/acp/cursor';

interface EffectivePathInfo {
    found: boolean;
    effective_path?: string;
    error?: string;
}

export function CursorACPSettings() {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');
    const [apiKey, setApiKey] = useState('');
    const [binaryPath, setBinaryPath] = useState('');
    const [effectivePath, setEffectivePath] = useState<EffectivePathInfo | null>(null);

    const loadSettings = useCallback(async () => {
        setLoading(true);
        try {
            const resp = await fetch(`${API_PREFIX}/settings`);
            const data = await resp.json();
            setApiKey(data.api_key || '');
            setBinaryPath(data.binary_path || '');
            if (data.effective_path) {
                setEffectivePath(data.effective_path);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load settings');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadSettings();
    }, [loadSettings]);

    const handleSave = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            const resp = await fetch(`${API_PREFIX}/settings`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ api_key: apiKey.trim(), binary_path: binaryPath.trim() }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => ({}));
                throw new Error(data.error || `Failed (${resp.status})`);
            }
            setSuccess('Settings saved');
            await loadSettings();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    const inputStyle: React.CSSProperties = {
        width: '100%',
        padding: '10px 12px',
        background: 'var(--mcc-bg-card, #1e293b)',
        border: '1px solid var(--mcc-border-default, #334155)',
        borderRadius: 8,
        color: 'var(--mcc-text-primary, #e2e8f0)',
        fontSize: 14,
        boxSizing: 'border-box',
    };

    return (
        <div className="acp-ui-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../acp/cursor')}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Cursor Agent Settings</h2>
            </div>

            <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
                {loading && <div style={{ color: 'var(--mcc-text-muted, #64748b)' }}>Loading settings...</div>}
                {error && <div style={{ color: 'var(--mcc-accent-red, #f87171)', fontSize: 13 }}>{error}</div>}
                {success && <div style={{ color: 'var(--mcc-accent-green, #22c55e)', fontSize: 13 }}>{success}</div>}

                {!loading && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                        {/* Effective Binary Path Display */}
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                            <label style={{ fontSize: 13, color: 'var(--mcc-text-secondary, #cbd5e1)' }}>
                                Effective Binary Path
                            </label>
                            <div style={{
                                padding: '10px 12px',
                                background: effectivePath?.found ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                                border: `1px solid ${effectivePath?.found ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                                borderRadius: 8,
                                fontFamily: 'monospace',
                                fontSize: 13,
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

                        {/* Binary Path */}
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                            <label style={{ fontSize: 13, color: 'var(--mcc-text-secondary, #cbd5e1)' }}>
                                Agent Binary Path
                            </label>
                            <input
                                type="text"
                                value={binaryPath}
                                onChange={e => setBinaryPath(e.target.value)}
                                disabled={saving}
                                placeholder="Leave empty to auto-detect from PATH"
                                style={inputStyle}
                            />
                            <span style={{ fontSize: 12, color: 'var(--mcc-text-muted, #64748b)' }}>
                                Custom path to cursor-agent binary. Leave empty to use the default from PATH.
                            </span>
                        </div>

                        {/* API Key */}
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                            <label style={{ fontSize: 13, color: 'var(--mcc-text-secondary, #cbd5e1)' }}>
                                Cursor API Key
                            </label>
                            <input
                                type="password"
                                value={apiKey}
                                onChange={e => setApiKey(e.target.value)}
                                disabled={saving}
                                placeholder="Enter API key (leave empty to use default login)"
                                style={inputStyle}
                            />
                            <span style={{ fontSize: 12, color: 'var(--mcc-text-muted, #64748b)' }}>
                                If set, this key will be passed to cursor-agent via --api-key flag.
                                Leave empty to use the default authentication (cursor-agent login).
                            </span>
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
