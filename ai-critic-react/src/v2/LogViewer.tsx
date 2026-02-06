import { useAutoScroll } from '../hooks/useAutoScroll';
import './LogViewer.css';

export interface LogLine {
    text: string;
    error?: boolean;
}

interface LogViewerProps {
    lines: LogLine[];
    /** Message shown when there are no lines. Default: "No logs yet..." */
    emptyMessage?: string;
    /** If true, shows a spinner with pending message at the bottom */
    pending?: boolean;
    /** Message shown next to the spinner when pending. Default: "Loading..." */
    pendingMessage?: string;
    /** Max height in px. Default: 200 */
    maxHeight?: number;
    /** Additional CSS class name */
    className?: string;
}

export function LogViewer({
    lines,
    emptyMessage = 'No logs yet...',
    pending,
    pendingMessage = 'Loading...',
    maxHeight = 200,
    className,
}: LogViewerProps) {
    const containerRef = useAutoScroll([lines, pending], 20);

    const isEmpty = lines.length === 0 && !pending;

    return (
        <div
            className={`mcc-log-viewer ${className || ''}`}
            ref={containerRef}
            style={{ maxHeight }}
        >
            {isEmpty ? (
                <div className="mcc-log-viewer-empty">{emptyMessage}</div>
            ) : (
                <>
                    {lines.map((line, i) => (
                        <div
                            key={i}
                            className={`mcc-log-viewer-line ${line.error ? 'mcc-log-viewer-line-error' : ''}`}
                        >
                            {line.text}
                        </div>
                    ))}
                    {pending && (
                        <div className="mcc-log-viewer-line mcc-log-viewer-pending">
                            <span className="mcc-log-viewer-spinner" />
                            {pendingMessage}
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
