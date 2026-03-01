import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../../../../pure-view/icons/BeakerIcon';
import './ACPUI.css';

interface SessionEntry {
    id: string;
    createdAt: number;
    model?: string;
}

export interface ACPSessionListProps {
    title: string;
    agentName: string;
    apiPrefix: string;
    backPath?: string;
    chatPath?: string;
    settingsPath?: string;
}

export function ACPSessionList({
    title,
    agentName,
    apiPrefix,
    backPath = '../experimental',
    chatPath = '',
    settingsPath,
}: ACPSessionListProps) {
    const navigate = useNavigate();
    const [statusMessage, setStatusMessage] = useState('');
    const [statusOk, setStatusOk] = useState(false);
    const [sessions, setSessions] = useState<SessionEntry[]>([]);
    const [cwd, setCwd] = useState('');

    const fetchSessions = useCallback(async () => {
        try {
            const resp = await fetch(`${apiPrefix}/sessions`);
            const data = await resp.json();
            setSessions(Array.isArray(data) ? data : []);
        } catch { /* ignore */ }
    }, [apiPrefix]);

    const checkStatus = useCallback(async () => {
        try {
            const resp = await fetch(`${apiPrefix}/status`);
            const data = await resp.json();
            if (data.available) {
                setStatusOk(true);
                setStatusMessage(`${agentName} agent available`);
                if (data.cwd) setCwd(data.cwd);
            } else {
                setStatusOk(false);
                setStatusMessage(data.message || `${agentName} agent not available`);
            }
        } catch {
            setStatusOk(false);
            setStatusMessage(`Failed to check ${agentName} agent status`);
        }
    }, [apiPrefix, agentName]);

    useEffect(() => {
        checkStatus();
        fetchSessions();
    }, [checkStatus, fetchSessions]);

    const formatTime = (ts: number) => {
        const d = new Date(ts);
        return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    };

    const navigateToChat = (sessionId: string) => {
        const base = chatPath || '.';
        navigate(`${base}/${sessionId}`);
    };

    return (
        <div className="acp-ui-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate(backPath)}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>{title}</h2>
                <div className="mcc-header-status">
                    <span className={`mcc-status-dot mcc-status-${statusOk ? 'checking' : 'not-running'}`}></span>
                    <span className="mcc-status-text">{statusMessage}</span>
                </div>
                {settingsPath && (
                    <button
                        className="acp-ui-settings-btn"
                        onClick={() => navigate(settingsPath)}
                        title="Settings"
                    >
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <circle cx="12" cy="12" r="3" />
                            <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
                        </svg>
                    </button>
                )}
            </div>

            <div className="acp-ui-sessions-page">
                <div className="acp-ui-sessions-toolbar">
                    <button
                        className="mcc-btn-primary"
                        onClick={() => navigateToChat('new')}
                        disabled={!statusOk}
                    >
                        + New Session
                    </button>
                    {cwd && <span className="acp-ui-cwd" title={cwd}>cwd: {cwd}</span>}
                </div>

                {sessions.length > 0 ? (
                    <div className="acp-ui-sessions">
                        <div className="acp-ui-sessions-header">Previous Sessions</div>
                        {[...sessions].reverse().map(s => (
                            <button
                                key={s.id}
                                className="acp-ui-session-item"
                                onClick={() => navigateToChat(s.id)}
                            >
                                <span className="acp-ui-session-id">{s.id.slice(0, 8)}...</span>
                                {s.model && <span className="acp-ui-session-model">{s.model}</span>}
                                <span className="acp-ui-session-time">{formatTime(s.createdAt)}</span>
                            </button>
                        ))}
                    </div>
                ) : (
                    <div className="acp-ui-sessions-empty">
                        No sessions yet. Click "+ New Session" to get started.
                    </div>
                )}
            </div>
        </div>
    );
}
