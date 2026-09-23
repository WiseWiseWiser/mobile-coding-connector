import { useEffect, useState } from 'react';
import { consumeSSEStream } from '../../api/sse';
import { streamLogFile } from '../../api/logs';
import type { ServiceStatus } from '../../api/services';
import { LogViewer } from '../LogViewer';
import { appendLogLine } from './serviceFormat';

export interface SystemServiceCardProps {
    service: ServiceStatus;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onDisable: () => void;
    onEnable: () => void;
}

/**
 * SystemServiceCard renders one server-owned subsystem. Unlike a user service it
 * has no command, PID or working dir, and it cannot be edited, renamed or
 * removed - only started, stopped and (when the subsystem supports it) toggled
 * for boot auto-start.
 */
export function SystemServiceCard({ service, onStart, onStop, onRestart, onDisable, onEnable }: SystemServiceCardProps) {
    const [showLogs, setShowLogs] = useState(false);
    const [logLines, setLogLines] = useState<{ text: string; error?: boolean }[]>([]);
    const [streaming, setStreaming] = useState(false);

    useEffect(() => {
        if (!showLogs || !service.logPath) return;

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
    }, [showLogs, service.logPath]);

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

    return (
        <div className={`mcc-port-card mcc-service-card mcc-service-card--system mcc-service-card--${service.status}`}>
            <div className="mcc-service-card-top">
                <div className="mcc-service-card-title-row">
                    <span className="mcc-port-label">{service.name}</span>
                    <div className="mcc-service-card-badges">
                        <span className="mcc-service-kind-badge">system</span>
                        {service.mocked && (
                            <span className="mcc-service-mocked-badge">mocked</span>
                        )}
                        <span className={`mcc-service-status-badge mcc-service-status-badge--${service.status}`}>{statusLabel}</span>
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
                    <button type="button" className="mcc-port-action-btn" onClick={onStart} disabled={isRunning}>Start</button>
                )}
                {can('restart') && (
                    <button type="button" className="mcc-port-action-btn" onClick={onRestart}>Restart</button>
                )}
                {can('stop') && (
                    <button type="button" className="mcc-port-action-btn" onClick={onStop} disabled={!canStop}>Stop</button>
                )}
                {canToggleAutoStart && !autoStartDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onDisable}>Disable</button>
                )}
                {canToggleAutoStart && autoStartDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onEnable}>Enable</button>
                )}
                {service.logPath && (
                    <button
                        type="button"
                        className={`mcc-port-action-btn mcc-port-logs-btn ${showLogs ? 'active' : ''}`}
                        onClick={() => setShowLogs((prev) => !prev)}
                    >
                        Logs
                    </button>
                )}
            </div>

            {showLogs && service.logPath && (
                <LogViewer
                    lines={logLines}
                    pending={streaming}
                    pendingMessage="Streaming service logs..."
                    className="mcc-port-logs-margin"
                    maxHeight={220}
                />
            )}
        </div>
    );
}
