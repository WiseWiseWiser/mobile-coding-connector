import { useState, useRef, useCallback } from 'react';
import { consumeSSEStream } from '../api/sse';
import type { LogLine } from '../v2/LogViewer';

export interface ReconnectingActionResult {
    ok: boolean;
    message: string;
    publicUrl?: string;
}

export interface ReconnectingActionState {
    running: boolean;
    logs: LogLine[];
    result: ReconnectingActionResult | null;
    showLogs: boolean;
    sessionId: string | null;
    reconnecting: boolean;
    reconnectionCount: number;
}

export interface ReconnectingActionControls {
    /** Trigger the action. Pass a function that returns a Response with SSE stream. */
    run: (action: (sessionId?: string, logIndex?: number) => Promise<Response>) => Promise<void>;
    /** Manually reconnect to an existing session */
    reconnect: () => Promise<void>;
    /** Reset all state */
    reset: () => void;
    /** Cancel/abort the current operation */
    cancel: () => void;
}

export type UseReconnectingStreamingActionReturn = [ReconnectingActionState, ReconnectingActionControls];

const MAX_RECONNECTIONS = 10;
const RECONNECT_DELAY_MS = 2000;

/**
 * Hook for streaming actions that supports automatic reconnection.
 * Useful for long-running operations like domain mapping where the connection might drop.
 */
export function useReconnectingStreamingAction(
    onComplete?: (result: ReconnectingActionResult) => void,
    options?: {
        maxReconnects?: number;
        reconnectDelayMs?: number;
    }
): UseReconnectingStreamingActionReturn {
    const maxReconnects = options?.maxReconnects ?? MAX_RECONNECTIONS;
    const reconnectDelayMs = options?.reconnectDelayMs ?? RECONNECT_DELAY_MS;

    const [running, setRunning] = useState(false);
    const [logs, setLogs] = useState<LogLine[]>([]);
    const [showLogs, setShowLogs] = useState(false);
    const [result, setResult] = useState<ReconnectingActionResult | null>(null);
    const [sessionId, setSessionId] = useState<string | null>(null);
    const [reconnecting, setReconnecting] = useState(false);
    const [reconnectionCount, setReconnectionCount] = useState(0);

    const abortControllerRef = useRef<AbortController | null>(null);
    const isAbortedRef = useRef(false);
    const sessionIdRef = useRef<string | null>(null);
    const logsRef = useRef<LogLine[]>([]);

    // Keep refs in sync with state
    sessionIdRef.current = sessionId;
    logsRef.current = logs;

    const reset = useCallback(() => {
        isAbortedRef.current = false;
        abortControllerRef.current?.abort();
        abortControllerRef.current = null;
        setRunning(false);
        setLogs([]);
        setShowLogs(false);
        setResult(null);
        setSessionId(null);
        setReconnecting(false);
        setReconnectionCount(0);
    }, []);

    const cancel = useCallback(() => {
        isAbortedRef.current = true;
        abortControllerRef.current?.abort();
        setRunning(false);
        setReconnecting(false);
    }, []);

    const processStream = useCallback(async (
        action: (sessionId?: string, logIndex?: number) => Promise<Response>,
        isReconnect: boolean
    ): Promise<boolean> => {
        try {
            const currentSessionId = sessionIdRef.current ?? undefined;
            const currentLogIndex = isReconnect ? logsRef.current.length : 0;

            if (isReconnect) {
                setReconnecting(true);
                await new Promise(resolve => setTimeout(resolve, reconnectDelayMs));
                if (isAbortedRef.current) return false;
            }

            abortControllerRef.current = new AbortController();
            const response = await action(currentSessionId, currentLogIndex);

            if (isAbortedRef.current) return false;
            setReconnecting(false);

            let completed = false;
            await consumeSSEStream(response, {
                onLog: (line) => {
                    if (!isAbortedRef.current) {
                        setLogs(prev => [...prev, line]);
                    }
                },
                onError: (line) => {
                    if (!isAbortedRef.current) {
                        setLogs(prev => [...prev, line]);
                    }
                },
                onCustom: (data) => {
                    // Handle session ID event
                    if (data.type === 'session' && data.session_id) {
                        setSessionId(data.session_id);
                        sessionIdRef.current = data.session_id;
                    }
                },
                onDone: (message, data) => {
                    completed = true;
                    if (!isAbortedRef.current) {
                        const isOk = data.success === undefined || data.success === 'true';
                        const actionResult: ReconnectingActionResult = {
                            ok: isOk,
                            message: message || (isOk ? 'Completed successfully' : 'Failed'),
                            publicUrl: data.public_url,
                        };
                        setResult(actionResult);
                        onComplete?.(actionResult);
                    }
                },
            });

            return completed;
        } catch (err: unknown) {
            if (isAbortedRef.current) return false;

            // Check if it's a connection error that we should retry
            const errorMessage = err instanceof Error ? err.message : 'Connection failed';
            const isConnectionError =
                errorMessage.includes('fetch') ||
                errorMessage.includes('network') ||
                errorMessage.includes('connection') ||
                errorMessage.includes('abort') ||
                errorMessage.includes('timeout');

            if (isConnectionError && !isReconnect) {
                // This was the initial connection attempt, try to reconnect
                return false; // Signal to retry
            }

            throw err;
        }
    }, [onComplete, reconnectDelayMs]);

    const run = useCallback(async (action: (sessionId?: string, logIndex?: number) => Promise<Response>) => {
        isAbortedRef.current = false;
        setRunning(true);
        setResult(null);
        setShowLogs(true);
        setReconnectionCount(0);

        let attempts = 0;
        let completed = false;

        while (!completed && !isAbortedRef.current && attempts <= maxReconnects) {
            if (attempts > 0) {
                setReconnectionCount(attempts);
            }

            try {
                completed = await processStream(action, attempts > 0);
                if (!completed && !isAbortedRef.current) {
                    // Stream ended without completion, try to reconnect
                    attempts++;
                    if (attempts > maxReconnects) {
                        const errorResult: ReconnectingActionResult = {
                            ok: false,
                            message: `Connection lost. Maximum reconnection attempts (${maxReconnects}) exceeded.`,
                        };
                        setResult(errorResult);
                        onComplete?.(errorResult);
                        break;
                    }
                }
            } catch (err: unknown) {
                if (isAbortedRef.current) break;

                const errorMessage = err instanceof Error ? err.message : 'Action failed';
                const errorResult: ReconnectingActionResult = {
                    ok: false,
                    message: errorMessage,
                };
                setResult(errorResult);
                onComplete?.(errorResult);
                break;
            }
        }

        setRunning(false);
        setReconnecting(false);
    }, [maxReconnects, onComplete, processStream]);

    const reconnect = useCallback(async () => {
        if (!sessionIdRef.current) return;

        setReconnecting(true);
        setReconnectionCount(prev => prev + 1);

        try {
            // This is a placeholder - the actual action function needs to be stored somewhere
            // For now, this is mainly used to indicate reconnection state
        } finally {
            setReconnecting(false);
        }
    }, []);

    const state: ReconnectingActionState = {
        running,
        logs,
        result,
        showLogs,
        sessionId,
        reconnecting,
        reconnectionCount,
    };

    const controls: ReconnectingActionControls = { run, reconnect, reset, cancel };

    return [state, controls];
}
