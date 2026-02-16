import { useState, useRef, useCallback } from 'react';
import { consumeSSEStream } from '../api/sse';
import { streamActionLogs, type ActionStatus } from '../api/actions';
import type { LogLine } from '../v2/LogViewer';

export interface StreamingActionResult {
    ok: boolean;
    message: string;
}

export interface StreamingActionState {
    running: boolean;
    logs: LogLine[];
    result: StreamingActionResult | null;
    showLogs: boolean;
}

export interface StreamingActionControls {
    /** Trigger the action. Pass a function that returns a Response with SSE stream. */
    run: (action: () => Promise<Response>) => Promise<void>;
    /** Resume watching a running action by action ID */
    resume: (actionId: string, initialStatus?: ActionStatus) => void;
    /** Reset all state */
    reset: () => void;
}

export type UseStreamingActionReturn = [StreamingActionState, StreamingActionControls];

export function useStreamingAction(onComplete?: (result: StreamingActionResult) => void): UseStreamingActionReturn {
    const [running, setRunning] = useState(false);
    const [logs, setLogs] = useState<LogLine[]>([]);
    const [showLogs, setShowLogs] = useState(false);
    const [result, setResult] = useState<StreamingActionResult | null>(null);
    const eventSourceRef = useRef<EventSource | null>(null);

    const cleanup = useCallback(() => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
    }, []);

    const resume = useCallback((actionId: string, initialStatus?: ActionStatus) => {
        cleanup();
        
        // Set initial state from status
        if (initialStatus?.logs) {
            const logBuffer = initialStatus.logs;
            const allLogs: LogLine[] = [];
            
            // Add first 100 lines
            logBuffer.first.forEach(line => allLogs.push({ text: line }));
            
            // If there's a gap (more than 200 total), add gap indicator
            if (logBuffer.total > 200) {
                allLogs.push({ text: `... ${logBuffer.total - 200} lines omitted ...`, error: false });
            }
            
            // Add last 100 lines (skipping duplicates from first)
            const firstSet = new Set(logBuffer.first);
            logBuffer.last.forEach(line => {
                if (!firstSet.has(line)) {
                    allLogs.push({ text: line });
                }
            });
            
            setLogs(allLogs);
        }
        
        setRunning(true);
        setShowLogs(true);
        setResult(null);

        // Connect to SSE stream
        eventSourceRef.current = streamActionLogs(actionId, {
            onLog: (message) => {
                setLogs(prev => [...prev, { text: message }]);
            },
            onDone: (data) => {
                const isOk = data.success === 'true';
                const actionResult: StreamingActionResult = {
                    ok: isOk,
                    message: data.message || (isOk ? 'Completed successfully' : 'Failed'),
                };
                setResult(actionResult);
                setRunning(false);
                onComplete?.(actionResult);
                cleanup();
            },
            onError: (message) => {
                setLogs(prev => [...prev, { text: `Error: ${message}`, error: true }]);
                setRunning(false);
                cleanup();
            },
            onStatus: (status) => {
                if (status === 'running') {
                    setRunning(true);
                }
            },
        });
    }, [cleanup, onComplete]);

    const run = async (action: () => Promise<Response>) => {
        cleanup();
        setRunning(true);
        setResult(null);
        setLogs([]);
        setShowLogs(true);

        try {
            const response = await action();
            await consumeSSEStream(response, {
                onLog: (line) => setLogs(prev => [...prev, line]),
                onError: (line) => setLogs(prev => [...prev, line]),
                onDone: (message, data) => {
                    const isOk = data.success === undefined || data.success === 'true';
                    const actionResult: StreamingActionResult = {
                        ok: isOk,
                        message: message || (isOk ? 'Completed successfully' : 'Failed'),
                    };
                    setResult(actionResult);
                    onComplete?.(actionResult);
                },
            });
        } catch (err: unknown) {
            const errorMessage = err instanceof Error ? err.message : 'Action failed';
            const actionResult: StreamingActionResult = { ok: false, message: errorMessage };
            setResult(actionResult);
            onComplete?.(actionResult);
        } finally {
            setRunning(false);
        }
    };

    const reset = () => {
        cleanup();
        setRunning(false);
        setLogs([]);
        setShowLogs(false);
        setResult(null);
    };

    const state: StreamingActionState = { running, logs, result, showLogs };
    const controls: StreamingActionControls = { run, resume, reset };

    return [state, controls];
}
