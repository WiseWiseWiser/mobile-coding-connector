import { useState } from 'react';
import Markdown from 'react-markdown';

interface ChatMessageProps {
    role: 'user' | 'assistant';
    content: string;
    thinking?: string;
}

export function ChatMessage({ role, content, thinking }: ChatMessageProps) {
    const [isThinkingExpanded, setIsThinkingExpanded] = useState(false);

    return (
        <div
            style={{
                display: 'flex',
                gap: '8px',
                alignItems: 'flex-start',
                flexDirection: role === 'user' ? 'row-reverse' : 'row',
            }}
        >
            {/* Avatar */}
            <div style={{
                width: '28px',
                height: '28px',
                borderRadius: '50%',
                backgroundColor: role === 'user' ? '#3b82f6' : '#10b981',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '14px',
                flexShrink: 0,
            }}>
                {role === 'user' ? '👤' : '🤖'}
            </div>
            {/* Message bubble */}
            <div
                style={{
                    maxWidth: 'calc(100% - 40px)',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '8px',
                }}
            >
                {/* Thinking section (collapsible) */}
                {role === 'assistant' && thinking && (
                    <div
                        style={{
                            borderRadius: '8px',
                            backgroundColor: '#fef3c7',
                            border: '1px solid #fcd34d',
                            overflow: 'hidden',
                        }}
                    >
                        <button
                            onClick={() => setIsThinkingExpanded(!isThinkingExpanded)}
                            style={{
                                width: '100%',
                                padding: '8px 12px',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px',
                                backgroundColor: 'transparent',
                                border: 'none',
                                cursor: 'pointer',
                                fontSize: '12px',
                                color: '#92400e',
                                fontWeight: 500,
                                textAlign: 'left',
                            }}
                        >
                            <span style={{
                                transform: isThinkingExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                                transition: 'transform 0.2s',
                            }}>▶</span>
                            <span>💭 AI Thinking</span>
                            <span style={{ color: '#b45309', fontWeight: 400 }}>
                                ({thinking.length} chars)
                            </span>
                        </button>
                        {isThinkingExpanded && (
                            <div
                                style={{
                                    padding: '8px 12px',
                                    borderTop: '1px solid #fcd34d',
                                    fontSize: '12px',
                                    lineHeight: 1.5,
                                    color: '#78350f',
                                    whiteSpace: 'pre-wrap',
                                    maxHeight: '200px',
                                    overflow: 'auto',
                                    textAlign: 'left',
                                }}
                            >
                                {thinking}
                            </div>
                        )}
                    </div>
                )}
                {/* Main content */}
                <div
                    style={{
                        padding: '10px 12px',
                        borderRadius: '8px',
                        backgroundColor: role === 'user' ? '#dbeafe' : '#f3f4f6',
                        fontSize: '13px',
                        lineHeight: 1.5,
                    }}
                >
                    {role === 'assistant' ? (
                        <div style={{ textAlign: 'left' }}>
                            <Markdown>{content}</Markdown>
                        </div>
                    ) : (
                        <div style={{ whiteSpace: 'pre-wrap', textAlign: 'left' }}>{content}</div>
                    )}
                </div>
            </div>
        </div>
    );
}
