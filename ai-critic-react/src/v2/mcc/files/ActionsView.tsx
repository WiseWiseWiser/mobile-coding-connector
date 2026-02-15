import { useState, useEffect } from 'react';
import { fetchActions, createAction, updateAction, deleteAction, runAction, fetchActionStatus, stopAction } from '../../../api/actions';
import type { Action, ActionStatus } from '../../../api/actions';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import type { LogLine } from '../../LogViewer';
import './ActionsView.css';

interface ActionsViewProps {
    projectName: string;
    projectDir: string;
}

const ICON_OPTIONS = [
    { value: '🔨', label: 'Hammer (Build)' },
    { value: '📋', label: 'Clipboard (Lint)' },
    { value: '🚀', label: 'Rocket (Deploy)' },
    { value: '▶️', label: 'Play (Run)' },
    { value: '🧪', label: 'Test Tube (Test)' },
    { value: '📦', label: 'Package (Package)' },
    { value: '🔄', label: 'Sync (Update)' },
    { value: '🧹', label: 'Broom (Clean)' },
    { value: '📊', label: 'Chart (Analyze)' },
    { value: '⚙️', label: 'Gear (Configure)' },
    { value: '🔍', label: 'Search (Find)' },
    { value: '✅', label: 'Check (Verify)' },
];

export function ActionsView({ projectName, projectDir }: ActionsViewProps) {
    const [actions, setActions] = useState<Action[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [editingAction, setEditingAction] = useState<Action | null>(null);
    const [isCreating, setIsCreating] = useState(false);
    const [actionStatuses, setActionStatuses] = useState<Record<string, ActionStatus>>({});
    const [expandedActionId, setExpandedActionId] = useState<string | null>(null);

    const [formName, setFormName] = useState('');
    const [formIcon, setFormIcon] = useState('▶️');
    const [formScript, setFormScript] = useState('');

    const [runState, runControls] = useStreamingAction();
    const [currentRunningId, setCurrentRunningId] = useState<string | null>(null);

    useEffect(() => {
        loadActions();
        loadStatuses();
    }, [projectName]);

    useEffect(() => {
        if (currentRunningId) {
            setExpandedActionId(currentRunningId);
        }
    }, [currentRunningId]);

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

    const handleDelete = async (actionId: string) => {
        if (!confirm('Are you sure you want to delete this action?')) {
            return;
        }

        try {
            await deleteAction(projectName, actionId);
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
            await stopAction(actionId);
            setCurrentRunningId(null);
            loadStatuses();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to stop action');
        }
    };

    const isEditing = isCreating || editingAction !== null;

    const isRunning = (actionId: string) => {
        return currentRunningId === actionId || actionStatuses[actionId]?.running;
    };

    const getActionLogs = (actionId: string): LogLine[] => {
        if (currentRunningId === actionId) {
            return runState.logs;
        }
        const logs = actionStatuses[actionId]?.logs || [];
        return logs.map((text: string) => ({ text, error: false }));
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
                                <input
                                    type="text"
                                    className="mcc-actions-form-input"
                                    value={formName}
                                    onChange={(e) => setFormName(e.target.value)}
                                    placeholder="e.g., Build Project"
                                />
                            </div>

                            <div className="mcc-actions-form-row">
                                <label className="mcc-actions-form-label">Icon</label>
                                <select
                                    className="mcc-actions-form-select"
                                    value={formIcon}
                                    onChange={(e) => setFormIcon(e.target.value)}
                                >
                                    {ICON_OPTIONS.map((opt) => (
                                        <option key={opt.value} value={opt.value}>
                                            {opt.value} {opt.label}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div className="mcc-actions-form-row">
                                <label className="mcc-actions-form-label">Script</label>
                                <textarea
                                    className="mcc-actions-form-textarea"
                                    value={formScript}
                                    onChange={(e) => setFormScript(e.target.value)}
                                    placeholder="e.g., npm run build&#10;or&#10;go build ./..."
                                    rows={4}
                                />
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
                                        const running = isRunning(action.id);
                                        const logs = getActionLogs(action.id);
                                        const expanded = expandedActionId === action.id;

                                        return (
                                            <div key={action.id} className={`mcc-action-item ${running ? 'running' : ''}`}>
                                                <div className="mcc-action-main">
                                                    <button
                                                        className="mcc-action-run-btn"
                                                        onClick={() => handleRun(action)}
                                                        disabled={running}
                                                    >
                                                        <span className="mcc-action-icon">{action.icon || '▶️'}</span>
                                                        <span className="mcc-action-name">{action.name}</span>
                                                        {running && (
                                                            <span className="mcc-action-running">Running</span>
                                                        )}
                                                    </button>
                                                    
                                                    {running && (
                                                        <button
                                                            className="mcc-action-stop-btn"
                                                            onClick={() => handleStop(action.id)}
                                                            title="Stop"
                                                        >
                                                            ⏹
                                                        </button>
                                                    )}

                                                    <div className="mcc-action-controls">
                                                        <button
                                                            className="mcc-action-expand-btn"
                                                            onClick={() => setExpandedActionId(expanded ? null : action.id)}
                                                            title={expanded ? 'Collapse' : 'Expand'}
                                                        >
                                                            {expanded ? '▼' : '▶'}
                                                        </button>
                                                        <button
                                                            className="mcc-action-edit-btn"
                                                            onClick={() => handleStartEdit(action)}
                                                            title="Edit"
                                                        >
                                                            ✎
                                                        </button>
                                                        <button
                                                            className="mcc-action-delete-btn"
                                                            onClick={() => handleDelete(action.id)}
                                                            title="Delete"
                                                        >
                                                            ×
                                                        </button>
                                                    </div>
                                                </div>

                                                <div className="mcc-action-script-row">
                                                    <code className="mcc-action-script">{action.script}</code>
                                                </div>

                                                {expanded && (
                                                    <div className="mcc-action-logs-section">
                                                        <div className="mcc-action-logs-header">
                                                            {running ? 'Running...' : logs.length > 0 ? 'Logs' : 'No logs'}
                                                            {actionStatuses[action.id]?.exit_code !== undefined && !running && (
                                                                <span className={`mcc-action-exit-code ${actionStatuses[action.id].exit_code === 0 ? 'success' : 'error'}`}>
                                                                    Exit: {actionStatuses[action.id].exit_code}
                                                                </span>
                                                            )}
                                                        </div>
                                                        {logs.length > 0 && (
                                                            <div className="mcc-action-logs">
                                                                {logs.map((log, i) => (
                                                                    <div key={i} className={`mcc-action-log-line ${log.error ? 'error' : ''}`}>{log.text}</div>
                                                                ))}
                                                            </div>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </>
                    )}
                </>
            )}
        </div>
    );
}
