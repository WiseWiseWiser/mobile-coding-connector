import { useState, useRef, useEffect, useImperativeHandle, forwardRef } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import { useCurrent } from '../../../../../hooks/useCurrent';
import { BeakerIcon } from '../../../../../pure-view/icons/BeakerIcon';
import { ModelSelector, type ModelOption } from '../../../../../pure-view/selector/ModelSelector';
import './ACPUI.css';

interface ChatMessage {
    role: 'user' | 'agent';
    content: string;
    toolCalls?: ToolCallInfo[];
    plan?: PlanEntry[];
}

interface ToolCallInfo {
    id: string;
    title: string;
    status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
    content?: string;
}

interface PlanEntry {
    content: string;
    status: 'pending' | 'in_progress' | 'completed';
    priority?: string;
}

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

export interface ACPChatHandle {
    connect(cwd?: string, resumeSessionId?: string, projectName?: string, worktreeId?: string): void;
}

export interface ACPChatProps {
    title: string;
    agentName: string;
    apiPrefix: string;
    defaultCwd?: string;
    emptyConnectedMessage?: string;
}

export const ACPChat = forwardRef<ACPChatHandle, ACPChatProps>(function ACPChat({
    title,
    agentName,
    apiPrefix,
    defaultCwd = '',
    emptyConnectedMessage = `Send a message to start coding with ${agentName} agent.`,
}, ref) {
    const navigate = useNavigate();
    const location = useLocation();
    const { sessionId: paramSessionId } = useParams<{ sessionId: string }>();
    const isNewSession = paramSessionId === 'new';

    console.log("DEBUG ACPChat render", { paramSessionId, isNewSession, defaultCwd });

    const [status, setStatus] = useState<ConnectionStatus>('disconnected');
    const [statusMessage, setStatusMessage] = useState('');
    const [sessionId, setSessionId] = useState<string | null>(null);
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [input, setInput] = useState('');
    const [isProcessing, setIsProcessing] = useState(false);
    const [cwd, setCwd] = useState(defaultCwd ?? '');
    const [dir, setDir] = useState('');
    const [model, setModel] = useState('');
    const [models, setModels] = useState<ModelOption[]>([]);
    const [selectedModel, setSelectedModel] = useState<{ modelID: string; providerID: string } | undefined>();
    const [connectLogs, setConnectLogs] = useState<string[]>([]);
    const [showConnectLogs, setShowConnectLogs] = useState(true);
    const [debugMode, setDebugMode] = useState(false);
    const [debugLogs, setDebugLogs] = useState<string[]>([]);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const chatContainerRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLTextAreaElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const connectStarted = useRef(false);
    const userScrolledUp = useRef(false);

    const scrollToBottom = () => {
        if (!userScrolledUp.current) {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    };

    const handleChatScroll = () => {
        const el = chatContainerRef.current;
        if (!el) return;
        const threshold = 80;
        const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < threshold;
        userScrolledUp.current = !isNearBottom;
    };

    useEffect(() => {
        if (defaultCwd) setCwd(defaultCwd);
    }, [defaultCwd]);

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const cwdRef = useCurrent(cwd);
    const modelsRef = useCurrent(models);
    const isNewSessionRef = useCurrent(isNewSession);
    const locationRef = useCurrent(location);
    const debugModeRef = useCurrent(debugMode);

    const connect = async (resumeSessionId?: string, cwdOverride?: string, projectName?: string, worktreeId?: string) => {
        const effectiveCwd = cwdOverride ?? cwdRef.current;
        console.log("DEBUG ACPChat.connect", { resumeSessionId, cwdOverride, cwdRefCurrent: cwdRef.current, effectiveCwd, projectName, worktreeId });
        if (cwdOverride !== undefined) setCwd(cwdOverride);

        setStatus('connecting');
        setStatusMessage(`Initializing ${agentName} agent...`);
        setConnectLogs([]);
        setDebugLogs([]);
        setShowConnectLogs(true);
        try {
            const body: Record<string, string | boolean> = {};
            if (resumeSessionId) body.sessionId = resumeSessionId;
            if (effectiveCwd) body.cwd = effectiveCwd;
            if (projectName) body.projectName = projectName;
            if (worktreeId) body.worktreeId = worktreeId;
            body.debug = debugModeRef.current;

            const resp = await fetch(`${apiPrefix}/connect`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            if (!resp.body) throw new Error('No response stream');

            const reader = resp.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    if (!line.startsWith('data: ')) continue;
                    const data = line.slice(6);
                    if (data === '[DONE]') continue;

                    try {
                        const update = JSON.parse(data);
                        if (debugModeRef.current) {
                            setDebugLogs(prev => [...prev, `[${update.type}] ${JSON.stringify(update)}`]);
                        }
                        if (update.type === 'log') {
                            setConnectLogs(prev => [...prev, update.message]);
                        } else if (update.type === 'connected') {
                            const newId = update.message;
                            setSessionId(newId);
                            if (update.model) {
                                setModel(update.model);
                                const match = modelsRef.current.find(m => m.name === update.model || m.id === update.model);
                                if (match) {
                                    setSelectedModel({ modelID: match.id, providerID: match.providerId });
                                }
                            }
                            // Handle the resolved directory from the backend
                            if (update.dir !== undefined) {
                                setDir(update.dir);
                            }
                            setStatus('connected');
                            setStatusMessage(`Session: ${newId.slice(0, 12)}...`);
                            if (isNewSessionRef.current) {
                                const newPath = locationRef.current.pathname.replace(/\/new$/, `/${newId}`);
                                navigate(newPath, { replace: true });
                            }
                        } else if (update.type === 'error') {
                            throw new Error(update.message);
                        }
                    } catch (e) {
                        if (e instanceof SyntaxError) continue;
                        throw e;
                    }
                }
            }
        } catch (e) {
            setStatus('error');
            setStatusMessage(e instanceof Error ? e.message : 'Connection failed');
            setConnectLogs(prev => [...prev, `Error: ${e instanceof Error ? e.message : String(e)}`]);
        }
    };

    useImperativeHandle(ref, () => ({
        connect(cwdOverride?: string, resumeSessionId?: string, projectName?: string, worktreeId?: string) {
            connect(resumeSessionId, cwdOverride, projectName, worktreeId);
        },
    }));

    useEffect(() => {
        if (connectStarted.current) return;
        connectStarted.current = true;

        // Extract project name and worktree ID from URL query params
        const urlParams = new URLSearchParams(location.search);
        const projectName = urlParams.get('project') || '';
        const worktreeId = urlParams.get('worktree') || '';

        (async () => {
            try {
                const resp = await fetch(`${apiPrefix}/status`);
                const data = await resp.json();
                if (data.cwd && !cwd) setCwd(data.cwd);
            } catch { /* ignore */ }
        })();

        (async () => {
            try {
                const resp = await fetch(`${apiPrefix}/models`);
                const data: ModelOption[] = await resp.json();
                setModels(data);
                const current = data.find((m: ModelOption & { is_current?: boolean }) => (m as any).is_current);
                if (current) {
                    setSelectedModel({ modelID: current.id, providerID: current.providerId });
                } else if (data.length > 0) {
                    setSelectedModel({ modelID: data[0].id, providerID: data[0].providerId });
                }
            } catch { /* ignore */ }
        })();

        if (!isNewSession && paramSessionId) {
            // For existing sessions, fetch session info to get the dir
            (async () => {
                try {
                    // Fetch session info first
                    const sessionResp = await fetch(`${apiPrefix}/session?sessionId=${encodeURIComponent(paramSessionId)}`);
                    if (sessionResp.ok) {
                        const sessionData = await sessionResp.json();
                        if (sessionData.dir) {
                            setDir(sessionData.dir);
                        }
                    }
                } catch { /* ignore */ }

                try {
                    const resp = await fetch(`${apiPrefix}/session/messages?sessionId=${encodeURIComponent(paramSessionId)}`);
                    if (resp.ok) {
                        const saved = await resp.json();
                        if (Array.isArray(saved) && saved.length > 0) {
                            setMessages(saved);
                        }
                    }
                } catch { /* ignore */ }
            })();
            connect(paramSessionId, undefined, projectName, worktreeId);
        }
    }, [isNewSession, paramSessionId, apiPrefix, location.search]);

    const inputRef2 = useCurrent(input);
    const sessionIdRef = useCurrent(sessionId);
    const isProcessingRef = useCurrent(isProcessing);
    const messagesLenRef = useCurrent(messages.length);
    const selectedModelRef = useCurrent(selectedModel);

    const saveMessages = (sid: string, msgs: ChatMessage[]) => {
        fetch(`${apiPrefix}/session/messages`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sessionId: sid, messages: msgs }),
        }).catch(() => {});
    };

    const sendPrompt = async () => {
        const currentInput = inputRef2.current;
        const currentSessionId = sessionIdRef.current;
        if (!currentInput.trim() || !currentSessionId || isProcessingRef.current) return;

        const userMessage = currentInput.trim();
        setInput('');
        setIsProcessing(true);
        userScrolledUp.current = false;

        setMessages(prev => [...prev, { role: 'user', content: userMessage }]);

        const agentMsgIndex = messagesLenRef.current + 1;
        setMessages(prev => [...prev, { role: 'agent', content: '', toolCalls: [], plan: [] }]);

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            const resp = await fetch(`${apiPrefix}/prompt`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sessionId: currentSessionId, prompt: userMessage, model: selectedModelRef.current?.modelID || '', debug: debugModeRef.current }),
                signal: controller.signal,
            });

            if (!resp.ok) {
                const errData = await resp.json().catch(() => ({}));
                throw new Error(errData.message || `Request failed (${resp.status})`);
            }

            const reader = resp.body?.getReader();
            if (!reader) throw new Error('No response stream');

            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    if (!line.startsWith('data: ')) continue;
                    const data = line.slice(6);
                    if (data === '[DONE]') continue;

                    try {
                        const update = JSON.parse(data);
                        if (debugModeRef.current) {
                            setDebugLogs(prev => [...prev, `[${update.type}] ${JSON.stringify(update)}`]);
                        }
                        if (update.type === 'session_info') {
                            if (update.model) {
                                setModel(update.model);
                                const match = modelsRef.current.find(m => m.name === update.model || m.id === update.model);
                                if (match) {
                                    setSelectedModel({ modelID: match.id, providerID: match.providerId });
                                }
                            }
                            continue;
                        }
                        setMessages(prev => {
                            const next = [...prev];
                            const msg = { ...next[agentMsgIndex] };

                            switch (update.type) {
                                case 'agent_message_chunk':
                                    msg.content += update.text || '';
                                    break;
                                case 'plan':
                                    msg.plan = update.entries || [];
                                    break;
                                case 'tool_call': {
                                    const existing = (msg.toolCalls || []).find(t => t.id === update.toolCallId);
                                    if (existing) {
                                        existing.status = update.status;
                                        if (update.content) existing.content = update.content;
                                    } else {
                                        msg.toolCalls = [...(msg.toolCalls || []), {
                                            id: update.toolCallId,
                                            title: update.title || 'Tool call',
                                            status: update.status || 'pending',
                                            content: update.content,
                                        }];
                                    }
                                    break;
                                }
                                case 'error':
                                    msg.content += `\n\n**Error:** ${update.message}`;
                                    break;
                            }

                            next[agentMsgIndex] = msg;
                            return next;
                        });
                    } catch { /* skip malformed JSON */ }
                }
            }
        } catch (e) {
            if ((e as Error).name !== 'AbortError') {
                setMessages(prev => {
                    const next = [...prev];
                    const msg = { ...next[agentMsgIndex] };
                    msg.content += `\n\n**Error:** ${e instanceof Error ? e.message : String(e)}`;
                    next[agentMsgIndex] = msg;
                    return next;
                });
            }
        } finally {
            setIsProcessing(false);
            abortRef.current = null;
            const sid = sessionIdRef.current;
            if (sid) {
                setMessages(prev => {
                    saveMessages(sid, prev);
                    return prev;
                });
            }
        }
    };

    const cancelPrompt = async () => {
        abortRef.current?.abort();
        const sid = sessionIdRef.current;
        if (sid) {
            try {
                await fetch(`${apiPrefix}/cancel`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ sessionId: sid }),
                });
            } catch { /* ignore */ }
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendPrompt();
        }
    };

    const handleBack = async () => {
        if (status === 'connected' || status === 'connecting') {
            try {
                await fetch(`${apiPrefix}/disconnect`, { method: 'POST' });
            } catch { /* ignore */ }
        }
        const backPath = location.pathname.replace(/\/[^/]+$/, '');
        navigate(backPath);
    };

    return (
        <div className="acp-ui-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={handleBack}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>{title}</h2>
                <div className="mcc-header-status">
                    <span className={`mcc-status-dot mcc-status-${status === 'connected' ? 'running' : status === 'error' ? 'not-running' : 'checking'}`}></span>
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
                            <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.6 9a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06-.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
                        </svg>
                    </button>
                )}
            </div>

            <div className="acp-ui-toolbar">
                <ModelSelector
                    models={models}
                    currentModel={selectedModel}
                    onSelect={(m) => {
                        setSelectedModel(m);
                        if (sessionId) {
                            fetch(`${apiPrefix}/session/model`, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ sessionId, model: m.modelID }),
                            }).catch(() => {});
                        }
                    }}
                    placeholder={model || 'Select model...'}
                    disabled={models.length === 0}
                />
                <button
                    className={`acp-ui-debug-toggle ${debugMode ? 'active' : ''}`}
                    onClick={() => setDebugMode(!debugMode)}
                    title="Toggle debug mode"
                >
                    Debug
                </button>
                <div className="acp-ui-cwd-container">
                    <span className="acp-ui-cwd-label">cwd:</span>
                    {status === 'connected' || status === 'connecting' ? (
                        <span className="acp-ui-cwd-display">{cwd || 'N/A'}</span>
                    ) : (
                        <input
                            className="acp-ui-cwd-input"
                            value={cwd}
                            onChange={e => setCwd(e.target.value)}
                            placeholder="Working directory..."
                        />
                    )}
                </div>
                <div className="acp-ui-cwd-container">
                    <span className="acp-ui-cwd-label">Dir:</span>
                    <span className="acp-ui-cwd-display">
                        {dir || (status === 'connected' || status === 'connecting' ? 'loading...' : 'N/A')}
                    </span>
                </div>
            </div>

            {(showConnectLogs && connectLogs.length > 0) && (
                <div className="acp-ui-connect-logs">
                    <button
                        className="acp-ui-connect-logs-dismiss"
                        onClick={() => setShowConnectLogs(false)}
                        title="Dismiss logs"
                    >
                        <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
                            <path d="M6 6l12 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                            <path d="M18 6L6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                        </svg>
                    </button>
                    {(debugMode ? debugLogs : connectLogs).map((log, i) => (
                        <div key={i} className="acp-ui-connect-log-line">{log}</div>
                    ))}
                </div>
            )}

            <div className="acp-ui-chat" ref={chatContainerRef} onScroll={handleChatScroll}>
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

            <div className="acp-ui-input-area">
                <textarea
                    ref={inputRef}
                    className="acp-ui-input"
                    placeholder={status === 'connected' ? 'Type a message... (Enter to send, Shift+Enter for newline)' : 'Connecting...'}
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={status !== 'connected' || isProcessing}
                    rows={3}
                />
                <div className="acp-ui-input-actions">
                    {isProcessing ? (
                        <button className="mcc-btn-secondary" onClick={cancelPrompt}>Cancel</button>
                    ) : (
                        <button
                            className="mcc-btn-primary"
                            onClick={sendPrompt}
                            disabled={!input.trim() || status !== 'connected'}
                        >
                            Send
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
});
