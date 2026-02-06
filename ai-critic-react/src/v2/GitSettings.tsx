import { useState, useEffect } from 'react';
import { useCurrent } from '../hooks/useCurrent';
import { encryptWithServerKey, EncryptionNotAvailableError } from './crypto';
import {
    testSshKey, fetchOAuthConfig, setOAuthConfig, exchangeOAuthToken,
    fetchGithubRepos, cloneRepo,
} from '../api/auth';
import type { GithubRepo, OAuthConfigStatus } from '../api/auth';
import { LogViewer } from './LogViewer';
import './GitSettings.css';

// ---- Constants ----

const GitSettingsTabs = {
    SSHKeys: 'ssh-keys',
    GitHub: 'github',
} as const;

type GitSettingsTab = typeof GitSettingsTabs[keyof typeof GitSettingsTabs];

// ---- SSH Key types (localStorage) ----

interface SSHKey {
    id: string;
    name: string;
    host: string;
    privateKey: string;
    createdAt: string;
}

const SSH_KEYS_STORAGE_KEY = 'ai-critic-ssh-keys';

function loadSSHKeys(): SSHKey[] {
    try {
        const data = localStorage.getItem(SSH_KEYS_STORAGE_KEY);
        if (!data) return [];
        return JSON.parse(data) as SSHKey[];
    } catch {
        return [];
    }
}

function saveSSHKeys(keys: SSHKey[]) {
    localStorage.setItem(SSH_KEYS_STORAGE_KEY, JSON.stringify(keys));
}

// ---- GitHub token (localStorage) ----

const GITHUB_TOKEN_STORAGE_KEY = 'ai-critic-github-token';

function loadGitHubToken(): string {
    return localStorage.getItem(GITHUB_TOKEN_STORAGE_KEY) || '';
}

function saveGitHubToken(token: string) {
    if (token) {
        localStorage.setItem(GITHUB_TOKEN_STORAGE_KEY, token);
    } else {
        localStorage.removeItem(GITHUB_TOKEN_STORAGE_KEY);
    }
}

// ---- Main Component ----

interface GitSettingsProps {
    onBack: () => void;
}

export function GitSettings({ onBack }: GitSettingsProps) {
    const [activeTab, setActiveTab] = useState<GitSettingsTab>(GitSettingsTabs.SSHKeys);

    return (
        <div className="mcc-git-settings">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Git Settings</h2>
            </div>
            <div className="mcc-git-tabs">
                <button
                    className={`mcc-git-tab ${activeTab === GitSettingsTabs.SSHKeys ? 'active' : ''}`}
                    onClick={() => setActiveTab(GitSettingsTabs.SSHKeys)}
                >
                    SSH Keys
                </button>
                <button
                    className={`mcc-git-tab ${activeTab === GitSettingsTabs.GitHub ? 'active' : ''}`}
                    onClick={() => setActiveTab(GitSettingsTabs.GitHub)}
                >
                    GitHub
                </button>
            </div>
            <div className="mcc-git-tab-content">
                {activeTab === GitSettingsTabs.SSHKeys && <SSHKeysPanel />}
                {activeTab === GitSettingsTabs.GitHub && <GitHubOAuthPanel />}
            </div>
        </div>
    );
}

// ---- SSH Key Card ----

interface SSHKeyCardProps {
    sshKey: SSHKey;
    onEdit: () => void;
    onDelete: () => void;
}

