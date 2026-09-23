import { useEffect, useState } from 'react';
import { consumeSSEStream } from '../../api/sse';
import { streamLogFile } from '../../api/logs';
import type { ServiceStatus } from '../../api/services';
import { LogViewer } from '../LogViewer';
import { appendLogLine, formatTime, stringifyEnvMap } from './serviceFormat';

export interface ServiceCardProps {
    service: ServiceStatus;
    onEdit: () => void;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onDisable: () => void;
    onEnable: () => void;
    onDelete: () => void;
}

/** ServiceCard renders one user-defined service: a command the server keeps alive. */
export function ServiceCard({ service, onEdit, onStart, onStop, onRestart, onDisable, onEnable, onDelete }: ServiceCardProps) {
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
            service.status === 'error' ? 'Error' : 'Stopped';

    const canStop = service.pid > 0 || service.desiredRunning;
    const isDisabled = service.enabled === false;

    return (
        <div className={`mcc-port-card mcc-service-card mcc-service-card--${service.status}`}>
            <div className="mcc-service-card-top">
                <div className="mcc-service-card-title-row">
                    <span className="mcc-port-label">{service.name}</span>
                    <div className="mcc-service-card-badges">
                        {isDisabled && (
                            <span className="mcc-service-enabled-badge--disabled">Disabled</span>
                        )}
                        <span className={`mcc-service-status-badge mcc-service-status-badge--${service.status}`}>{statusLabel}</span>
                    </div>
                </div>
                <div className="mcc-service-command">{service.command}</div>
            </div>

            <div className="mcc-service-meta">
                <span>PID: {service.pid || 'n/a'}</span>
                <span>Working Dir: {service.workingDir || 'home dir'}</span>
            </div>

            <div className="mcc-service-env-block">
                <div className="mcc-service-env-title">Effective PATH</div>
                <div className="mcc-service-env-value">{service.effectivePath || 'Unavailable'}</div>
                {service.extraEnv && Object.keys(service.extraEnv).length > 0 && (
                    <>
                        <div className="mcc-service-env-title">Extra Env</div>
                        <div className="mcc-service-env-value">{stringifyEnvMap(service.extraEnv)}</div>
                    </>
                )}
            </div>

            {service.portForward ? (
                <div className="mcc-service-forward">
                    <div className="mcc-service-forward-line">
                        <span>Port {service.portForward.port}</span>
                        <span>{service.portForward.provider || 'localtunnel'}</span>
                    </div>
                    {service.portForward.publicUrl ? (
                        <a href={service.portForward.publicUrl} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                            {service.portForward.publicUrl}
                        </a>
                    ) : (
                        <div className="mcc-port-url mcc-port-url-connecting">
                            {service.portForward.error || service.portForward.status || 'Waiting for tunnel'}
                        </div>
                    )}
                    {service.requireAuth && (
                        <div className="mcc-service-auth-hint">
                            Auth: {service.authTokenMode === 'custom' ? 'custom' : 'shared'}
                            {service.authUser ? ` · user ${service.authUser}` : ' · any user'}
                        </div>
                    )}
                </div>
            ) : (
                <div className="mcc-service-forward mcc-service-forward--empty">No port forwarding configured.</div>
            )}

            {(service.lastStartedAt || service.lastExitedAt || service.lastExitError) && (
                <div className="mcc-service-times">
                    {service.lastStartedAt && <div>Started: {formatTime(service.lastStartedAt)}</div>}
                    {service.lastExitedAt && <div>Exited: {formatTime(service.lastExitedAt)}</div>}
                    {service.lastExitError && <div className="mcc-service-error-text">Last error: {service.lastExitError}</div>}
                </div>
            )}

            <div className="mcc-port-actions">
                <button type="button" className="mcc-port-action-btn" onClick={onEdit}>Edit</button>
                <button type="button" className="mcc-port-action-btn" onClick={onStart}>Start</button>
                <button type="button" className="mcc-port-action-btn" onClick={onRestart}>Restart</button>
                <button type="button" className="mcc-port-action-btn" onClick={onStop} disabled={!canStop}>Stop</button>
                {!isDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onDisable}>Disable</button>
                )}
                {isDisabled && (
                    <button type="button" className="mcc-port-action-btn" onClick={onEnable}>Enable</button>
                )}
                <button
                    type="button"
                    className={`mcc-port-action-btn mcc-port-logs-btn ${showLogs ? 'active' : ''}`}
                    onClick={() => setShowLogs((prev) => !prev)}
                >
                    Logs
                </button>
                <button type="button" className="mcc-port-action-btn mcc-port-stop" onClick={onDelete}>Delete</button>
            </div>

            {showLogs && (
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
