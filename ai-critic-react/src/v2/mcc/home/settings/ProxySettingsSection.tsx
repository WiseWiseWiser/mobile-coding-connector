import { useState, useEffect } from 'react';
import { fetchProxyConfig, saveProxyConfig, type ProxyServer, generateProxyServerId } from '../../../../api/proxyConfig';
import { consumeSSEStream } from '../../../../api/sse';
import { FlexInput } from '../../../../pure-view/FlexInput';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { TestButton } from '../../../../pure-view/buttons';
import { parseProxyUrl } from '../../../../lib/url';
import './ProxySettingsSection.css';

export function ProxySettingsSection() {
    const [servers, setServers] = useState<ProxyServer[]>([]);
    const [enabled, setEnabled] = useState(true);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    
    // Form state for new/edit server
    const [showForm, setShowForm] = useState(false);
    const [editingId, setEditingId] = useState<string | null>(null);
    const [parseInput, setParseInput] = useState('');
    const [formName, setFormName] = useState('');
    const [formHost, setFormHost] = useState('');
    const [formPort, setFormPort] = useState('');
    const [formProtocol, setFormProtocol] = useState('http');
    const [formUsername, setFormUsername] = useState('');
    const [formPassword, setFormPassword] = useState('');
    const [formDomains, setFormDomains] = useState('');

    // Test proxy state
    const [testUrl, setTestUrl] = useState('');
    const [testRunning, setTestRunning] = useState(false);
    const [testLogs, setTestLogs] = useState<LogLine[]>([]);

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = async () => {
        try {
            const cfg = await fetchProxyConfig();
            setEnabled(cfg.enabled);
            setServers(cfg.servers || []);
            setLoading(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load proxy config');
            setLoading(false);
        }
    };

    const resetForm = () => {
        setShowForm(false);
        setEditingId(null);
        setFormName('');
        setFormHost('');
        setFormPort('');
        setFormProtocol('http');
        setFormUsername('');
        setFormPassword('');
        setFormDomains('');
    };

    const handleAdd = () => {
        setEditingId(null);
        setFormName('');
        setFormHost('');
        setFormPort('');
        setFormProtocol('http');
        setFormUsername('');
        setFormPassword('');
        setFormDomains('');
        setShowForm(true);
    };

    const handleParseUrl = () => {
        const raw = parseInput.trim();
        if (!raw) return;

        try {
            const result = parseProxyUrl(raw);
            setFormProtocol(result.protocol);
            setFormHost(result.host);
            if (result.port) setFormPort(result.port);
            if (result.username) setFormUsername(result.username);
            if (result.password) setFormPassword(result.password);
            if (!formName) setFormName(result.host);
            setError(null);
        } catch {
            setError(`Failed to parse URL: "${raw}"`);
        }
    };

    const buildProxyUrl = () => {
        const auth = formUsername ? `${formUsername}${formPassword ? `:${formPassword}` : ''}@` : '';
        return `${formProtocol}://${auth}${formHost}:${formPort}`;
    };

    const handleTestProxy = async () => {
        const target = testUrl.trim() || 'https://api.ipify.org?format=json';
        setTestRunning(true);
        setTestLogs([]);

        try {
            const resp = await fetch('/api/proxy/test', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    proxyUrl: formHost && formPort ? buildProxyUrl() : '',
                    targetUrl: target,
                    enabled: !!(formHost && formPort),
                }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                setTestLogs([{ text: text || `Request failed (${resp.status})`, error: true }]);
                setTestRunning(false);
                return;
            }

            await consumeSSEStream(resp, {
                onLog: (line) => setTestLogs(prev => [...prev, line]),
                onError: (line) => setTestLogs(prev => [...prev, line]),
                onDone: (_message, data) => {
                    const msg = data.message || '';
                    if (msg) {
                        setTestLogs(prev => [...prev, { text: msg, error: data.success === 'false' }]);
                    }
                },
            });
        } catch (e) {
            setTestLogs(prev => [...prev, { text: e instanceof Error ? e.message : String(e), error: true }]);
        }

        setTestRunning(false);
    };

    const handleEdit = (server: ProxyServer) => {
        setEditingId(server.id);
        setFormName(server.name);
        setFormHost(server.host);
        setFormPort(server.port.toString());
        setFormProtocol(server.protocol || 'http');
        setFormUsername(server.username || '');
        setFormPassword(server.password || '');
        setFormDomains(server.domains?.join('\n') || '');
        setShowForm(true);
    };

    const handleDelete = (id: string) => {
        setServers(servers.filter(s => s.id !== id));
        if (editingId === id) {
            resetForm();
        }
        setSuccess(false);
    };

    const handleSaveForm = () => {
        const name = formName.trim();
        const host = formHost.trim();
        const portNum = parseInt(formPort, 10);
        
        if (!name) {
            setError('Proxy name is required');
            return;
        }
        if (!host) {
            setError('Host is required');
            return;
        }
        if (!portNum || portNum < 1 || portNum > 65535) {
            setError('Port must be between 1 and 65535');
            return;
        }

        const domains = formDomains
            .split('\n')
            .map(d => d.trim())
            .filter(d => d.length > 0);

        const server: ProxyServer = {
            id: editingId || generateProxyServerId(),
            name,
            host,
            port: portNum,
            protocol: formProtocol || 'http',
            username: formUsername.trim() || undefined,
            password: formPassword || undefined,
            domains,
        };

        if (editingId) {
            setServers(servers.map(s => s.id === editingId ? server : s));
        } else {
            setServers([...servers, server]);
        }

        resetForm();
        setSuccess(false);
        setError(null);
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setSuccess(false);
        try {
            await saveProxyConfig({ enabled, servers });
            setSuccess(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save proxy config');
        }
        setSaving(false);
    };

    if (loading) {
        return (
            <div className="diagnose-section">
                <h3 className="diagnose-section-title">Proxy Servers</h3>
                <div className="diagnose-loading">Loading...</div>
            </div>
        );
    }

    return (
        <div className="diagnose-section">
            <h3 className="diagnose-section-title">Proxy Servers</h3>
            
            <div className="proxy-section-content">
                {/* Global Enable Toggle */}
                <div className="proxy-section-group">
                    <label className="proxy-section-checkbox-label">
                        <input
                            type="checkbox"
                            checked={enabled}
                            onChange={e => { setEnabled(e.target.checked); setSuccess(false); }}
                        />
                        <span>Enable proxy configuration for git and other tools</span>
                    </label>
                    <p className="proxy-section-desc">
                        When enabled, the system will automatically set <code>http_proxy</code> and <code>https_proxy</code>
                        environment variables based on the target domain.
                    </p>
                </div>

                {/* Proxy Servers List */}
                <div className="proxy-section-group">
                    <div className="proxy-section-list-header">
                        <label className="proxy-section-label">Configured Proxy Servers</label>
                        {!showForm && (
                            <button className="proxy-section-btn proxy-section-btn-add" onClick={handleAdd}>
                                + Add
                            </button>
                        )}
                    </div>
                    
                    {servers.length === 0 && !showForm ? (
                        <p className="proxy-section-empty">No proxy servers configured.</p>
                    ) : (
                        <div className="proxy-section-servers">
                            {servers.map(server => (
                                <div key={server.id} className="proxy-section-server-card">
                                    <div className="proxy-section-server-header">
                                        <span className="proxy-section-server-name">{server.name}</span>
                                        <div className="proxy-section-server-actions">
                                            <button
                                                className="proxy-section-server-btn"
                                                onClick={() => handleEdit(server)}
                                                title="Edit"
                                            >
                                                Edit
                                            </button>
                                            <button
                                                className="proxy-section-server-btn proxy-section-server-btn-danger"
                                                onClick={() => handleDelete(server.id)}
                                                title="Delete"
                                            >
                                                Delete
                                            </button>
                                        </div>
                                    </div>
                                    <div className="proxy-section-server-details">
                                        <code className="proxy-section-server-url">
                                            {server.protocol || 'http'}://{server.username ? `${server.username}@` : ''}{server.host}:{server.port}
                                        </code>
                                        {server.domains && server.domains.length > 0 && (
                                            <div className="proxy-section-server-domains">
                                                <span className="proxy-section-server-domains-label">Domains:</span>
                                                {server.domains.map((domain, idx) => (
                                                    <span key={idx} className="proxy-section-server-domain-tag">{domain}</span>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Add/Edit Form */}
                {showForm && <div className="proxy-section-group proxy-section-form">
                    <label className="proxy-section-label">
                        {editingId ? 'Edit Proxy Server' : 'Add New Proxy Server'}
                    </label>

                    {/* Quick Parse URL */}
                    <div className="proxy-section-parse-row">
                        <FlexInput
                            inputClassName="proxy-section-input proxy-section-parse-input"
                            value={parseInput}
                            onChange={setParseInput}
                            placeholder="http://user:pass@proxy.example.com:8080"
                            onKeyDown={e => { if (e.key === 'Enter') handleParseUrl(); }}
                        />
                        <button className="proxy-section-btn proxy-section-btn-parse" onClick={handleParseUrl}>
                            Parse
                        </button>
                    </div>
                    
                    <div className="proxy-section-form-grid">
                        <div className="proxy-section-form-field">
                            <label>Name *</label>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                value={formName}
                                onChange={setFormName}
                                placeholder="Office Proxy"
                            />
                        </div>
                        
                        <div className="proxy-section-form-field">
                            <label>Protocol</label>
                            <select
                                className="proxy-section-select"
                                value={formProtocol}
                                onChange={e => setFormProtocol(e.target.value)}
                            >
                                <option value="http">HTTP</option>
                                <option value="https">HTTPS</option>
                                <option value="socks5">SOCKS5</option>
                            </select>
                        </div>
                        
                        <div className="proxy-section-form-field">
                            <label>Host *</label>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                value={formHost}
                                onChange={setFormHost}
                                placeholder="proxy.example.com"
                            />
                        </div>
                        
                        <div className="proxy-section-form-field">
                            <label>Port *</label>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                value={formPort}
                                onChange={setFormPort}
                                placeholder="8080"
                            />
                        </div>
                        
                        <div className="proxy-section-form-field">
                            <label>Username (optional)</label>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                value={formUsername}
                                onChange={setFormUsername}
                                placeholder="username"
                            />
                        </div>
                        
                        <div className="proxy-section-form-field">
                            <label>Password (optional)</label>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                type="password"
                                value={formPassword}
                                onChange={setFormPassword}
                                placeholder="password"
                            />
                        </div>
                    </div>
                    
                    <div className="proxy-section-form-field proxy-section-form-field-full">
                        <label>Domains (one per line)</label>
                        <textarea
                            className="proxy-section-textarea"
                            value={formDomains}
                            onChange={e => setFormDomains(e.target.value)}
                            placeholder="git.example.com&#10;github.example.com"
                            rows={4}
                        />
                        <p className="proxy-section-field-desc">
                            List of domains that should use this proxy. One domain per line. 
                            For example: <code>git.garena.com</code> or <code>*.internal.example.com</code>
                        </p>
                    </div>
                    
                    {/* Usage hint */}
                    {formHost && formPort && (
                        <div className="proxy-section-usage-hint">
                            <span className="proxy-section-usage-hint-title">How this proxy will be used:</span>
                            <code>export http_proxy={formProtocol}://{formUsername ? `${formUsername}@` : ''}{formHost}:{formPort}</code>
                            <code>curl -x {formProtocol}://{formHost}:{formPort} https://example.com</code>
                            <code>git -c http.proxy={formProtocol}://{formHost}:{formPort} clone ...</code>
                        </div>
                    )}

                    {/* Test Proxy */}
                    {formHost && formPort && (
                        <div className="proxy-section-test">
                            <span className="proxy-section-test-title">Test Connection</span>
                            <FlexInput
                                inputClassName="proxy-section-input"
                                value={testUrl}
                                onChange={setTestUrl}
                                placeholder="https://api.ipify.org?format=json"
                                multiline
                                rows={1}
                                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey && testUrl.trim()) { e.preventDefault(); handleTestProxy(); } }}
                            />
                            <TestButton
                                onClick={handleTestProxy}
                                disabled={!testUrl.trim()}
                                running={testRunning}
                            />
                            {testLogs.length > 0 && (
                                <LogViewer lines={testLogs} pending={testRunning} pendingMessage="Running..." maxHeight={200} />
                            )}
                        </div>
                    )}

                    <div className="proxy-section-form-actions">
                        <button
                            className="proxy-section-btn proxy-section-btn-secondary"
                            onClick={resetForm}
                        >
                            Cancel
                        </button>
                        <button
                            className="proxy-section-btn"
                            onClick={handleSaveForm}
                        >
                            {editingId ? 'Update Proxy Server' : 'Add Proxy Server'}
                        </button>
                    </div>
                </div>}

                {error && <div className="proxy-section-error">{error}</div>}
                {success && <div className="proxy-section-success">Proxy configuration saved successfully!</div>}

                <div className="proxy-section-actions">
                    <button
                        className="mcc-port-action-btn"
                        onClick={handleSave}
                        disabled={saving}
                    >
                        {saving ? 'Saving...' : 'Save Configuration'}
                    </button>
                </div>
            </div>
        </div>
    );
}