function SSHKeyCard({ sshKey, onEdit, onDelete }: SSHKeyCardProps) {
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; output: string } | null>(null);

    const handleTest = async () => {
        setTesting(true);
        setTestResult(null);

        try {
            // Encrypt the private key before sending
            const keyData = await encryptWithServerKey(sshKey.privateKey);

            const data = await testSshKey(sshKey.host, keyData);
            setTestResult(data);
        } catch (err) {
            if (err instanceof EncryptionNotAvailableError) {
                setTestResult({ success: false, output: 'Server encryption keys not configured. Ask the server admin to run: go run ./script/crypto/gen' });
            } else {
                setTestResult({ success: false, output: String(err) });
            }
        }
        setTesting(false);
    };

    return (
        <div className="mcc-ssh-key-card">
            <div className="mcc-ssh-key-header">
                <KeyIcon />
                <span className="mcc-ssh-key-name">{sshKey.name}</span>
                <span className="mcc-ssh-key-host">{sshKey.host}</span>
            </div>
            <div className="mcc-ssh-key-actions">
                <button className="mcc-port-action-btn" onClick={handleTest} disabled={testing}>
                    {testing ? 'Testing...' : 'Test'}
                </button>
                <button className="mcc-port-action-btn" onClick={onEdit}>Edit</button>
                <button className="mcc-port-action-btn mcc-port-stop" onClick={onDelete}>Delete</button>
            </div>
            {testResult && (
                <div className={`mcc-ssh-test-result ${testResult.success ? 'mcc-ssh-test-success' : 'mcc-ssh-test-error'}`}>
                    {testResult.output || (testResult.success ? 'Connection successful' : 'Connection failed')}
                </div>
            )}
        </div>
    );
}

// ---- SSH Keys Panel ----

function SSHKeysPanel() {
    const [keys, setKeys] = useState<SSHKey[]>(() => loadSSHKeys());
    const [showAddForm, setShowAddForm] = useState(false);
    const [editingKey, setEditingKey] = useState<SSHKey | null>(null);
    const [name, setName] = useState('');
    const [host, setHost] = useState('github.com');
    const [privateKey, setPrivateKey] = useState('');

    const keysRef = useCurrent(keys);

    const resetForm = () => {
        setName('');
        setHost('github.com');
        setPrivateKey('');
        setShowAddForm(false);
        setEditingKey(null);
    };

    const handleSaveKey = () => {
        if (!name.trim() || !host.trim() || !privateKey.trim()) return;

        const currentKeys = keysRef.current;
        let newKeys: SSHKey[];

        if (editingKey) {
            newKeys = currentKeys.map(k =>
                k.id === editingKey.id
                    ? { ...k, name: name.trim(), host: host.trim(), privateKey: privateKey.trim() }
                    : k
            );
        } else {
            const newKey: SSHKey = {
                id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
                name: name.trim(),
                host: host.trim(),
                privateKey: privateKey.trim(),
                createdAt: new Date().toISOString(),
            };
            newKeys = [...currentKeys, newKey];
        }

        saveSSHKeys(newKeys);
        setKeys(newKeys);
        resetForm();
    };

    const handleEditKey = (key: SSHKey) => {
        setEditingKey(key);
        setName(key.name);
        setHost(key.host);
        setPrivateKey(key.privateKey);
        setShowAddForm(true);
    };

    const handleDeleteKey = (id: string) => {
        const newKeys = keysRef.current.filter(k => k.id !== id);
        saveSSHKeys(newKeys);
        setKeys(newKeys);
    };

    return (
        <div className="mcc-ssh-keys-panel">
            <p className="mcc-git-desc">
                SSH private keys are stored in your browser's local storage. They are sent to the server only during git clone operations.
            </p>

            {keys.length > 0 && (
                <div className="mcc-ssh-key-list">
                    {keys.map(key => (
                        <SSHKeyCard
                            key={key.id}
                            sshKey={key}
                            onEdit={() => handleEditKey(key)}
                            onDelete={() => handleDeleteKey(key.id)}
                        />
                    ))}
                </div>
            )}

            {keys.length === 0 && !showAddForm && (
                <div className="mcc-git-empty">No SSH keys configured. Add one to use SSH-based git operations.</div>
            )}

            {showAddForm ? (
                <div className="mcc-add-port-form">
                    <div className="mcc-add-port-header">
                        <span>{editingKey ? 'Edit SSH Key' : 'Add SSH Key'}</span>
                        <button className="mcc-close-btn" onClick={resetForm}>×</button>
                    </div>
                    <div className="mcc-git-form-fields">
                        <div className="mcc-form-field">
                            <label>Name</label>
                            <input
                                type="text"
                                placeholder="My GitHub Key"
                                value={name}
                                onChange={e => setName(e.target.value)}
                            />
                        </div>
                        <div className="mcc-form-field">
                            <label>Host</label>
                            <input
                                type="text"
                                placeholder="github.com"
                                value={host}
                                onChange={e => setHost(e.target.value)}
                            />
                        </div>
                        <div className="mcc-form-field">
                            <label>Private Key</label>
                            <textarea
                                className="mcc-ssh-key-textarea"
                                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
                                value={privateKey}
                                onChange={e => setPrivateKey(e.target.value)}
                                rows={6}
                            />
                        </div>
                    </div>
                    <button className="mcc-forward-btn" onClick={handleSaveKey}>
                        {editingKey ? 'Update Key' : 'Save Key'}
                    </button>
                </div>
            ) : (
                <button className="mcc-add-port-btn" onClick={() => setShowAddForm(true)}>
                    <PlusIcon />
                    <span>Add SSH Key</span>
                </button>
            )}
        </div>
    );
}

