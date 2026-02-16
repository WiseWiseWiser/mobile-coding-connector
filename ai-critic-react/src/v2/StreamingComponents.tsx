import { LogViewer } from '../pure-view/LogViewer';
import type { StreamingActionState } from '../hooks/useStreamingAction';
import './StreamingActionButton.css';

interface StreamingButtonProps {
    /** Button label when idle */
    label: string;
    /** Button label when running */
    runningLabel: string;
    /** Click handler - typically calls controls.run(action) */
    onClick: () => void;
    /** Whether the action is currently running */
    running: boolean;
    /** Additional CSS class for the button */
    className?: string;
    /** Whether the button is disabled */
    disabled?: boolean;
    /** Icon to show in button (optional) */
    icon?: React.ReactNode;
}

/** Standalone streaming action button (no logs area). Use with useStreamingAction hook. */
export function StreamingButton({
    label,
    runningLabel,
    onClick,
    running,
    className = '',
    disabled = false,
    icon,
}: StreamingButtonProps) {
    return (
        <button
            className={`streaming-action-button ${className}`}
            onClick={onClick}
            disabled={disabled || running}
        >
            {running ? runningLabel : (
                <>
                    {label}
                    {icon && <span className="streaming-action-button-icon">{icon}</span>}
                </>
            )}
        </button>
    );
}

interface StreamingLogsProps {
    /** State from useStreamingAction hook */
    state: StreamingActionState;
    /** Pending message for log viewer */
    pendingMessage?: string;
    /** Max height for log viewer. Default: 150 */
    maxHeight?: number;
}

/** Standalone streaming logs display. Use with useStreamingAction hook. */
export function StreamingLogs({
    state,
    pendingMessage = 'Running...',
    maxHeight = 150,
}: StreamingLogsProps) {
    const { running, logs, result, showLogs } = state;

    return (
        <>
            {showLogs && logs.length > 0 && (
                <div className="streaming-action-logs">
                    <LogViewer
                        lines={logs}
                        pending={running}
                        pendingMessage={pendingMessage}
                        maxHeight={maxHeight}
                    />
                </div>
            )}

            {result && (
                <div className={`streaming-action-result ${result.ok ? 'success' : 'error'}`}>
                    {result.message}
                </div>
            )}
        </>
    );
}
