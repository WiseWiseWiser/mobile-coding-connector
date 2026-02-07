import { useState, useEffect } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import { useAutoScroll } from '../../../hooks/useAutoScroll';
import {
    fetchAgentSessions, fetchMessages, sendPromptAsync, agentEventUrl,
    fetchOpencodeConfig, fetchOpencodeProviders,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentSessionInfo, AgentMessage, OpencodeConfig } from '../../../api/agents';
import { AgentChatHeader } from './AgentChatHeader';
import { ChatMessageGroup, groupMessagesByRole } from './ChatMessage';

export interface AgentChatProps {
    session: AgentSessionInfo;
    projectName: string | null;
    opencodeSID: string;
    onStop: () => void;
    onBack: () => void;
    onSessionUpdate: (session: AgentSessionInfo) => void;
}

export function AgentChat({ session, projectName, opencodeSID, onStop, onBack, onSessionUpdate }: AgentChatProps) {
    const [messages, setMessages] = useState<AgentMessage[]>([]);
    const [input, setInput] = useState('');
    const [sending, setSending] = useState(false);
    const [agentConfig, setAgentConfig] = useState<OpencodeConfig | null>(null);
    const [contextLimit, setContextLimit] = useState<number>(0);
    const messagesContainerRef = useAutoScroll([messages]);
    const sessionRef = useCurrent(session);

    // Poll session status while starting
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Starting) return;

        const timer = setInterval(async () => {
            try {
                const sessions = await fetchAgentSessions();
                const updated = sessions.find(s => s.id === sessionRef.current.id);
                if (updated) {
                    onSessionUpdate(updated);
                }
            } catch { /* ignore */ }
        }, 1500);

        return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    // Fetch config and provider info once agent is running
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Running) return;

        // Fetch config and providers in parallel to determine model and context limit
        Promise.all([
            fetchOpencodeConfig(session.id),
            fetchOpencodeProviders(session.id),
        ]).then(([cfg, providersData]) => {
            setAgentConfig(cfg);

            // Determine the model: explicit config or default from providers
            let modelID = cfg.model?.modelID;
            let providerID = cfg.model?.providerID;

            // If no explicit model, use the default from providers
            if (!modelID && providersData.default) {
                const defaults = providersData.default;
                // Pick the first default entry
                const firstKey = Object.keys(defaults)[0];
                if (firstKey) {
                    providerID = firstKey;
                    modelID = defaults[firstKey];
                }
            }

            if (!providerID || !modelID) return;

            const provider = providersData.providers?.find(p => p.id === providerID);
            const model = provider?.models?.[modelID];
            if (model?.limit?.context) {
                setContextLimit(model.limit.context);
            }
            // Store resolved model name
            if (modelID) {
                setAgentConfig(prev => prev ? { ...prev, model: { modelID: modelID!, providerID: providerID! } } : prev);
            }
        }).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    // Initial message load (one-time HTTP fetch for existing messages)
    useEffect(() => {
        if (!opencodeSID || session.status !== AgentSessionStatuses.Running) return;
        fetchMessages(session.id, opencodeSID)
            .then(msgs => setMessages(msgs))
            .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [opencodeSID, session.status]);

    // SSE-driven incremental message updates
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Running) return;

        const eventSource = new EventSource(agentEventUrl(session.id));

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                const evt = data.payload || data;
                const eventType: string = evt.type || '';
                const props = evt.properties;
                if (!props) return;

                if (eventType === 'message.updated' && props.info) {
                    const msgInfo = props.info;
                    setMessages(prev => {
                        const idx = prev.findIndex(m => m.info.id === msgInfo.id);
                        if (idx >= 0) {
                            const updated = [...prev];
                            updated[idx] = { ...updated[idx], info: msgInfo };
                            return updated;
                        }
                        return [...prev, { info: msgInfo, parts: [] }];
                    });
                }

                if (eventType === 'message.part.updated' && props.part) {
                    const part = props.part;
                    const messageID: string = part.messageID;
                    setMessages(prev => {
                        const msgIdx = prev.findIndex(m => m.info.id === messageID);
                        if (msgIdx < 0) {
                            return [...prev, {
                                info: { id: messageID, role: 'assistant', time: '' },
                                parts: [part],
                            }];
                        }
                        const msg = prev[msgIdx];
                        const partIdx = msg.parts.findIndex(p => p.id === part.id);
                        const newParts = [...msg.parts];
                        if (partIdx >= 0) {
                            newParts[partIdx] = part;
                        } else {
                            newParts.push(part);
                        }
                        const updated = [...prev];
                        updated[msgIdx] = { ...msg, parts: newParts };
                        return updated;
                    });
                }

                if (eventType === 'message.removed' && props.messageID) {
                    setMessages(prev => prev.filter(m => m.info.id !== props.messageID));
                }

                if (eventType === 'message.part.removed' && props.partID) {
                    setMessages(prev => prev.map(m => ({
                        ...m,
                        parts: m.parts.filter(p => p.id !== props.partID),
                    })));
                }

            } catch { /* ignore non-JSON events */ }
        };

        return () => eventSource.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    const handleSend = async () => {
        if (!input.trim() || !opencodeSID) return;
        const text = input.trim();
        setInput('');
        setSending(true);
        try {
            await sendPromptAsync(session.id, opencodeSID, text);
        } catch { /* SSE will show updates */ }
        setSending(false);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSend();
        }
    };

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

    if (session.status === AgentSessionStatuses.Error) {
        return (
            <div className="mcc-agent-view">
                <AgentChatHeader agentName={session.agent_name} projectName={projectName} onStop={onStop} onBack={onBack} stopLabel="Close" />
                <div className="mcc-agent-error-state">
                    <span>Agent failed to start</span>
                    {session.error && <p className="mcc-agent-error">{session.error}</p>}
                    <button className="mcc-forward-btn" onClick={onStop}>Try Again</button>
                </div>
            </div>
        );
    }

    // Compute model name and context usage from messages
    const lastAssistantMsg = [...messages].reverse().find(m => m.info.role === 'assistant' && m.info.tokens);
    const modelName = agentConfig?.model?.modelID || lastAssistantMsg?.info.modelID || undefined;
    const lastInputTokens = lastAssistantMsg?.info.tokens?.input || 0;
    const contextPercent = contextLimit > 0 && lastInputTokens > 0
        ? Math.round((lastInputTokens / contextLimit) * 100)
        : undefined;

    return (
        <div className="mcc-agent-view mcc-agent-view-chat">
            <AgentChatHeader
                agentName={session.agent_name}
                projectName={projectName}
                onBack={onBack}
                modelName={modelName}
                contextPercent={contextPercent}
            />

            <div className="mcc-agent-messages" ref={messagesContainerRef}>
                <div className="mcc-agent-msg mcc-agent-msg-assistant">
                    <div className="mcc-agent-msg-avatar">🤖</div>
                    <div className="mcc-agent-msg-content">
                        <div className="mcc-agent-msg-text">
                            Hi! I'm {session.agent_name}. I'm ready to help with your project. What would you like to work on?
                        </div>
                    </div>
                </div>

                {groupMessagesByRole(messages).map((group, idx) => (
                    <ChatMessageGroup key={group[0].info.id || idx} messages={group} />
                ))}
            </div>

            <div className="mcc-agent-input-area">
                <textarea
                    className="mcc-agent-input"
                    placeholder="Type a message..."
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={2}
                    disabled={sending}
                />
                <button
                    className="mcc-agent-send-btn"
                    onClick={handleSend}
                    disabled={!input.trim() || sending}
                >
                    {sending ? '...' : 'Send'}
                </button>
            </div>
        </div>
    );
}
