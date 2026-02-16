import { useState, useEffect, useRef } from 'react';
import { fetchActions, createAction, updateAction, deleteAction, runAction, fetchActionStatus, stopAction, streamActionLogs } from '../../../api/actions';
import type { Action, ActionStatus, LogBuffer } from '../../../api/actions';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import type { LogLine } from '../../LogViewer';
import { NoZoomingInput } from '../components/NoZoomingInput';
import { ActionIconSelector, ActionCard, ConfirmModal } from '../../../pure-view';
import './ActionsView.css';

interface ActionsViewProps {
    projectName: string;
    projectDir: string;
}

// Store for resumed streaming logs (actionId -> logs)
const streamingLogsStore = new Map<string, LogLine[]>();
const streamingRunningStore = new Map<string, boolean>();

export function ActionsView({ projectName, projectDir }: ActionsViewProps) {
    const [actions, setActions] = useState<Action[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [editingAction, setEditingAction] = useState<Action | null>(null);
    const [isCreating, setIsCreating] = useState(false);
    const [actionStatuses, setActionStatuses] = useState<Record<string, ActionStatus>>({});
    const [deleteConfirm, setDeleteConfirm] = useState<{ actionId: string; name: string } | null>(null);

    const [formName, setFormName] = useState('');
    const [formIcon, setFormIcon] = useState('▶️');
    const [formScript, setFormScript] = useState('');

    const [runState, runControls] = useStreamingAction();
    const [currentRunningId, setCurrentRunningId] = useState<string | null>(null);
    
    // Track which actions we're already streaming
    const streamingActionsRef = useRef<Set<string>>(new Set());
    const eventSourcesRef = useRef<Map<string, EventSource>>(new Map());
    // Force re-render for resumed logs
    const [, setResumedLogCount] = useState(0);

    useEffect(() => {
        loadActions();
        loadStatuses();
        
        // Poll for running actions every 5 seconds
        const pollInterval = setInterval(() => {
            loadStatuses();
        }, 5000);
        
        return () => clearInterval(pollInterval);
    }, [projectName]);

    // Effect to handle resuming running actions
    useEffect(() => {
        const runningActions = Object.entries(actionStatuses).filter(([_, status]) => status.running);
        
        runningActions.forEach(([actionId, status]) => {
            // Skip if this is the currently running action in this session
            if (actionId === currentRunningId) return;
            
            // Skip if already streaming this action
            if (streamingActionsRef.current.has(actionId)) return;
            
            // Mark as streaming
            streamingActionsRef.current.add(actionId);
            
            // Initialize logs from status
            const logBuffer = status.logs as LogBuffer | undefined;
            if (logBuffer) {
                const logs: LogLine[] = [];
                logBuffer.first.forEach(line => logs.push({ text: line }));
                if (logBuffer.total > 200) {
                    logs.push({ text: `... ${logBuffer.total - 200} lines omitted ...`, error: false });
                }
                const firstSet = new Set(logBuffer.first);
                logBuffer.last.forEach(line => {
                    if (!firstSet.has(line)) {
                        logs.push({ text: line });
                    }
                });
                streamingLogsStore.set(actionId, logs);
            }
            streamingRunningStore.set(actionId, true);
            setResumedLogCount(n => n + 1);
            
            // Connect to SSE stream
            const eventSource = streamActionLogs(actionId, {
                onLog: (message) => {
                    const logs = streamingLogsStore.get(actionId) || [];
                    logs.push({ text: message });
                    streamingLogsStore.set(actionId, logs);
                    setResumedLogCount(n => n + 1);
                },
                onDone: () => {
                    streamingRunningStore.set(actionId, false);
                    streamingActionsRef.current.delete(actionId);
                    eventSourcesRef.current.delete(actionId);
                    loadStatuses();
                    setResumedLogCount(n => n + 1);
                },
                onError: () => {
                    streamingRunningStore.set(actionId, false);
                    streamingActionsRef.current.delete(actionId);
                    eventSourcesRef.current.delete(actionId);
                    setResumedLogCount(n => n + 1);
                },
                onStatus: (status) => {
                    streamingRunningStore.set(actionId, status === 'running');
                    setResumedLogCount(n => n + 1);
                },
            });
            eventSourcesRef.current.set(actionId, eventSource);
        });
        
        // Cleanup: close streams for actions that are no longer running
        streamingActionsRef.current.forEach(actionId => {
            if (!actionStatuses[actionId]?.running) {
                const es = eventSourcesRef.current.get(actionId);
                if (es) {
                    es.close();
                    eventSourcesRef.current.delete(actionId);
                }
                streamingActionsRef.current.delete(actionId);
                streamingRunningStore.delete(actionId);
            }
        });
    }, [actionStatuses, currentRunningId]);

    // Cleanup on unmount
    useEffect(() => {
        return () => {
            eventSourcesRef.current.forEach(es => es.close());
            eventSourcesRef.current.clear();
            streamingLogsStore.clear();
            streamingRunningStore.clear();
        };
    }, []);

    const loadActions = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await fetchActions(projectName);
            setActions(data);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to load actions');
        } finally {
            setLoading(false);
        }
    };

    const loadStatuses = async () => {
        try {
            const statuses = await fetchActionStatus(projectName);
            if (statuses && typeof statuses === 'object') {
                setActionStatuses(statuses as Record<string, ActionStatus>);
            }
        } catch (e) {
            console.error('Failed to load action statuses:', e);
        }
    };

    const resetForm = () => {
        setFormName('');
        setFormIcon('▶️');
        setFormScript('');
        setEditingAction(null);
        setIsCreating(false);
    };

    const handleStartCreate = () => {
        resetForm();
        setIsCreating(true);
    };

    const handleStartEdit = (action: Action) => {
        setEditingAction(action);
        setFormName(action.name);
        setFormIcon(action.icon || '▶️');
        setFormScript(action.script);
        setIsCreating(false);
    };

    const handleSave = async () => {
        if (!formName.trim() || !formScript.trim()) {
            setError('Name and script are required');
            return;
        }

        setError('');
        try {
            if (editingAction) {
                await updateAction(projectName, {
                    ...editingAction,
                    name: formName.trim(),
                    icon: formIcon,
                    script: formScript.trim(),
                });
            } else {
                await createAction(projectName, {
                    name: formName.trim(),
                    icon: formIcon,
                    script: formScript.trim(),
                });
            }
            resetForm();
            await loadActions();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to save action');
        }
    };

    const handleDelete = async () => {
        if (!deleteConfirm) return;

        try {
            await deleteAction(projectName, deleteConfirm.actionId);
            setDeleteConfirm(null);
            await loadActions();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to delete action');
        }
    };

    const handleRun = async (action: Action) => {
        setCurrentRunningId(action.id);
        runControls.reset();
        try {
            await runControls.run(async () => {
                return runAction({
                    project_dir: projectDir,
                    script: action.script,
                    action_id: action.id,
                });
            });
        } finally {
            setCurrentRunningId(null);
            loadStatuses();
        }
    };

    const handleStop = async (actionId: string) => {
        try {
            await stopAction(projectName, actionId);
            setCurrentRunningId(null);
            loadStatuses();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to stop action');
        }
    };

    const isEditing = isCreating || editingAction !== null;

    const getActionLogs = (actionId: string): LogLine[] => {
        // Priority: live streaming logs > resumed streaming logs > cached logs
        if (currentRunningId === actionId) {
            return runState.logs;
        }
        
        // Check if we have resumed streaming logs for this action
        if (streamingLogsStore.has(actionId)) {
            return streamingLogsStore.get(actionId)!;
        }
        
        // Fall back to cached logs from status
        const logBuffer = actionStatuses[actionId]?.logs as LogBuffer | undefined;
        if (!logBuffer) return [];
        
        const logs: LogLine[] = [];
        
        // Add first 100 lines
        logBuffer.first.forEach(line => logs.push({ text: line }));
        
        // If there's a gap (more than 200 total), add gap indicator
        if (logBuffer.total > 200) {
            logs.push({ text: `... ${logBuffer.total - 200} lines omitted ...`, error: false });
        }
        
        // Add last 100 lines (skipping duplicates from first)
        const firstSet = new Set(logBuffer.first);
        logBuffer.last.forEach(line => {
            if (!firstSet.has(line)) {
                logs.push({ text: line });
            }
        });
        
        return logs;
    };

    const isActionRunning = (actionId: string): boolean => {
        // Check current session first
        if (currentRunningId === actionId) return runState.running;
        
        // Check resumed streaming
        if (streamingRunningStore.has(actionId)) {
            return streamingRunningStore.get(actionId)!;
        }
        
        // Fall back to status
        return actionStatuses[actionId]?.running || false;
    };

    return (
        <div className="mcc-actions-view">
            <div className="mcc-actions-header">
                <h3 className="mcc-actions-title">Actions</h3>
                {!isEditing && (
                    <button className="mcc-actions-add-btn" onClick={handleStartCreate}>
                        + Add Action
                    </button>
                )}
            </div>

            {loading ? (
                <div className="mcc-actions-empty">Loading...</div>
            ) : (
                <>
                    {error && <div className="mcc-actions-error">{error}</div>}

                    {isEditing && (
                        <div className="mcc-actions-form">
                            <div className="mcc-actions-form-row">
                                <label className="mcc-actions-form-label">Name</label>
                                <NoZoomingInput>
                                    <input
                                        type="text"
                                        className="mcc-actions-form-input"
                                        value={formName}
                                        onChange={(e) => setFormName(e.target.value)}
                                        placeholder="e.g., Build Project"
                                    />
                                </NoZoomingInput>
                            </div>

                            <div className="mcc-actions-form-row">
                                <label className="mcc-actions-form-label">Icon</label>
                                <ActionIconSelector
                                    value={formIcon}
                                    onChange={setFormIcon}
                                />
                            </div>

                            <div className="mcc-actions-form-row">
                                <label className="mcc-actions-form-label">Script</label>
                                <NoZoomingInput>
                                    <textarea
                                        className="mcc-actions-form-textarea"
                                        value={formScript}
                                        onChange={(e) => setFormScript(e.target.value)}
                                        placeholder="e.g., npm run build&#10;or&#10;go build ./..."
                                        rows={4}
                                    />
                                </NoZoomingInput>
                            </div>

                            <div className="mcc-actions-form-buttons">
                                <button className="mcc-actions-form-save" onClick={handleSave}>
                                    Save
                                </button>
                                <button className="mcc-actions-form-cancel" onClick={resetForm}>
                                    Cancel
                                </button>
                            </div>
                        </div>
                    )}

                    {!isEditing && (
                        <>
                            {actions.length === 0 ? (
                                <div className="mcc-actions-empty">
                                    No actions defined yet. Click "Add Action" to create one.
                                </div>
                            ) : (
                                <div className="mcc-actions-list">
                                    {actions.map((action) => {
                                        const running = isActionRunning(action.id);
                                        const logs = getActionLogs(action.id);
                                        const status = actionStatuses[action.id];

                                        return (
                                            <div key={action.id} style={{ marginBottom: 12 }}>
                                                <ActionCard
                                                    name={action.name}
                                                    icon={action.icon || '▶️'}
                                                    script={action.script}
                                                    running={running}
                                                    exitCode={status?.exit_code}
                                                    logs={logs}
                                                    onRun={() => handleRun(action)}
                                                    onStop={() => handleStop(action.id)}
                                                    onEdit={() => handleStartEdit(action)}
                                                    onDelete={() => setDeleteConfirm({ actionId: action.id, name: action.name })}
                                                />
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </>
                    )}

                    {/* Delete Confirmation Modal */}
                    {deleteConfirm && (
                        <ConfirmModal
                            title="Delete Action"
                            message={`Are you sure you want to delete "${deleteConfirm.name}"? This action cannot be undone.`}
                            info={{
                                Action: deleteConfirm.name,
                            }}
                            command={`Delete action "${deleteConfirm.name}"`}
                            confirmLabel="Delete"
                            confirmVariant="danger"
                            onConfirm={handleDelete}
                            onClose={() => setDeleteConfirm(null)}
                        />
                    )}
                </>
            )}
        </div>
    );
}
