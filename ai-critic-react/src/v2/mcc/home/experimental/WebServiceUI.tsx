import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { OpencodeWebTargetPreferences } from '../../../../api/agents.ts';
import type { OpencodeWebTargetPreference } from '../../../../api/agents.ts';
import { consumeSSEStream } from '../../../../api/sse.ts';
import { LogViewer } from '../../../LogViewer.tsx';
import type { LogLine } from '../../../LogViewer.tsx';
import { BeakerIcon } from '../../../../pure-view/icons/BeakerIcon';
import { OpenInNewIcon } from '../../../../pure-view/icons/OpenInNewIcon';
import { SettingsIcon } from '../../../../pure-view/icons/SettingsIcon';
import { RefreshIcon } from '../../../../pure-view/icons/RefreshIcon';
import { EnterFocusIcon } from '../../../../pure-view/icons/EnterFocusIcon';
import { ExitFocusIcon } from '../../../../pure-view/icons/ExitFocusIcon';
import { ExternalIFrame } from './ExternalIFrame.tsx';
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
    settingsPath?: string;
}

interface WebServiceStatusResponse {
    running: boolean;
    port: number;
    domain?: string;
    port_mapped?: boolean;
    target_preference?: OpencodeWebTargetPreference;
    exposed_domain?: string;
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

function normalizeWebTargetUrl(urlLike: string, protocol: 'http' | 'https'): string {
    const trimmed = urlLike.trim();
    if (!trimmed) {
        return '';
    }
    if (/^https?:\/\//i.test(trimmed)) {
        return trimmed;
    }
    return `${protocol}://${trimmed}`;
}

function resolveProtocol(urlLike: string): 'http' | 'https' {
    const trimmed = urlLike.trim();
    if (!trimmed) {
        return 'https';
    }
    const withoutScheme = trimmed.replace(/^https?:\/\//i, '');
    const hostPart = withoutScheme.split('/')[0].split('?')[0];
    const host = hostPart.split(':')[0].toLowerCase();
    if (host === 'localhost' || host === '127.0.0.1') {
        return 'http';
    }
    return 'https';
}

function resolveMappedTargetUrl(status: WebServiceStatusResponse): string {
    const exposedDomain = status.exposed_domain?.trim();
    if (exposedDomain) {
        return normalizeWebTargetUrl(exposedDomain, resolveProtocol(exposedDomain));
    }

    const configuredDomain = status.domain?.trim();
    if (!configuredDomain) {
        return '';
    }

    return normalizeWebTargetUrl(configuredDomain, resolveProtocol(configuredDomain));
}

function resolveWebServiceTargetUrl(status: WebServiceStatusResponse, fallbackPort: number): string {
    if (status.target_preference === OpencodeWebTargetPreferences.Localhost) {
        return `http://localhost:${fallbackPort}`;
    }

    const mappedTargetUrl = resolveMappedTargetUrl(status);
    if (mappedTargetUrl) {
        return mappedTargetUrl;
    }
    return `http://localhost:${fallbackPort}`;
}

function DismissIcon() {
    return (
        <svg className="codex-web-dismiss-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12" />
            <path d="M18 6L6 18" />
        </svg>
    );
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
    settingsPath,
}: WebServiceUIProps) {
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [serverStatus, setServerStatus] = useState<'checking' | 'running' | 'not-running'>('checking');
    const [resolvedPort, setResolvedPort] = useState(port);
    const [targetUrl, setTargetUrl] = useState(`http://localhost:${port}`);
    const [actionState, setActionState] = useState<'refresh' | 'start' | 'stop' | null>(null);
    const [actionMessage, setActionMessage] = useState<string | null>(null);
    const [actionError, setActionError] = useState<string | null>(null);
    const [actionLogs, setActionLogs] = useState<LogLine[]>([]);
    const [showActionLogs, setShowActionLogs] = useState(true);

    const refreshStatus = async (opts?: { silent?: boolean }) => {
        const silent = opts?.silent === true;
        if (!silent) {
            setServerStatus('checking');
        }
        try {
            const status = await fetchWebServiceStatus(statusEndpoint, title);
            const nextPort = status.port || port;
            setResolvedPort(nextPort);
            setTargetUrl(resolveWebServiceTargetUrl(status, nextPort));
            if (status.running) {
                setServerStatus('running');
                setError(null);
                if (!silent) {
                    setIsLoading(true);
                }
                return;
            }
            setServerStatus('not-running');
        } catch (e) {
            const message = e instanceof Error ? e.message : `Failed to fetch ${title} status`;
            setServerStatus('not-running');
            if (!silent) {
                setError(message);
                setActionError(message);
            }
        }
    };

    useEffect(() => {
        void refreshStatus();
    }, [port]);

    useEffect(() => {
        const timer = window.setInterval(() => {
            if (actionState !== null) {
                return;
            }
            void refreshStatus({ silent: true });
        }, 5000);
        return () => window.clearInterval(timer);
    }, [actionState, port, statusEndpoint, title]);

    const appendActionLog = (line: LogLine) => {
        setActionLogs(prev => {
            const last = prev[prev.length - 1];
            if (last && last.text === line.text && Boolean(last.error) === Boolean(line.error)) {
                return prev;
            }
            return [...prev, line];
        });
    };

    const runStreamAction = async (
        endpoint: string,
        method: 'GET' | 'POST',
        pendingMessage: string,
        opts?: { clearLogs?: boolean },
    ) => {
        if (opts?.clearLogs) {
            setActionLogs([]);
        }
        setShowActionLogs(true);
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
                onLog: appendActionLog,
                onError: (line) => {
                    appendActionLog(line);
                    setActionError(line.text);
                    setActionMessage(null);
                },
                onDone: (message, data) => {
                    success = data.success !== 'false';
                    const doneMessage = (message || data.message || '').trim();
                    if (doneMessage) {
                        appendActionLog({ text: doneMessage, error: !success });
                    }
                    if (success) {
                        setActionMessage(doneMessage || null);
                        setActionError(null);
                    } else {
                        setActionMessage(null);
                        if (doneMessage) {
                            setActionError(doneMessage);
                        }
                    }
                },
            });
            return success;
        } catch (e) {
            const message = e instanceof Error ? e.message : String(e);
            appendActionLog({ text: message, error: true });
            setActionError(message);
            setActionMessage(null);
            return false;
        }
    };

    const handleStartServer = async () => {
        setActionState('start');
        setActionError(null);
        setActionMessage(`Starting ${title} server on port ${resolvedPort}...`);
        try {
            if (startStreamEndpoint) {
                await runStreamAction(
                    startStreamEndpoint,
                    'POST',
                    `Starting ${title} server on port ${resolvedPort}...`,
                    { clearLogs: true },
                );
                await refreshStatus();
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
        setError(null);
        try {
            if (stopStreamEndpoint) {
                await runStreamAction(
                    stopStreamEndpoint,
                    'POST',
                    `Stopping ${title} server...`,
                );
                await refreshStatus();
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
        setActionMessage(null);
        try {
            if (statusStreamEndpoint) {
                await runStreamAction(statusStreamEndpoint, 'GET', 'Refreshing server status...');
            }
            await refreshStatus();
            setActionMessage(null);
        } finally {
            setActionState(null);
        }
    };

    const actionBusy = actionState !== null;
    const isServerRunning = serverStatus === 'running';
    const toggleServerButtonLabel =
        actionState === 'start'
            ? 'Starting...'
            : actionState === 'stop'
                ? 'Stopping...'
                : isServerRunning
                    ? 'Stop Server'
                    : 'Start Server';
    const toggleServerOperationClass = isServerRunning ? 'mcc-btn-op-stop' : 'mcc-btn-op-start';
    const toggleServerIcon = isServerRunning ? '■' : '▶';
    const handleToggleServer = async () => {
        if (isServerRunning) {
            await handleStopServer();
            return;
        }
        await handleStartServer();
    };
    const handleOpenTargetInNewTab = () => {
        window.open(targetUrl, '_blank', 'noopener,noreferrer');
    };
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
    const handleIframeLoadingChange = (loading: boolean) => {
        setIsLoading(loading);
    };
    const handleIframeError = (message: string) => {
        setIsLoading(false);
        setError(message);
    };
    const handleDismissActionMessage = () => {
        setActionMessage(null);
        setActionError(null);
    };
    const handleDismissActionLogs = () => {
        setShowActionLogs(false);
    };

    return (
        <div className={`codex-web-view${hidePanels ? ' codex-web-view-focus' : ''}`}>
            {!hidePanels && (
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={() => navigate(backPath)}>&larr;</button>
                    <BeakerIcon className="mcc-header-icon" />
                    <h2>{title}</h2>
                    <div className="mcc-header-status">
                        <span className={`mcc-status-dot mcc-status-${serverStatus}`}></span>
                        <button
                            className="mcc-status-refresh-btn"
                            onClick={handleRefreshStatus}
                            disabled={actionBusy}
                            title="Refresh status"
                            aria-label="Refresh status"
                        >
                            <span className={`mcc-status-refresh-icon${actionState === 'refresh' ? ' spinning' : ''}`}>
                                <RefreshIcon />
                            </span>
                        </button>
                    </div>
                    {serverStatus === 'running' && (
                        <button
                            className="codex-web-focus-toggle"
                            onClick={handleOpenTargetInNewTab}
                            title="Open target in new tab"
                            aria-label="Open target in new tab"
                        >
                            <OpenInNewIcon className="codex-web-focus-icon" />
                        </button>
                    )}
                    {settingsPath && (
                        <button
                            className="codex-web-focus-toggle"
                            onClick={() => navigate(settingsPath)}
                            title="Open settings"
                            aria-label="Open settings"
                        >
                            <span className="codex-web-settings-icon">
                                <SettingsIcon />
                            </span>
                        </button>
                    )}
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
                            <button
                                className={`mcc-btn-secondary ${toggleServerOperationClass} codex-web-action-btn`}
                                onClick={handleToggleServer}
                                disabled={actionBusy || serverStatus === 'checking'}
                            >
                                <span className="codex-web-action-btn-icon" aria-hidden="true">{toggleServerIcon}</span>
                                <span>{toggleServerButtonLabel}</span>
                            </button>
                        </div>
                        {showActionLogs && (actionLogs.length > 0 || actionBusy) && (
                            <div className="codex-web-log-panel">
                                <button
                                    className="codex-web-action-dismiss codex-web-log-dismiss"
                                    onClick={handleDismissActionLogs}
                                    aria-label="Dismiss action logs"
                                >
                                    <DismissIcon />
                                </button>
                                <LogViewer
                                    lines={actionLogs}
                                    pending={actionBusy}
                                    pendingMessage="Streaming logs..."
                                    maxHeight={180}
                                />
                            </div>
                        )}
                        {actionMessage && (
                            <div className="codex-web-action-status codex-web-action-banner">
                                <span>{actionMessage}</span>
                                <button
                                    className="codex-web-action-dismiss"
                                    onClick={handleDismissActionMessage}
                                    aria-label="Dismiss status message"
                                >
                                    <DismissIcon />
                                </button>
                            </div>
                        )}
                        {actionError && (
                            <div className="codex-web-action-error codex-web-action-banner">
                                <span>{actionError}</span>
                                <button
                                    className="codex-web-action-dismiss"
                                    onClick={handleDismissActionMessage}
                                    aria-label="Dismiss error message"
                                >
                                    <DismissIcon />
                                </button>
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
                    {error && (
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
                                    <li>Click the refresh icon in the status indicator</li>
                                </ol>
                            </div>
                        </div>
                    )}
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
                </div>
            </div>
        </div>
    );
}
