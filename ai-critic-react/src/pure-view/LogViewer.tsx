import { useEffect, useRef } from 'react';
import './LogViewer.css';

export interface LogLine {
    text: string;
    error?: boolean;
}

export interface LogViewerProps {
    lines: LogLine[];
    emptyMessage?: string;
    pending?: boolean;
    pendingMessage?: string;
    maxHeight?: number;
    className?: string;
}

function useAutoScroll<T extends HTMLElement = HTMLDivElement>(
    deps: unknown[],
    threshold = 50
) {
    const containerRef = useRef<T | null>(null);
    const isAtBottomRef = useRef(true);

    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        const handleScroll = () => {
            const { scrollTop, scrollHeight, clientHeight } = container;
            isAtBottomRef.current = scrollHeight - scrollTop - clientHeight <= threshold;
        };

        container.addEventListener('scroll', handleScroll, { passive: true });
        return () => container.removeEventListener('scroll', handleScroll);
    }, [threshold]);

    useEffect(() => {
        if (!isAtBottomRef.current) return;
        const container = containerRef.current;
        if (!container) return;

        requestAnimationFrame(() => {
            container.scrollTop = container.scrollHeight;
        });
    }, deps);

    return containerRef;
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
