import { useState, useRef, useEffect, useImperativeHandle, forwardRef, useMemo } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import { useCurrent } from '../../../../../../hooks/useCurrent';
import type { ModelOption } from '../../../../../../pure-view/selector/ModelSelector';
import { createACPAPI, type CursorACPModelInfo, fetchCursorACPSessionSettings, saveCursorACPSessionSettings } from '../../../../../../api/cursor-acp';
import type { ChatMessage } from './ACPChatTypes';
import { ACPChatHeader } from './ACPChatHeader';
import { ACPChatToolbar } from './ACPChatToolbar';
import { ACPChatMessages } from './ACPChatMessages';
import { ACPChatInput } from './ACPChatInput';
import '../ACPUI.css';

export type { ChatMessage } from './ACPChatTypes';

export interface ACPChatHandle {
    connect(resumeSessionId?: string, projectName?: string, worktreeId?: string): void;
}

export interface ACPChatProps {
    title: string;
    agentName: string;
    apiPrefix: string;
    emptyConnectedMessage?: string;
}

export const ACPChat = forwardRef<ACPChatHandle, ACPChatProps>(function ACPChat({
    title,
    agentName,
    apiPrefix,
    emptyConnectedMessage = `Send a message to start coding with ${agentName} agent.`,
}, ref) {
    const navigate = useNavigate();
    const location = useLocation();
    const { sessionId: paramSessionId } = useParams<{ sessionId: string }>();
    const isNewSession = paramSessionId === 'new';
    const api = useMemo(() => createACPAPI(apiPrefix), [apiPrefix]);

    const [status, setStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
    const [statusMessage, setStatusMessage] = useState('');
    const [sessionId, setSessionId] = useState<string | null>(null);
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [input, setInput] = useState('');
    const [isProcessing, setIsProcessing] = useState(false);
    const [dir, setDir] = useState('');
    const [model, setModel] = useState('');
    const [models, setModels] = useState<ModelOption[]>([]);
    const [selectedModel, setSelectedModel] = useState<{ modelID: string; providerID: string } | undefined>();
    const [connectLogs, setConnectLogs] = useState<string[]>([]);
    const [showConnectLogs, setShowConnectLogs] = useState(true);
    const [debugMode, setDebugMode] = useState(false);
    const [debugLogs, setDebugLogs] = useState<string[]>([]);
    const [yoloMode, setYoloMode] = useState(false);
    const [loadErrors, setLoadErrors] = useState<string[]>([]);

    const chatContainerRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const connectStarted = useRef(false);
    const userScrolledUp = useRef(false);
    const pendingYolo = useRef(false);

    const modelsRef = useCurrent(models);
    const isNewSessionRef = useCurrent(isNewSession);
    const locationRef = useCurrent(location);
    const debugModeRef = useCurrent(debugMode);
    const inputRef = useCurrent(input);
    const sessionIdRef = useCurrent(sessionId);
    const isProcessingRef = useCurrent(isProcessing);
    const messagesLenRef = useCurrent(messages.length);
    const selectedModelRef = useCurrent(selectedModel);

    const handleChatScroll = () => {
        const el = chatContainerRef.current;
        if (!el) return;
        const threshold = 80;
        const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < threshold;
        userScrolledUp.current = !isNearBottom;
    };

    const connect = async (resumeSessionId?: string, projectName?: string, worktreeId?: string) => {
        setStatus('connecting');
        setStatusMessage(`Initializing ${agentName} agent...`);
        setConnectLogs([]);
        setDebugLogs([]);
        setShowConnectLogs(true);
        try {
            const body: Record<string, string | boolean> = {};
            if (resumeSessionId) body.sessionId = resumeSessionId;
            if (projectName) body.projectName = projectName;
            if (worktreeId) body.worktreeId = worktreeId;
            body.debug = debugModeRef.current;

            const resp = await api.connect(body);
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
        connect(resumeSessionId?: string, projectName?: string, worktreeId?: string) {
            connect(resumeSessionId, projectName, worktreeId);
        },
    }));

    useEffect(() => {
        if (connectStarted.current) return;
        connectStarted.current = true;

        const urlParams = new URLSearchParams(location.search);
        const projectName = urlParams.get('project') || '';
        const worktreeId = urlParams.get('worktree') || '';

        (async () => {
            try {
                const data = await api.fetchModels();
                const modelOptions: ModelOption[] = data.map((m: CursorACPModelInfo) => ({
                    id: m.id,
                    name: m.name || m.id,
                    providerId: m.providerId || 'cursor',
                    providerName: m.providerName || 'Cursor',
                }));
                setModels(modelOptions);
                const current = data.find(m => m.is_current);
                if (current) {
                    setSelectedModel({ modelID: current.id, providerID: current.providerId || 'cursor' });
                } else if (modelOptions.length > 0) {
                    setSelectedModel({ modelID: modelOptions[0].id, providerID: modelOptions[0].providerId });
                }
            } catch (err) {
                setLoadErrors(prev => [...prev, `Models: ${err instanceof Error ? err.message : String(err)}`]);
            }
        })();

        if (!isNewSession && paramSessionId) {
            (async () => {
                try {
                    const sessionData = await api.fetchSession(paramSessionId);
                    if (sessionData.dir) setDir(sessionData.dir);
                } catch (err) {
                    setLoadErrors(prev => [...prev, `Session: ${err instanceof Error ? err.message : String(err)}`]);
                }

                try {
                    const saved = await api.fetchSessionMessages(paramSessionId);
                    if (saved.length > 0) setMessages(saved as ChatMessage[]);
                } catch (err) {
                    setLoadErrors(prev => [...prev, `Messages: ${err instanceof Error ? err.message : String(err)}`]);
                }

                try {
                    const settings = await fetchCursorACPSessionSettings(paramSessionId);
                    setYoloMode(settings.yoloMode || false);
                } catch (err) {
                    setLoadErrors(prev => [...prev, `Settings: ${err instanceof Error ? err.message : String(err)}`]);
                }
            })();
            connect(paramSessionId, projectName, worktreeId);
        }
    }, [isNewSession, paramSessionId, apiPrefix, location.search]);

    const saveMessages = (sid: string, msgs: ChatMessage[]) => {
        api.saveSessionMessages(sid, msgs).catch(() => {});
    };

    const sendPrompt = async () => {
        const currentInput = inputRef.current;
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
            const resp = await api.sendPrompt({
                sessionId: currentSessionId,
                prompt: userMessage,
                model: selectedModelRef.current?.modelID || '',
                debug: debugModeRef.current,
            }, controller.signal);

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
            try { await api.cancel(sid); } catch { /* best-effort */ }
        }
    };

    const handleBack = async () => {
        if (status === 'connected' || status === 'connecting') {
            try { await api.disconnect(); } catch { /* best-effort */ }
        }
        const backPath = location.pathname.replace(/\/[^/]+$/, '');
        navigate(backPath);
    };

    const yoloRef = useCurrent(yoloMode);

    const handleYoloToggle = () => {
        const newValue = !yoloMode;
        setYoloMode(newValue);
        pendingYolo.current = true;
        const sid = sessionIdRef.current;
        if (sid) {
            pendingYolo.current = false;
            saveCursorACPSessionSettings({ sessionId: sid, yoloMode: newValue }).catch(() => {});
        }
    };

    useEffect(() => {
        if (!sessionId || !pendingYolo.current) return;
        pendingYolo.current = false;
        saveCursorACPSessionSettings({ sessionId, yoloMode: yoloRef.current }).catch(() => {});
    }, [sessionId]);

    const handleModelSelect = (m: { modelID: string; providerID: string }) => {
        setSelectedModel(m);
        if (sessionId) {
            api.updateSessionModel(sessionId, m.modelID).catch(() => {});
        }
    };

    return (
        <div className="acp-ui-container">
            <ACPChatHeader
                title={title}
                status={status}
                statusMessage={statusMessage}
                sessionId={sessionId}
                isNewSession={isNewSession}
                onBack={handleBack}
            />

            <ACPChatToolbar
                models={models}
                selectedModel={selectedModel}
                onModelSelect={handleModelSelect}
                modelPlaceholder={model || 'Select model...'}
                yoloMode={yoloMode}
                onYoloToggle={handleYoloToggle}
                debugMode={debugMode}
                onDebugToggle={() => setDebugMode(!debugMode)}
                dir={dir}
                status={status}
            />

            {loadErrors.length > 0 && (
                <div style={{ padding: '6px 12px', background: 'rgba(248,113,113,0.1)', borderRadius: 6, margin: '0 12px' }}>
                    {loadErrors.map((err, i) => (
                        <div key={i} style={{ fontSize: 12, color: 'var(--mcc-accent-red, #f87171)' }}>{err}</div>
                    ))}
                </div>
            )}

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

            <ACPChatMessages
                messages={messages}
                status={status}
                agentName={agentName}
                emptyConnectedMessage={emptyConnectedMessage}
                chatContainerRef={chatContainerRef}
                onScroll={handleChatScroll}
            />

            <ACPChatInput
                input={input}
                onInputChange={setInput}
                onSend={sendPrompt}
                onCancel={cancelPrompt}
                isProcessing={isProcessing}
                status={status}
            />
        </div>
    );
});
