import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCurrent } from '../../../../hooks/useCurrent';
import { encryptWithServerKey, EncryptionNotAvailableError } from '../crypto';
import { fetchGithubRepos, cloneRepo } from '../../../../api/auth';
import type { GithubRepo } from '../../../../api/auth';
import { LogViewer } from '../../../LogViewer';
import { LockIcon } from '../../../icons';
import { loadSSHKeys, loadGitHubToken } from './gitStorage';
import type { SSHKey } from './gitStorage';
import { FlexInput } from '../../../../pure-view/FlexInput';
import './GitSettings.css';

export function CloneRepoView() {
    const navigate = useNavigate();

    return (
        <div className="mcc-git-settings">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>← Back</button>
                <h2>Clone Repository</h2>
            </div>
            <div className="mcc-git-tab-content">
                <CloneRepoPanel />
            </div>
        </div>
    );
}

function CloneRepoPanel() {
    const [token] = useState(() => loadGitHubToken());
    const [repos, setRepos] = useState<GithubRepo[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [search, setSearch] = useState('');
    const [cloning, setCloning] = useState<string | null>(null);
    const [cloneResult, setCloneResult] = useState<{ status: string; dir?: string; error?: string } | null>(null);
    const [cloneLogs, setCloneLogs] = useState<string[]>([]);
    const [sshKeys] = useState<SSHKey[]>(() => loadSSHKeys());
    const [selectedKeyId, setSelectedKeyId] = useState('');
    const [useSSH, setUseSSH] = useState(false);
    const [manualUrl, setManualUrl] = useState('');

    const tokenRef = useCurrent(token);

    // Load repos when panel opens
    useEffect(() => {
        const currentToken = tokenRef.current;
        if (!currentToken) return;

        setLoading(true);
        fetchGithubRepos(currentToken)
            .then((data) => {
                setRepos(data);
                setLoading(false);
            })
            .catch(err => {
                setError(String(err));
                setLoading(false);
            });
    }, [tokenRef]);

    const handleClone = async (repoUrl: string) => {
        setCloning(repoUrl);
        setCloneResult(null);
        setCloneLogs([]);

        const body: Record<string, unknown> = { repo_url: repoUrl };

        if (useSSH && selectedKeyId) {
            const key = sshKeys.find(k => k.id === selectedKeyId);
            if (key) {
                try {
                    body.ssh_key = await encryptWithServerKey(key.privateKey);
                } catch (err) {
                    if (err instanceof EncryptionNotAvailableError) {
                        setCloneResult({ status: 'error', error: 'Server encryption keys not configured. Ask the server admin to run: go run ./script/crypto/gen' });
                        setCloning(null);
                        return;
                    }
                    setCloneResult({ status: 'error', error: String(err) });
                    setCloning(null);
                    return;
                }
                body.use_ssh = true;
                body.ssh_key_id = selectedKeyId;
            }
        }

        try {
            const resp = await cloneRepo(body);

            const contentType = resp.headers.get('Content-Type') || '';
            if (contentType.includes('text/event-stream')) {
                const reader = resp.body?.getReader();
                if (!reader) {
                    setCloneResult({ status: 'error', error: 'Failed to read response stream' });
                    setCloning(null);
                    return;
                }

                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            try {
                                const data = JSON.parse(line.slice(6));
                                if (data.type === 'log') {
                                    setCloneLogs(prev => [...prev, data.message]);
                                } else if (data.type === 'error') {
                                    setCloneLogs(prev => [...prev, `ERROR: ${data.message}`]);
                                    setCloneResult({ status: 'error', error: data.message });
                                } else if (data.type === 'done') {
                                    setCloneResult({ status: 'ok', dir: data.dir });
                                }
                            } catch {
                                // Skip malformed SSE data
                            }
                        }
                    }
                }
            } else {
                const data = await resp.json();
                setCloneResult(data);
            }
        } catch (err) {
            setCloneResult({ status: 'error', error: String(err) });
        }
        setCloning(null);
    };

    const filteredRepos = repos.filter(r =>
        r.full_name.toLowerCase().includes(search.toLowerCase())
    );

    // Find SSH key matching github.com host for default selection
    useEffect(() => {
        if (sshKeys.length > 0 && !selectedKeyId) {
            const ghKey = sshKeys.find(k => k.host === 'github.com');
            if (ghKey) {
                setSelectedKeyId(ghKey.id);
            } else {
                setSelectedKeyId(sshKeys[0].id);
            }
        }
    }, [sshKeys, selectedKeyId]);

    return (
        <div className="mcc-clone-panel">
            {sshKeys.length > 0 && (
                <div className="mcc-clone-ssh-section">
                    <label className="mcc-clone-ssh-toggle">
                        <input
                            type="checkbox"
                            checked={useSSH}
                            onChange={e => setUseSSH(e.target.checked)}
                        />
                        <span>Use SSH key for cloning</span>
                    </label>
                    {useSSH && (
                        <select
                            className="mcc-clone-ssh-select"
                            value={selectedKeyId}
                            onChange={e => setSelectedKeyId(e.target.value)}
                        >
                            {sshKeys.map(k => (
                                <option key={k.id} value={k.id}>{k.name} ({k.host})</option>
                            ))}
                        </select>
                    )}
                </div>
            )}

            <div className="mcc-clone-manual">
                <div className="mcc-form-field">
                    <label>Clone by URL</label>
                    <div className="mcc-clone-manual-row">
                        <FlexInput
                            value={manualUrl}
                            onChange={setManualUrl}
                            placeholder="https://github.com/user/repo.git or git@github.com:user/repo.git"
                        />
                        <button
                            className="mcc-forward-btn mcc-clone-btn"
                            onClick={() => handleClone(manualUrl)}
                            disabled={!manualUrl.trim() || !!cloning}
                        >
                            {cloning === manualUrl ? 'Cloning...' : 'Clone'}
                        </button>
                    </div>
                </div>
            </div>

            {(cloneLogs.length > 0 || !!cloning) && (
                <LogViewer
                    lines={cloneLogs.map(text => ({ text, error: text.startsWith('ERROR:') }))}
                    pending={!!cloning}
                    pendingMessage="Cloning in progress..."
                />
            )}

            {cloneResult && (
                <div className={`mcc-clone-result ${cloneResult.status === 'ok' ? 'mcc-clone-success' : 'mcc-clone-error'}`}>
                    {cloneResult.status === 'ok'
                        ? `Cloned to: ${cloneResult.dir}`
                        : `Error: ${cloneResult.error}`}
                </div>
            )}

            {!token ? (
                <div className="mcc-git-empty">
                    Login with GitHub in the "GitHub" tab to list your repositories.
                </div>
            ) : (
                <>
                    <div className="mcc-clone-search">
                        <FlexInput
                            value={search}
                            onChange={setSearch}
                            placeholder="Search repositories..."
                        />
                    </div>

                    {error && <div className="mcc-ports-error">{error}</div>}

                    {loading ? (
                        <div className="mcc-git-loading">Loading repositories...</div>
                    ) : (
                        <div className="mcc-clone-repo-list">
                            {filteredRepos.map(repo => (
                                <div key={repo.full_name} className="mcc-clone-repo-card">
                                    <div className="mcc-clone-repo-info">
                                        <div className="mcc-clone-repo-name">
                                            {repo.private && <LockIcon />}
                                            <span>{repo.full_name}</span>
                                        </div>
                                        {repo.description && (
                                            <div className="mcc-clone-repo-desc">{repo.description}</div>
                                        )}
                                        <div className="mcc-clone-repo-meta">
                                            {repo.language && <span className="mcc-clone-repo-lang">{repo.language}</span>}
                                            <span>{new Date(repo.updated_at).toLocaleDateString()}</span>
                                        </div>
                                    </div>
                                    <button
                                        className="mcc-forward-btn mcc-clone-btn"
                                        onClick={() => handleClone(useSSH ? repo.ssh_url : repo.clone_url)}
                                        disabled={!!cloning}
                                    >
                                        {cloning === (useSSH ? repo.ssh_url : repo.clone_url) ? 'Cloning...' : 'Clone'}
                                    </button>
                                </div>
                            ))}
                            {filteredRepos.length === 0 && !loading && (
                                <div className="mcc-git-empty">
                                    {search ? 'No matching repositories.' : 'No repositories found.'}
                                </div>
                            )}
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
