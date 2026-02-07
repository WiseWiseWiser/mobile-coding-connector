import { useState, useEffect } from 'react';
import { fetchTerminalConfig, saveTerminalConfig } from '../../../../api/terminalConfig';
import './TerminalSection.css';

export function TerminalSection() {
    const [paths, setPaths] = useState<string[]>([]);
    const [shell, setShell] = useState('');
    const [shellFlags, setShellFlags] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const [newPath, setNewPath] = useState('');

    useEffect(() => {
        fetchTerminalConfig()
            .then(cfg => {
                setPaths(cfg.extra_paths || []);
                setShell(cfg.shell || '');
                setShellFlags(cfg.shell_flags?.join(' ') ?? '');
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
    }, []);

    const handleAddPath = () => {
        const trimmed = newPath.trim();
        if (!trimmed) return;
        if (paths.includes(trimmed)) {
            setError('This path is already in the list');
            return;
        }
        setPaths([...paths, trimmed]);
        setNewPath('');
        setError(null);
        setSuccess(false);
    };

    const handleRemovePath = (index: number) => {
        setPaths(paths.filter((_, i) => i !== index));
        setSuccess(false);
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setSuccess(false);
        try {
            const flags = shellFlags.trim();
            await saveTerminalConfig({
                extra_paths: paths,
                shell: shell.trim() || undefined,
                shell_flags: flags ? flags.split(/\s+/) : undefined,
            });
            setSuccess(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setSaving(false);
    };

    const handlePathKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleAddPath();
        }
    };

    return (
        <div className="diagnose-section">
            <h3 className="diagnose-section-title">Terminal</h3>

            {loading ? (
                <div className="diagnose-loading">Loading...</div>
            ) : (
                <div className="terminal-section-content">
                    {/* Shell Configuration */}
                    <div className="terminal-section-group">
                        <label className="terminal-section-label">Default Shell</label>
                        <p className="terminal-section-desc">
                            Shell path or name. Leave empty for default (<code>bash</code>).
                        </p>
                        <input
                            className="terminal-section-input"
                            type="text"
                            value={shell}
                            onChange={e => { setShell(e.target.value); setSuccess(false); }}
                            placeholder="bash"
                        />
                    </div>

                    <div className="terminal-section-group">
                        <label className="terminal-section-label">Shell Flags</label>
                        <p className="terminal-section-desc">
                            Space-separated flags passed to the shell. Leave empty for default (<code>-i</code>).
                        </p>
                        <input
                            className="terminal-section-input"
                            type="text"
                            value={shellFlags}
                            onChange={e => { setShellFlags(e.target.value); setSuccess(false); }}
                            placeholder="--login -i"
                        />
                    </div>

                    {/* Extra PATHs */}
                    <div className="terminal-section-group">
                        <label className="terminal-section-label">Extra PATHs</label>
                        <p className="terminal-section-desc">
                            Extra directories to append to the <code>PATH</code> environment variable in terminal sessions.
                        </p>

                        {paths.length > 0 && (
                            <div className="terminal-section-paths">
                                {paths.map((p, i) => (
                                    <div key={i} className="terminal-section-path-row">
                                        <code className="terminal-section-path-value">{p}</code>
                                        <button
                                            className="terminal-section-path-remove"
                                            onClick={() => handleRemovePath(i)}
                                            title="Remove"
                                        >
                                            &times;
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}

                        <div className="terminal-section-add-row">
                            <input
                                className="terminal-section-input"
                                type="text"
                                value={newPath}
                                onChange={e => setNewPath(e.target.value)}
                                onKeyDown={handlePathKeyDown}
                                placeholder="/usr/local/custom/bin"
                            />
                            <button
                                className="mcc-port-action-btn"
                                onClick={handleAddPath}
                                disabled={!newPath.trim()}
                            >
                                Add
                            </button>
                        </div>
                    </div>

                    {error && <div className="terminal-section-error">{error}</div>}
                    {success && <div className="terminal-section-success">Saved! New terminal sessions will use the updated settings.</div>}

                    <div className="terminal-section-actions">
                        <button
                            className="mcc-port-action-btn"
                            onClick={handleSave}
                            disabled={saving}
                        >
                            {saving ? 'Saving...' : 'Save'}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
