import { useState } from 'react';
import { consumeSSEStream } from '../api/sse';
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
    /** Reset all state */
    reset: () => void;
}

export type UseStreamingActionReturn = [StreamingActionState, StreamingActionControls];

export function useStreamingAction(onComplete?: (result: StreamingActionResult) => void): UseStreamingActionReturn {
    const [running, setRunning] = useState(false);
    const [logs, setLogs] = useState<LogLine[]>([]);
    const [showLogs, setShowLogs] = useState(false);
    const [result, setResult] = useState<StreamingActionResult | null>(null);

    const run = async (action: () => Promise<Response>) => {
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
                    // Treat as success if success is 'true' or if success field is absent (e.g. clone handler)
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
        setRunning(false);
        setLogs([]);
        setShowLogs(false);
        setResult(null);
    };

    const state: StreamingActionState = { running, logs, result, showLogs };
    const controls: StreamingActionControls = { run, reset };

    return [state, controls];
}
