import { useState, useEffect } from 'react';
import { getServerConfig, setServerConfig, type ServerConfig } from '../../../../api/serverSettings';
import './ServerSettingsSection.css';

export function ServerSettingsSection() {
    const [config, setConfig] = useState<ServerConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);
    const [projectDir, setProjectDir] = useState('');
    const [useExplicitDir, setUseExplicitDir] = useState(false);
    const [saveMessage, setSaveMessage] = useState<string | null>(null);

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await getServerConfig();
            setConfig(data);
            setProjectDir(data.project_dir || '');
            setUseExplicitDir(data.using_explicit_dir);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setSaveMessage(null);
        try {
            const dirToSave = useExplicitDir ? projectDir.trim() : '';
            await setServerConfig(dirToSave);
            setSaveMessage('Settings saved successfully');
            // Reload to get updated state
            await loadConfig();
            setTimeout(() => setSaveMessage(null), 3000);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleClear = () => {
        setUseExplicitDir(false);
        setProjectDir('');
    };

    if (loading) {
        return <div className="diagnose-loading">Loading server settings...</div>;
    }

    return (
        <div className="server-settings-section">
            {error && <div className="server-settings-error">{error}</div>}
            {saveMessage && <div className="server-settings-success">{saveMessage}</div>}

            <div className="server-settings-info">
                <div className="server-settings-row">
                    <span className="server-settings-label">Auto-detected directory:</span>
                    <span className="server-settings-value server-settings-mono">{config?.auto_detected_dir}</span>
                </div>
                {config?.using_explicit_dir && (
                    <div className="server-settings-row">
                        <span className="server-settings-label">Currently using:</span>
                        <span className="server-settings-value server-settings-mono server-settings-highlight">
                            {config.project_dir}
                        </span>
                    </div>
                )}
            </div>

            <div className="server-settings-config">
                <label className="server-settings-checkbox">
                    <input
                        type="checkbox"
                        checked={useExplicitDir}
                        onChange={(e) => setUseExplicitDir(e.target.checked)}
                    />
                    <span>Use explicit project directory</span>
                </label>

                {useExplicitDir && (
                    <div className="server-settings-field">
                        <label htmlFor="project-dir">Project Directory:</label>
                        <input
                            id="project-dir"
                            type="text"
                            value={projectDir}
                            onChange={(e) => setProjectDir(e.target.value)}
                            placeholder="e.g., /path/to/mobile-coding-connector"
                            className="server-settings-input"
                        />
                        <small>
                            When set, this directory will be used instead of the auto-detected directory.
                            This is useful when running the server from a different location.
                        </small>
                    </div>
                )}
            </div>

            <div className="server-settings-actions">
                {useExplicitDir && config?.using_explicit_dir && (
                    <button
                        type="button"
                        className="server-settings-btn server-settings-btn--secondary"
                        onClick={handleClear}
                        disabled={saving}
                    >
                        Clear & Use Auto-detected
                    </button>
                )}
                <button
                    type="button"
                    className="server-settings-btn server-settings-btn--primary"
                    onClick={handleSave}
                    disabled={saving}
                >
                    {saving ? 'Saving...' : 'Save'}
                </button>
            </div>
        </div>
    );
}
