import { useState } from 'react';
import { LogViewer } from './LogViewer';
import type { LogLine } from './LogViewer';
import './ActionCard.css';

export interface ActionCardProps {
    name: string;
    icon: string;
    script: string;
    running?: boolean;
    exitCode?: number;
    logs?: LogLine[];
    onRun?: () => void;
    onStop?: () => void;
    onEdit?: () => void;
    onDelete?: () => void;
}

export function ActionCard({
    name,
    icon,
    script,
    running = false,
    exitCode,
    logs = [],
    onRun,
    onStop,
    onEdit,
    onDelete,
}: ActionCardProps) {
    const [scriptExpanded, setScriptExpanded] = useState(false);

    const scriptLines = script.split('\n');
    const isLongScript = scriptLines.length > 3;

    const showExpandScript = isLongScript;

    return (
        <div className={`action-card ${running ? 'action-card--running' : ''} ${exitCode !== undefined ? (exitCode === 0 ? 'action-card--success' : 'action-card--error') : ''}`}>
            {/* Row 1: Action header with icon, name, and action buttons */}
            <div className="action-card__header">
                <div className="action-card__info">
                    <span className="action-card__icon">{icon}</span>
                    <span className="action-card__name">{name}</span>
                    {running && (
                        <span className="action-card__status action-card__status--running">
                            <span className="action-card__spinner"></span>
                            Running
                        </span>
                    )}
                </div>
                <div className="action-card__actions">
                    {onEdit && (
                        <button
                            className="action-card__action-btn"
                            onClick={onEdit}
                            title="Edit"
                        >
                            ✏️
                        </button>
                    )}
                    {onDelete && (
                        <button
                            className="action-card__action-btn action-card__action-btn--delete"
                            onClick={onDelete}
                            title="Delete"
                        >
                            🗑️
                        </button>
                    )}
                    {running ? (
                        onStop && (
                            <button
                                className="action-card__run-btn running"
                                onClick={onStop}
                                title="Stop"
                            >
                                <span className="action-card__stop-icon">⏹</span>
                            </button>
                        )
                    ) : (
                        onRun && (
                            <button
                                className="action-card__run-btn"
                                onClick={onRun}
                                title="Run"
                            >
                                <span className="action-card__play-icon">▶</span>
                            </button>
                        )
                    )}
                </div>
            </div>

            {/* Row 2: Script preview with expand/collapse */}
            <div className="action-card__script-row">
                <code className="action-card__script">
                    {showExpandScript && !scriptExpanded ? (
                        <>
                            {scriptLines.slice(0, 3).join('\n')}
                            {scriptLines.length > 3 && (
                                <span className="action-card__script-more">
                                    ... +{scriptLines.length - 3} more lines
                                </span>
                            )}
                        </>
                    ) : (
                        script
                    )}
                </code>
                {isLongScript && (
                    <button
                        className="action-card__expand-btn"
                        onClick={() => setScriptExpanded(!scriptExpanded)}
                    >
                        {scriptExpanded ? 'Show less' : 'Show more'}
                    </button>
                )}
            </div>

            {/* Row 3: Logs section using shared LogViewer */}
            {(running || logs.length > 0 || exitCode !== undefined) && (
                <div className="action-card__logs-section">
                    {exitCode !== undefined && !running && (
                        <div className={`action-card__exit-code ${exitCode === 0 ? 'success' : 'error'}`}>
                            {exitCode === 0 ? '✓ Success' : `✗ Exit ${exitCode}`}
                        </div>
                    )}
                    <LogViewer
                        lines={logs}
                        pending={running}
                        pendingMessage="Streaming logs..."
                        maxHeight={200}
                    />
                </div>
            )}
        </div>
    );
}
