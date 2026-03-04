import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../../../../../pure-view/icons/BeakerIcon';
import type { ConnectionStatus } from './ACPChatTypes';

export interface ACPChatHeaderProps {
    title: string;
    status: ConnectionStatus;
    statusMessage: string;
    sessionId: string | null;
    isNewSession: boolean;
    onBack: () => void;
}

export function ACPChatHeader({ title, status, statusMessage, sessionId, isNewSession, onBack }: ACPChatHeaderProps) {
    const navigate = useNavigate();

    const statusClass = status === 'connected' ? 'running' : status === 'error' ? 'not-running' : 'checking';

    return (
        <div className="mcc-section-header">
            <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
            <BeakerIcon className="mcc-header-icon" />
            <h2>{title}</h2>
            <div className="mcc-header-status">
                <span className={`mcc-status-dot mcc-status-${statusClass}`}></span>
                <span className="mcc-status-text">{statusMessage}</span>
            </div>
            {sessionId && !isNewSession && (
                <button
                    className="acp-ui-settings-btn"
                    onClick={() => navigate(`./settings`)}
                    title="Session Settings"
                >
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <circle cx="12" cy="12" r="3" />
                        <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.6 9a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
                    </svg>
                </button>
            )}
        </div>
    );
}
