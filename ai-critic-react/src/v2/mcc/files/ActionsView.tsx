import { useState, useEffect } from 'react';
import { fetchActions, createAction, updateAction, deleteAction, runAction, fetchActionStatus, stopAction } from '../../../api/actions';
import type { Action, ActionStatus } from '../../../api/actions';
import { usePerActionStreaming } from '../../../hooks/usePerActionStreaming';
import type { LogLine } from '../../LogViewer';
import { NoZoomingInput } from '../components/NoZoomingInput';
import { ActionIconSelector, ActionCard, ConfirmModal } from '../../../pure-view';
import './ActionsView.css';

interface ActionsViewProps {
    projectName: string;
    projectDir: string;
}

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

    // Use per-action streaming hook - each action has its own independent state
    const perActionStreaming = usePerActionStreaming();

    useEffect(() => {
        loadActions();
        loadStatuses();
        
        // Poll for running actions every 5 seconds
        const pollInterval = setInterval(() => {
            loadStatuses();
        }, 5000);
        
        return () => clearInterval(pollInterval);
    }, [projectName]);



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
        const [, controls] = perActionStreaming.getActionState(action.id);
        
        // Reset this specific action's state
        controls.reset();
        
        try {
            await controls.run(async () => {
                return runAction({
                    project_dir: projectDir,
                    script: action.script,
                    action_id: action.id,
                });
            });
        } finally {
            loadStatuses();
        }
    };

    const handleStop = async (actionId: string) => {
        try {
            await stopAction(projectName, actionId);
            loadStatuses();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to stop action');
        }
    };

    const isEditing = isCreating || editingAction !== null;

    const getActionLogs = (actionId: string): LogLine[] => {
        // Priority: per-action streaming state > cached logs from status
        const [state] = perActionStreaming.getActionState(actionId);
        if (state.logs.length > 0) {
            return state.logs;
        }
        
        // Fall back to cached logs from status
        const logBuffer = actionStatuses[actionId]?.logs;
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
        // Check per-action streaming state first
        const [state] = perActionStreaming.getActionState(actionId);
        if (state.running) {
            return true;
        }
        
        // Fall back to status from server
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
