import { useState, useEffect } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import {
    fetchAgentSessions, listOpencodeSessions, createOpencodeSession,
    fetchMessages, AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentSessionInfo } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { truncate } from './utils';

export interface SessionListProps {
    session: AgentSessionInfo;
    projectName: string | null;
    onBack: () => void;
    onStop: () => void;
    onSelectSession: (opencodeSID: string) => void;
    onSessionUpdate: (session: AgentSessionInfo) => void;
}

interface SessionPreview {
    id: string;
    firstMessage: string;
}

export function SessionList({ session, projectName, onBack, onStop, onSelectSession, onSessionUpdate }: SessionListProps) {
    const [sessions, setSessions] = useState<SessionPreview[]>([]);
    const [loading, setLoading] = useState(true);
    const [creating, setCreating] = useState(false);
    const onSelectSessionRef = useCurrent(onSelectSession);
    const sessionRef = useCurrent(session);

    // Poll session status while starting
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Starting) return;

        const timer = setInterval(async () => {
            try {
                const allSessions = await fetchAgentSessions();
                const updated = allSessions.find(s => s.id === sessionRef.current.id);
                if (updated) {
                    onSessionUpdate(updated);
                }
            } catch { /* ignore */ }
        }, 1500);

        return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    // Load sessions and fetch first user message for each
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Running) return;

        let cancelled = false;
        const load = async () => {
            setLoading(true);
            try {
                const list = await listOpencodeSessions(session.id);
                if (cancelled) return;

                // If no sessions exist, auto-create one and navigate into it
                if (list.length === 0) {
                    const newSession = await createOpencodeSession(session.id);
                    if (!cancelled) {
                        onSelectSessionRef.current(newSession.id);
                    }
                    return;
                }

                // Fetch first user message for each session as preview
                const previews = await Promise.all(
                    list.map(async (s) => {
                        try {
                            const msgs = await fetchMessages(session.id, s.id);
                            const firstUserMsg = msgs.find(m => m.info.role === 'user');
                            const text = firstUserMsg?.parts
                                .map(p => p.text || p.content || '')
                                .join(' ')
                                .trim() || '';
                            return { id: s.id, firstMessage: text };
                        } catch {
                            return { id: s.id, firstMessage: '' };
                        }
                    })
                );
                if (!cancelled) {
                    setSessions(previews);
                    setLoading(false);
                }
            } catch {
                if (!cancelled) setLoading(false);
            }
        };
        load();
        return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    const handleNewChat = async () => {
        setCreating(true);
        try {
            const newSession = await createOpencodeSession(session.id);
            onSelectSession(newSession.id);
        } catch { /* ignore */ }
        setCreating(false);
    };

    // Show spinner while starting
    if (session.status === AgentSessionStatuses.Starting) {
        return (
            <div className="mcc-agent-view">
                <AgentChatHeader agentName={session.agent_name} projectName={projectName} onStop={onStop} onBack={onBack} />
                <div className="mcc-agent-starting">
                    <div className="mcc-agent-spinner" />
                    <span>Starting agent server...</span>
                </div>
            </div>
        );
    }

    return (
        <div className="mcc-agent-view">
            <AgentChatHeader agentName={session.agent_name} projectName={projectName} onBack={onBack} />
            <div className="mcc-agent-header" style={{ paddingTop: 4 }}>
                <h2>Sessions</h2>
            </div>
            <div className="mcc-agent-new-chat-row">
                <button className="mcc-forward-btn mcc-agent-new-chat-btn" onClick={handleNewChat} disabled={creating}>
                    {creating ? '...' : '+ New Chat'}
                </button>
            </div>
            {loading ? (
                <div className="mcc-agent-loading">Loading sessions...</div>
            ) : sessions.length === 0 ? (
                <div className="mcc-agent-loading">No sessions yet</div>
            ) : (
                <div className="mcc-agent-session-list">
                    {sessions.map((s, idx) => (
                        <button
                            key={s.id}
                            className="mcc-agent-session-card"
                            onClick={() => onSelectSession(s.id)}
                        >
                            <div className="mcc-agent-session-card-title">
                                Session {sessions.length - idx}
                            </div>
                            <div className="mcc-agent-session-card-preview">
                                {s.firstMessage
                                    ? truncate(s.firstMessage, 100)
                                    : 'No messages yet'}
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
