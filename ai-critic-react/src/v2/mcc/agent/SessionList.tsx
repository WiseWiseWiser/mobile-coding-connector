import { useState, useEffect } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import {
    fetchAgentSessions, createOpencodeSession,
    fetchMessages, AgentSessionStatuses, listOpencodeSessionsPaginated,
    fetchOpencodeSettings,
} from '../../../api/agents';
import type { AgentSessionInfo, OpencodeSettings } from '../../../api/agents';
import { ACPRoles } from '../../../api/acp';
import { convertMessages } from '../../../api/acp_adapter';
import { AgentChatHeader } from './AgentChatHeader';
import { SettingsIcon } from '../../../pure-view/icons/SettingsIcon';
import { truncate } from './utils';

export interface SessionListProps {
    session: AgentSessionInfo;
    projectName: string | null;
    onBack: () => void;
    onStop: () => void;
    onSelectSession: (opencodeSID: string) => void;
    onSessionUpdate: (session: AgentSessionInfo) => void;
    onSettings?: () => void;
}

interface SessionPreview {
    id: string;
    firstMessage: string;
    created_at?: string;
}

// Parse saved model from settings
function getSavedModelFromSettings(settings: OpencodeSettings | null) {
    if (!settings?.model) return undefined;
    const parts = settings.model.split('/');
    if (parts.length >= 2) {
        return { providerID: parts[0], modelID: parts[1] };
    }
    return undefined;
}

export function SessionList({ session, projectName, onBack, onStop, onSelectSession, onSessionUpdate, onSettings }: SessionListProps) {
    const [sessions, setSessions] = useState<SessionPreview[]>([]);
    const [loading, setLoading] = useState(true);
    const [creating, setCreating] = useState(false);
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const [savedSettings, setSavedSettings] = useState<OpencodeSettings | null>(null);
    const pageSize = 5;
    const onSelectSessionRef = useCurrent(onSelectSession);
    const sessionRef = useCurrent(session);

    // Load saved settings for model preference
    useEffect(() => {
        fetchOpencodeSettings()
            .then(settings => setSavedSettings(settings))
            .catch(() => {});
    }, []);

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
                const response = await listOpencodeSessionsPaginated(session.id, currentPage, pageSize);
                if (cancelled) return;

                // If no sessions exist, auto-create one and navigate into it
                if (response.total === 0) {
                    const model = getSavedModelFromSettings(savedSettings);
                    const newSession = await createOpencodeSession(session.id, model);
                    if (!cancelled) {
                        onSelectSessionRef.current(newSession.id);
                    }
                    return;
                }

                setTotalPages(response.total_pages);
                setTotalCount(response.total);

                // Fetch first user message for each session as preview
                const previews = await Promise.all(
                    response.items.map(async (s) => {
                        try {
                            const rawMsgs = await fetchMessages(session.id, s.id);
                            const msgs = convertMessages(rawMsgs);
                            const firstUserMsg = msgs.find(m => m.role === ACPRoles.User);
                            const text = firstUserMsg?.parts
                                .map(p => p.content || '')
                                .join(' ')
                                .trim() || '';
                            return { id: s.id, firstMessage: text, created_at: s.created_at };
                        } catch {
                            return { id: s.id, firstMessage: '', created_at: s.created_at };
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
    }, [session.id, session.status, currentPage]);

    const handleNewChat = async () => {
        setCreating(true);
        try {
            const model = getSavedModelFromSettings(savedSettings);
            const newSession = await createOpencodeSession(session.id, model);
            onSelectSession(newSession.id);
        } catch { /* ignore */ }
        setCreating(false);
    };

    const handlePageChange = (newPage: number) => {
        if (newPage >= 1 && newPage <= totalPages) {
            setCurrentPage(newPage);
        }
    };

    const renderPagination = () => {
        if (totalPages <= 1) return null;

        return (
            <div className="mcc-agent-pagination">
                <div className="mcc-agent-pagination-info">
                    Showing {sessions.length} of {totalCount} sessions
                </div>
                <div className="mcc-agent-pagination-controls">
                    <button
                        className="mcc-agent-pagination-btn"
                        onClick={() => handlePageChange(currentPage - 1)}
                        disabled={currentPage === 1}
                    >
                        ←
                    </button>
                    <span className="mcc-agent-pagination-page">
                        Page {currentPage} of {totalPages}
                    </span>
                    <button
                        className="mcc-agent-pagination-btn"
                        onClick={() => handlePageChange(currentPage + 1)}
                        disabled={currentPage === totalPages}
                    >
                        →
                    </button>
                </div>
            </div>
        );
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
                {onSettings && (
                    <button className="mcc-agent-settings-icon-btn" onClick={onSettings} title="Agent Settings">
                        <SettingsIcon />
                    </button>
                )}
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
                <>
                    {renderPagination()}
                    <div className="mcc-agent-session-list">
                        {sessions.map((s, idx) => {
                            const globalIndex = totalCount - ((currentPage - 1) * pageSize) - idx;
                            return (
                                <button
                                    key={s.id}
                                    className="mcc-agent-session-card"
                                    onClick={() => onSelectSession(s.id)}
                                >
                                    <div className="mcc-agent-session-card-title">
                                        Session {globalIndex}
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
                            );
                        })}
                    </div>
                    {renderPagination()}
                </>
            )}
        </div>
    );
}
