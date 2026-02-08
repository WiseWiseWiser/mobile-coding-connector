import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCurrent } from '../../../../hooks/useCurrent';
import { encryptWithServerKey, EncryptionNotAvailableError } from '../crypto';
import {
    testSshKey, fetchOAuthConfig, setOAuthConfig, exchangeOAuthToken,
} from '../../../../api/auth';
import { consumeSSEStream } from '../../../../api/sse';
import type { LogLine } from '../../../LogViewer';
import { LogViewer } from '../../../LogViewer';
import type { OAuthConfigStatus } from '../../../../api/auth';
import { KeyIcon, PlusIcon, GitHubIcon } from '../../../icons';
import { loadSSHKeys, saveSSHKeys, loadGitHubToken, saveGitHubToken, loadGitUserConfig, saveGitUserConfig } from './gitStorage';
import type { SSHKey, GitUserConfig } from './gitStorage';
import './GitSettings.css';

// ---- Constants ----

const GitSettingsTabs = {
    SSHKeys: 'ssh-keys',
    GitHub: 'github',
    GitConfig: 'git-config',
} as const;

type GitSettingsTab = typeof GitSettingsTabs[keyof typeof GitSettingsTabs];

// ---- Main Component ----

/** Standalone page with back button + header. Used at /settings/git route. */
export function GitSettings() {
    const navigate = useNavigate();

    return (
        <div className="mcc-git-settings">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>← Back</button>
                <h2>Git Settings</h2>
            </div>
            <GitSettingsContent />
        </div>
    );
}

/** Embeddable content (tabs + panels) without page header. Used inside SettingsView as a section. */
export function GitSettingsContent() {
    const [activeTab, setActiveTab] = useState<GitSettingsTab>(GitSettingsTabs.SSHKeys);

    return (
        <>
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
                <button
                    className={`mcc-git-tab ${activeTab === GitSettingsTabs.GitConfig ? 'active' : ''}`}
                    onClick={() => setActiveTab(GitSettingsTabs.GitConfig)}
                >
                    Git Config
                </button>
            </div>
            <div className="mcc-git-tab-content">
                {activeTab === GitSettingsTabs.SSHKeys && <SSHKeysPanel />}
                {activeTab === GitSettingsTabs.GitHub && <GitHubOAuthPanel />}
                {activeTab === GitSettingsTabs.GitConfig && <GitConfigPanel />}
            </div>
        </>
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
    const [testLogs, setTestLogs] = useState<LogLine[]>([]);
    const [testDone, setTestDone] = useState(false);
    const [testSuccess, setTestSuccess] = useState<boolean | null>(null);

    const handleTest = async () => {
        setTesting(true);
        setTestLogs([]);
        setTestDone(false);
        setTestSuccess(null);

        try {
            const keyData = await encryptWithServerKey(sshKey.privateKey);
            const resp = await testSshKey(sshKey.host, keyData);
            await consumeSSEStream(resp, {
                onLog: (line) => setTestLogs(prev => [...prev, line]),
                onError: (line) => setTestLogs(prev => [...prev, line]),
                onDone: (message, data) => {
                    setTestLogs(prev => [...prev, { text: message }]);
                    setTestDone(true);
                    setTestSuccess(data.success === 'true');
                },
            });
        } catch (err) {
            if (err instanceof EncryptionNotAvailableError) {
                setTestLogs([{ text: 'Server encryption keys not configured. Ask the server admin to run: go run ./script/crypto/gen', error: true }]);
            } else {
                setTestLogs(prev => [...prev, { text: String(err), error: true }]);
            }
            setTestDone(true);
            setTestSuccess(false);
        }
        setTesting(false);
    };

    const handleDownload = () => {
        const blob = new Blob([sshKey.privateKey], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        const filename = (sshKey.name || 'id_rsa').replace(/[^a-zA-Z0-9_.-]/g, '_');
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const showLogs = testLogs.length > 0;

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
            {showLogs && (
                <div className="mcc-ssh-test-logs">
                    {testDone && testSuccess !== null && (
                        <div className={`mcc-ssh-test-result ${testSuccess ? 'mcc-ssh-test-success' : 'mcc-ssh-test-error'}`}>
                            {testSuccess ? 'Connection successful' : 'Connection failed'}
                        </div>
                    )}
                    <LogViewer
                        lines={testLogs}
                        pending={testing}
                        pendingMessage="Testing SSH connection..."
                        maxHeight={150}
                    />
                </div>
            )}
            <div className="mcc-ssh-key-extra-actions">
                <button className="mcc-port-action-btn" onClick={handleDownload}>Download</button>
            </div>
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
    const keyFileInputRef = useRef<HTMLInputElement>(null);

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
                            <input
                                ref={keyFileInputRef}
                                type="file"
                                style={{ display: 'none' }}
                                onChange={e => {
                                    const file = e.target.files?.[0];
                                    if (!file) return;
                                    const reader = new FileReader();
                                    reader.onload = () => {
                                        setPrivateKey(reader.result as string);
                                        // Auto-fill name from filename if name is empty
                                        if (!name.trim()) {
                                            setName(file.name.replace(/\.[^.]+$/, ''));
                                        }
                                    };
                                    reader.readAsText(file);
                                    if (keyFileInputRef.current) keyFileInputRef.current.value = '';
                                }}
                            />
                            <button
                                type="button"
                                className="mcc-port-action-btn mcc-ssh-upload-btn"
                                onClick={() => keyFileInputRef.current?.click()}
                            >
                                Upload from file
                            </button>
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

    useEffect(() => {
        const params = new URLSearchParams(window.location.search);
        const code = params.get('code');
        if (!code) return;

        const url = new URL(window.location.href);
        url.searchParams.delete('code');
        url.searchParams.delete('state');
        window.history.replaceState({}, '', url.toString());

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
        const redirectUri = window.location.origin + '/?tab=home&view=git-settings';
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
                            . Set the callback URL to: <code>{window.location.origin}/?tab=home&view=git-settings</code>
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

// ---- Git Config Panel ----

function GitConfigPanel() {
    const [config, setConfig] = useState<GitUserConfig>(() => loadGitUserConfig());
    const [name, setName] = useState(config.name);
    const [email, setEmail] = useState(config.email);
    const [saved, setSaved] = useState(false);

    const handleSave = () => {
        const newConfig: GitUserConfig = {
            name: name.trim(),
            email: email.trim(),
        };
        saveGitUserConfig(newConfig);
        setConfig(newConfig);
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
    };

    const isValid = name.trim() && email.trim();

    return (
        <div className="mcc-git-config-panel">
            <p className="mcc-git-desc">
                Configure your Git user name and email. These will be used for all Git commits.
            </p>

            <div className="mcc-add-port-form">
                <div className="mcc-git-form-fields">
                    <div className="mcc-form-field">
                        <label>User Name *</label>
                        <input
                            type="text"
                            placeholder="John Doe"
                            value={name}
                            onChange={e => setName(e.target.value)}
                        />
                    </div>
                    <div className="mcc-form-field">
                        <label>Email *</label>
                        <input
                            type="email"
                            placeholder="john@example.com"
                            value={email}
                            onChange={e => setEmail(e.target.value)}
                        />
                    </div>
                </div>

                {saved && (
                    <div className="mcc-git-success" style={{ marginTop: '12px' }}>
                        Configuration saved successfully!
                    </div>
                )}

                <button
                    className="mcc-forward-btn"
                    onClick={handleSave}
                    disabled={!isValid}
                    style={{ marginTop: '16px' }}
                >
                    Save Configuration
                </button>
            </div>
        </div>
    );
}
