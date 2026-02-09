import { useState } from 'react';
import { consumeSSEStream } from '../api/sse';
import { LogViewer } from './LogViewer';
import type { LogLine } from './LogViewer';
import './StreamingActionButton.css';

export interface StreamingActionResult {
    ok: boolean;
    message: string;
}

export interface StreamingActionButtonProps {
    /** Button label when idle */
    label: string;
    /** Button label when running */
    runningLabel: string;
    /** Function that returns a Response with SSE stream */
    action: () => Promise<Response>;
    /** Optional callback when action completes */
    onComplete?: (result: StreamingActionResult) => void;
    /** Additional CSS class for the button */
    className?: string;
    /** Whether the button is disabled */
    disabled?: boolean;
    /** Max height for log viewer. Default: 150 */
    logMaxHeight?: number;
    /** Icon to show in button (optional) */
    icon?: React.ReactNode;
}

export function StreamingActionButton({
    label,
    runningLabel,
    action,
    onComplete,
    className = '',
    disabled = false,
    logMaxHeight = 150,
    icon,
}: StreamingActionButtonProps) {
    const [running, setRunning] = useState(false);
    const [logs, setLogs] = useState<LogLine[]>([]);
    const [showLogs, setShowLogs] = useState(false);
    const [result, setResult] = useState<StreamingActionResult | null>(null);

    const handleClick = async () => {
        setRunning(true);
        setResult(null);
        setLogs([]);
        setShowLogs(true);

        try {
            const response = await action();

            await consumeSSEStream(response, {
                onLog: (line) => setLogs(prev => [...prev, line]),
                onError: (line) => setLogs(prev => [...prev, line]),
                onDone: (message, data) => {
                    const actionResult: StreamingActionResult = {
                        ok: data.success === 'true',
                        message: message || (data.success === 'true' ? 'Completed successfully' : 'Failed'),
                    };
                    setResult(actionResult);
                    onComplete?.(actionResult);
                },
            });
        } catch (err: unknown) {
            const errorMessage = err instanceof Error ? err.message : 'Action failed';
            const actionResult: StreamingActionResult = { ok: false, message: errorMessage };
            setResult(actionResult);
            onComplete?.(actionResult);
        } finally {
            setRunning(false);
        }
    };

    return (
        <div className="streaming-action-button-container">
            <button
                className={`streaming-action-button ${className}`}
                onClick={handleClick}
                disabled={disabled || running}
            >
                {running ? runningLabel : (
                    <>
                        {label}
                        {icon && <span className="streaming-action-button-icon">{icon}</span>}
                    </>
                )}
            </button>

            {showLogs && logs.length > 0 && (
                <div className="streaming-action-logs">
                    <LogViewer
                        lines={logs}
                        pending={running}
                        pendingMessage={runningLabel}
                        maxHeight={logMaxHeight}
                    />
                </div>
            )}

            {result && (
                <div className={`streaming-action-result ${result.ok ? 'success' : 'error'}`}>
                    {result.message}
                </div>
            )}
        </div>
    );
}
