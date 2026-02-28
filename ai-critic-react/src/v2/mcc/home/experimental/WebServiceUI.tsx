import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { consumeSSEStream } from '../../../../api/sse';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { BeakerIcon } from '../../../icons';
import { EnterFocusIcon, ExitFocusIcon } from '../../../../pure-view/icon';
import { ExternalAgentLink } from '../../../../pure-view/link';
import { ExternalIFrame } from './ExternalIFrame';
import './WebServiceUI.css';

export interface WebServiceUIProps {
    port?: number;
    title?: string;
    statusEndpoint?: string;
    startEndpoint?: string;
    stopEndpoint?: string;
    statusStreamEndpoint?: string;
    startStreamEndpoint?: string;
    stopStreamEndpoint?: string;
    installCommand?: string;
    authHint?: string;
    startCommandPrefix?: string;
    backPath?: string;
    enableFocusMode?: boolean;
    iframePersistenceKey?: string;
}

interface WebServiceStatusResponse {
    running: boolean;
    port: number;
}

interface WebServiceActionResponse {
    success?: boolean;
    message?: string;
    error?: string;
}

async function fetchWebServiceStatus(statusEndpoint: string, serviceTitle: string): Promise<WebServiceStatusResponse> {
    const resp = await fetch(statusEndpoint);
    if (!resp.ok) {
        throw new Error(`Failed to fetch ${serviceTitle} status`);
    }
    return resp.json();
}

