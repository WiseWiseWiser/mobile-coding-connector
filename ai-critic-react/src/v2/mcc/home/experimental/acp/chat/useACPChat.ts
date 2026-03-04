import { useState, useRef, useEffect, useMemo } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import { useCurrent } from '../../../../../../hooks/useCurrent';
import type { ModelOption } from '../../../../../../pure-view/selector/ModelSelector';
import { createACPAPI, type CursorACPModelInfo, fetchCursorACPSettings, fetchCursorACPSessionSettings, saveCursorACPSessionSettings } from '../../../../../../api/cursor-acp';
import type { ChatMessage, ConnectionStatus } from './ACPChatTypes';

export interface UseACPChatOptions {
    agentName: string;
    apiPrefix: string;
}

export function useACPChat({ agentName, apiPrefix }: UseACPChatOptions) {
    const navigate = useNavigate();
    const location = useLocation();
    const { sessionId: paramSessionId } = useParams<{ sessionId: string }>();
    const isNewSession = paramSessionId === 'new';
    const api = useMemo(() => createACPAPI(apiPrefix), [apiPrefix]);

    const [status, setStatus] = useState<ConnectionStatus>('disconnected');
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
    const yoloRef = useCurrent(yoloMode);

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

    useEffect(() => {
        if (connectStarted.current) return;
        connectStarted.current = true;

        const urlParams = new URLSearchParams(location.search);
        const projectName = urlParams.get('project') || '';
        const worktreeId = urlParams.get('worktree') || '';

        (async () => {
            try {
                const [data, settings] = await Promise.all([
                    api.fetchModels(),
                    fetchCursorACPSettings().catch(() => null),
                ]);
                const modelOptions: ModelOption[] = data.map((m: CursorACPModelInfo) => ({
                    id: m.id,
                    name: m.name || m.id,
                    providerId: m.providerId || 'cursor',
                    providerName: m.providerName || 'Cursor',
                }));
                setModels(modelOptions);

                const configuredDefault = settings?.default_model
                    ? modelOptions.find(m => m.id === settings.default_model)
                    : undefined;
                if (configuredDefault) {
                    setSelectedModel({ modelID: configuredDefault.id, providerID: configuredDefault.providerId });
                } else {
                    const current = data.find(m => m.is_current);
                    if (current) {
                        setSelectedModel({ modelID: current.id, providerID: current.providerId || 'cursor' });
                    } else if (modelOptions.length > 0) {
                        setSelectedModel({ modelID: modelOptions[0].id, providerID: modelOptions[0].providerId });
                    }
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

    return {
        status,
        statusMessage,
        sessionId,
        isNewSession,
        messages,
        input,
        setInput,
        isProcessing,
        dir,
        model,
        models,
        selectedModel,
        connectLogs,
        showConnectLogs,
        setShowConnectLogs,
        debugMode,
        setDebugMode,
        debugLogs,
        yoloMode,
        loadErrors,
        chatContainerRef,

        connect,
        sendPrompt,
        cancelPrompt,
        handleBack,
        handleYoloToggle,
        handleModelSelect,
        handleChatScroll,
    };
}