// ---- GitHub OAuth Panel ----

function GitHubOAuthPanel() {
    const [oauthStatus, setOauthStatus] = useState<OAuthConfigStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [clientId, setClientId] = useState('');
    const [clientSecret, setClientSecret] = useState('');
    const [saving, setSaving] = useState(false);
    const [saveError, setSaveError] = useState('');
    const [token, setToken] = useState(() => loadGitHubToken());
    const [showConfigForm, setShowConfigForm] = useState(false);

    const tokenRef = useCurrent(token);

    // Load OAuth config status
    useEffect(() => {
        fetchOAuthConfig()
            .then((data) => {
                setOauthStatus(data);
                if (data.configured && data.client_id) {
                    setClientId(data.client_id);
                }
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, []);

    // Handle OAuth callback (check URL for ?code=...)
    useEffect(() => {
        const params = new URLSearchParams(window.location.search);
        const code = params.get('code');
        if (!code) return;

        // Clean up URL
        const url = new URL(window.location.href);
        url.searchParams.delete('code');
        url.searchParams.delete('state');
        window.history.replaceState({}, '', url.toString());

        // Exchange code for token
        exchangeOAuthToken(code)
            .then((data) => {
                if (data.access_token) {
                    saveGitHubToken(data.access_token);
                    setToken(data.access_token);
                } else if (data.error) {
                    setSaveError(data.error);
                }
            })
            .catch(err => setSaveError(String(err)));
    }, []);

    const handleSaveConfig = async () => {
        if (!clientId.trim() || !clientSecret.trim()) return;
        setSaving(true);
        setSaveError('');

        try {
            const resp = await setOAuthConfig(clientId.trim(), clientSecret.trim());
            const data = await resp.json();
            if (!resp.ok) {
                setSaveError(data.error || 'Failed to save');
            } else {
                setOauthStatus({ configured: true, client_id: clientId.trim() });
                setShowConfigForm(false);
                setClientSecret('');
            }
        } catch (err) {
            setSaveError(String(err));
        }
        setSaving(false);
    };

    const handleLogin = () => {
        if (!oauthStatus?.client_id) return;
        const redirectUri = window.location.origin + '/v2?tab=home&view=git-settings';
        const scope = 'repo';
        const authUrl = `https://github.com/login/oauth/authorize?client_id=${oauthStatus.client_id}&redirect_uri=${encodeURIComponent(redirectUri)}&scope=${scope}`;
        window.location.href = authUrl;
    };

    const handleLogout = () => {
        saveGitHubToken('');
        setToken('');
    };

    if (loading) {
        return <div className="mcc-git-loading">Loading GitHub configuration...</div>;
    }

    const currentToken = tokenRef.current;

    return (
        <div className="mcc-github-panel">
            {/* OAuth Config Section */}
            <div className="mcc-git-section">
                <div className="mcc-git-section-header">
                    <h3>OAuth App Configuration</h3>
                    {oauthStatus?.configured && !showConfigForm && (
                        <button className="mcc-port-action-btn" onClick={() => setShowConfigForm(true)}>Edit</button>
                    )}
                </div>

                {oauthStatus?.configured && !showConfigForm ? (
                    <div className="mcc-git-config-status">
                        <span className="mcc-git-status-dot mcc-git-status-ok" />
                        <span>Configured (Client ID: {oauthStatus.client_id?.slice(0, 8)}...)</span>
                    </div>
                ) : (
                    <div className="mcc-add-port-form">
                        <p className="mcc-git-desc">
                            Create a GitHub OAuth App at{' '}
                            <a href="https://github.com/settings/developers" target="_blank" rel="noopener noreferrer">
                                github.com/settings/developers
                            </a>
                            . Set the callback URL to: <code>{window.location.origin}/v2?tab=home&view=git-settings</code>
                        </p>
                        <div className="mcc-git-form-fields">
                            <div className="mcc-form-field">
                                <label>Client ID</label>
                                <input
                                    type="text"
                                    placeholder="Ov23li..."
                                    value={clientId}
                                    onChange={e => setClientId(e.target.value)}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Client Secret</label>
                                <input
                                    type="password"
                                    placeholder="Enter client secret"
                                    value={clientSecret}
                                    onChange={e => setClientSecret(e.target.value)}
                                />
                            </div>
                        </div>
                        {saveError && <div className="mcc-ports-error">{saveError}</div>}
                        <div className="mcc-git-form-actions">
                            <button className="mcc-forward-btn" onClick={handleSaveConfig} disabled={saving}>
                                {saving ? 'Saving...' : 'Save Configuration'}
                            </button>
                            {showConfigForm && (
                                <button className="mcc-port-action-btn" onClick={() => setShowConfigForm(false)}>Cancel</button>
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Login Section */}
            <div className="mcc-git-section">
                <h3>GitHub Account</h3>
                {currentToken ? (
                    <div className="mcc-git-logged-in">
                        <div className="mcc-git-config-status">
                            <span className="mcc-git-status-dot mcc-git-status-ok" />
                            <span>Logged in to GitHub</span>
                        </div>
                        <button className="mcc-port-action-btn mcc-port-stop" onClick={handleLogout}>Logout</button>
                    </div>
                ) : (
                    <div>
                        <p className="mcc-git-desc">Login with GitHub to list and clone your repositories.</p>
                        <button
                            className="mcc-forward-btn mcc-github-login-btn"
                            onClick={handleLogin}
                            disabled={!oauthStatus?.configured}
                        >
                            <GitHubIcon />
                            <span>Login with GitHub</span>
                        </button>
                        {!oauthStatus?.configured && (
                            <p className="mcc-git-hint">Configure the OAuth App above first.</p>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
}

// ---- Clone Repo View (standalone) ----

interface CloneRepoViewProps {
    onBack: () => void;
}

export function CloneRepoView({ onBack }: CloneRepoViewProps) {
    return (
        <div className="mcc-git-settings">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Clone Repository</h2>
            </div>
            <div className="mcc-git-tab-content">
                <CloneRepoPanel />
            </div>
        </div>
    );
}

// ---- Clone Repo Panel ----

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
                // Encrypt the private key before sending
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

            // Check if response is SSE stream
            const contentType = resp.headers.get('Content-Type') || '';
            if (contentType.includes('text/event-stream')) {
                // Read SSE stream
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

                    // Parse SSE lines
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || ''; // Keep incomplete line in buffer

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
                // Fallback: JSON response (shouldn't happen with new backend, but just in case)
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
            {/* SSH Key selector */}
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

            {/* Manual URL clone */}
            <div className="mcc-clone-manual">
                <div className="mcc-form-field">
                    <label>Clone by URL</label>
                    <div className="mcc-clone-manual-row">
                        <input
                            type="text"
                            placeholder="https://github.com/user/repo.git or git@github.com:user/repo.git"
                            value={manualUrl}
                            onChange={e => setManualUrl(e.target.value)}
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

            {/* Clone logs */}
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

            {/* Repo list from GitHub */}
            {!token ? (
                <div className="mcc-git-empty">
                    Login with GitHub in the "GitHub" tab to list your repositories.
                </div>
            ) : (
                <>
                    <div className="mcc-clone-search">
                        <input
                            type="text"
                            placeholder="Search repositories..."
                            value={search}
                            onChange={e => setSearch(e.target.value)}
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

// ---- Icons ----

function KeyIcon() {
    return (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
        </svg>
    );
}

function PlusIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
    );
}

function GitHubIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
        </svg>
    );
}

function LockIcon() {
    return (
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ marginRight: 4, flexShrink: 0 }}>
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        </svg>
    );
}