export function WebServiceUI({
    port = 3000,
    title = 'Web Service',
    statusEndpoint = '/api/codex-web/status',
    startEndpoint = '/api/codex-web/start',
    stopEndpoint = '/api/codex-web/stop',
    statusStreamEndpoint,
    startStreamEndpoint,
    stopStreamEndpoint,
    installCommand = 'npm install -g codex-web-local',
    authHint = 'Make sure the CLI is installed and authenticated',
    startCommandPrefix = 'codex-web-local --port',
    backPath = '../experimental',
    enableFocusMode = false,
    iframePersistenceKey,
}: WebServiceUIProps) {
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [serverStatus, setServerStatus] = useState<'checking' | 'running' | 'not-running'>('checking');
    const [resolvedPort, setResolvedPort] = useState(port);
    const [actionState, setActionState] = useState<'refresh' | 'start' | 'stop' | null>(null);
    const [actionMessage, setActionMessage] = useState<string | null>(null);
    const [actionError, setActionError] = useState<string | null>(null);
    const [actionLogs, setActionLogs] = useState<LogLine[]>([]);

    const refreshStatus = async () => {
        setServerStatus('checking');
        try {
            const status = await fetchWebServiceStatus(statusEndpoint, title);
            const nextPort = status.port || port;
            setResolvedPort(nextPort);
            if (status.running) {
                setServerStatus('running');
                setError(null);
                setIsLoading(true);
                return;
            }
            setServerStatus('not-running');
            setError(`${title} server is not running on port ${nextPort}. Click Start Server to launch it.`);
        } catch (e) {
            const message = e instanceof Error ? e.message : `Failed to fetch ${title} status`;
            setServerStatus('not-running');
            setError(message);
            setActionError(message);
        }
    };

    useEffect(() => {
        void refreshStatus();
    }, [port]);

    const runStreamAction = async (
        endpoint: string,
        method: 'GET' | 'POST',
        pendingMessage: string,
    ) => {
        setActionLogs([]);
        setActionError(null);
        setActionMessage(pendingMessage);
        let success = false;

        try {
            const resp = await fetch(endpoint, { method });
            if (!resp.ok) {
                const text = await resp.text();
                throw new Error(text || `Request failed (${resp.status})`);
            }

            await consumeSSEStream(resp, {
                onLog: (line) => setActionLogs(prev => [...prev, line]),
                onError: (line) => {
                    setActionLogs(prev => [...prev, line]);
                    setActionError(line.text);
                },
                onDone: (message, data) => {
                    if (message) {
                        setActionLogs(prev => [...prev, { text: message }]);
                        setActionMessage(message);
                    }
                    success = data.success !== 'false';
                    if (!success && data.message) {
                        setActionError(data.message);
                    }
                },
            });
            return success;
        } catch (e) {
            const message = e instanceof Error ? e.message : String(e);
            setActionLogs(prev => [...prev, { text: message, error: true }]);
            setActionError(message);
            return false;
        }
    };

    const handleStartServer = async () => {
        setActionState('start');
        setActionError(null);
        setActionMessage(`Starting ${title} server on port ${resolvedPort}...`);
        try {
            if (startStreamEndpoint) {
                const success = await runStreamAction(
                    startStreamEndpoint,
                    'POST',
                    `Starting ${title} server on port ${resolvedPort}...`,
                );
                if (success) {
                    await refreshStatus();
                }
                return;
            }

            const resp = await fetch(startEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ port }),
            });
            const data = await resp.json().catch(() => ({} as WebServiceActionResponse));
            if (!resp.ok) {
                throw new Error(data?.message || `Failed to start ${title} server`);
            }
            setActionMessage(data?.message || `${title} server started`);
            await refreshStatus();
        } catch (e) {
            setActionError(e instanceof Error ? e.message : String(e));
        } finally {
            setActionState(null);
        }
    };

    const handleStopServer = async () => {
        setActionState('stop');
        setActionError(null);
        setActionMessage(`Stopping ${title} server...`);
        try {
            if (stopStreamEndpoint) {
                const success = await runStreamAction(
                    stopStreamEndpoint,
                    'POST',
                    `Stopping ${title} server...`,
                );
                if (success) {
                    await refreshStatus();
                }
                return;
            }

            const resp = await fetch(stopEndpoint, {
                method: 'POST',
            });
            const data = await resp.json().catch(() => ({} as WebServiceActionResponse));
            if (!resp.ok) {
                throw new Error(data?.message || `Failed to stop ${title} server`);
            }
            setActionMessage(data?.message || `${title} server stopped`);
            await refreshStatus();
        } catch (e) {
            setActionError(e instanceof Error ? e.message : String(e));
        } finally {
            setActionState(null);
        }
    };

    const handleRefreshStatus = async () => {
        setActionState('refresh');
        setActionError(null);
        setActionMessage('Refreshing server status...');
        try {
            if (statusStreamEndpoint) {
                await runStreamAction(statusStreamEndpoint, 'GET', 'Refreshing server status...');
            }
            await refreshStatus();
        } finally {
            setActionState(null);
        }
    };

    const actionBusy = actionState !== null;
    const targetUrl = `http://localhost:${resolvedPort}`;
    const effectiveIframeKey = iframePersistenceKey || statusEndpoint || title;
    const focusMode = enableFocusMode && searchParams.get('focus') === '1';
    const hidePanels = enableFocusMode && focusMode;
    const setFocusMode = (next: boolean) => {
        const nextParams = new URLSearchParams(searchParams);
        if (next) {
            nextParams.set('focus', '1');
        } else {
            nextParams.delete('focus');
        }
        setSearchParams(nextParams, { replace: true });
    };
    const handleIframeLoadingChange = useCallback((loading: boolean) => {
        setIsLoading(loading);
    }, []);
    const handleIframeError = useCallback((message: string) => {
        setIsLoading(false);
        setError(message);
    }, []);

    return (
        <div className={`codex-web-view${hidePanels ? ' codex-web-view-focus' : ''}`}>
            {!hidePanels && (
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={() => navigate(backPath)}>&larr;</button>
                    <BeakerIcon className="mcc-header-icon" />
                    <h2>{title}</h2>
                    <div className="mcc-header-status">
                        <span className={`mcc-status-dot mcc-status-${serverStatus}`}></span>
                        <span className="mcc-status-text">
                            {serverStatus === 'checking' && 'Checking...'}
                            {serverStatus === 'running' && 'Connected'}
                            {serverStatus === 'not-running' && 'Disconnected'}
                        </span>
                    </div>
                    {enableFocusMode && (
                        <button
                            className="codex-web-focus-toggle"
                            onClick={() => setFocusMode(true)}
                            title="Enter focus mode"
                            aria-label="Enter focus mode"
                        >
                            <EnterFocusIcon className="codex-web-focus-icon" />
                        </button>
                    )}
                </div>
            )}

            <div className="codex-web-content">
                {!hidePanels && (
                    <>
                        <div className="codex-web-controls">
                            <button className="mcc-btn-secondary" onClick={handleRefreshStatus} disabled={actionBusy}>
                                {actionState === 'refresh' ? 'Refreshing...' : 'Refresh Status'}
                            </button>
                            <button
                                className="mcc-btn-primary"
                                onClick={handleStartServer}
                                disabled={actionBusy || serverStatus === 'running'}
                            >
                                {actionState === 'start' ? 'Starting...' : 'Start Server'}
                            </button>
                            <button
                                className="mcc-btn-secondary"
                                onClick={handleStopServer}
                                disabled={actionBusy || serverStatus !== 'running'}
                            >
                                {actionState === 'stop' ? 'Stopping...' : 'Stop Server'}
                            </button>
                        </div>
                        {actionMessage && <div className="codex-web-action-status">{actionMessage}</div>}
                        {actionError && <div className="codex-web-action-error">{actionError}</div>}
                        {(actionLogs.length > 0 || actionBusy) && (
                            <div className="codex-web-log-panel">
                                <div className="codex-web-log-title">Action Logs</div>
                                <LogViewer
                                    lines={actionLogs}
                                    pending={actionBusy}
                                    pendingMessage="Streaming logs..."
                                    maxHeight={180}
                                />
                            </div>
                        )}
                        {serverStatus === 'running' && (
                            <div className="codex-web-target-link">
                                <ExternalAgentLink href={targetUrl}>
                                    Open target server in new tab ↗
                                </ExternalAgentLink>
                            </div>
                        )}
                    </>
                )}
                <div className="codex-web-main">
                    {hidePanels && (
                        <button
                            className="codex-web-focus-toggle codex-web-focus-toggle-floating"
                            onClick={() => setFocusMode(false)}
                            title="Exit focus mode"
                            aria-label="Exit focus mode"
                        >
                            <ExitFocusIcon className="codex-web-focus-icon" />
                        </button>
                    )}
                    {error ? (
                        <div className="codex-web-error">
                            <div className="codex-web-error-icon">⚠️</div>
                            <h3>Connection Error</h3>
                            <p>{error}</p>
                            <div className="codex-web-error-actions">
                                <button className="mcc-btn-primary" onClick={() => setError(null)}>
                                    Dismiss
                                </button>
                                <button className="mcc-btn-secondary" onClick={handleRefreshStatus} disabled={actionBusy}>
                                    Refresh Status
                                </button>
                            </div>
                            <div className="codex-web-error-help">
                                <h4>Quick Start Guide:</h4>
                                <ol>
                                    <li>Install required package: <code>{installCommand}</code></li>
                                    <li>{authHint}</li>
                                    <li>Start the server: <code>{startCommandPrefix} {resolvedPort}</code></li>
                                    <li>Click <strong>Refresh Status</strong> above</li>
                                </ol>
                            </div>
                        </div>
                    ) : (
                        <>
                            {isLoading && serverStatus === 'running' && (
                                <div className="codex-web-loading">
                                    <div className="mcc-loading-spinner"></div>
                                    <span>{`Loading ${title} UI...`}</span>
                                </div>
                            )}
                            <ExternalIFrame
                                active={serverStatus === 'running'}
                                targetUrl={targetUrl}
                                title={title}
                                persistenceKey={effectiveIframeKey}
                                onLoadingChange={handleIframeLoadingChange}
                                onError={handleIframeError}
                            />
                        </>
                    )}
                </div>
            </div>
        </div>
    );
}
