import { useState, useEffect } from 'react';
import type { ExternalOpencodeSession } from '../../../api/agents';
import { fetchExternalSessions } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { truncate } from './utils';
import { Pagination } from './Pagination';

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
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const pageSize = 5;

    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        fetchExternalSessions(currentPage, pageSize)
            .then(data => {
                if (cancelled) return;
                if (data && data.items) {
                    const previews = data.items.map((s: ExternalOpencodeSession) => ({
                        id: s.id,
                        title: s.title || 'Untitled Session',
                        firstMessage: s.title || '',
                        created_at: s.time?.created ? new Date(s.time.created).toISOString() : undefined,
                    }));
                    setSessions(previews);
                    setTotalPages(data.total_pages);
                    setTotalCount(data.total);
                }
                setLoading(false);
            })
            .catch(() => {
                if (!cancelled) setLoading(false);
            });
        return () => { cancelled = true; };
    }, [currentPage]);

    const handlePageChange = (newPage: number) => {
        if (newPage >= 1 && newPage <= totalPages) {
            setCurrentPage(newPage);
        }
    };

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
                <>
                    <Pagination
                        currentPage={currentPage}
                        totalPages={totalPages}
                        totalCount={totalCount}
                        pageSize={pageSize}
                        onPageChange={handlePageChange}
                        loading={loading}
                    />
                    <div className="mcc-agent-session-list">
                        {sessions.map((s) => {
                            return (
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
                            );
                        })}
                    </div>
                    <Pagination
                        currentPage={currentPage}
                        totalPages={totalPages}
                        totalCount={totalCount}
                        pageSize={pageSize}
                        onPageChange={handlePageChange}
                        loading={loading}
                    />
                </>
            )}
        </div>
    );
}
