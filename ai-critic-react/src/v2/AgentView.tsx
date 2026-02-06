import { useState, useEffect } from 'react';
import { useCurrent } from '../hooks/useCurrent';
import { useAutoScroll } from '../hooks/useAutoScroll';
import {
    fetchAgents, fetchAgentSessions, launchAgentSession, stopAgentSession,
    getOrCreateOpencodeSession, fetchMessages, sendPromptAsync, agentEventUrl,
    fetchOpencodeConfig, fetchOpencodeProviders,
    AgentSessionStatuses,
} from '../api/agents';
import type { AgentDef, AgentSessionInfo, AgentMessage, MessagePart, OpencodeConfig } from '../api/agents';
import './AgentView.css';

// ---- Props ----

interface AgentViewProps {
    projectDir: string | null;
    projectName: string | null;
    /** Current sub-view from URL (e.g., "chat") */
    currentView: string;
    /** Navigate to a sub-view within the Agent tab */
    onNavigateToView: (view: string) => void;
}

export function AgentView({ projectDir, projectName, currentView, onNavigateToView }: AgentViewProps) {
    const [agents, setAgents] = useState<AgentDef[]>([]);
    const [agentsLoading, setAgentsLoading] = useState(true);
    const [session, setSession] = useState<AgentSessionInfo | null>(null);
    const [launchError, setLaunchError] = useState('');

    // Fetch agents list
    useEffect(() => {
        fetchAgents()
            .then(data => { setAgents(data); setAgentsLoading(false); })
            .catch(() => setAgentsLoading(false));
    }, []);

    // Check for existing sessions matching this project
    const projectDirRef = useCurrent(projectDir);
    useEffect(() => {
        if (!projectDir) return;
        fetchAgentSessions()
            .then(sessions => {
                const existing = sessions.find(
                    s => s.project_dir === projectDirRef.current &&
                        (s.status === AgentSessionStatuses.Running || s.status === AgentSessionStatuses.Starting)
                );
                if (existing) {
                    setSession(existing);
                }
            })
            .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [projectDir]);

    const handleLaunchHeadless = async (agent: AgentDef) => {
        if (!projectDir) return;
        setLaunchError('');
        try {
            const sessionInfo = await launchAgentSession(agent.id, projectDir);
            setSession(sessionInfo);
            onNavigateToView('chat');
        } catch (err) {
            setLaunchError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleStopSession = async () => {
        if (!session) return;
        try {
            await stopAgentSession(session.id);
        } catch { /* ignore */ }
        setSession(null);
        onNavigateToView('');
    };

    if (!projectDir) {
        return (
            <div className="mcc-agent-view">
                <div className="mcc-empty-state">
                    <AgentIcon />
                    <h3>No Project Selected</h3>
                    <p>Select a project from the Home tab to start an agent.</p>
                </div>
            </div>
        );
    }

    if (currentView === 'chat' && session) {
        return (
            <AgentChat
                session={session}
                projectName={projectName}
                onStop={handleStopSession}
                onBack={() => onNavigateToView('')}
                onSessionUpdate={setSession}
            />
        );
    }

    return (
        <AgentPicker
            agents={agents}
            loading={agentsLoading}
            projectName={projectName}
            launchError={launchError}
            activeSession={session}
            onLaunchHeadless={handleLaunchHeadless}
            onResumeChat={() => onNavigateToView('chat')}
            onStopSession={handleStopSession}
        />
    );
}

// ---- Agent Picker ----

interface AgentPickerProps {
    agents: AgentDef[];
    loading: boolean;
    projectName: string | null;
    launchError: string;
    activeSession: AgentSessionInfo | null;
    onLaunchHeadless: (agent: AgentDef) => void;
    onResumeChat: () => void;
    onStopSession: () => void;
}

function AgentPicker({ agents, loading, projectName, launchError, activeSession, onLaunchHeadless, onResumeChat, onStopSession }: AgentPickerProps) {
    return (
        <div className="mcc-agent-view">
            <div className="mcc-agent-header">
                <h2>Agents</h2>
                <div className="mcc-agent-project-badge">
                    <FolderIcon />
                    <span>{projectName}</span>
                </div>
            </div>

            {loading && <div className="mcc-agent-loading">Loading agents...</div>}
            {launchError && <div className="mcc-agent-error">{launchError}</div>}

            {/* Active session banner */}
            {activeSession && (
                <div className="mcc-agent-active-session">
                    <div className="mcc-agent-active-session-info">
                        <span className="mcc-agent-active-session-label">Active session</span>
                        <span className={`mcc-agent-active-session-status ${activeSession.status}`}>
                            {activeSession.status}
                        </span>
                    </div>
                    <div className="mcc-agent-active-session-actions">
                        <button className="mcc-forward-btn" onClick={onResumeChat}>
                            Resume Chat
                        </button>
                        <button className="mcc-agent-stop-btn" onClick={onStopSession}>
                            Stop
                        </button>
                    </div>
                </div>
            )}

            <div className="mcc-agent-list">
                {agents.map(agent => (
                    <div key={agent.id} className="mcc-agent-card">
                        <div className="mcc-agent-card-header">
                            <div className="mcc-agent-card-info">
                                <span className="mcc-agent-card-name">{agent.name}</span>
                                <span className={`mcc-agent-card-status ${agent.installed ? 'installed' : 'not-installed'}`}>
                                    {agent.installed ? 'Installed' : 'Not installed'}
                                </span>
                            </div>
                        </div>
                        <div className="mcc-agent-card-desc">{agent.description}</div>
                        <div className="mcc-agent-card-actions">
                            {agent.headless && agent.installed && !activeSession && (
                                <button
                                    className="mcc-forward-btn mcc-agent-launch-btn"
                                    onClick={() => onLaunchHeadless(agent)}
                                >
                                    Start Chat
                                </button>
                            )}
                            {agent.headless && agent.installed && activeSession && (
                                <span className="mcc-agent-card-note">Session already active</span>
                            )}
                            {!agent.headless && agent.installed && (
                                <span className="mcc-agent-card-note">Terminal-only agent</span>
                            )}
                            {!agent.installed && (
                                <span className="mcc-agent-card-note">Not available</span>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ---- Agent Chat ----

interface AgentChatProps {
    session: AgentSessionInfo;
    projectName: string | null;
    onStop: () => void;
    onBack: () => void;
    onSessionUpdate: (session: AgentSessionInfo) => void;
}

function AgentChat({ session, projectName, onStop, onBack, onSessionUpdate }: AgentChatProps) {
    const [messages, setMessages] = useState<AgentMessage[]>([]);
    const [input, setInput] = useState('');
    const [sending, setSending] = useState(false);
    const [opencodeSID, setOpencodeSID] = useState<string | null>(null);
    const [agentConfig, setAgentConfig] = useState<OpencodeConfig | null>(null);
    const [contextLimit, setContextLimit] = useState<number>(0);
    const messagesContainerRef = useAutoScroll([messages]);
    const sessionRef = useCurrent(session);
    const opencodeSIDRef = useCurrent(opencodeSID);

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

    // Create an opencode session once agent is running
    useEffect(() => {
        if (session.status !== AgentSessionStatuses.Running) return;
        if (opencodeSIDRef.current) return;

        getOrCreateOpencodeSession(session.id)
            .then(data => {
                if (data?.id) setOpencodeSID(data.id);
            })
            .catch(() => {});
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
        const sid = opencodeSID;
        if (!sid || session.status !== AgentSessionStatuses.Running) return;
        fetchMessages(session.id, sid)
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
                    placeholder={opencodeSID ? 'Type a message...' : 'Connecting to agent...'}
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={2}
                    disabled={!opencodeSID || sending}
                />
                <button
                    className="mcc-agent-send-btn"
                    onClick={handleSend}
                    disabled={!input.trim() || !opencodeSID || sending}
                >
                    {sending ? '...' : 'Send'}
                </button>
            </div>
        </div>
    );
}

// ---- Chat Header (shared) ----

function AgentChatHeader({ agentName, projectName, onStop, onBack, stopLabel = 'Stop', modelName, contextPercent }: {
    agentName: string;
    projectName: string | null;
    onStop?: () => void;
    onBack?: () => void;
    stopLabel?: string;
    modelName?: string;
    contextPercent?: number;
}) {
    return (
        <div className="mcc-agent-chat-header">
            {onBack && (
                <button className="mcc-agent-back-btn" onClick={onBack} title="Back to agents">
                    <BackIcon />
                </button>
            )}
            <div className="mcc-agent-chat-title">
                <span>{agentName}</span>
                {modelName && (
                    <div className="mcc-agent-model-info">
                        <span className="mcc-agent-model-name">{modelName}</span>
                        {contextPercent !== undefined && (
                            <span className="mcc-agent-context-usage">{contextPercent}%</span>
                        )}
                    </div>
                )}
            </div>
            {onStop && <button className="mcc-agent-stop-btn" onClick={onStop}>{stopLabel}</button>}
        </div>
    );
}

// ---- Message Grouping ----

function groupMessagesByRole(messages: AgentMessage[]): AgentMessage[][] {
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

function ChatMessageGroup({ messages }: { messages: AgentMessage[] }) {
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

    if (part.type === 'tool-invocation' || part.type === 'tool_use' || part.type === 'tool-result') {
        const toolName = part.tool || 'tool';
        const isRunning = part.state === 'running' || part.state === 'partial-call';
        return (
            <div className={`mcc-agent-msg-tool ${isRunning ? 'running' : ''}`}>
                <div className="mcc-agent-msg-tool-header">
                    <span className="mcc-agent-msg-tool-icon">{isRunning ? '⏳' : '⚙️'}</span>
                    <span className="mcc-agent-msg-tool-name">{toolName}</span>
                </div>
                {part.output && (
                    <pre className="mcc-agent-msg-tool-output">{truncate(part.output, 500)}</pre>
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

function truncate(text: string, maxLen: number): string {
    if (text.length <= maxLen) return text;
    return text.slice(0, maxLen) + '...';
}

// ---- Icons ----

function BackIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="15 18 9 12 15 6" />
        </svg>
    );
}

function AgentIcon() {
    return (
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="mcc-empty-icon">
            <path d="M12 2a4 4 0 0 1 4 4v2a4 4 0 0 1-8 0V6a4 4 0 0 1 4-4z" />
            <path d="M16 14H8a4 4 0 0 0-4 4v2h16v-2a4 4 0 0 0-4-4z" />
            <circle cx="9" cy="7" r="1" fill="currentColor" />
            <circle cx="15" cy="7" r="1" fill="currentColor" />
        </svg>
    );
}

function FolderIcon() {
    return (
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
        </svg>
    );
}
