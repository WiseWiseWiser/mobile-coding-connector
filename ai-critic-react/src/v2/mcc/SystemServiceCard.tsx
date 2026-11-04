import { useEffect, useRef, useState } from 'react';
import { consumeSSEStream } from '../../api/sse';
import { streamLogFile } from '../../api/logs';
import { systemOperationStreamURL, type ServiceStatus } from '../../api/services';
import { LogViewer } from '../LogViewer';
import { appendLogLine } from './serviceFormat';

export interface SystemServiceCardProps {
    service: ServiceStatus;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onDisable: () => void;
    onEnable: () => void;
    /** Opens the detail view; only proxy services provide one today. */
    onOpen?: () => void;
    /** Refresh the services list after a streamed operation finishes. */
    onRefresh?: () => void;
}

type ActionName = 'start' | 'stop' | 'restart';

/**
 * SystemServiceCard renders one server-owned subsystem. Unlike a user service it
 * has no command, PID or working dir, and it cannot be edited, renamed or
 * removed - only started, stopped and (when the subsystem supports it) toggled
 * for boot auto-start.
 *
 * Every card carries an inline log area: streamed actions (operationStream)
 * run through /api/services/system/stream and their logs land here; services
 * without an operation stream fall back to tailing their LogPath file, with
 * operation errors surfacing via the section banner.
 */
