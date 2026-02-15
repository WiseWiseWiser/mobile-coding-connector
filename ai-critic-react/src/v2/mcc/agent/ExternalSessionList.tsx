import { useState, useEffect } from 'react';
import type { ExternalOpencodeSession } from '../../../api/agents';
import { fetchExternalSessions } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { truncate } from './utils';

export interface ExternalSessionListProps {
    projectName: string | null;
    onBack: () => void;
    onSelectSession: (sessionId: string) => void;
    onNewSession?: () => void;
}

interface SessionPreview {
    id: string;
    title: string;
    firstMessage: string;
    created_at?: string;
}

export function ExternalSessionList({ projectName, onBack, onSelectSession, onNewSession }: ExternalSessionListProps) {
    const [sessions, setSessions] = useState<SessionPreview[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        fetchExternalSessions()
            .then(data => {
                if (cancelled) return;
                if (data && data.sessions) {
                    const previews = data.sessions.map((s: ExternalOpencodeSession) => ({
                        id: s.id,
                        title: s.title || 'Untitled Session',
                        firstMessage: s.title || '',
                        created_at: s.time?.created ? new Date(s.time.created).toISOString() : undefined,
                    }));
                    setSessions(previews);
                }
                setLoading(false);
            })
            .catch(() => {
                if (!cancelled) setLoading(false);
            });
        return () => { cancelled = true; };
    }, []);

    return (
        <div className="mcc-agent-view">
            <AgentChatHeader agentName="OpenCode (External)" projectName={projectName} onBack={onBack} />
            <div className="mcc-agent-header" style={{ paddingTop: 4 }}>
                <h2>External Sessions</h2>
            </div>
            <div className="mcc-agent-new-chat-row">
                {onNewSession && (
                    <button className="mcc-agent-new-chat-btn" onClick={onNewSession}>
                        + New Session
                    </button>
                )}
                <span className="mcc-agent-card-note">Sessions from CLI or Web</span>
            </div>
            {loading ? (
                <div className="mcc-agent-loading">Loading sessions...</div>
            ) : sessions.length === 0 ? (
                <div className="mcc-agent-loading">No external sessions found</div>
            ) : (
                <div className="mcc-agent-session-list">
                    {sessions.map((s) => (
                        <button
                            key={s.id}
                            className="mcc-agent-session-card"
                            onClick={() => onSelectSession(s.id)}
                        >
                            <div className="mcc-agent-session-card-title">
                                {s.title}
                            </div>
                            <div className="mcc-agent-session-card-preview">
                                {s.firstMessage
                                    ? truncate(s.firstMessage, 100)
                                    : 'No preview available'}
                            </div>
                            <div className="mcc-agent-session-card-id">
                                {s.id.slice(0, 8)}...
                            </div>
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}
