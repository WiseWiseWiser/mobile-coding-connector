import { useState, useEffect } from 'react';
import {
    fetchCheckpoints,
    deleteCheckpoint,
    fetchCurrentChanges,
    fetchCurrentDiff,
} from '../../../api/checkpoints';
import type { CheckpointSummary, ChangedFile, FileDiff } from '../../../api/checkpoints';
import { DiffViewer } from '../../DiffViewer';
import { statusBadge } from './utils';
import './FilesView.css';

export interface CheckpointListViewProps {
    projectName: string;
    projectDir: string;
    onCreateCheckpoint: () => void;
    onSelectCheckpoint: (id: number) => void;
}

export function CheckpointListView({ projectName, projectDir, onCreateCheckpoint, onSelectCheckpoint }: CheckpointListViewProps) {
    const [checkpoints, setCheckpoints] = useState<CheckpointSummary[]>([]);
    const [currentChanges, setCurrentChanges] = useState<ChangedFile[]>([]);
    const [currentDiffs, setCurrentDiffs] = useState<FileDiff[]>([]);
    const [loading, setLoading] = useState(true);
    const [showDiffs, setShowDiffs] = useState(false);
    const [loadingDiffs, setLoadingDiffs] = useState(false);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchCheckpoints(projectName),
            fetchCurrentChanges(projectName, projectDir),
        ])
            .then(([cpData, changes]) => {
                setCheckpoints(cpData || []);
                setCurrentChanges(changes || []);
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [projectName, projectDir]);

    // Load diffs when toggle is enabled
    useEffect(() => {
        if (!showDiffs || currentChanges.length === 0) return;
        setLoadingDiffs(true);
        fetchCurrentDiff(projectName, projectDir)
            .then(diffs => { setCurrentDiffs(diffs || []); setLoadingDiffs(false); })
            .catch(() => setLoadingDiffs(false));
    }, [showDiffs, projectName, projectDir, currentChanges.length]);

    const handleDelete = async (id: number) => {
        try {
            await deleteCheckpoint(projectName, id);
            setCheckpoints(prev => prev.filter(cp => cp.id !== id));
        } catch {
            // ignore
        }
    };

    const hasChanges = currentChanges.length > 0;

    return (
        <>
            {loading ? (
                <div className="mcc-files-empty">Loading...</div>
            ) : (
                <>
                    {/* Current changes section */}
                    <div className="mcc-checkpoint-section-header">
                        <span className="mcc-checkpoint-section-label">Current Changes</span>
                        {hasChanges && (
                            <button
                                className={`mcc-diff-toggle-btn ${showDiffs ? 'active' : ''}`}
                                onClick={() => setShowDiffs(!showDiffs)}
                            >
                                {showDiffs ? 'Hide Diffs' : 'Show Diffs'}
                            </button>
                        )}
                    </div>
                    {hasChanges ? (
                        <>
                            <div className="mcc-changed-files-list mcc-changed-files-compact">
                                {currentChanges.map(f => (
                                    <div key={f.path} className="mcc-changed-file-item mcc-changed-file-item-readonly">
                                        {statusBadge(f.status)}
                                        <span className="mcc-changed-file-path">{f.path}</span>
                                    </div>
                                ))}
                            </div>
                            {showDiffs && (
                                <div className="mcc-current-diffs">
                                    {loadingDiffs ? (
                                        <div className="mcc-files-empty">Loading diffs...</div>
                                    ) : (
                                        <DiffViewer diffs={currentDiffs} />
                                    )}
                                </div>
                            )}
                        </>
                    ) : (
                        <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>No changes in working tree</div>
                    )}

                    <button
                        className="mcc-create-checkpoint-btn"
                        onClick={onCreateCheckpoint}
                        disabled={!hasChanges}
                        title={hasChanges ? undefined : 'No changes in working tree'}
                    >
                        <span>{hasChanges ? `+ Create Checkpoint (${currentChanges.length} changes)` : 'No changes to checkpoint'}</span>
                    </button>

                    {/* Checkpoint history */}
                    {checkpoints.length > 0 && (
                        <>
                            <div className="mcc-checkpoint-section-label">History</div>
                            <div className="mcc-checkpoint-list">
                                {[...checkpoints].reverse().map(cp => (
                                    <div key={cp.id} className="mcc-checkpoint-card" onClick={() => onSelectCheckpoint(cp.id)}>
                                        <div className="mcc-checkpoint-card-header">
                                            <span className="mcc-checkpoint-name">{cp.name}</span>
                                            <span className="mcc-checkpoint-file-count">{cp.file_count} file{cp.file_count !== 1 ? 's' : ''}</span>
                                        </div>
                                        <div className="mcc-checkpoint-card-meta">
                                            <span>{new Date(cp.timestamp).toLocaleString()}</span>
                                            <button className="mcc-checkpoint-delete-btn" onClick={e => { e.stopPropagation(); handleDelete(cp.id); }}>Delete</button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </>
                    )}
                </>
            )}
        </>
    );
}
