import { useState, useEffect } from 'react';
import {
    fetchExposedURLs,
    addExposedURL,
    updateExposedURL,
    deleteExposedURL,
    toggleExposedURL,
    fetchExposedURLsCloudflareStatus,
    startExposedURLTunnel,
    stopExposedURLTunnel,
} from '../../../../api/exposedUrls';
import type { ExposedURLWithStatus, CloudflareStatus } from '../../../../api/exposedUrls';
import { FlexInput } from '../../../../pure-view/FlexInput';
import './ExposedUrlsSection.css';

export function ExposedUrlsSection() {
    const [urls, setUrls] = useState<ExposedURLWithStatus[]>([]);
    const [cfStatus, setCfStatus] = useState<CloudflareStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Add form state
    const [showAddForm, setShowAddForm] = useState(false);
    const [newExternalDomain, setNewExternalDomain] = useState('');
    const [newInternalURL, setNewInternalURL] = useState('');

    // Edit form state
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editExternalDomain, setEditExternalDomain] = useState('');
    const [editInternalURL, setEditInternalURL] = useState('');

    const loadData = async () => {
        setLoading(true);
        setError(null);
        try {
            const [urlsData, statusData] = await Promise.all([
                fetchExposedURLs(),
                fetchExposedURLsCloudflareStatus(),
            ]);
            setUrls(urlsData);
            setCfStatus(statusData);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, []);

    // Poll for status updates when there are connecting/active tunnels
    useEffect(() => {
        const hasActiveOrConnecting = urls.some(d => d.status === 'connecting' || d.status === 'active');
        if (!hasActiveOrConnecting) return;

        const timer = setInterval(() => {
            fetchExposedURLs()
                .then(data => setUrls(data))
                .catch(() => {});
        }, 3000);
        return () => clearInterval(timer);
    }, [urls]);

    const handleAdd = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newExternalDomain || !newInternalURL) return;

        setError(null);
        try {
            await addExposedURL(newExternalDomain, newInternalURL);
            const data = await fetchExposedURLs();
            setUrls(data);
            setNewExternalDomain('');
            setNewInternalURL('');
            setShowAddForm(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleEdit = (url: ExposedURLWithStatus) => {
        setEditingId(url.id);
        setEditExternalDomain(url.external_domain);
        setEditInternalURL(url.internal_url);
    };

    const handleSaveEdit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!editingId || !editExternalDomain || !editInternalURL) return;

        setError(null);
        try {
            await updateExposedURL(editingId, editExternalDomain, editInternalURL);
            const data = await fetchExposedURLs();
            setUrls(data);
            setEditingId(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this exposed URL?')) return;

        setError(null);
        try {
            await deleteExposedURL(id);
            const data = await fetchExposedURLs();
            setUrls(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleStart = async (id: string) => {
        setError(null);
        try {
            await startExposedURLTunnel(id);
            const data = await fetchExposedURLs();
            setUrls(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleStop = async (id: string) => {
        setError(null);
        try {
            await stopExposedURLTunnel(id);
            const data = await fetchExposedURLs();
            setUrls(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleToggle = async (id: string, disabled: boolean) => {
        setError(null);
        try {
            await toggleExposedURL(id, disabled);
            const data = await fetchExposedURLs();
            setUrls(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const getStatusClass = (status: string) => {
        switch (status) {
            case 'active': return 'status-active';
            case 'connecting': return 'status-connecting';
            case 'error': return 'status-error';
            default: return 'status-stopped';
        }
    };

    const getStatusText = (status: string) => {
        switch (status) {
            case 'active': return 'Active';
            case 'connecting': return 'Connecting...';
            case 'error': return 'Error';
            default: return 'Stopped';
        }
    };

    return (
        <div className="diagnose-section">
            <h3 className="diagnose-section-title">Exposed URLs</h3>

            {loading ? (
                <div className="diagnose-loading">Loading exposed URLs...</div>
            ) : (
                <div className="exposed-urls-card">
                    {/* Cloudflare Status */}
                    <div className="exposed-urls-cf-status">
                        <span className="exposed-urls-cf-label">Cloudflare Status:</span>
                        {cfStatus ? (
                            <span className={`exposed-urls-cf-badge ${cfStatus.installed ? 'installed' : 'not-installed'}`}>
                                {cfStatus.installed 
                                    ? (cfStatus.authenticated ? 'Installed & Authenticated' : 'Installed (Not Authenticated)')
                                    : 'Not Installed'}
                            </span>
                        ) : (
                            <span className="exposed-urls-cf-badge unknown">Unknown</span>
                        )}
                    </div>

                    {/* URL List */}
                    {urls.length === 0 && !showAddForm && (
                        <div className="exposed-urls-empty">
                            No exposed URLs configured. Add one to expose internal services via Cloudflare tunnel.
                        </div>
                    )}

                    {urls.length > 0 && (
                        <div className="exposed-urls-list">
                            {urls.map((url) => (
                                <div key={url.id} className="exposed-url-item">
                                    {editingId === url.id ? (
                                        <form onSubmit={handleSaveEdit} className="exposed-url-edit-form">
                                            <div className="exposed-url-field">
                                                <label>External Domain:</label>
                                                <FlexInput
                                                    value={editExternalDomain}
                                                    onChange={setEditExternalDomain}
                                                    placeholder="e.g., myapp.example.com"
                                                />
                                            </div>
                                            <div className="exposed-url-field">
                                                <label>Internal URL:</label>
                                                <FlexInput
                                                    value={editInternalURL}
                                                    onChange={setEditInternalURL}
                                                    placeholder="e.g., http://localhost:3000 or tcp://my.squid.com:3128"
                                                />
                                            </div>
                                            <div className="exposed-url-actions">
                                                <button type="submit" className="exposed-url-btn save">Save</button>
                                                <button type="button" className="exposed-url-btn cancel" onClick={() => setEditingId(null)}>Cancel</button>
                                            </div>
                                        </form>
                                    ) : (
                                        <div className="exposed-url-view">
                                            <div className="exposed-url-info">
                                                <div className="exposed-url-row">
                                                    <span className="exposed-url-label">External:</span>
                                                    <a 
                                                        href={`https://${url.external_domain}`} 
                                                        target="_blank" 
                                                        rel="noopener noreferrer"
                                                        className="exposed-url-link"
                                                    >
                                                        {url.external_domain}
                                                    </a>
                                                </div>
                                                <div className="exposed-url-row">
                                                    <span className="exposed-url-label">Internal:</span>
                                                    <span className="exposed-url-value">{url.internal_url}</span>
                                                </div>
                                                <div className="exposed-url-row">
                                                    <span className="exposed-url-label">Status:</span>
                                                    <span className={`exposed-url-status ${getStatusClass(url.status)}`}>
                                                        {getStatusText(url.status)}
                                                    </span>
                                                </div>
                                                {url.tunnel_url && (
                                                    <div className="exposed-url-row">
                                                        <span className="exposed-url-label">Tunnel:</span>
                                                        <a 
                                                            href={url.tunnel_url} 
                                                            target="_blank" 
                                                            rel="noopener noreferrer"
                                                            className="exposed-url-link"
                                                        >
                                                            {url.tunnel_url}
                                                        </a>
                                                    </div>
                                                )}
                                                {url.error && (
                                                    <div className="exposed-url-row">
                                                        <span className="exposed-url-label">Error:</span>
                                                        <span className="exposed-url-error">{url.error}</span>
                                                    </div>
                                                )}
                                            </div>
                                            <div className="exposed-url-toggle">
                                                <label>
                                                    <input
                                                        type="checkbox"
                                                        checked={!url.disabled}
                                                        onChange={(e) => handleToggle(url.id, !e.target.checked)}
                                                    />
                                                    <span>Enabled</span>
                                                </label>
                                            </div>
                                            <div className="exposed-url-actions">
                                                {url.status === 'active' ? (
                                                    <button 
                                                        className="exposed-url-btn stop"
                                                        onClick={() => handleStop(url.id)}
                                                    >
                                                        Stop
                                                    </button>
                                                ) : (
                                                    <button 
                                                        className="exposed-url-btn start"
                                                        onClick={() => handleStart(url.id)}
                                                        disabled={!cfStatus?.installed || !cfStatus?.authenticated}
                                                    >
                                                        Start
                                                    </button>
                                                )}
                                                <button 
                                                    className="exposed-url-btn edit"
                                                    onClick={() => handleEdit(url)}
                                                >
                                                    Edit
                                                </button>
                                                <button 
                                                    className="exposed-url-btn delete"
                                                    onClick={() => handleDelete(url.id)}
                                                >
                                                    Delete
                                                </button>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}

                    {error && <div className="exposed-urls-error">{error}</div>}

                    {/* Add Form */}
                    {showAddForm ? (
                        <form onSubmit={handleAdd} className="exposed-urls-add-form">
                            <h4>Add New Exposed URL</h4>
                            <div className="exposed-url-field">
                                <label>External Domain:</label>
                                <FlexInput
                                    value={newExternalDomain}
                                    onChange={setNewExternalDomain}
                                    placeholder="e.g., myapp.example.com"
                                />
                                <small>The domain that will be exposed publicly</small>
                            </div>
                            <div className="exposed-url-field">
                                <label>Internal URL:</label>
                                <FlexInput
                                    value={newInternalURL}
                                    onChange={setNewInternalURL}
                                    placeholder="e.g., http://localhost:3000 or tcp://my.squid.com:3128"
                                />
                                <small>Internal service (use tcp:// for proxy servers)</small>
                            </div>
                            <div className="exposed-urls-add-actions">
                                <button type="submit" className="exposed-url-btn save">Add URL</button>
                                <button type="button" className="exposed-url-btn cancel" onClick={() => setShowAddForm(false)}>Cancel</button>
                            </div>
                        </form>
                    ) : (
                        <button className="exposed-urls-add-toggle" onClick={() => setShowAddForm(true)}>
                            + Add Exposed URL
                        </button>
                    )}
                </div>
            )}
        </div>
    );
}
