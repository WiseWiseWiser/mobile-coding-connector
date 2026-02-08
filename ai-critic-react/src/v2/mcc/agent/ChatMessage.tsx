import { useState } from 'react';
import type { ACPMessage, ACPMessagePart } from '../../../api/acp';
import { ACPContentTypes, ACPRoles } from '../../../api/acp';
import { truncate } from './utils';

// ---- Message Grouping ----

export function groupMessagesByRole(messages: ACPMessage[]): ACPMessage[][] {
    const groups: ACPMessage[][] = [];
    for (const msg of messages) {
        const lastGroup = groups[groups.length - 1];
        if (lastGroup && lastGroup[0].role === msg.role) {
            lastGroup.push(msg);
        } else {
            groups.push([msg]);
        }
    }
    return groups;
}

export function ChatMessageGroup({ messages }: { messages: ACPMessage[] }) {
    const isUser = messages[0].role === ACPRoles.User;

    const thinkingParts: ACPMessagePart[] = [];
    const contentParts: ACPMessagePart[] = [];
    for (const msg of messages) {
        for (const part of msg.parts) {
            if (part.content_type === ACPContentTypes.Thinking) {
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
                    <ACPMessagePartView key={part.id || idx} part={part} />
                ))}
            </div>
        </div>
    );
}

// ---- Thinking Block ----

function ThinkingBlock({ parts }: { parts: ACPMessagePart[] }) {
    const [expanded, setExpanded] = useState(false);

    const thinkingText = parts.map(p => p.content || '').join('\n').trim();
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

// ---- ACP Message Part View ----

function ACPMessagePartView({ part }: { part: ACPMessagePart }) {
    if (part.content_type === ACPContentTypes.TextPlain) {
        if (!part.content) return null;
        return <div className="mcc-agent-msg-text">{part.content}</div>;
    }

    if (part.content_type === ACPContentTypes.ToolCall) {
        const toolName = part.name || 'tool';
        const status = part.metadata?.status as string | undefined;
        const isRunning = status === 'running' || status === 'pending';
        const hasError = status === 'error';
        const output = part.metadata?.output as string | undefined;
        const error = part.metadata?.error as string | undefined;
        const title = part.metadata?.title as string | undefined;

        // Try to extract filename from tool input
        let fileName = '';
        const toolInput = part.metadata?.input || tryParseJSON(part.content);
        if (toolInput && typeof toolInput === 'object') {
            const inp = toolInput as Record<string, unknown>;
            fileName = (inp.filePath || inp.path || inp.file || '') as string;
        }

        return (
            <div className={`mcc-agent-msg-tool ${isRunning ? 'running' : ''} ${hasError ? 'error' : ''}`}>
                <div className="mcc-agent-msg-tool-header">
                    <span className="mcc-agent-msg-tool-icon">
                        {isRunning ? '⏳' : hasError ? '❌' : '⚙️'}
                    </span>
                    <span className="mcc-agent-msg-tool-name">{title || toolName}</span>
                    {fileName && (
                        <span className="mcc-agent-msg-tool-file" style={{
                            marginLeft: '8px',
                            color: 'var(--mcc-text-secondary)',
                            fontSize: '0.9em'
                        }}>
                            {fileName}
                        </span>
                    )}
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

    // Fallback: render content as text
    if (part.content) {
        return <div className="mcc-agent-msg-text">{part.content}</div>;
    }

    return null;
}

function tryParseJSON(s: string): unknown {
    try {
        return JSON.parse(s);
    } catch {
        return null;
    }
}
