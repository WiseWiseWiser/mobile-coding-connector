import { useState, useEffect } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import { useAutoScroll } from '../../../hooks/useAutoScroll';
import {
    fetchAgentSessions, fetchMessages, sendPromptAsync, agentEventUrl,
    fetchOpencodeConfig, fetchOpencodeProviders, updateAgentConfig,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentSessionInfo, OpencodeConfig } from '../../../api/agents';
import type { ACPMessage } from '../../../api/acp';
import { parseSSEEvent, convertMessages } from '../../../api/acp_adapter';
import { AgentChatHeader } from './AgentChatHeader';
import type { ModelOption } from './AgentChatHeader';
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
    const [messages, setMessages] = useState<ACPMessage[]>([]);
    const [input, setInput] = useState('');
    const [sending, setSending] = useState(false);
    const [agentConfig, setAgentConfig] = useState<OpencodeConfig | null>(null);
    const [contextLimit, setContextLimit] = useState<number>(0);
    const [availableModels, setAvailableModels] = useState<ModelOption[]>([]);
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

        Promise.all([
            fetchOpencodeConfig(session.id),
            fetchOpencodeProviders(session.id),
        ]).then(([cfg, providersData]) => {
            setAgentConfig(cfg);

            let modelID = cfg.model?.modelID;
            let providerID = cfg.model?.providerID;

            if (!modelID && providersData.default) {
                const defaults = providersData.default;
                const firstKey = Object.keys(defaults)[0];
                if (firstKey) {
                    providerID = firstKey;
                    modelID = defaults[firstKey];
                }
            }

            // Build available models list from all providers
            const models: ModelOption[] = [];
            for (const provider of (providersData.providers || [])) {
                for (const [, modelInfo] of Object.entries(provider.models || {})) {
                    models.push({
                        id: modelInfo.id,
                        name: modelInfo.name,
                        is_default: (modelInfo as Record<string, unknown>).is_default as boolean | undefined,
                        is_current: (modelInfo as Record<string, unknown>).is_current as boolean | undefined,
                    });
                }
            }
            setAvailableModels(models);

            if (!providerID || !modelID) return;

            const provider = providersData.providers?.find(p => p.id === providerID);
            const model = provider?.models?.[modelID];
            if (model?.limit?.context) {
                setContextLimit(model.limit.context);
            }
            if (modelID) {
                setAgentConfig(prev => prev ? { ...prev, model: { modelID: modelID!, providerID: providerID! } } : prev);
            }
        }).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    // Initial message load
    useEffect(() => {
        if (!opencodeSID || session.status !== AgentSessionStatuses.Running) return;
        fetchMessages(session.id, opencodeSID)
            .then(msgs => setMessages(convertMessages(msgs)))
            .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [opencodeSID, session.status]);

    // SSE-driven incremental message updates (standard ACP events)
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Running) return;

        const eventSource = new EventSource(agentEventUrl(session.id));

        eventSource.onmessage = (event) => {
            const updater = parseSSEEvent(event.data);
            if (updater) {
                setMessages(updater);
            }
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

    const handleModelChange = async (modelId: string) => {
        try {
            await updateAgentConfig(session.id, { model: { modelID: modelId } });
            setAgentConfig(prev => prev ? { ...prev, model: { modelID: modelId, providerID: 'cursor' } } : prev);
        } catch { /* ignore */ }
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

    const modelName = agentConfig?.model?.modelID || undefined;
    const contextPercent = contextLimit > 0 ? undefined : undefined; // TODO: compute from token usage

    return (
        <div className="mcc-agent-view mcc-agent-view-chat">
            <AgentChatHeader
                agentName={session.agent_name}
                projectName={projectName}
                onBack={onBack}
                modelName={modelName}
                contextPercent={contextPercent}
                availableModels={availableModels}
                onModelChange={handleModelChange}
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
                    <ChatMessageGroup key={group[0].id || idx} messages={group} />
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
