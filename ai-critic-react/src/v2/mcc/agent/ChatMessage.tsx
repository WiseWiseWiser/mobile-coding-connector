import { useState } from 'react';
import type { AgentMessage, MessagePart } from '../../../api/agents';
import { truncate } from './utils';

// ---- Message Grouping ----

export function groupMessagesByRole(messages: AgentMessage[]): AgentMessage[][] {
    const groups: AgentMessage[][] = [];
    for (const msg of messages) {
        const lastGroup = groups[groups.length - 1];
        if (lastGroup && lastGroup[0].info.role === msg.info.role) {
            lastGroup.push(msg);
        } else {
            groups.push([msg]);
        }
    }
    return groups;
}

export function ChatMessageGroup({ messages }: { messages: AgentMessage[] }) {
    const isUser = messages[0].info.role === 'user';

    const thinkingParts: MessagePart[] = [];
    const contentParts: MessagePart[] = [];
    for (const msg of messages) {
        for (const part of msg.parts) {
            if (part.type === 'reasoning' || part.type === 'thinking' || part.thinking || part.reasoning) {
                thinkingParts.push(part);
            } else {
                contentParts.push(part);
            }
        }
    }

    return (
        <div className={`mcc-agent-msg ${isUser ? 'mcc-agent-msg-user' : 'mcc-agent-msg-assistant'}`}>
            <div className="mcc-agent-msg-avatar">
                {isUser ? '👤' : '🤖'}
            </div>
            <div className="mcc-agent-msg-content">
                {thinkingParts.length > 0 && (
                    <ThinkingBlock parts={thinkingParts} />
                )}
                {contentParts.map((part, idx) => (
                    <MessagePartView key={part.id || idx} part={part} />
                ))}
            </div>
        </div>
    );
}

// ---- Thinking Block ----

function ThinkingBlock({ parts }: { parts: MessagePart[] }) {
    const [expanded, setExpanded] = useState(false);

    const thinkingText = parts.map(p => p.thinking || p.reasoning || p.text || p.content || '').join('\n').trim();
    if (!thinkingText) return null;

    const lines = thinkingText.split('\n');
    const needsExpand = lines.length > 3;
    const previewText = needsExpand && !expanded ? lines.slice(0, 3).join('\n') : thinkingText;

    return (
        <div className="mcc-agent-msg-thinking">
            <div className="mcc-agent-msg-thinking-label">
                <span className="mcc-agent-msg-thinking-icon">💭</span>
                <span>Thinking</span>
            </div>
            <div className={`mcc-agent-msg-thinking-content ${!expanded && needsExpand ? 'clamped' : ''}`}>
                {previewText}
            </div>
            {needsExpand && (
                <button className="mcc-agent-msg-thinking-toggle" onClick={() => setExpanded(!expanded)}>
                    {expanded ? 'Show less' : 'Show more'}
                </button>
            )}
        </div>
    );
}

function MessagePartView({ part }: { part: MessagePart }) {
    if (part.type === 'text') {
        const text = part.text || part.content || '';
        if (!text) return null;
        return <div className="mcc-agent-msg-text">{text}</div>;
    }

    if (part.type === 'tool') {
        // OpenCode tool part structure
        const toolState = typeof part.state === 'object' ? part.state : undefined;
        const toolName = part.tool || (toolState?.title) || 'tool';
        const isRunning = toolState?.status === 'running' || toolState?.status === 'pending';
        const hasError = toolState?.status === 'error';
        const output = toolState?.output || part.output;
        const error = toolState?.error;
        return (
            <div className={`mcc-agent-msg-tool ${isRunning ? 'running' : ''} ${hasError ? 'error' : ''}`}>
                <div className="mcc-agent-msg-tool-header">
                    <span className="mcc-agent-msg-tool-icon">
                        {isRunning ? '⏳' : hasError ? '❌' : '⚙️'}
                    </span>
                    <span className="mcc-agent-msg-tool-name">{toolName}</span>
                </div>
                {output && (
                    <pre className="mcc-agent-msg-tool-output">{truncate(output, 500)}</pre>
                )}
                {error && (
                    <pre className="mcc-agent-msg-tool-error">{truncate(error, 500)}</pre>
                )}
            </div>
        );
    }

    const text = part.text || part.content;
    if (text) {
        return <div className="mcc-agent-msg-text">{text}</div>;
    }

    return null;
}
