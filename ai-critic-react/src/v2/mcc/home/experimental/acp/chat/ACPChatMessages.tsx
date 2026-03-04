import { useRef, useEffect } from 'react';
import type { ChatMessage, ConnectionStatus } from './ACPChatTypes';

export interface ACPChatMessagesProps {
    messages: ChatMessage[];
    status: ConnectionStatus;
    agentName: string;
    emptyConnectedMessage: string;
    chatContainerRef: React.RefObject<HTMLDivElement | null>;
    onScroll: () => void;
}

export function ACPChatMessages({ messages, status, agentName, emptyConnectedMessage, chatContainerRef, onScroll }: ACPChatMessagesProps) {
    const messagesEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    return (
        <div className="acp-ui-chat" ref={chatContainerRef} onScroll={onScroll}>
            {messages.length === 0 && status === 'connected' && (
                <div className="acp-ui-empty">{emptyConnectedMessage}</div>
            )}
            {messages.length === 0 && status === 'connecting' && (
                <div className="acp-ui-empty">Connecting to {agentName}...</div>
            )}
            {messages.length === 0 && status === 'error' && (
                <div className="acp-ui-empty">Connection failed. Go back and try again.</div>
            )}
            {messages.map((msg, i) => (
                <div key={i} className={`acp-ui-message acp-ui-message-${msg.role}`}>
                    <div className="acp-ui-message-role">{msg.role === 'user' ? 'You' : agentName}</div>
                    {msg.plan && msg.plan.length > 0 && (
                        <div className="acp-ui-plan">
                            <div className="acp-ui-plan-title">Plan</div>
                            {msg.plan.map((entry, j) => (
                                <div key={j} className={`acp-ui-plan-entry acp-ui-plan-${entry.status}`}>
                                    <span className="acp-ui-plan-status">
                                        {entry.status === 'completed' ? '\u2713' : entry.status === 'in_progress' ? '\u25CB' : '\u2022'}
                                    </span>
                                    {entry.content}
                                </div>
                            ))}
                        </div>
                    )}
                    {msg.content && (
                        <div className="acp-ui-message-content">
                            <pre>{msg.content}</pre>
                        </div>
                    )}
                    {msg.toolCalls && msg.toolCalls.length > 0 && (
                        <div className="acp-ui-tools">
                            {msg.toolCalls.map(tc => (
                                <div key={tc.id} className={`acp-ui-tool acp-ui-tool-${tc.status}`}>
                                    <div className="acp-ui-tool-header">
                                        <span className="acp-ui-tool-icon">
                                            {tc.status === 'completed' ? '\u2713' : tc.status === 'in_progress' ? '\u23F3' : tc.status === 'failed' ? '\u2717' : '\u2022'}
                                        </span>
                                        <span className="acp-ui-tool-title">{tc.title}</span>
                                        <span className="acp-ui-tool-status">{tc.status}</span>
                                    </div>
                                    {tc.content && (
                                        <pre className="acp-ui-tool-content">{tc.content}</pre>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            ))}
            <div ref={messagesEndRef} />
        </div>
    );
}
