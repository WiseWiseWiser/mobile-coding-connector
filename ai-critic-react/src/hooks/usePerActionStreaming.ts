import { useState, useRef, useCallback } from 'react';
import { consumeSSEStream } from '../api/sse';
import type { LogLine } from '../v2/LogViewer';

export interface PerActionState {
    running: boolean;
    logs: LogLine[];
    showLogs: boolean;
    exitCode: number | null;
    error: string | null;
}

export interface PerActionControls {
    run: (action: () => Promise<Response>) => Promise<void>;
    stop: () => void;
    reset: () => void;
    setShowLogs: (show: boolean) => void;
    addLog: (line: string, isError?: boolean) => void;
}

export type PerActionEntry = [PerActionState, PerActionControls];

export function usePerActionStreaming(): {
    getActionState: (actionId: string) => PerActionEntry;
    hasAction: (actionId: string) => boolean;
    removeAction: (actionId: string) => void;
} {
    const [actionStates, setActionStates] = useState<Map<string, PerActionState>>(new Map());
    const eventSourcesRef = useRef<Map<string, EventSource>>(new Map());

    const cleanup = useCallback((actionId: string) => {
        const es = eventSourcesRef.current.get(actionId);
        if (es) {
            es.close();
            eventSourcesRef.current.delete(actionId);
        }
    }, []);

    const reset = useCallback((actionId: string) => {
        cleanup(actionId);
        setActionStates(prev => {
            const next = new Map(prev);
            next.set(actionId, {
                running: false,
                logs: [],
                showLogs: false,
                exitCode: null,
                error: null,
            });
            return next;
        });
    }, [cleanup]);

    const run = useCallback(async (actionId: string, action: () => Promise<Response>) => {
        // Initialize state for this action
        setActionStates(prev => {
            const next = new Map(prev);
            next.set(actionId, {
                running: true,
                logs: [],
                showLogs: true,
                exitCode: null,
                error: null,
            });
            return next;
        });

        try {
            const response = await action();
            await consumeSSEStream(response, {
                onLog: (line) => {
                    setActionStates(prev => {
                        const next = new Map(prev);
                        const state = next.get(actionId);
                        if (state) {
                            next.set(actionId, {
                                ...state,
                                logs: [...state.logs, line],
                            });
                        }
                        return next;
                    });
                },
                onError: (line) => {
                    setActionStates(prev => {
                        const next = new Map(prev);
                        const state = next.get(actionId);
                        if (state) {
                            next.set(actionId, {
                                ...state,
                                logs: [...state.logs, line],
                            });
                        }
                        return next;
                    });
                },
                onDone: (_, data) => {
                    const isOk = data.success === undefined || data.success === 'true';
                    setActionStates(prev => {
                        const next = new Map(prev);
                        const state = next.get(actionId);
                        if (state) {
                            next.set(actionId, {
                                ...state,
                                running: false,
                                exitCode: isOk ? 0 : 1,
                                error: isOk ? null : (data.message || 'Action failed'),
                            });
                        }
                        return next;
                    });
                },
            });
        } catch (err: unknown) {
            const errorMessage = err instanceof Error ? err.message : 'Action failed';
            setActionStates(prev => {
                const next = new Map(prev);
                const state = next.get(actionId);
                if (state) {
                    next.set(actionId, {
                        ...state,
                        running: false,
                        error: errorMessage,
                        exitCode: 1,
                    });
                }
                return next;
            });
        }
    }, []);

    const stop = useCallback((actionId: string) => {
        cleanup(actionId);
        setActionStates(prev => {
            const next = new Map(prev);
            const state = next.get(actionId);
            if (state) {
                next.set(actionId, {
                    ...state,
                    running: false,
                });
            }
            return next;
        });
    }, [cleanup]);

    const setShowLogs = useCallback((actionId: string, show: boolean) => {
        setActionStates(prev => {
            const next = new Map(prev);
            const state = next.get(actionId);
            if (state) {
                next.set(actionId, {
                    ...state,
                    showLogs: show,
                });
            }
            return next;
        });
    }, []);

    const addLog = useCallback((actionId: string, line: string, isError: boolean = false) => {
        setActionStates(prev => {
            const next = new Map(prev);
            const state = next.get(actionId);
            if (state) {
                next.set(actionId, {
                    ...state,
                    logs: [...state.logs, { text: line, error: isError }],
                });
            }
            return next;
        });
    }, []);

    const getActionState = (actionId: string): PerActionEntry => {
        const state = actionStates.get(actionId) || {
            running: false,
            logs: [],
            showLogs: false,
            exitCode: null,
            error: null,
        };

        const controls: PerActionControls = {
            run: (action) => run(actionId, action),
            stop: () => stop(actionId),
            reset: () => reset(actionId),
            setShowLogs: (show) => setShowLogs(actionId, show),
            addLog: (line, isError) => addLog(actionId, line, isError),
        };

        return [state, controls];
    };

    const hasAction = (actionId: string) => actionStates.has(actionId);

    const removeAction = (actionId: string) => {
        cleanup(actionId);
        setActionStates(prev => {
            const next = new Map(prev);
            next.delete(actionId);
            return next;
        });
    };

    return {
        getActionState,
        hasAction,
        removeAction,
    };
}
