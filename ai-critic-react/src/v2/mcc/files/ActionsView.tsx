import { useState, useEffect } from 'react';
import { fetchActions, createAction, updateAction, deleteAction, runAction } from '../../../api/actions';
import type { Action } from '../../../api/actions';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import './ActionsView.css';

interface ActionsViewProps {
    projectName: string;
    projectDir: string;
}

// Icon options for actions
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
    const [runningActionId, setRunningActionId] = useState<string | null>(null);

    // Form state
    const [formName, setFormName] = useState('');
    const [formIcon, setFormIcon] = useState('▶️');
    const [formScript, setFormScript] = useState('');

    const [runState, runControls] = useStreamingAction();

    useEffect(() => {
        loadActions();
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
        setRunningActionId(action.id);
        try {
            await runControls.run(async () => {
                return runAction({
                    project_dir: projectDir,
                    script: action.script,
                });
            });
        } finally {
            setRunningActionId(null);
        }
    };

    const isEditing = isCreating || editingAction !== null;

    return (
        <div className="mcc-actions-view">
            {/* Header */}
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

                    {/* Edit Form */}
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

                    {/* Actions List */}
                    {!isEditing && (
                        <>
                            {actions.length === 0 ? (
                                <div className="mcc-actions-empty">
                                    No actions defined yet. Click "Add Action" to create one.
                                </div>
                            ) : (
                                <div className="mcc-actions-list">
                                    {actions.map((action) => (
                                        <div key={action.id} className="mcc-action-item">
                                            <div className="mcc-action-main">
                                                <button
                                                    className="mcc-action-run-btn"
                                                    onClick={() => handleRun(action)}
                                                    disabled={runningActionId === action.id || runState.running}
                                                >
                                                    <span className="mcc-action-icon">{action.icon || '▶️'}</span>
                                                    <span className="mcc-action-name">{action.name}</span>
                                                    {runningActionId === action.id && (
                                                        <span className="mcc-action-running">Running...</span>
                                                    )}
                                                </button>
                                                <div className="mcc-action-controls">
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
                                            <div className="mcc-action-script-preview">
                                                <code>{action.script}</code>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}

                            {/* Streaming Logs */}
                            {(runState.running || runState.logs.length > 0) && (
                                <div className="mcc-actions-logs">
                                    <StreamingLogs
                                        state={runState}
                                        pendingMessage="Running action..."
                                        maxHeight={200}
                                    />
                                </div>
                            )}
                        </>
                    )}
                </>
            )}
        </div>
    );
}
