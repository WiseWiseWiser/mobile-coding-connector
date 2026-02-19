import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { getLogs, clearLogs, getLogsAsText, type LogEntry } from '../../../../logs';
import { BackIcon } from '../../../icons';
import { FlexInput } from '../../../../pure-view/FlexInput';
import './LogsView.css';

export function LogsView() {
    const navigate = useNavigate();
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [filter, setFilter] = useState('');
    const [levelFilter, setLevelFilter] = useState<string>('all');
    const [copied, setCopied] = useState(false);

    const refreshLogs = useCallback(() => {
        setLogs(getLogs());
    }, []);

    useEffect(() => {
        refreshLogs();
        const interval = setInterval(refreshLogs, 2000);
        return () => clearInterval(interval);
    }, [refreshLogs]);

    const handleClear = () => {
        clearLogs();
        refreshLogs();
    };

    const handleCopy = async () => {
        const text = getLogsAsText();
        await navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const filteredLogs = logs.filter(log => {
        const matchesLevel = levelFilter === 'all' || log.level === levelFilter;
        const matchesText = !filter || log.message.toLowerCase().includes(filter.toLowerCase());
        return matchesLevel && matchesText;
    });

    const getLevelClass = (level: string) => {
        switch (level) {
            case 'error': return 'logs-level-error';
            case 'warn': return 'logs-level-warn';
            case 'info': return 'logs-level-info';
            case 'debug': return 'logs-level-debug';
            default: return '';
        }
    };

    return (
        <div className="logs-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../')}>
                    <BackIcon />
                </button>
                <h2>Frontend Logs</h2>
            </div>

            <div className="logs-controls">
                <FlexInput
                    inputClassName="logs-filter-input"
                    placeholder="Filter logs..."
                    value={filter}
                    onChange={setFilter}
                />
                <select
                    value={levelFilter}
                    onChange={(e) => setLevelFilter(e.target.value)}
                    className="logs-level-select"
                >
                    <option value="all">All Levels</option>
                    <option value="error">Error</option>
                    <option value="warn">Warning</option>
                    <option value="info">Info</option>
                    <option value="log">Log</option>
                    <option value="debug">Debug</option>
                </select>
            </div>

            <div className="logs-actions">
                <button className="logs-action-btn" onClick={refreshLogs}>
                    Refresh
                </button>
                <button className="logs-action-btn" onClick={handleClear}>
                    Clear
                </button>
                <button className="logs-action-btn logs-copy-btn" onClick={handleCopy}>
                    {copied ? 'Copied!' : 'Copy All'}
                </button>
            </div>

            <div className="logs-count">
                {filteredLogs.length} of {logs.length} entries
            </div>

            <div className="logs-list">
                {filteredLogs.length === 0 ? (
                    <div className="logs-empty">No logs to display</div>
                ) : (
                    filteredLogs.map(log => (
                        <div key={log.id} className={`logs-entry ${getLevelClass(log.level)}`}>
                            <span className="logs-time">
                                {log.timestamp.toLocaleTimeString()}
                            </span>
                            <span className={`logs-level ${getLevelClass(log.level)}`}>
                                {log.level}
                            </span>
                            <span className="logs-message">{log.message}</span>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}
