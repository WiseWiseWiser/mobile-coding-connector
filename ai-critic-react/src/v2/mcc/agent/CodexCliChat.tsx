import { useCallback, useEffect, useRef, useState } from 'react';
import { AgentChatHeader } from './AgentChatHeader';
import { NoZoomingInput } from '../components/NoZoomingInput';
import { useAutoScroll } from '../../../hooks/useAutoScroll';
import { PlusIcon } from '../../../pure-view/icons/PlusIcon';
import { SettingsIcon } from '../../../pure-view/icons/SettingsIcon';
import {
    fetchCodexModels,
    loadCodexApprovalPolicy,
    loadCodexDefaultModel,
    loadCodexSandbox,
    type CodexModel,
} from './codexSettings';
import './AgentView.css';

type ConnectionState = 'connecting' | 'connected' | 'disconnected';
type MessageKind = 'user' | 'assistant' | 'event' | 'error' | 'tool' | 'todo' | 'thinking';
type CodexToolStatus = 'running' | 'completed' | 'failed';
type CodexTodoStatus = 'pending' | 'in_progress' | 'completed';

interface CodexCliChatProps {
    projectName: string | null;
    projectDir: string;
    onBack: () => void;
    onSettings?: () => void;
}

interface CodexServerMessage {
    type: string;
    event?: unknown;
    raw?: string;
    data?: string;
    message?: string;
    session_id?: string;
    command?: string[];
    code?: number;
    error?: string;
    resume?: boolean;
    running?: boolean;
}

interface CodexChatMessage {
    id: string;
    kind: MessageKind;
    title?: string;
    text: string;
    raw?: string;
    tool?: CodexToolDetails;
    todos?: CodexTodoItem[];
}

interface CodexToolDetails {
    callID: string;
    name: string;
    title: string;
    status: CodexToolStatus;
    input?: string;
    output?: string;
    error?: string;
    command?: string;
    cwd?: string;
    file?: string;
    summary?: string;
    changes?: CodexFileChange[];
}

interface CodexFileChange {
    path: string;
    kind?: string;
}

interface CodexTodoItem {
    step: string;
    status: CodexTodoStatus;
}

interface CodexSession {
    id: string;
    title: string;
    project_dir?: string;
    model?: string;
    created_at?: string;
    updated_at?: string;
}

interface CodexHistoryMessage {
    role: string;
    text: string;
    time?: string;
}

interface CodexSessionMessagesResponse {
    messages?: CodexHistoryMessage[];
}

interface RawCodexEvent {
    id: string;
    type: string;
    text: string;
}

interface QueuedCodexPrompt {
    id: string;
    prompt: string;
}

