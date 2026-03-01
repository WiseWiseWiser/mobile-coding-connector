import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchProxyConfig, type ProxyServer } from '../../../../api/proxyConfig';
import { FlexInput } from '../../../../pure-view/FlexInput';
import { BackIcon } from '../../../icons';
import './ProxyTestView.css';

export function ProxyTestView() {
    const navigate = useNavigate();
    const [servers, setServers] = useState<ProxyServer[]>([]);
    const [loading, setLoading] = useState(true);
    
    const [selectedServerId, setSelectedServerId] = useState<string>('');
    const [proxyEnabled, setProxyEnabled] = useState(true);
    
    const [formHost, setFormHost] = useState('');
    const [formPort, setFormPort] = useState('');
    const [formProtocol, setFormProtocol] = useState('http');
    const [formUsername, setFormUsername] = useState('');
    const [formPassword, setFormPassword] = useState('');
    
    const [targetUrl, setTargetUrl] = useState('');
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = async () => {
        try {
            const cfg = await fetchProxyConfig();
            setServers(cfg.servers || []);
            setProxyEnabled(cfg.enabled);
            setLoading(false);
        } catch (err) {
            setLoading(false);
        }
    };

    const handleServerSelect = (serverId: string) => {
        setSelectedServerId(serverId);
        if (serverId) {
            const server = servers.find(s => s.id === serverId);
            if (server) {
                setFormHost(server.host);
                setFormPort(server.port.toString());
                setFormProtocol(server.protocol || 'http');
                setFormUsername(server.username || '');
                setFormPassword(server.password || '');
            }
        }
    };

    const handleTest = async () => {
        if (!formHost || !formPort || !targetUrl) {
            setTestResult({ success: false, message: 'Please fill in proxy details and target URL' });
            return;
        }

        setTesting(true);
        setTestResult(null);

        try {
            const portNum = parseInt(formPort, 10);
            const proxyUrl = `${formProtocol}://${formUsername ? `${formUsername}:${formPassword}@` : ''}${formHost}:${portNum}`;
            
            const response = await fetch('/api/proxy/test', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    proxyUrl,
                    targetUrl: targetUrl.trim(),
                    enabled: proxyEnabled,
                }),
            });

            const data = await response.json();
            
            if (response.ok && data.success) {
                setTestResult({ success: true, message: data.message || 'Connection successful!' });
            } else {
                setTestResult({ success: false, message: data.message || 'Connection failed' });
            }
        } catch (err) {
            setTestResult({ success: false, message: err instanceof Error ? err.message : 'Test failed' });
        }

        setTesting(false);
    };

    if (loading) {
        return (
            <div className="proxy-test-view">
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={() => navigate('../')}>
                        <BackIcon />
                    </button>
                    <h2>Proxy Test</h2>
                </div>
                <div className="diagnose-loading">Loading...</div>
            </div>
        );
    }

    return (
        <div className="proxy-test-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../')}>
                    <BackIcon />
                </button>
                <h2>Proxy Test</h2>
            </div>

            <div className="proxy-test-content">
                {/* Toggle to temporarily disable proxy */}
                <div className="proxy-test-section">
                    <label className="proxy-section-checkbox-label">
                        <input
                            type="checkbox"
                            checked={proxyEnabled}
                            onChange={e => setProxyEnabled(e.target.checked)}
                        />
                        <span>Enable proxy for testing</span>
                    </label>
                    <p className="proxy-section-desc">
                        Uncheck this to test connection without proxy (bypass proxy)
                    </p>
                </div>

                {/* Select existing config */}
                <div className="proxy-test-section">
                    <label className="proxy-section-label">Select Existing Proxy Config</label>
                    <select
                        className="proxy-section-select proxy-test-select"
                        value={selectedServerId}
                        onChange={e => handleServerSelect(e.target.value)}
                    >
                        <option value="">-- Select a saved proxy --</option>
                        {servers.map(server => (
                            <option key={server.id} value={server.id}>
                                {server.name} ({server.protocol || 'http'}://{server.host}:{server.port})
                            </option>
                        ))}
                    </select>
                </div>

                {/* Proxy Details Form */}
                <div className="proxy-test-section proxy-test-form">
                    <label className="proxy-section-label">Proxy Details</label>
                    
                    <div className="proxy-section-form-grid">
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
                </div>

                {/* Target URL */}
                <div className="proxy-test-section">
                    <label className="proxy-section-label">Target URL *</label>
                    <FlexInput
                        inputClassName="proxy-section-input proxy-test-target-input"
                        value={targetUrl}
                        onChange={setTargetUrl}
                        placeholder="https://api.ipify.org?format=json"
                    />
                    <p className="proxy-section-desc">
                        Enter the URL you want to test the proxy with
                    </p>
                </div>

                {/* Test Button */}
                <div className="proxy-test-section">
                    <button
                        className="proxy-section-btn proxy-test-btn"
                        onClick={handleTest}
                        disabled={testing}
                    >
                        {testing ? 'Testing...' : 'Test Connection'}
                    </button>
                </div>

                {/* Test Result */}
                {testResult && (
                    <div className={`proxy-test-result ${testResult.success ? 'success' : 'error'}`}>
                        {testResult.message}
                    </div>
                )}
            </div>
        </div>
    );
}
