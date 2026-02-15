import { useState, useEffect } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import { useAutoScroll } from '../../../hooks/useAutoScroll';
import {
    fetchAgentSessions, fetchMessages, sendPromptAsync, agentEventUrl,
    fetchOpencodeConfig, fetchOpencodeProviders, updateAgentConfig,
    stopAgentSession, fetchOpencodeSettings,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentSessionInfo, OpencodeConfig } from '../../../api/agents';
import type { ACPMessage } from '../../../api/acp';
import { ACPEventTypes, ACPRoles } from '../../../api/acp';
import { parseSSEEvent, convertMessages } from '../../../api/acp_adapter';
import { AgentChatHeader } from './AgentChatHeader';
import type { ModelOption } from '../components/ModelSelector';
import { ChatMessageGroup, groupMessagesByRole } from './ChatMessage';

export interface AgentChatProps {
    session: AgentSessionInfo;
    projectName: string | null;
    opencodeSID: string;
    onStop: () => void;
    onBack: () => void;
    onSessionUpdate: (session: AgentSessionInfo) => void;
    connecting?: boolean;
}

export function AgentChat({ session, projectName, opencodeSID, onStop, onBack, onSessionUpdate, connecting }: AgentChatProps) {
    const [messages, setMessages] = useState<ACPMessage[]>([]);
    const [input, setInput] = useState('');
    const [sending, setSending] = useState(false);
    const [agentProcessing, setAgentProcessing] = useState(false);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
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
            fetchOpencodeSettings(),
        ]).then(([cfg, providersData, savedSettings]) => {
            // Build available models list from all providers
            const models: ModelOption[] = [];
            for (const provider of (providersData.providers || [])) {
                for (const [, modelInfo] of Object.entries(provider.models || {})) {
                    models.push({
                        id: modelInfo.id,
                        name: modelInfo.name,
                        providerId: provider.id,
                        providerName: provider.name || provider.id,
                        is_default: (modelInfo as Record<string, unknown>).is_default as boolean | undefined,
                        is_current: (modelInfo as Record<string, unknown>).is_current as boolean | undefined,
                    });
                }
            }
            // Sort models by provider name, then by model name
            models.sort((a, b) => {
                const providerCompare = a.providerName.localeCompare(b.providerName);
                if (providerCompare !== 0) return providerCompare;
                return a.name.localeCompare(b.name);
            });
            setAvailableModels(models);

            // Get model from config, saved settings, or provider default
            let modelID = cfg.model?.modelID;
            let providerID = cfg.model?.providerID;

            // If no model in config, use saved preference
            if (!modelID && savedSettings?.model) {
                const parts = savedSettings.model.split('/');
                if (parts.length >= 2) {
                    providerID = parts[0];
                    modelID = parts[1];
                }
            }

            // If still no model, fall back to provider default
            if (!modelID && providersData.default) {
                const defaults = providersData.default;
                const firstKey = Object.keys(defaults)[0];
                if (firstKey) {
                    providerID = firstKey;
                    modelID = defaults[firstKey];
                }
            }

            // Set context limit if we have a model
            if (providerID && modelID) {
                const provider = providersData.providers?.find(p => p.id === providerID);
                const model = provider?.models?.[modelID];
                if (model?.limit?.context) {
                    setContextLimit(model.limit.context);
                }
            }

            // Set config with model for UI display (NOT sent to backend)
            setAgentConfig({
                ...cfg,
                model: modelID ? { modelID, providerID: providerID || '' } : cfg.model,
            });
        }).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    // Initial message load
    useEffect(() => {
        if (!opencodeSID || session.status !== AgentSessionStatuses.Running) return;
        fetchMessages(session.id, opencodeSID)
            .then(msgs => {
                const converted = convertMessages(msgs);
                setMessages(converted);
                
                // If messages exist, try to get model from last user message
                if (converted.length > 0) {
                    // Find last user message with a model
                    for (let i = converted.length - 1; i >= 0; i--) {
                        const msg = converted[i];
                        if (msg.role === ACPRoles.User && msg.model) {
                            // Extract provider and model from the model string (format: "provider/model")
                            const parts = msg.model.split('/');
                            if (parts.length >= 2) {
                                setAgentConfig(prev => prev ? {
                                    ...prev,
                                    model: { modelID: parts[1], providerID: parts[0] }
                                } : prev);
                            }
                            break;
                        }
                    }
                }
            })
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

            // Track agent processing state from SSE events
            try {
                const parsed = JSON.parse(event.data);
                const eventType: string = parsed?.type;
                const role: string = parsed?.message?.role;
                
                // Handle message events
                if (role === ACPRoles.Agent) {
                    if (eventType === ACPEventTypes.MessageCreated || eventType === ACPEventTypes.MessageUpdated) {
                        setAgentProcessing(true);
                    } else if (eventType === ACPEventTypes.MessageCompleted) {
                        setAgentProcessing(false);
                    }
                }
                
                // Handle session status events to detect when agent becomes idle
                if (eventType === 'session.idle') {
                    setAgentProcessing(false);
                }
                if (eventType === 'session.status') {
                    const status = parsed?.properties?.status?.type;
                    if (status === 'idle') {
                        setAgentProcessing(false);
                    } else if (status === 'busy') {
                        setAgentProcessing(true);
                    }
                }
            } catch { /* ignore parse errors */ }
        };

        return () => eventSource.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [session.id, session.status]);

    const handleSend = async () => {
        if (!input.trim() || !opencodeSID) return;
        const text = input.trim();
        setInput('');
        setSending(true);
        setAgentProcessing(true); // Show handling status immediately
        try {
            const currentModel = agentConfig?.model;
            await sendPromptAsync(session.id, opencodeSID, text, currentModel);
        } catch { /* SSE will show updates */ }
        setSending(false);
    };

    const handleModelChange = async (model: { modelID: string; providerID: string }) => {
        try {
            await updateAgentConfig(session.id, { model: { modelID: model.modelID } });
            setAgentConfig(prev => prev ? { ...prev, model: { modelID: model.modelID, providerID: model.providerID } } : prev);
        } catch (err) {
            setErrorMessage(`Failed to change model: ${err instanceof Error ? err.message : String(err)}`);
        }
    };

    const handleStop = async () => {
        try {
            await stopAgentSession(session.id);
            onStop();
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
                currentModel={agentConfig?.model}
                onModelChange={handleModelChange}
            />

            <div className="mcc-agent-messages" ref={messagesContainerRef}>
                {connecting && (
                    <div className="mcc-agent-loading" style={{ padding: '16px', textAlign: 'center' }}>
                        Connecting to session...
                    </div>
                )}
                {!connecting && (
                    <>
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
                    </>
                )}
            </div>

            {errorMessage && (
                <div className="mcc-agent-error-toast">
                    <span>{errorMessage}</span>
                    <button className="mcc-agent-error-toast-close" onClick={() => setErrorMessage(null)}>×</button>
                </div>
            )}

            <div className="mcc-agent-input-area">
                <textarea
                    className="mcc-agent-input"
                    placeholder="Type a message..."
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={2}
                    disabled={sending || agentProcessing || connecting}
                />
                {agentProcessing ? (
                    <button
                        className="mcc-agent-send-btn mcc-agent-stop-btn"
                        onClick={handleStop}
                    >
                        Stop
                    </button>
                ) : (
                    <button
                        className="mcc-agent-send-btn"
                        onClick={handleSend}
                        disabled={!input.trim() || sending}
                    >
                        {sending ? '...' : 'Send'}
                    </button>
                )}
            </div>
        </div>
    );
}
