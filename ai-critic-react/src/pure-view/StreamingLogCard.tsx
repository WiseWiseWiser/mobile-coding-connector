import { useState, useEffect, useRef } from 'react';
import './StreamingLogCard.css';

export interface LogLine {
    text: string;
    error?: boolean;
}

export interface StreamingLogCardProps {
    logs: LogLine[];
    running?: boolean;
    exitCode?: number;
    title?: string;
    defaultExpanded?: boolean;
    onToggle?: (expanded: boolean) => void;
}

export function StreamingLogCard({
    logs,
    running = false,
    exitCode,
    title = 'Logs',
    defaultExpanded = false,
    onToggle,
}: StreamingLogCardProps) {
    const [expanded, setExpanded] = useState(defaultExpanded);
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (running && expanded && containerRef.current) {
            containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
    }, [logs, running, expanded]);

    const handleToggle = () => {
        const newExpanded = !expanded;
        setExpanded(newExpanded);
        onToggle?.(newExpanded);
    };

    const lineCount = logs.length;

    return (
        <div className={`streaming-log-card ${running ? 'running' : ''} ${exitCode !== undefined ? (exitCode === 0 ? 'success' : 'error') : ''}`}>
            <button className="streaming-log-card-header" onClick={handleToggle}>
                <span className="streaming-log-card-arrow">{expanded ? '▼' : '▶'}</span>
                <span className="streaming-log-card-title">
                    {running ? 'Streaming Logs...' : title}
                </span>
                {lineCount > 0 && (
                    <span className="streaming-log-card-count">{lineCount} lines</span>
                )}
                {exitCode !== undefined && !running && (
                    <span className={`streaming-log-card-exit ${exitCode === 0 ? 'success' : 'error'}`}>
                        {exitCode === 0 ? '✓ Success' : `✗ Exit ${exitCode}`}
                    </span>
                )}
                {running && <span className="streaming-log-card-spinner" />}
            </button>
            
            {expanded && (
                <div className="streaming-log-card-content" ref={containerRef}>
                    {logs.map((log, i) => (
                        <div 
                            key={i} 
                            className={`streaming-log-card-line ${log.error ? 'error' : ''}`}
                        >
                            {log.text}
                            {running && i === logs.length - 1 && (
                                <span className="streaming-log-card-cursor">▋</span>
                            )}
                        </div>
                    ))}
                    {running && logs.length === 0 && (
                        <div className="streaming-log-card-line pending">
                            <span className="streaming-log-card-spinner" />
                            Waiting for output...
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