export function SystemServiceCard({ service, onStart, onStop, onRestart, onDisable, onEnable, onOpen, onRefresh }: SystemServiceCardProps) {
    const [logOpen, setLogOpen] = useState(false);
    const [logLines, setLogLines] = useState<{ text: string; error?: boolean }[]>([]);
    const [streaming, setStreaming] = useState(false);
    const [pendingAction, setPendingAction] = useState<ActionName | null>(null);
    const logBodyRef = useRef<HTMLDivElement>(null);

    const canStream = !!service.operationStream;
    const isStreamLog = !!pendingAction; // during an operation the SSE lines are authoritative

    // File-tail streaming (fallback / idle view).
    useEffect(() => {
        if (!logOpen || isStreamLog || !service.logPath) return;

        const controller = new AbortController();
        setLogLines([]);
        setStreaming(true);

        streamLogFile({ path: service.logPath, lines: 100, signal: controller.signal })
            .then(async (response) => {
                await consumeSSEStream(response, {
                    onLog: (line) => setLogLines((prev) => appendLogLine(prev, line)),
                    onError: (line) => setLogLines((prev) => appendLogLine(prev, line)),
                    onDone: (message) => setLogLines((prev) => appendLogLine(prev, { text: message })),
                });
            })
            .catch((err) => {
                if (controller.signal.aborted) return;
                setLogLines((prev) => appendLogLine(prev, { text: err instanceof Error ? err.message : String(err), error: true }));
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setStreaming(false);
                }
            });

        return () => {
            controller.abort();
            setStreaming(false);
        };
    }, [logOpen, isStreamLog, service.logPath]);

    // Auto-scroll the inline log body.
    useEffect(() => {
        if (logOpen && logBodyRef.current) {
            logBodyRef.current.scrollTop = logBodyRef.current.scrollHeight;
        }
    }, [logLines, logOpen]);

    const openLog = () => setLogOpen(true);

    const runStream = async (action: ActionName) => {
        openLog();
        setLogLines([]);
        setPendingAction(action);
        setStreaming(true);
        let hadError = false;
        try {
            const resp = await fetch(systemOperationStreamURL(service.id, action));
            if (!resp.ok) {
                const text = await resp.text();
                setLogLines((prev) => appendLogLine(prev, { text: text || `${resp.status} ${resp.statusText}`, error: true }));
                return;
            }
            await consumeSSEStream(resp, {
                onLog: (line) => setLogLines((prev) => appendLogLine(prev, line)),
                onError: (line) => {
                    hadError = true;
                    setLogLines((prev) => appendLogLine(prev, line));
                },
                onDone: (message, data) => {
                    const status = data?.status ?? 'done';
                    setLogLines((prev) => appendLogLine(prev, {
                        text: message || `${action}: ${status}`,
                        error: status === 'error',
                    }));
                },
            });
        } catch (err) {
            hadError = true;
            setLogLines((prev) => appendLogLine(prev, { text: err instanceof Error ? err.message : String(err), error: true }));
        } finally {
            setStreaming(false);
            setPendingAction(null);
            if (hadError) {
                setLogLines((prev) => appendLogLine(prev, {
                    text: serviceHint(service),
                    error: false,
                }));
            }
            onRefresh?.();
        }
    };

    const onAction = (action: ActionName) => {
        if (canStream) {
            void runStream(action);
            return;
        }
        // Fallback: plain POST via the parent's handlers; errors surface in the
        // section banner. Open the log area when there is a file to tail.
        openLog();
        if (action === 'start') onStart();
        else if (action === 'stop') onStop();
        else onRestart();
    };

    const statusLabel = service.status === 'running' ? 'Running' :
        service.status === 'starting' ? 'Starting' :
            service.status === 'error' ? 'Error' :
                service.status === 'unknown' ? 'Unknown' : 'Stopped';

    const isRunning = service.status === 'running' || service.status === 'starting';
    const canStop = isRunning || service.desiredRunning;
    const canToggleAutoStart = !!service.autoStartSwitch;
    const autoStartDisabled = service.enabled === false;
    const actions = service.actions ?? ['start', 'restart', 'stop'];
    const can = (action: string) => actions.includes(action);
    const busy = (action: ActionName) => pendingAction === action;
    const anyBusy = pendingAction !== null;

    return (
        <div className={`mcc-port-card mcc-service-card mcc-service-card--system mcc-service-card--${service.status}`}>
            <div className="mcc-service-card-top">
                <div className="mcc-service-card-title-row">
                    <span
                        className={`mcc-port-label${onOpen ? ' mcc-service-openable' : ''}`}
                        onClick={onOpen}
                        role={onOpen ? 'button' : undefined}
                        title={onOpen ? 'Open detail view' : undefined}
                    >
                        {service.name}
                    </span>
                    <div className="mcc-service-card-badges">
                        <span className="mcc-service-kind-badge">system</span>
                        {service.mocked && (
                            <span className="mcc-service-mocked-badge">mocked</span>
                        )}
                        <span className={`mcc-service-status-badge mcc-service-status-badge--${service.status}`}>{statusLabel}</span>
                        {onOpen && (
                            <button className="mcc-btn mcc-service-open-btn" onClick={onOpen}>Detail</button>
                        )}
                    </div>
                </div>
                {service.description && (
                    <div className="mcc-service-description">{service.description}</div>
                )}
            </div>

            <div className="mcc-service-meta">
                <span>{service.detail || (isRunning ? 'Running' : 'Not running')}</span>
                {canToggleAutoStart && (
                    <span>Auto-start: {autoStartDisabled ? 'off' : 'on'}</span>
                )}
            </div>

            {service.id === 'sys-ws-proxy' && (service.detail ?? '').includes('not configured') && (
                <div className="mcc-sys-hint">
                    上游代理未设置,Start 会失败。修复:在 <b>Detail</b> 里填写 upstream proxy(例如 http://127.0.0.1:23456),
                    或执行 <code>remote-agent ws-proxy config set --upstream-proxy …</code>
                </div>
            )}

            {service.mocked && (
                <div className="mcc-service-mocked-hint">
                    Simulated for now; the real integration is not wired up yet.
                </div>
            )}

            {service.edge && (
                <div className="mcc-service-forward">
                    <div className="mcc-service-forward-line">
                        <span>Edge</span>
                    </div>
                    <a href={service.edge} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                        {service.edge}
                    </a>
                </div>
            )}

            {service.publicUrl && (
                <div className="mcc-service-forward">
                    <div className="mcc-service-forward-line">
                        <span>Public</span>
                    </div>
                    <a href={service.publicUrl} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                        {service.publicUrl}
                    </a>
                </div>
            )}

            {service.hosts && service.hosts.length > 0 && (
                <ul className="mcc-service-hosts">
                    {service.hosts.map((host) => (
                        <li key={host.host} className={`mcc-service-host mcc-service-host--${host.state}`}>
                            <span className="mcc-service-host-dot" aria-hidden="true" />
                            <a href={`https://${host.host}`} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                                {host.host}
                            </a>
                            <span className="mcc-service-host-dials">{host.dials} dials</span>
                            <span className="mcc-service-host-state">{host.state}</span>
                        </li>
                    ))}
                </ul>
            )}

            <div className="mcc-port-actions">
                {can('start') && (
                    <button
                        type="button"
                        className="mcc-port-action-btn"
                        onClick={() => onAction('start')}
                        disabled={isRunning || anyBusy}
                    >
                        {busy('start') ? 'Starting…' : 'Start'}
                    </button>
                )}
                {can('restart') && (
                    <button
                        type="button"
                        className="mcc-port-action-btn"
                        onClick={() => onAction('restart')}
                        disabled={anyBusy}
                    >
                        {busy('restart') ? 'Restarting…' : 'Restart'}
                    </button>
                )}
                {can('stop') && (
                    <button
                        type="button"
                        className="mcc-port-action-btn"
                        onClick={() => onAction('stop')}
                        disabled={!canStop || anyBusy}
                    >
                        {busy('stop') ? 'Stopping…' : 'Stop'}
                    </button>
                )}
                {canToggleAutoStart && !autoStartDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onDisable} disabled={anyBusy}>Disable</button>
                )}
                {canToggleAutoStart && autoStartDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onEnable} disabled={anyBusy}>Enable</button>
                )}
                <button
                    type="button"
                    className={`mcc-port-action-btn mcc-port-logs-btn ${logOpen ? 'active' : ''}`}
                    onClick={() => setLogOpen((prev) => !prev)}
                >
                    Logs
                </button>
            </div>

            {logOpen && (
                <div className="mcc-sys-log">
                    <div className="mcc-sys-log-head">
                        <span className={`mcc-sys-log-dot ${streaming || pendingAction ? 'on' : ''}`} />
                        <span>{service.id} · {isStreamLog ? `operation ${pendingAction}` : service.logPath ? 'log tail' : 'no log source'}</span>
                        <span className="mcc-sys-log-spacer" />
                        <button type="button" onClick={() => setLogLines([])}>Clear</button>
                        <button type="button" onClick={() => setLogOpen(false)}>Hide</button>
                    </div>
                    {isStreamLog || logLines.length > 0 || !service.logPath ? (
                        <div className="mcc-sys-log-body" ref={logBodyRef}>
                            {logLines.length === 0 && !streaming && (
                                <div className="mcc-sys-log-empty">
                                    {isStreamLog ? 'waiting for operation output…' : 'no operation output yet — Start/Stop/Restart will stream here'}
                                </div>
                            )}
                            {logLines.map((line, i) => (
                                <div key={i} className={line.error ? 'mcc-sys-log-line err' : 'mcc-sys-log-line'}>{line.text}</div>
                            ))}
                        </div>
                    ) : (
                        <LogViewer
                            lines={logLines}
                            pending={streaming}
                            pendingMessage="Streaming service logs..."
                            className="mcc-port-logs-margin"
                            maxHeight={220}
                        />
                    )}
                </div>
            )}
        </div>
    );
}

/** serviceHint returns the fix-it guidance appended after an in-stream error. */
function serviceHint(service: ServiceStatus): string {
    if (service.id === 'sys-ws-proxy') {
        return 'hint: set upstream proxy in Detail (or `remote-agent ws-proxy config set --upstream-proxy …`), then Start again';
    }
    if (service.id === 'sys-http-proxy') {
        return 'hint: set a password in Detail first (config.password), then Start again';
    }
    return `hint: open ${service.name} Detail for configuration`;
}
