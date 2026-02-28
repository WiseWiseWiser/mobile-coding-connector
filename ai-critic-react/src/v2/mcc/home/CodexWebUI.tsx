import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../icons';
import './CodexWebUI.css';

interface CodexWebUIProps {
    port?: number;
}

export function CodexWebUI({ port = 3000 }: CodexWebUIProps) {
    const navigate = useNavigate();
    const iframeRef = useRef<HTMLIFrameElement>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [serverStatus, setServerStatus] = useState<'checking' | 'running' | 'not-running'>('checking');

    // Check if codex-web-local server is running
    useEffect(() => {
        const checkServer = async () => {
            try {
                // Try to fetch from the codex-web-local server
                await fetch(`http://localhost:${port}`, {
                    method: 'HEAD',
                    mode: 'no-cors',
                });
                setServerStatus('running');
            } catch (e) {
                // Server not running
                setServerStatus('not-running');
                setError(`Codex Web server not running on port ${port}. Please start it with: npx codex-web-local --port ${port}`);
            }
        };

        checkServer();
    }, [port]);

    const handleIframeLoad = () => {
        setIsLoading(false);
    };

    const handleIframeError = () => {
        setIsLoading(false);
        setError('Failed to load Codex Web UI');
    };

    return (
        <div className="codex-web-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('/home/experimental')}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Codex Web</h2>
                <div className="mcc-header-status">
                    <span className={`mcc-status-dot mcc-status-${serverStatus}`}></span>
                    <span className="mcc-status-text">
                        {serverStatus === 'checking' && 'Checking...'}
                        {serverStatus === 'running' && 'Connected'}
                        {serverStatus === 'not-running' && 'Disconnected'}
                    </span>
                </div>
            </div>

            <div className="codex-web-content">
                {error ? (
                    <div className="codex-web-error">
                        <div className="codex-web-error-icon">⚠️</div>
                        <h3>Connection Error</h3>
                        <p>{error}</p>
                        <div className="codex-web-error-actions">
                            <button className="mcc-btn-primary" onClick={() => setError(null)}>
                                Dismiss
                            </button>
                            <button className="mcc-btn-secondary" onClick={() => window.location.reload()}>
                                Retry Connection
                            </button>
                        </div>
                        <div className="codex-web-error-help">
                            <h4>Quick Start Guide:</h4>
                            <ol>
                                <li>Install codex-web-local: <code>npm install -g codex-web-local</code></li>
                                <li>Make sure you have Codex CLI installed and authenticated</li>
                                <li>Start the server: <code>codex-web-local --port 3000</code></li>
                                <li>Refresh this page</li>
                            </ol>
                        </div>
                    </div>
                ) : (
                    <>
                        {isLoading && serverStatus === 'running' && (
                            <div className="codex-web-loading">
                                <div className="mcc-loading-spinner"></div>
                                <span>Loading Codex Web UI...</span>
                            </div>
                        )}
                        <iframe
                            ref={iframeRef}
                            src={`http://localhost:${port}`}
                            className="codex-web-iframe"
                            onLoad={handleIframeLoad}
                            onError={handleIframeError}
                            sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads"
                            title="Codex Web UI"
                        />
                    </>
                )}
            </div>
        </div>
    );
}
