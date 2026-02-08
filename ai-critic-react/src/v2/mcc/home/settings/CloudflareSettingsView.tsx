import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchCloudflareStatus, cloudflareLogin, fetchTunnels, createTunnel, deleteTunnel } from '../../../../api/cloudflare';
import type { CloudflareStatus, TunnelInfo } from '../../../../api/cloudflare';
import { consumeSSEStream } from '../../../../api/sse';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { fetchDomains, saveDomains, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import './CloudflareSettingsView.css';

/** Embeddable Cloudflare settings content (no page header) */
export function CloudflareSettingsContent() {
    const [status, setStatus] = useState<CloudflareStatus | null>(null);
    const [tunnels, setTunnels] = useState<TunnelInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Login state
    const [loggingIn, setLoggingIn] = useState(false);
    const [loginLogs, setLoginLogs] = useState<LogLine[]>([]);
    const [loginUrl, setLoginUrl] = useState<string | null>(null);

    // Create tunnel state
    const [showCreate, setShowCreate] = useState(false);
    const [newTunnelName, setNewTunnelName] = useState('');
    const [creating, setCreating] = useState(false);

    // Delete state
    const [deleting, setDeleting] = useState<string | null>(null);

    // Upload state
    const [uploading, setUploading] = useState(false);
    const [uploadMessage, setUploadMessage] = useState<string | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    // My Own Domains state
    const [userDomains, setUserDomains] = useState<string[]>([]);
    const [newDomain, setNewDomain] = useState('');
    const [addingDomain, setAddingDomain] = useState(false);
    const [domainsError, setDomainsError] = useState<string | null>(null);
    const [domainsLoading, setDomainsLoading] = useState(false);

    const loadStatus = async () => {
        try {
            const s = await fetchCloudflareStatus();
            setStatus(s);
            if (s.authenticated) {
                const t = await fetchTunnels();
                setTunnels(t || []);
            }
        } catch (err) {
            setError(String(err));
        }
        setLoading(false);
    };

    const loadUserDomains = async () => {
        setDomainsLoading(true);
        try {
            const data = await fetchDomains();
            // Extract only the user's custom domains (filter out auto-generated ones)
            const domains = data.domains
                .filter(d => !d.domain.includes('trycloudflare.com') && !d.domain.includes('loca.lt'))
                .map(d => d.domain);
            setUserDomains(domains);
        } catch (err) {
            setDomainsError(String(err));
        }
        setDomainsLoading(false);
    };

    useEffect(() => {
        loadStatus();
        loadUserDomains();
    }, []);

    const handleLogin = async () => {
        setLoggingIn(true);
        setLoginLogs([]);
        setLoginUrl(null);
        try {
            const resp = await cloudflareLogin();
            await consumeSSEStream(resp, {
                onLog: (line) => setLoginLogs(prev => [...prev, line]),
                onError: (line) => setLoginLogs(prev => [...prev, line]),
                onDone: (message) => {
                    setLoginLogs(prev => [...prev, { text: message }]);
                    loadStatus();
                },
                onCustom: (data) => {
                    if (data.type === 'auth_url') {
                        setLoginUrl(data.url);
                    }
                },
            });
        } catch (err) {
            setLoginLogs(prev => [...prev, { text: String(err), error: true }]);
        }
        setLoggingIn(false);
    };

    const handleCreate = async () => {
        if (!newTunnelName.trim()) return;
        setCreating(true);
        try {
            await createTunnel(newTunnelName.trim());
            setNewTunnelName('');
            setShowCreate(false);
            const t = await fetchTunnels();
            setTunnels(t || []);
        } catch (err) {
            setError(String(err));
        }
        setCreating(false);
    };

    const handleUpload = async (files: FileList | null) => {
        if (!files || files.length === 0) return;
        setUploading(true);
        setUploadMessage(null);
        try {
            const formData = new FormData();
            for (let i = 0; i < files.length; i++) {
                formData.append('files', files[i]);
            }
            const resp = await fetch('/api/cloudflare/upload', {
                method: 'POST',
                body: formData,
            });
            const data = await resp.json();
            if (!resp.ok) {
                setUploadMessage(`Error: ${data.message || resp.statusText}`);
            } else {
                setUploadMessage(data.message);
                loadStatus();
            }
        } catch (err) {
            setUploadMessage(`Upload failed: ${String(err)}`);
        }
        setUploading(false);
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }
    };

    const handleDelete = async (name: string) => {
        if (!confirm(`Delete tunnel "${name}"? This cannot be undone.`)) return;
        setDeleting(name);
        try {
            await deleteTunnel(name);
            const t = await fetchTunnels();
            setTunnels(t || []);
        } catch (err) {
            setError(String(err));
        }
        setDeleting(null);
    };

    const handleAddDomain = async () => {
        if (!newDomain.trim()) return;
        const domain = newDomain.trim();
        if (userDomains.includes(domain)) {
            setDomainsError('Domain already exists');
            return;
        }
        
        setAddingDomain(true);
        setDomainsError(null);
        
        try {
            const updatedDomains = [...userDomains, domain];
            const domainEntries: DomainEntry[] = updatedDomains.map(d => ({ domain: d, provider: 'cloudflare' }));
            await saveDomains({ domains: domainEntries });
            setUserDomains(updatedDomains);
            setNewDomain('');
        } catch (err) {
            setDomainsError(err instanceof Error ? err.message : String(err));
        }
        setAddingDomain(false);
    };

    const handleRemoveDomain = async (domain: string) => {
        if (!confirm(`Remove domain "${domain}"?`)) return;
        
        try {
            const updatedDomains = userDomains.filter(d => d !== domain);
            const domainEntries: DomainEntry[] = updatedDomains.map(d => ({ domain: d, provider: 'cloudflare' }));
            await saveDomains({ domains: domainEntries });
            setUserDomains(updatedDomains);
        } catch (err) {
            setDomainsError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleGenerateRandomDomain = async () => {
        try {
            const randomDomain = await fetchRandomDomain();
            setNewDomain(randomDomain);
        } catch (err) {
            setDomainsError(err instanceof Error ? err.message : String(err));
        }
    };

    if (loading) {
        return <div className="cf-loading">Checking cloudflared status...</div>;
    }
    if (error) {
        return <div className="cf-error">{error}</div>;
    }
    if (!status) {
        return null;
    }

    return (
        <>
            {/* Auth Status */}
            <div className="cf-section">
                <div className="cf-section-title">Authentication</div>
                <div className="cf-auth-status">
                    <span className={`cf-status-dot ${status.authenticated ? 'ok' : 'error'}`} />
                    <span className="cf-status-text">
                        {status.authenticated
                            ? 'Authenticated with Cloudflare'
                            : status.error || 'Not authenticated'}
                    </span>
                </div>
                {!status.authenticated && (
                    <button
                        className="cf-login-btn"
                        onClick={handleLogin}
                        disabled={loggingIn}
                    >
                        {loggingIn ? 'Logging in...' : 'Login'}
                    </button>
                )}
                {loginUrl && (
                    <div className="cf-login-url">
                        <p>Open this link to authorize:</p>
                        <a href={loginUrl} target="_blank" rel="noopener noreferrer" className="cf-login-link">
                            Authorize with Cloudflare
                        </a>
                    </div>
                )}
                {loginLogs.length > 0 && (
                    <div className="cf-login-logs">
                        <LogViewer
                            lines={loginLogs}
                            pending={loggingIn}
                            pendingMessage="Waiting for authentication..."
                            maxHeight={200}
                        />
                    </div>
                )}
            </div>

            {/* Upload Existing Auth Files */}
            <div className="cf-section">
                <div className="cf-section-title">Use Existing Authentication</div>
                <p className="cf-section-desc">
                    If you have previously downloaded Cloudflare auth files (cert.pem, tunnel JSON), you can upload them here to reuse.
                </p>
                <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    accept=".pem,.json"
                    style={{ display: 'none' }}
                    onChange={(e) => handleUpload(e.target.files)}
                />
                <button
                    className="cf-upload-btn"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading}
                >
                    {uploading ? 'Uploading...' : 'Upload Auth Files'}
                </button>
                {uploadMessage && (
                    <div className={`cf-upload-message ${uploadMessage.startsWith('Error') || uploadMessage.startsWith('Upload failed') ? 'error' : 'success'}`}>
                        {uploadMessage}
                    </div>
                )}
            </div>

            {/* Credential Files */}
            {status.authenticated && status.cert_files && status.cert_files.length > 0 && (
                <div className="cf-section">
                    <div className="cf-section-title">Credential Files</div>
                    <div className="cf-cert-list">
                        {status.cert_files.map(file => (
                            <div key={file.name} className="cf-cert-card">
                                <div className="cf-cert-info">
                                    <span className="cf-cert-name">{file.name}</span>
                                    <span className="cf-cert-path">{file.path}</span>
                                    <span className="cf-cert-size">{formatBytes(file.size)}</span>
                                </div>
                                <a
                                    className="cf-download-btn"
                                    href={`/api/cloudflare/download?name=${encodeURIComponent(file.name)}`}
                                    download={file.name}
                                >
                                    Download
                                </a>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* My Own Domains */}
            {status.authenticated && (
                <div className="cf-section">
                    <div className="cf-section-header">
                        <div className="cf-section-title">My Own Domains</div>
                        <span className="cf-section-subtitle">
                            Configure domains to use for port forwarding
                        </span>
                    </div>
                    <p className="cf-section-desc">
                        When Cloudflare is authenticated, these domains will be preferred for port forwarding instead of random subdomains.
                    </p>
                    
                    {/* Add Domain Form */}
                    <div className="cf-domain-add-form">
                        <input
                            className="cf-input"
                            type="text"
                            placeholder="e.g., app.example.com"
                            value={newDomain}
                            onChange={(e) => setNewDomain(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleAddDomain()}
                            disabled={addingDomain}
                        />
                        <button
                            className="cf-generate-btn"
                            onClick={handleGenerateRandomDomain}
                            disabled={addingDomain}
                            title="Generate random subdomain"
                        >
                            🎲 Random
                        </button>
                        <button
                            className="cf-add-domain-btn"
                            onClick={handleAddDomain}
                            disabled={addingDomain || !newDomain.trim()}
                        >
                            {addingDomain ? 'Adding...' : 'Add Domain'}
                        </button>
                    </div>
                    
                    {domainsError && (
                        <div className="cf-error-message">{domainsError}</div>
                    )}
                    
                    {/* Domain List */}
                    {domainsLoading ? (
                        <div className="cf-empty">Loading domains...</div>
                    ) : userDomains.length === 0 ? (
                        <div className="cf-empty">No custom domains configured.</div>
                    ) : (
                        <div className="cf-domains-list">
                            {userDomains.map(domain => (
                                <div key={domain} className="cf-domain-card">
                                    <div className="cf-domain-info">
                                        <span className="cf-domain-name">{domain}</span>
                                    </div>
                                    <button
                                        className="cf-remove-btn"
                                        onClick={() => handleRemoveDomain(domain)}
                                    >
                                        Remove
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {/* Tunnels */}
            {status.authenticated && (
                <div className="cf-section">
                    <div className="cf-section-header">
                        <div className="cf-section-title">Tunnels</div>
                        <button
                            className="cf-add-btn"
                            onClick={() => setShowCreate(!showCreate)}
                        >
                            {showCreate ? 'Cancel' : '+ Create'}
                        </button>
                    </div>

                    {showCreate && (
                        <div className="cf-create-form">
                            <input
                                className="cf-input"
                                type="text"
                                placeholder="Tunnel name"
                                value={newTunnelName}
                                onChange={(e) => setNewTunnelName(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                                disabled={creating}
                            />
                            <button
                                className="cf-create-btn"
                                onClick={handleCreate}
                                disabled={creating || !newTunnelName.trim()}
                            >
                                {creating ? 'Creating...' : 'Create'}
                            </button>
                        </div>
                    )}

                    {tunnels.length === 0 ? (
                        <div className="cf-empty">No tunnels configured.</div>
                    ) : (
                        <div className="cf-tunnels-list">
                            {tunnels.map(tunnel => (
                                <div key={tunnel.id} className="cf-tunnel-card">
                                    <div className="cf-tunnel-info">
                                        <span className="cf-tunnel-name">{tunnel.name}</span>
                                        <span className="cf-tunnel-id">{tunnel.id.slice(0, 8)}...</span>
                                    </div>
                                    <button
                                        className="cf-delete-btn"
                                        onClick={() => handleDelete(tunnel.name)}
                                        disabled={deleting === tunnel.name}
                                    >
                                        {deleting === tunnel.name ? 'Deleting...' : 'Delete'}
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </>
    );
}

/** Full page wrapper with header and back button */
export function CloudflareSettingsView() {
    const navigate = useNavigate();

    return (
        <div className="cf-settings-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Cloudflare Settings</h2>
            </div>
            <CloudflareSettingsContent />
        </div>
    );
}

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
