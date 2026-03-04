import { useEffect, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { BeakerIcon } from '../../../../../pure-view/icons/BeakerIcon';
import { fetchCursorACPSessionSettings, saveCursorACPSessionSettings } from '../../../../../api/cursor-acp';
import './ACPUI.css';

export function CursorACPSessionSettings() {
    const navigate = useNavigate();
    const { sessionId } = useParams<{ sessionId: string }>();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');
    const [trustWorkspace, setTrustWorkspace] = useState(false);
    const [yoloMode, setYoloMode] = useState(false);

    const loadSettings = useCallback(async () => {
        if (!sessionId) return;
        setLoading(true);
        try {
            const data = await fetchCursorACPSessionSettings(sessionId);
            setTrustWorkspace(data.trustWorkspace || false);
            setYoloMode(data.yoloMode || false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load settings');
        } finally {
            setLoading(false);
        }
    }, [sessionId]);

    useEffect(() => {
        loadSettings();
    }, [loadSettings]);

    const handleSave = async () => {
        if (!sessionId) return;
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await saveCursorACPSessionSettings({
                sessionId,
                trustWorkspace,
                yoloMode,
            });
            setSuccess('Settings saved successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="acp-ui-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate(-1)}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Session Settings</h2>
            </div>

            <div style={{ padding: 16, maxWidth: 600 }}>
                {loading && <div style={{ color: 'var(--mcc-text-muted, #64748b)' }}>Loading settings...</div>}
                {error && <div style={{ color: 'var(--mcc-accent-red, #f87171)', fontSize: 13, marginBottom: 12 }}>{error}</div>}
                {success && <div style={{ color: 'var(--mcc-accent-green, #22c55e)', fontSize: 13, marginBottom: 12 }}>{success}</div>}

                {!loading && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                        <div style={{ 
                            padding: 16, 
                            background: 'var(--mcc-bg-card, #1e293b)', 
                            borderRadius: 8,
                            border: '1px solid var(--mcc-border-default, #334155)'
                        }}>
                            <h3 style={{ margin: '0 0 12px 0', fontSize: 14, color: 'var(--mcc-text-primary, #e2e8f0)' }}>
                                Workspace Trust
                            </h3>
                            <p style={{ margin: '0 0 16px 0', fontSize: 13, color: 'var(--mcc-text-secondary, #cbd5e1)', lineHeight: 1.5 }}>
                                Trust the current workspace to allow cursor-agent to access and modify files. 
                                You can enable this permanently or respond to trust prompts per-session.
                            </p>
                            <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
                                <input
                                    type="checkbox"
                                    checked={trustWorkspace}
                                    onChange={e => setTrustWorkspace(e.target.checked)}
                                    disabled={saving}
                                    style={{ width: 18, height: 18, cursor: 'pointer' }}
                                />
                                <span style={{ fontSize: 14, color: 'var(--mcc-text-primary, #e2e8f0)' }}>
                                    Trust workspace for this session
                                </span>
                            </label>
                        </div>

                        <div style={{ 
                            padding: 16, 
                            background: 'var(--mcc-bg-card, #1e293b)', 
                            borderRadius: 8,
                            border: '1px solid var(--mcc-border-default, #334155)'
                        }}>
                            <h3 style={{ margin: '0 0 12px 0', fontSize: 14, color: 'var(--mcc-text-primary, #e2e8f0)' }}>
                                YOLO Mode
                            </h3>
                            <p style={{ margin: '0 0 16px 0', fontSize: 13, color: 'var(--mcc-text-secondary, #cbd5e1)', lineHeight: 1.5 }}>
                                Enable YOLO mode to bypass all confirmations including workspace trust prompts. 
                                This is equivalent to passing <code>--yolo</code> flag to cursor-agent.
                            </p>
                            <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
                                <input
                                    type="checkbox"
                                    checked={yoloMode}
                                    onChange={e => setYoloMode(e.target.checked)}
                                    disabled={saving}
                                    style={{ width: 18, height: 18, cursor: 'pointer' }}
                                />
                                <span style={{ fontSize: 14, color: 'var(--mcc-text-primary, #e2e8f0)' }}>
                                    Enable YOLO mode (--yolo)
                                </span>
                            </label>
                        </div>

                        <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                            <button 
                                className="mcc-btn-primary" 
                                onClick={handleSave} 
                                disabled={saving}
                                style={{ minWidth: 100 }}
                            >
                                {saving ? 'Saving...' : 'Save Settings'}
                            </button>
                            <button 
                                className="mcc-btn-secondary" 
                                onClick={() => navigate(-1)}
                                disabled={saving}
                                style={{ minWidth: 80 }}
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