export function CodexCliChat({ projectName, projectDir, onBack, onSettings }: CodexCliChatProps) {
    const [connectionState, setConnectionState] = useState<ConnectionState>('connecting');
    const [busy, setBusy] = useState(false);
    const [input, setInput] = useState('');
    const [model, setModel] = useState('');
    const [models, setModels] = useState<CodexModel[]>([]);
    const [sandbox] = useState(loadCodexSandbox);
    const [approvalPolicy] = useState(loadCodexApprovalPolicy);
    const [sessionID, setSessionID] = useState('');
    const [sessions, setSessions] = useState<CodexSession[]>([]);
    const [sessionsLoading, setSessionsLoading] = useState(false);
    const [sessionsOpen, setSessionsOpen] = useState(false);
    const [sessionTitleOpen, setSessionTitleOpen] = useState(false);
    const [queuedPrompts, setQueuedPrompts] = useState<QueuedCodexPrompt[]>([]);
    const [stopModalOpen, setStopModalOpen] = useState(false);
    const [messages, setMessages] = useState<CodexChatMessage[]>([
        {
            id: 'welcome',
            kind: 'event',
            title: 'Codex',
            text: 'Ready for a Codex prompt.',
        },
    ]);
    const [rawEvents, setRawEvents] = useState<RawCodexEvent[]>([]);

    const wsRef = useRef<WebSocket | null>(null);
    const inputRef = useRef<HTMLTextAreaElement | null>(null);
    const sessionTitleRef = useRef<HTMLDivElement | null>(null);
    const modelRef = useRef('');
    const sessionIDRef = useRef('');
    const queuedPromptsRef = useRef<QueuedCodexPrompt[]>([]);
    const handleServerMessageRef = useRef<(msg: CodexServerMessage) => void>(() => {});
    const activeAssistantIDRef = useRef<string | null>(null);
    const historyLoadSeqRef = useRef(0);
    const sessionStorageKey = `mcc.codex.session.${projectDir}`;
    const messagesContainerRef = useAutoScroll([messages, rawEvents]);

    const appendMessage = useCallback((message: Omit<CodexChatMessage, 'id'>) => {
        setMessages(prev => [...prev, { id: makeID(), ...message }]);
    }, []);

    const refreshSessions = useCallback(async () => {
        setSessionsLoading(true);
        try {
            const params = new URLSearchParams({ project_dir: projectDir });
            const response = await fetch(`/api/agents/codex/sessions?${params.toString()}`);
            if (!response.ok) throw new Error(await response.text());
            const data = await response.json();
            setSessions(Array.isArray(data.sessions) ? data.sessions : []);
        } catch {
            setSessions([]);
        } finally {
            setSessionsLoading(false);
        }
    }, [projectDir]);

    const appendAssistantText = useCallback((text: string, replace: boolean) => {
        const trimmed = text.trim();
        if (!trimmed) return;

        setMessages(prev => {
            const activeID = activeAssistantIDRef.current;
            if (activeID) {
                return prev.map(msg => {
                    if (msg.id !== activeID) return msg;
                    if (msg.text === trimmed) return msg;
                    return {
                        ...msg,
                        text: replace ? trimmed : msg.text + trimmed,
                    };
                });
            }

            const last = prev[prev.length - 1];
            if (last?.kind === 'assistant' && last.text === trimmed) {
                activeAssistantIDRef.current = last.id;
                return prev;
            }

            const id = makeID();
            activeAssistantIDRef.current = id;
            return [...prev, { id, kind: 'assistant', text: trimmed }];
        });
    }, []);

    const appendAssistantMessage = useCallback((text: string) => {
        const trimmed = text.trim();
        if (!trimmed) return;

        activeAssistantIDRef.current = null;
        setMessages(prev => {
            const last = prev[prev.length - 1];
            if (last?.kind === 'assistant' && last.text === trimmed) {
                return prev;
            }
            return [...prev, { id: makeID(), kind: 'assistant', text: trimmed }];
        });
    }, []);

    const resetAssistantStream = useCallback(() => {
        activeAssistantIDRef.current = null;
    }, []);

    const resizeInput = useCallback((target = inputRef.current) => {
        if (!target) return;
        target.style.height = 'auto';
        const style = window.getComputedStyle(target);
        const lineHeight = Number.parseFloat(style.lineHeight) || 22;
        const minHeight = Number.parseFloat(style.minHeight) || 40;
        const verticalSpace = target.offsetHeight - target.clientHeight;
        if (target.value.length === 0) {
            target.style.height = `${minHeight}px`;
            target.style.overflowY = 'hidden';
            return;
        }
        const maxHeight = Math.ceil(lineHeight * 3 + verticalSpace);
        const nextHeight = Math.min(target.scrollHeight, maxHeight);
        target.style.height = `${nextHeight}px`;
        target.style.overflowY = target.scrollHeight > maxHeight ? 'auto' : 'hidden';
    }, []);

    const upsertToolMessage = useCallback((tool: CodexToolDetails) => {
        setMessages(prev => {
            const index = prev.findIndex(msg => msg.kind === 'tool' && msg.tool?.callID && msg.tool.callID === tool.callID);
            if (index >= 0) {
                const next = [...prev];
                const existing = next[index].tool;
                const merged = mergeToolDetails(existing, tool);
                next[index] = {
                    ...next[index],
                    text: merged.summary || merged.title,
                    tool: merged,
                };
                return next;
            }
            return [...prev, {
                id: makeID(),
                kind: 'tool',
                title: tool.title,
                text: tool.summary || tool.title,
                tool,
            }];
        });
    }, []);

    const upsertTodoMessage = useCallback((callID: string, todos: CodexTodoItem[], title = 'Plan') => {
        setMessages(prev => {
            const index = callID ? prev.findIndex(msg => msg.kind === 'todo' && msg.tool?.callID === callID) : -1;
            const message: CodexChatMessage = {
                id: index >= 0 ? prev[index].id : makeID(),
                kind: 'todo',
                title,
                text: todos.map(todo => todo.step).join('\n'),
                todos,
                tool: {
                    callID,
                    name: 'update_plan',
                    title,
                    status: todos.some(todo => todo.status === 'in_progress') ? 'running' : 'completed',
                },
            };
            if (index < 0) return [...prev, message];
            const next = [...prev];
            next[index] = message;
            return next;
        });
    }, []);

    const handleCodexEvent = useCallback((event: unknown, raw?: string) => {
        const eventType = readString(event, 'type') || 'codex_event';
        setRawEvents(prev => [...prev.slice(-199), {
            id: makeID(),
            type: eventType,
            text: raw || stringifyEvent(event),
        }]);

        const richEvent = parseCodexRichEvent(event);
        if (richEvent?.type === 'tool') {
            resetAssistantStream();
            upsertToolMessage(richEvent.tool);
            return;
        }
        if (richEvent?.type === 'todo') {
            resetAssistantStream();
            upsertTodoMessage(richEvent.callID, richEvent.todos, richEvent.title);
            return;
        }
        if (richEvent?.type === 'thinking') {
            appendMessage({ kind: 'thinking', title: 'Reasoning', text: richEvent.text });
            return;
        }

        const resultText = readString(event, 'result');
        if (eventType === 'result' && resultText) {
            appendAssistantText(resultText, true);
            resetAssistantStream();
            return;
        }

        const text = extractCodexText(event);
        if (text) {
            if (isCompleteCodexAssistantMessage(event)) {
                appendAssistantMessage(text);
            } else {
                appendAssistantText(text, false);
            }
            return;
        }

        const eventSummary = summarizeCodexEvent(event);
        if (eventSummary) {
            appendMessage({
                kind: 'event',
                title: eventSummary.title,
                text: eventSummary.text,
                raw,
            });
        }
    }, [appendAssistantMessage, appendAssistantText, appendMessage, resetAssistantStream, upsertTodoMessage, upsertToolMessage]);

    const startCodexPrompt = useCallback((prompt: string, newSession = false, appendUserMessage = true) => {
        const ws = wsRef.current;
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            appendMessage({ kind: 'error', title: 'Disconnected', text: 'Reconnect before sending another prompt.' });
            setConnectionState('disconnected');
            return false;
        }

        if (newSession) {
            sessionIDRef.current = '';
            setSessionID('');
            window.localStorage.removeItem(sessionStorageKey);
        }

        setBusy(true);
        resetAssistantStream();
        if (appendUserMessage) {
            appendMessage({ kind: 'user', text: prompt });
        }
        ws.send(JSON.stringify({
            type: 'prompt',
            prompt,
            project_dir: projectDir,
            model: modelRef.current.trim() || undefined,
            sandbox,
            approval_policy: approvalPolicy,
            session_id: newSession ? undefined : sessionIDRef.current || undefined,
            new_session: newSession,
        }));
        return true;
    }, [appendMessage, approvalPolicy, projectDir, resetAssistantStream, sandbox, sessionStorageKey]);

    const runNextQueuedPrompt = useCallback(() => {
        const [nextPrompt, ...remainingPrompts] = queuedPromptsRef.current;
        if (!nextPrompt) return false;

        queuedPromptsRef.current = remainingPrompts;
        setQueuedPrompts(remainingPrompts);
        appendMessage({
            kind: 'event',
            title: 'Running queued prompt',
            text: nextPrompt.prompt,
        });
        window.setTimeout(() => {
            startCodexPrompt(nextPrompt.prompt, false, false);
        }, 0);
        return true;
    }, [appendMessage, startCodexPrompt]);

    const handleServerMessage = useCallback((msg: CodexServerMessage) => {
        switch (msg.type) {
        case 'ready':
            setConnectionState('connected');
            return;
        case 'started':
            setBusy(true);
            resetAssistantStream();
            appendMessage({
                kind: 'event',
                title: msg.resume ? 'Resumed Codex' : 'Started Codex',
                text: formatCommand(msg.command),
            });
            return;
        case 'attached':
            if (msg.session_id) {
                sessionIDRef.current = msg.session_id;
                setSessionID(msg.session_id);
                window.localStorage.setItem(sessionStorageKey, msg.session_id);
            }
            if (msg.running) {
                setBusy(true);
                resetAssistantStream();
            } else {
                setBusy(false);
                resetAssistantStream();
            }
            return;
        case 'session':
            if (msg.session_id) {
                sessionIDRef.current = msg.session_id;
                setSessionID(msg.session_id);
                window.localStorage.setItem(sessionStorageKey, msg.session_id);
                void refreshSessions();
            }
            return;
        case 'codex_event':
            handleCodexEvent(msg.event, msg.raw);
            return;
        case 'stderr':
            if (msg.data) {
                appendMessage({ kind: 'error', title: 'Codex stderr', text: msg.data });
            }
            return;
        case 'stdout':
            if (msg.data) {
                appendMessage({ kind: 'event', title: 'Codex output', text: msg.data });
            }
            return;
        case 'cancelled':
            setBusy(false);
            resetAssistantStream();
            appendMessage({ kind: 'event', title: 'Cancelled', text: 'Codex run was cancelled.' });
            runNextQueuedPrompt();
            return;
        case 'exit':
            setBusy(false);
            resetAssistantStream();
            void refreshSessions();
            if (msg.code === 0) {
                appendMessage({ kind: 'event', title: 'Completed', text: 'Codex finished.' });
            } else {
                appendMessage({
                    kind: 'error',
                    title: 'Codex exited',
                    text: msg.error || `Exit code ${msg.code ?? 'unknown'}`,
                });
            }
            runNextQueuedPrompt();
            return;
        case 'error':
            setBusy(false);
            resetAssistantStream();
            appendMessage({ kind: 'error', title: 'Error', text: msg.message || 'Codex request failed.' });
            return;
        default:
            appendMessage({ kind: 'event', title: msg.type, text: stringifyEvent(msg) });
        }
    }, [appendMessage, handleCodexEvent, refreshSessions, resetAssistantStream, runNextQueuedPrompt, sessionStorageKey]);

    useEffect(() => {
        handleServerMessageRef.current = handleServerMessage;
    }, [handleServerMessage]);

    const connect = useCallback(() => {
        const existing = wsRef.current;
        if (existing && (existing.readyState === WebSocket.OPEN || existing.readyState === WebSocket.CONNECTING)) return;

        setConnectionState('connecting');
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${protocol}//${window.location.host}/api/agents/codex/ws`);
        wsRef.current = ws;

        ws.onopen = () => {
            if (wsRef.current !== ws) return;
            setConnectionState('connected');
            if (sessionIDRef.current) {
                ws.send(JSON.stringify({
                    type: 'attach',
                    session_id: sessionIDRef.current,
                }));
            }
        };
        ws.onmessage = event => {
            if (wsRef.current !== ws) return;
            try {
                handleServerMessageRef.current(JSON.parse(event.data));
            } catch {
                appendMessage({ kind: 'error', title: 'Protocol error', text: String(event.data) });
            }
        };
        ws.onerror = () => {
            if (wsRef.current !== ws) return;
            setConnectionState('disconnected');
            appendMessage({ kind: 'error', title: 'WebSocket error', text: 'Connection to Codex bridge failed.' });
        };
        ws.onclose = () => {
            if (wsRef.current !== ws) return;
            setConnectionState('disconnected');
            setBusy(false);
            resetAssistantStream();
        };
    }, [appendMessage, resetAssistantStream]);

    const reconnectCodex = useCallback(() => {
        const existing = wsRef.current;
        if (existing?.readyState === WebSocket.OPEN) {
            setConnectionState('connected');
            return;
        }
        if (existing?.readyState === WebSocket.CONNECTING) return;

        if (existing) {
            existing.onopen = null;
            existing.onmessage = null;
            existing.onerror = null;
            existing.onclose = null;
            try {
                existing.close();
            } catch {
                // Ignore close failures for already-broken sockets.
            }
            wsRef.current = null;
        }
        connect();
    }, [connect]);

    useEffect(() => {
        const storedSessionID = window.localStorage.getItem(sessionStorageKey);
        if (storedSessionID) {
            sessionIDRef.current = storedSessionID;
            setSessionID(storedSessionID);
        }
    }, [sessionStorageKey]);

    useEffect(() => {
        modelRef.current = model;
    }, [model]);

    useEffect(() => {
        sessionIDRef.current = sessionID;
    }, [sessionID]);

    useEffect(() => {
        fetchCodexModels()
            .then(data => {
                const nextModels = data.models;
                setModels(nextModels);
                setModel((current) => {
                    const nextModel = current || loadCodexDefaultModel() || data.currentModel || nextModels[0]?.id || '';
                    modelRef.current = nextModel;
                    return nextModel;
                });
            })
            .catch(() => {
                setModels([]);
            });
        void refreshSessions();
    }, [refreshSessions]);

    useEffect(() => {
        connect();
        return () => {
            const ws = wsRef.current;
            if (ws) {
                ws.onopen = null;
                ws.onmessage = null;
                ws.onerror = null;
                ws.onclose = null;
                ws.close();
            }
            wsRef.current = null;
        };
    }, [connect]);

    useEffect(() => {
        resizeInput();
    }, [input, resizeInput]);

    useEffect(() => {
        if (!busy) {
            setStopModalOpen(false);
        }
    }, [busy]);

    useEffect(() => {
        setSessionTitleOpen(false);
    }, [sessionID]);

    useEffect(() => {
        if (!stopModalOpen) return;

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setStopModalOpen(false);
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
        };
    }, [stopModalOpen]);

    useEffect(() => {
        if (!sessionTitleOpen) return;

        const handlePointerDown = (event: PointerEvent) => {
            const target = event.target;
            if (target instanceof Node && sessionTitleRef.current?.contains(target)) {
                return;
            }
            setSessionTitleOpen(false);
        };
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setSessionTitleOpen(false);
            }
        };

        document.addEventListener('pointerdown', handlePointerDown);
        document.addEventListener('keydown', handleKeyDown);
        return () => {
            document.removeEventListener('pointerdown', handlePointerDown);
            document.removeEventListener('keydown', handleKeyDown);
        };
    }, [sessionTitleOpen]);

    const sendPrompt = (newSession = false) => {
        const prompt = input.trim();
        if (!prompt) return;

        const ws = wsRef.current;
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            appendMessage({ kind: 'error', title: 'Disconnected', text: 'Reconnect before sending another prompt.' });
            setConnectionState('disconnected');
            return;
        }

        setInput('');
        window.requestAnimationFrame(() => resizeInput());

        if (busy) {
            const queuedPrompt = { id: makeID(), prompt };
            const nextQueue = [...queuedPromptsRef.current, queuedPrompt];
            queuedPromptsRef.current = nextQueue;
            setQueuedPrompts(nextQueue);
            appendMessage({ kind: 'user', text: prompt });
            appendMessage({
                kind: 'event',
                title: 'Queued',
                text: 'This prompt will run after the current Codex turn finishes.',
            });
            return;
        }

        startCodexPrompt(prompt, newSession, true);
    };

    const cancelPrompt = () => {
        const ws = wsRef.current;
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            appendMessage({ kind: 'error', title: 'Disconnected', text: 'Reconnect before stopping the current Codex run.' });
            setConnectionState('disconnected');
            return;
        }
        appendMessage({ kind: 'event', title: 'Stopping', text: 'Stop requested for the current Codex run.' });
        ws.send(JSON.stringify({ type: 'cancel' }));
    };

    const requestCancelPrompt = () => {
        setStopModalOpen(true);
    };

    const confirmCancelPrompt = () => {
        setStopModalOpen(false);
        cancelPrompt();
    };

    const createNewSession = () => {
        if (busy) return;

        historyLoadSeqRef.current += 1;
        sessionIDRef.current = '';
        queuedPromptsRef.current = [];
        setSessionID('');
        setQueuedPrompts([]);
        setSessionsOpen(false);
        setSessionTitleOpen(false);
        setRawEvents([]);
        resetAssistantStream();
        window.localStorage.removeItem(sessionStorageKey);
        setMessages([{
            id: makeID(),
            kind: 'event',
            title: 'New Codex session',
            text: 'Send a prompt to start a fresh Codex session.',
        }]);
        window.requestAnimationFrame(() => inputRef.current?.focus());
    };

    const attachCodexSession = useCallback((targetSessionID: string) => {
        const ws = wsRef.current;
        if (!targetSessionID || !ws || ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({
            type: 'attach',
            session_id: targetSessionID,
        }));
    }, []);

    const selectSession = (session: CodexSession) => {
        const loadSeq = historyLoadSeqRef.current + 1;
        historyLoadSeqRef.current = loadSeq;
        const sessionTitle = session.title || session.id;

        sessionIDRef.current = session.id;
        setSessionID(session.id);
        window.localStorage.setItem(sessionStorageKey, session.id);
        if (session.model) {
            modelRef.current = session.model;
            setModel(session.model);
        }
        setSessionsOpen(false);
        resetAssistantStream();
        setMessages([{
            id: makeID(),
            kind: 'event',
            title: 'Loading session',
            text: sessionTitle,
        }]);

        const params = new URLSearchParams({ session_id: session.id });
        fetch(`/api/agents/codex/session-messages?${params.toString()}`)
            .then(async response => {
                if (!response.ok) throw new Error(await response.text());
                return response.json() as Promise<CodexSessionMessagesResponse>;
            })
            .then(data => {
                if (historyLoadSeqRef.current !== loadSeq) return;
                const loadedMessages = (Array.isArray(data.messages) ? data.messages : [])
                    .filter(message => typeof message.text === 'string' && Boolean(message.text.trim()))
                    .slice(-10)
                    .map<CodexChatMessage>(message => ({
                        id: makeID(),
                        kind: message.role === 'user' ? 'user' : 'assistant',
                        text: message.text.trim(),
                    }));
                setMessages(loadedMessages.length > 0 ? loadedMessages : [{
                    id: makeID(),
                    kind: 'event',
                    title: 'Session selected',
                    text: 'No user or assistant messages were found in this session.',
                }]);
            })
            .catch(error => {
                if (historyLoadSeqRef.current !== loadSeq) return;
                setMessages([{
                    id: makeID(),
                    kind: 'error',
                    title: 'Session history unavailable',
                    text: error instanceof Error ? error.message : 'Failed to load session messages.',
                }]);
            })
            .finally(() => {
                attachCodexSession(session.id);
            });
    };

    const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            if (connectionState !== 'connected') {
                return;
            }
            sendPrompt(false);
        }
    };

    const activeSessionTitle = sessionID ? currentSessionTitle(sessions, sessionID) : '';
    const canSendPrompt = connectionState === 'connected';
    const sendButtonDisabled = !canSendPrompt || !input.trim();
    const sendButtonLabel = connectionState === 'connecting' ? 'Connecting' : busy ? 'Queue' : 'Send';
    const headerActions = (activeSessionTitle || onSettings) ? (
        <div className="mcc-codex-header-actions">
            {activeSessionTitle && (
                <div className="mcc-codex-session-title-anchor" ref={sessionTitleRef}>
                    <button
                        type="button"
                        className="mcc-codex-session-title-chip"
                        onClick={() => setSessionTitleOpen(open => !open)}
                        aria-expanded={sessionTitleOpen}
                        title="Show full session title"
                    >
                        {activeSessionTitle}
                    </button>
                    {sessionTitleOpen && (
                        <div className="mcc-codex-session-title-popover" role="dialog">
                            {activeSessionTitle}
                        </div>
                    )}
                </div>
            )}
            {onSettings && (
                <button className="mcc-agent-settings-icon-btn" onClick={onSettings} title="Codex settings">
                    <SettingsIcon />
                </button>
            )}
        </div>
    ) : undefined;

    return (
        <div className="mcc-agent-view mcc-agent-view-chat mcc-codex-chat">
            <AgentChatHeader
                agentName="Codex"
                projectName={projectName}
                onBack={onBack}
                rightActions={headerActions}
            />

            <div className="mcc-codex-toolbar">
                <div className="mcc-codex-toolbar-main">
                    <button
                        className={`mcc-codex-session-toggle${sessionsOpen ? ' active' : ''}`}
                        onClick={() => setSessionsOpen(open => !open)}
                    >
                        Sessions {sessions.length}
                    </button>
                    <select
                        className="mcc-codex-model-input"
                        value={model}
                        onChange={event => {
                            modelRef.current = event.target.value;
                            setModel(event.target.value);
                        }}
                        disabled={busy}
                        title="Model"
                    >
                        {models.length === 0 && <option value="">Default model</option>}
                        {models.map(item => (
                            <option key={item.id} value={item.id}>{item.name}</option>
                        ))}
                    </select>
                    <button
                        type="button"
                        className="mcc-codex-icon-btn mcc-codex-new-session-btn"
                        onClick={createNewSession}
                        disabled={busy}
                        title="New Codex session"
                        aria-label="New Codex session"
                    >
                        <PlusIcon />
                    </button>
                </div>
            </div>

            {sessionsOpen && (
                <button
                    className="mcc-codex-session-backdrop"
                    onClick={() => setSessionsOpen(false)}
                    aria-label="Close sessions"
                />
            )}
            <div className={`mcc-codex-session-panel${sessionsOpen ? ' open' : ''}`}>
                <div className="mcc-codex-session-panel-header">
                    <span>Sessions</span>
                    <div className="mcc-codex-session-panel-actions">
                        <button className="mcc-codex-icon-btn" onClick={refreshSessions} disabled={sessionsLoading} title="Refresh sessions">
                            ↻
                        </button>
                        <button className="mcc-codex-icon-btn" onClick={() => setSessionsOpen(false)} title="Close sessions">
                            x
                        </button>
                    </div>
                </div>
                {sessionsLoading && <div className="mcc-codex-session-empty">Loading sessions...</div>}
                {!sessionsLoading && sessions.length === 0 && <div className="mcc-codex-session-empty">No Codex sessions for this project.</div>}
                {!sessionsLoading && sessions.map(session => (
                    <button
                        key={session.id}
                        className={`mcc-codex-session-item${session.id === sessionID ? ' active' : ''}`}
                        onClick={() => selectSession(session)}
                    >
                        <span className="mcc-codex-session-title">{session.title || session.id}</span>
                        <span className="mcc-codex-session-meta">{formatSessionTime(session.updated_at || session.created_at)}</span>
                    </button>
                ))}
            </div>

            <div className="mcc-codex-body" ref={messagesContainerRef}>
                <div className="mcc-codex-transcript">
                    {messages.map(message => (
                        <CodexMessageView key={message.id} message={message} />
                    ))}
                    {busy && (
                        <div className="mcc-codex-running">
                            <div className="mcc-agent-spinner" />
                            <span>Codex is working...</span>
                            {queuedPrompts.length > 0 && (
                                <span className="mcc-codex-queue-count">{queuedPrompts.length} queued</span>
                            )}
                            <button
                                type="button"
                                className="mcc-codex-stop-inline"
                                onClick={requestCancelPrompt}
                                title="Stop Codex run"
                                aria-label="Stop Codex run"
                            >
                                <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                                    <rect x="2" y="2" width="8" height="8" rx="1.5" fill="currentColor" />
                                </svg>
                            </button>
                        </div>
                    )}
                    {connectionState !== 'connected' && (
                        <div className={`mcc-codex-connection-banner mcc-codex-connection-${connectionState}`}>
                            <span>{connectionState === 'connecting' ? 'Connecting to Codex bridge...' : 'Codex bridge disconnected.'}</span>
                            <button type="button" onClick={reconnectCodex} disabled={connectionState === 'connecting'}>
                                Reconnect
                            </button>
                        </div>
                    )}
                </div>
            </div>

            <div className="mcc-agent-input-area">
                <NoZoomingInput>
                    <textarea
                        ref={inputRef}
                        className="mcc-agent-input"
                        placeholder={connectionState === 'disconnected' ? 'Reconnect to send this prompt...' : busy ? 'Queue another Codex prompt...' : 'Ask Codex to inspect, edit, or explain this project...'}
                        value={input}
                        onChange={event => {
                            setInput(event.target.value);
                            resizeInput(event.currentTarget);
                        }}
                        onKeyDown={handleKeyDown}
                        rows={1}
                    />
                </NoZoomingInput>
                <button
                    className="mcc-agent-send-btn"
                    onClick={() => sendPrompt(false)}
                    disabled={sendButtonDisabled}
                >
                    {sendButtonLabel}
                </button>
            </div>
            {stopModalOpen && (
                <div className="mcc-codex-modal-backdrop" onClick={() => setStopModalOpen(false)}>
                    <div
                        className="mcc-codex-modal"
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby="mcc-codex-stop-title"
                        onClick={event => event.stopPropagation()}
                    >
                        <div className="mcc-codex-modal-title" id="mcc-codex-stop-title">Stop Codex run?</div>
                        <div className="mcc-codex-modal-text">
                            In-progress tool execution will be interrupted. Queued prompts will remain and run after this stop completes.
                        </div>
                        <div className="mcc-codex-modal-actions">
                            <button type="button" className="mcc-codex-modal-secondary" onClick={() => setStopModalOpen(false)}>
                                Keep running
                            </button>
                            <button type="button" className="mcc-codex-modal-danger" onClick={confirmCancelPrompt}>
                                Stop run
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

function CodexMessageView({ message }: { message: CodexChatMessage }) {
    if (message.kind === 'user' || message.kind === 'assistant') {
        return (
            <div className={`mcc-agent-msg ${message.kind === 'user' ? 'mcc-agent-msg-user' : 'mcc-agent-msg-assistant'}`}>
                <div className="mcc-agent-msg-avatar">{message.kind === 'user' ? 'U' : 'C'}</div>
                <div className="mcc-agent-msg-content">
                    <div className="mcc-agent-msg-text">{message.text}</div>
                </div>
            </div>
        );
    }

    if (message.kind === 'tool' && message.tool) {
        return <CodexToolCard tool={message.tool} />;
    }

    if (message.kind === 'todo' && message.todos) {
        return <CodexTodoCard title={message.title || 'Plan'} todos={message.todos} />;
    }

    if (message.kind === 'thinking') {
        return (
            <details className="mcc-codex-thinking">
                <summary>{message.title || 'Reasoning'}</summary>
                <div>{message.text}</div>
            </details>
        );
    }

    return (
        <div className={`mcc-codex-event mcc-codex-event-${message.kind}`}>
            {message.title && <div className="mcc-codex-event-title">{message.title}</div>}
            <div className="mcc-codex-event-text">{message.text}</div>
        </div>
    );
}

function CodexToolCard({ tool }: { tool: CodexToolDetails }) {
    const output = trimLongText(tool.output || '', 5000);
    const error = trimLongText(tool.error || '', 3000);
    const changes = tool.changes || [];
    return (
        <div className={`mcc-codex-tool-card mcc-codex-tool-${tool.status}`}>
            <div className="mcc-codex-tool-header">
                <span className="mcc-codex-tool-status">{formatToolStatus(tool.status)}</span>
                <span className="mcc-codex-tool-name">{tool.title || tool.name}</span>
            </div>
            {(tool.summary || tool.command || tool.file || tool.cwd) && (
                <div className="mcc-codex-tool-meta">
                    {tool.summary && <span>{tool.summary}</span>}
                    {tool.command && <code>{tool.command}</code>}
                    {tool.file && <code>{tool.file}</code>}
                    {tool.cwd && <span>{tool.cwd}</span>}
                </div>
            )}
            {changes.length > 0 && (
                <div className="mcc-codex-file-change-list">
                    {changes.map((change, index) => (
                        <div key={`${change.kind || 'change'}-${change.path}-${index}`} className="mcc-codex-file-change">
                            <span>{formatFileChangeKind(change.kind)}</span>
                            <code>{change.path}</code>
                        </div>
                    ))}
                </div>
            )}
            {tool.input && (
                <details className="mcc-codex-tool-details">
                    <summary>Input</summary>
                    <pre>{trimLongText(tool.input, 3000)}</pre>
                </details>
            )}
            {output && (
                <details className="mcc-codex-tool-details" open={tool.status === 'failed'}>
                    <summary>Output</summary>
                    <pre>{output}</pre>
                </details>
            )}
            {error && (
                <details className="mcc-codex-tool-details mcc-codex-tool-error" open>
                    <summary>Error</summary>
                    <pre>{error}</pre>
                </details>
            )}
        </div>
    );
}

function CodexTodoCard({ title, todos }: { title: string; todos: CodexTodoItem[] }) {
    return (
        <div className="mcc-codex-todo-card">
            <div className="mcc-codex-todo-title">{title}</div>
            <div className="mcc-codex-todo-list">
                {todos.map((todo, index) => (
                    <div key={`${todo.status}-${todo.step}-${index}`} className={`mcc-codex-todo-item ${todo.status}`}>
                        <span className="mcc-codex-todo-status">{formatTodoStatus(todo.status)}</span>
                        <span>{todo.step}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

type CodexRichEvent =
    | { type: 'tool'; tool: CodexToolDetails }
    | { type: 'todo'; callID: string; title: string; todos: CodexTodoItem[] }
    | { type: 'thinking'; text: string };

function parseCodexRichEvent(event: unknown): CodexRichEvent | null {
    const topType = readString(event, 'type');
    const payload = recordAt(event, ['payload']);
    const item = (topType === 'response_item' || topType === 'event_msg') && payload ? payload : asRecord(event);
    if (!item) return null;

    const liveItem = recordAt(event, ['item']);
    if (liveItem && (topType === 'item.started' || topType === 'item.completed')) {
        const parsed = parseCodexLiveItem(topType, liveItem);
        if (parsed) return parsed;
    }

    const itemType = readString(item, 'type');
    if (topType === 'event_msg' && itemType === 'agent_reasoning') {
        const text = readString(item, 'text');
        return text ? { type: 'thinking', text } : null;
    }

    if (itemType === 'function_call' || itemType === 'custom_tool_call') {
        const callID = readString(item, 'call_id') || makeID();
        const name = readString(item, 'name') || 'tool';
        const inputValue = readString(item, 'arguments') || readString(item, 'input');
        const parsedInput = parseMaybeJSON(inputValue);
        const todos = extractTodosFromTool(name, parsedInput);
        if (todos.length > 0) {
            return { type: 'todo', callID, title: 'Plan', todos };
        }

        const patchChanges = name === 'apply_patch' ? extractPatchChanges(inputValue) : [];
        const command = extractCommandText(parsedInput);
        const commandPresentation = command ? inferCommandFileOperation(command) : null;
        return {
            type: 'tool',
            tool: {
                callID,
                name: commandPresentation?.name || name,
                title: patchChanges.length > 0 ? formatFileChangeTitle(patchChanges) : commandPresentation?.title || formatToolName(name),
                status: normalizeToolStatus(readString(item, 'status')) || 'running',
                input: formatToolInput(inputValue, parsedInput),
                command,
                file: patchChanges.length > 0 ? formatFileChangeFile(patchChanges) : commandPresentation?.file || extractFileText(parsedInput),
                summary: patchChanges.length > 0 ? formatFileChangeSummary(patchChanges) : commandPresentation?.summary || summarizeToolInput(name, parsedInput),
                changes: patchChanges.length > 0 ? patchChanges : undefined,
            },
        };
    }

    if (itemType === 'function_call_output' || itemType === 'custom_tool_call_output') {
        const callID = readString(item, 'call_id') || makeID();
        const output = readString(item, 'output');
        const parsedOutput = parseMaybeJSON(output);
        const outputText = formatToolOutput(output, parsedOutput);
        const status = toolStatusFromOutput(output, parsedOutput);
        return {
            type: 'tool',
            tool: {
                callID,
                name: 'tool',
                title: 'Tool',
                status,
                output: outputText,
                error: status === 'failed' ? extractToolError(output, parsedOutput) : undefined,
            },
        };
    }

    if (topType === 'event_msg' && itemType === 'exec_command_end') {
        const callID = readString(item, 'call_id') || makeID();
        const command = extractCommandFromArray(valueAt(item, ['command'])) || extractCommandFromParsed(valueAt(item, ['parsed_cmd']));
        const stdout = readString(item, 'aggregated_output') || readString(item, 'stdout') || readString(item, 'formatted_output');
        const stderr = readString(item, 'stderr');
        const exitCode = readNumber(item, 'exit_code');
        return {
            type: 'tool',
            tool: {
                callID,
                name: 'exec_command',
                title: 'Shell command',
                status: exitCode === 0 ? 'completed' : 'failed',
                command,
                cwd: readString(item, 'cwd'),
                output: stdout,
                error: stderr || (exitCode !== undefined && exitCode !== 0 ? `Exit code ${exitCode}` : undefined),
                summary: readString(item, 'status') || undefined,
            },
        };
    }

    if (topType === 'event_msg' && itemType === 'patch_apply_end') {
        const callID = readString(item, 'call_id') || makeID();
        const changes = Object.keys(recordAt(item, ['changes']) || {}).map(path => ({ path, kind: 'update' }));
        const success = Boolean(valueAt(item, ['success']));
        return {
            type: 'tool',
            tool: {
                callID,
                name: 'apply_patch',
                title: changes.length > 0 ? formatFileChangeTitle(changes) : 'Apply patch',
                status: success ? 'completed' : 'failed',
                output: readString(item, 'stdout'),
                error: readString(item, 'stderr') || undefined,
                file: changes.length > 0 ? formatFileChangeFile(changes) : undefined,
                summary: success ? 'Patch applied' : 'Patch failed',
                changes,
            },
        };
    }

    return null;
}

function parseCodexLiveItem(topType: string, item: Record<string, unknown>): CodexRichEvent | null {
    const itemType = readString(item, 'type');
    const callID = readString(item, 'id') || readString(item, 'call_id') || makeID();
    const isCompleted = topType === 'item.completed';

    if (itemType === 'file_change') {
        const changes = extractFileChanges(valueAt(item, ['changes']));
        const status = normalizeToolStatus(readString(item, 'status')) || (isCompleted ? 'completed' : 'running');
        return {
            type: 'tool',
            tool: {
                callID,
                name: 'file_change',
                title: changes.length > 0 ? formatFileChangeTitle(changes) : 'File change',
                status,
                file: changes.length > 0 ? formatFileChangeFile(changes) : undefined,
                summary: changes.length > 0 ? formatFileChangeSummary(changes) : readString(item, 'status') || undefined,
                output: readString(item, 'diff') || readString(item, 'summary'),
                changes,
            },
        };
    }

    if (itemType === 'command_execution') {
        const exitCode = readNumber(item, 'exit_code');
        const rawStatus = readString(item, 'status').toLowerCase();
        const status = isCompleted
            ? (exitCode === undefined ? (rawStatus === 'failed' ? 'failed' : 'completed') : (exitCode === 0 ? 'completed' : 'failed'))
            : 'running';
        const command = readString(item, 'command');
        const commandPresentation = inferCommandFileOperation(command);
        return {
            type: 'tool',
            tool: {
                callID,
                name: commandPresentation?.name || 'exec_command',
                title: commandPresentation?.title || 'Shell command',
                status,
                command,
                output: readString(item, 'aggregated_output'),
                error: isCompleted && exitCode !== undefined && exitCode !== 0 ? `Exit code ${exitCode}` : undefined,
                file: commandPresentation?.file,
                summary: commandPresentation?.summary || readString(item, 'status') || undefined,
            },
        };
    }

    if (itemType === 'function_call' || itemType === 'custom_tool_call' || itemType === 'tool_call') {
        const name = readString(item, 'name') || 'tool';
        const inputValue = readString(item, 'arguments') || readString(item, 'input');
        const parsedInput = parseMaybeJSON(inputValue);
        const todos = extractTodosFromTool(name, parsedInput);
        if (todos.length > 0) {
            return { type: 'todo', callID, title: 'Plan', todos };
        }
        const patchChanges = name === 'apply_patch' ? extractPatchChanges(inputValue) : [];
        const command = extractCommandText(parsedInput);
        const commandPresentation = command ? inferCommandFileOperation(command) : null;
        return {
            type: 'tool',
            tool: {
                callID,
                name: commandPresentation?.name || name,
                title: patchChanges.length > 0 ? formatFileChangeTitle(patchChanges) : commandPresentation?.title || formatToolName(name),
                status: isCompleted ? 'completed' : 'running',
                input: formatToolInput(inputValue, parsedInput),
                command,
                file: patchChanges.length > 0 ? formatFileChangeFile(patchChanges) : commandPresentation?.file || extractFileText(parsedInput),
                summary: patchChanges.length > 0 ? formatFileChangeSummary(patchChanges) : commandPresentation?.summary || summarizeToolInput(name, parsedInput),
                changes: patchChanges.length > 0 ? patchChanges : undefined,
            },
        };
    }

    if (itemType === 'function_call_output' || itemType === 'custom_tool_call_output' || itemType === 'tool_call_output') {
        const output = readString(item, 'output');
        const parsedOutput = parseMaybeJSON(output);
        const status = toolStatusFromOutput(output, parsedOutput);
        return {
            type: 'tool',
            tool: {
                callID,
                name: 'tool',
                title: 'Tool',
                status,
                output: formatToolOutput(output, parsedOutput),
                error: status === 'failed' ? extractToolError(output, parsedOutput) : undefined,
            },
        };
    }

    if (itemType === 'reasoning') {
        const text = readString(item, 'text') || readString(item, 'summary');
        return text ? { type: 'thinking', text } : null;
    }

    return null;
}

function mergeToolDetails(existing: CodexToolDetails | undefined, next: CodexToolDetails): CodexToolDetails {
    if (!existing) return next;
    return {
        ...existing,
        ...next,
        name: next.name === 'tool' ? existing.name : next.name,
        title: next.title === 'Tool' ? existing.title : next.title,
        input: next.input || existing.input,
        output: next.output || existing.output,
        error: next.error || existing.error,
        command: next.command || existing.command,
        cwd: next.cwd || existing.cwd,
        file: next.file || existing.file,
        summary: next.summary || existing.summary,
        changes: next.changes || existing.changes,
    };
}

function extractTodosFromTool(name: string, input: unknown): CodexTodoItem[] {
    if (name !== 'update_plan') return [];
    const record = asRecord(input);
    const rawPlan = Array.isArray(record?.plan) ? record.plan : [];
    return rawPlan.map(item => {
        const todo = asRecord(item);
        const step = typeof todo?.step === 'string' ? todo.step.trim() : '';
        const status = normalizeTodoStatus(typeof todo?.status === 'string' ? todo.status : '');
        return step ? { step, status } : null;
    }).filter((item): item is CodexTodoItem => Boolean(item));
}

function normalizeTodoStatus(status: string): CodexTodoStatus {
    if (status === 'completed') return 'completed';
    if (status === 'in_progress') return 'in_progress';
    return 'pending';
}

function normalizeToolStatus(status: string): CodexToolStatus | '' {
    const normalized = status.toLowerCase();
    if (normalized === 'completed' || normalized === 'success' || normalized === 'succeeded') return 'completed';
    if (normalized === 'failed' || normalized === 'error') return 'failed';
    if (normalized === 'running' || normalized === 'pending' || normalized === 'in_progress') return 'running';
    return '';
}

function toolStatusFromOutput(output: string, parsedOutput: unknown): CodexToolStatus {
    const metadata = recordAt(parsedOutput, ['metadata']);
    const exitCode = typeof metadata?.exit_code === 'number' ? metadata.exit_code : undefined;
    if (exitCode !== undefined) return exitCode === 0 ? 'completed' : 'failed';
    const match = output.match(/Process exited with code (-?\d+)/);
    if (match) return Number(match[1]) === 0 ? 'completed' : 'failed';
    if (/error|failed|permission denied/i.test(output)) return 'failed';
    return 'completed';
}

function extractToolError(output: string, parsedOutput: unknown): string | undefined {
    const record = asRecord(parsedOutput);
    if (typeof record?.error === 'string') return record.error;
    const stderr = recordAt(parsedOutput, ['metadata']);
    if (typeof stderr?.stderr === 'string') return stderr.stderr;
    const lines = output.split('\n').filter(line => /error|failed|permission denied|exit code/i.test(line));
    return lines.slice(0, 6).join('\n') || undefined;
}

function formatToolInput(raw: string, parsed: unknown): string | undefined {
    if (!raw) return undefined;
    if (parsed && typeof parsed === 'object') {
        return stringifyEvent(parsed);
    }
    return raw;
}

function formatToolOutput(raw: string, parsed: unknown): string {
    const record = asRecord(parsed);
    if (typeof record?.output === 'string') return record.output;
    return raw;
}

function summarizeToolInput(name: string, input: unknown): string | undefined {
    const command = extractCommandText(input);
    if (command) return command;
    const file = extractFileText(input);
    if (file) return file;
    return name === 'apply_patch' ? 'Patch operation' : undefined;
}

function extractCommandText(input: unknown): string | undefined {
    const record = asRecord(input);
    if (!record) return undefined;
    if (typeof record.cmd === 'string') return record.cmd;
    if (typeof record.command === 'string') return record.command;
    if (Array.isArray(record.command)) return record.command.map(String).join(' ');
    return undefined;
}

function extractFileText(input: unknown): string | undefined {
    const record = asRecord(input);
    if (!record) return undefined;
    for (const key of ['filePath', 'file_path', 'path', 'file']) {
        const value = record[key];
        if (typeof value === 'string' && value.trim()) return value;
    }
    return undefined;
}

function extractFileChanges(value: unknown): CodexFileChange[] {
    if (!Array.isArray(value)) return [];
    const changes: CodexFileChange[] = [];
    for (const item of value) {
        const record = asRecord(item);
        const path = firstStringAt(record, [['path'], ['file'], ['file_path'], ['filePath']]);
        if (!path) continue;
        const kind = firstStringAt(record, [['kind'], ['type'], ['action']]);
        changes.push({ path, kind: kind || undefined });
    }
    return uniqueFileChanges(changes);
}

function extractPatchChanges(patch: string): CodexFileChange[] {
    if (!patch) return [];

    const changes: CodexFileChange[] = [];
    for (const line of patch.split('\n')) {
        let match = line.match(/^\*\*\* Add File:\s+(.+)$/);
        if (match) {
            changes.push({ path: match[1].trim(), kind: 'create' });
            continue;
        }
        match = line.match(/^\*\*\* Update File:\s+(.+)$/);
        if (match) {
            changes.push({ path: match[1].trim(), kind: 'update' });
            continue;
        }
        match = line.match(/^\*\*\* Delete File:\s+(.+)$/);
        if (match) {
            changes.push({ path: match[1].trim(), kind: 'delete' });
            continue;
        }
        match = line.match(/^\*\*\* Move to:\s+(.+)$/);
        if (match) {
            changes.push({ path: match[1].trim(), kind: 'move' });
        }
    }
    return uniqueFileChanges(changes);
}

function uniqueFileChanges(changes: CodexFileChange[]): CodexFileChange[] {
    const seen = new Set<string>();
    return changes.filter(change => {
        const key = `${change.kind || 'change'}:${change.path}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
    });
}

function inferCommandFileOperation(command: string | undefined): Pick<CodexToolDetails, 'name' | 'title' | 'file' | 'summary'> | null {
    if (!command) return null;

    const normalized = unwrapShellCommand(command);
    const words = splitShellWords(normalized);
    if (words.length === 0) return null;

    const executable = basename(words[0]);
    if (['cat', 'bat', 'batcat', 'nl', 'head', 'tail'].includes(executable)) {
        const file = lastPathArgument(words.slice(1));
        return {
            name: 'file_read',
            title: 'Read file',
            file,
            summary: file ? `Read ${shortPath(file)}` : 'Read file',
        };
    }

    if (executable === 'sed') {
        const file = lastPathArgument(words.slice(1));
        const isEdit = words.some(word => word === '-i' || word.startsWith('-i'));
        return {
            name: isEdit ? 'file_edit' : 'file_read',
            title: isEdit ? 'Edit file' : 'Read file',
            file,
            summary: file ? `${isEdit ? 'Edit' : 'Read'} ${shortPath(file)}` : (isEdit ? 'Edit file' : 'Read file'),
        };
    }

    if (executable === 'apply_patch' || /\bapply_patch\b/.test(normalized)) {
        return {
            name: 'file_edit',
            title: 'Edit files',
            summary: 'Apply patch',
        };
    }

    if (['perl', 'python', 'python3', 'node', 'ruby'].includes(executable) && /\b(writeFile|write_text|open\(.+,\s*['"]w|Path\(.+\)\.write|fs\.writeFile)/.test(normalized)) {
        const file = extractQuotedPath(normalized);
        return {
            name: 'file_edit',
            title: 'Edit file',
            file,
            summary: file ? `Edit ${shortPath(file)}` : 'Edit file',
        };
    }

    if (['tee', 'touch', 'rm', 'mv', 'cp'].includes(executable) || words.some(word => word === '>' || word === '>>') || /(^|\s)(>|>>)\s*\S+/.test(normalized)) {
        const file = executable === 'tee' ? lastPathArgument(words.slice(1)) : extractRedirectTarget(normalized) || lastPathArgument(words.slice(1));
        return {
            name: 'file_edit',
            title: executable === 'rm' ? 'Delete file' : 'Edit file',
            file,
            summary: file ? `${executable === 'rm' ? 'Delete' : 'Edit'} ${shortPath(file)}` : (executable === 'rm' ? 'Delete file' : 'Edit file'),
        };
    }

    if (['rg', 'grep', 'find'].includes(executable)) {
        const file = lastPathArgument(words.slice(1), true);
        return {
            name: 'file_search',
            title: 'Search files',
            file,
            summary: file ? `Search ${shortPath(file)}` : 'Search files',
        };
    }

    return null;
}

function unwrapShellCommand(command: string): string {
    const words = splitShellWords(command.trim());
    const executable = words.length > 0 ? basename(words[0]) : '';
    if ((executable === 'bash' || executable === 'sh') && words[1] === '-lc' && words[2]) {
        return words[2].trim();
    }
    return command.trim();
}

function splitShellWords(command: string): string[] {
    const words: string[] = [];
    let current = '';
    let quote: '"' | "'" | '' = '';
    let escaping = false;

    for (const char of command) {
        if (escaping) {
            current += char;
            escaping = false;
            continue;
        }
        if (char === '\\') {
            escaping = true;
            continue;
        }
        if (quote) {
            if (char === quote) {
                quote = '';
            } else {
                current += char;
            }
            continue;
        }
        if (char === '"' || char === "'") {
            quote = char;
            continue;
        }
        if (/\s/.test(char)) {
            if (current) {
                words.push(current);
                current = '';
            }
            continue;
        }
        current += char;
    }

    if (current) words.push(current);
    return words;
}

function lastPathArgument(words: string[], requirePathLike = false): string | undefined {
    for (let i = words.length - 1; i >= 0; i--) {
        const word = words[i];
        if (!word || word.startsWith('-') || ['|', '&&', ';', '>', '>>', '<'].includes(word)) continue;
        if (requirePathLike && !looksLikePath(word)) continue;
        return word;
    }
    return undefined;
}

function extractRedirectTarget(command: string): string | undefined {
    const match = command.match(/(?:^|\s)(?:>|>>)\s*(['"]?)([^'"\s]+)\1/);
    return match?.[2];
}

function extractQuotedPath(command: string): string | undefined {
    const match = command.match(/['"]([^'"]+\.[A-Za-z0-9_-]+)['"]/);
    return match?.[1];
}

function looksLikePath(value: string): boolean {
    return value.includes('/') || value.includes('.') || /^[A-Z0-9_ -]+file$/i.test(value);
}

function basename(path: string): string {
    const clean = path.split('/').filter(Boolean).pop() || path;
    return clean.trim();
}

function shortPath(path: string): string {
    const parts = path.split('/').filter(Boolean);
    return parts.slice(-2).join('/') || path;
}

function formatFileChangeTitle(changes: CodexFileChange[]): string {
    if (changes.length !== 1) return 'File changes';
    const kind = normalizeFileChangeKind(changes[0].kind);
    if (kind === 'create') return 'Create file';
    if (kind === 'delete') return 'Delete file';
    if (kind === 'move') return 'Move file';
    if (kind === 'read') return 'Read file';
    return 'Edit file';
}

function formatFileChangeSummary(changes: CodexFileChange[]): string {
    if (changes.length === 0) return 'File change';
    if (changes.length > 1) return `${changes.length} files changed`;
    const change = changes[0];
    const action = formatFileChangeKind(change.kind).toLowerCase();
    return `${action === 'edit' ? 'Update' : titleCase(action)} ${shortPath(change.path)}`;
}

function formatFileChangeFile(changes: CodexFileChange[]): string {
    if (changes.length === 1) return changes[0].path;
    return `${changes.length} files changed`;
}

function formatFileChangeKind(kind: string | undefined): string {
    const normalized = normalizeFileChangeKind(kind);
    if (normalized === 'create') return 'ADD';
    if (normalized === 'delete') return 'DEL';
    if (normalized === 'move') return 'MOVE';
    if (normalized === 'read') return 'READ';
    return 'EDIT';
}

function normalizeFileChangeKind(kind: string | undefined): string {
    const normalized = (kind || '').toLowerCase();
    if (['add', 'added', 'create', 'created', 'new'].includes(normalized)) return 'create';
    if (['delete', 'deleted', 'remove', 'removed'].includes(normalized)) return 'delete';
    if (['move', 'moved', 'rename', 'renamed'].includes(normalized)) return 'move';
    if (['read', 'view', 'viewed'].includes(normalized)) return 'read';
    return 'update';
}

function titleCase(value: string): string {
    return value ? `${value[0].toUpperCase()}${value.slice(1)}` : value;
}

function extractCommandFromArray(value: unknown): string | undefined {
    return Array.isArray(value) ? value.map(String).join(' ') : undefined;
}

function extractCommandFromParsed(value: unknown): string | undefined {
    if (!Array.isArray(value)) return undefined;
    return value.map(item => {
        const record = asRecord(item);
        return typeof record?.cmd === 'string' ? record.cmd : '';
    }).filter(Boolean).join(' && ') || undefined;
}

function formatToolName(name: string): string {
    if (name === 'exec_command') return 'Shell command';
    if (name === 'apply_patch') return 'Apply patch';
    if (name === 'file_change' || name === 'file_edit') return 'Edit file';
    if (name === 'file_read') return 'Read file';
    if (name === 'file_search') return 'Search files';
    return name.replace(/_/g, ' ');
}

function formatToolStatus(status: CodexToolStatus): string {
    if (status === 'completed') return 'DONE';
    if (status === 'failed') return 'ERR';
    return 'RUN';
}

function formatTodoStatus(status: CodexTodoStatus): string {
    if (status === 'completed') return 'DONE';
    if (status === 'in_progress') return 'NOW';
    return 'TODO';
}

function trimLongText(text: string, maxLength: number): string {
    if (text.length <= maxLength) return text;
    return `${text.slice(0, maxLength)}\n... truncated ${text.length - maxLength} chars`;
}

function parseMaybeJSON(value: string): unknown {
    if (!value) return null;
    try {
        return JSON.parse(value);
    } catch {
        return null;
    }
}

function makeID(): string {
    return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function formatCommand(command?: string[]): string {
    if (!command || command.length === 0) return 'Codex CLI started.';
    return command.map(part => part.includes(' ') ? JSON.stringify(part) : part).join(' ');
}

function shortSessionID(sessionID: string): string {
    if (sessionID.length <= 13) return sessionID;
    return `${sessionID.slice(0, 8)}...${sessionID.slice(-4)}`;
}

function currentSessionTitle(sessions: CodexSession[], sessionID: string): string {
    return sessions.find(session => session.id === sessionID)?.title || shortSessionID(sessionID);
}

function formatSessionTime(value?: string): string {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
}

function stringifyEvent(value: unknown): string {
    try {
        return JSON.stringify(value, null, 2);
    } catch {
        return String(value);
    }
}

function readString(value: unknown, key: string): string {
    if (!value || typeof value !== 'object') return '';
    const record = value as Record<string, unknown>;
    const raw = record[key];
    return typeof raw === 'string' ? raw : '';
}

function readNumber(value: unknown, key: string): number | undefined {
    const record = asRecord(value);
    const raw = record?.[key];
    return typeof raw === 'number' ? raw : undefined;
}

function extractCodexText(event: unknown): string {
    const payload = recordAt(event, ['payload']);
    const topType = readString(event, 'type');
    const payloadType = payload ? readString(payload, 'type') : '';
    if (topType === 'event_msg' && payloadType === 'agent_message') {
        return readString(payload, 'message');
    }
    if (topType === 'response_item' && payloadType === 'message') {
        const role = readString(payload, 'role');
        if (role !== 'assistant') return '';
        return extractTextFromContent(valueAt(payload, ['content']));
    }

    const direct = firstStringAt(event, [
        ['delta'],
        ['text_delta'],
        ['content_delta'],
        ['message', 'text'],
        ['message', 'content', 'text'],
        ['item', 'text'],
        ['item', 'content', 'text'],
    ]);
    if (direct) return direct;

    const messageContent = valueAt(event, ['message', 'content']);
    const messageText = extractTextFromContent(messageContent);
    if (messageText) return messageText;

    const itemContent = valueAt(event, ['item', 'content']);
    return extractTextFromContent(itemContent);
}

function isCompleteCodexAssistantMessage(event: unknown): boolean {
    const payload = recordAt(event, ['payload']);
    const topType = readString(event, 'type');
    const payloadType = payload ? readString(payload, 'type') : '';
    const item = recordAt(event, ['item']);
    if (topType === 'item.completed' && item && readString(item, 'type') === 'agent_message') {
        return true;
    }
    if (topType === 'event_msg' && payloadType === 'agent_message') {
        return true;
    }
    if (topType === 'response_item' && payloadType === 'message') {
        return readString(payload, 'role') === 'assistant';
    }
    return false;
}

function extractTextFromContent(value: unknown): string {
    if (typeof value === 'string') return value;
    if (!Array.isArray(value)) return '';
    return value.map(part => {
        if (typeof part === 'string') return part;
        if (!part || typeof part !== 'object') return '';
        const record = part as Record<string, unknown>;
        if (typeof record.text === 'string') return record.text;
        if (typeof record.content === 'string') return record.content;
        return '';
    }).join('');
}

function summarizeCodexEvent(event: unknown): { title: string; text: string } | null {
    const type = readString(event, 'type');
    if (!type) return null;

    const lower = type.toLowerCase();
    if (lower.includes('tool') || lower.includes('exec') || lower.includes('command')) {
        return { title: 'Tool event', text: compactEvent(event) };
    }
    if (lower.includes('patch') || lower.includes('file')) {
        return { title: 'File event', text: compactEvent(event) };
    }
    if (lower.includes('error')) {
        return { title: 'Codex error', text: compactEvent(event) };
    }
    return null;
}

function compactEvent(event: unknown): string {
    const type = readString(event, 'type');
    const name = firstStringAt(event, [['name'], ['tool_name'], ['item', 'name'], ['call', 'name']]);
    const command = firstStringAt(event, [['command'], ['cmd'], ['arguments', 'command'], ['input', 'command']]);
    const path = firstStringAt(event, [['path'], ['file'], ['file_path'], ['input', 'path']]);
    return [type, name, command, path].filter(Boolean).join(' · ') || stringifyEvent(event);
}

function firstStringAt(value: unknown, paths: string[][]): string {
    for (const path of paths) {
        const found = valueAt(value, path);
        if (typeof found === 'string' && found.trim()) {
            return found;
        }
    }
    return '';
}

function valueAt(value: unknown, path: string[]): unknown {
    let current = value;
    for (const key of path) {
        if (!current || typeof current !== 'object') return undefined;
        current = (current as Record<string, unknown>)[key];
    }
    return current;
}

function asRecord(value: unknown): Record<string, unknown> | null {
    return value && typeof value === 'object' && !Array.isArray(value)
        ? value as Record<string, unknown>
        : null;
}

function recordAt(value: unknown, path: string[]): Record<string, unknown> | null {
    return asRecord(valueAt(value, path));
}
