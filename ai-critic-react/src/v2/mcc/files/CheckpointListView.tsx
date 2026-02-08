import { useState, useEffect } from 'react';
import {
    fetchCheckpoints,
    deleteCheckpoint,
    fetchCurrentChanges,
    fetchCurrentDiff,
} from '../../../api/checkpoints';
import type { CheckpointSummary, ChangedFile, FileDiff } from '../../../api/checkpoints';
import { gitFetch } from '../../../api/review';
import { encryptWithServerKey, EncryptionNotAvailableError } from '../home/crypto';
import { DiffViewer } from '../../DiffViewer';
import { statusBadge } from './utils';
import { loadSSHKeys } from '../home/settings/gitStorage';
import { fetchProjects } from '../../../api/projects';
import './FilesView.css';

export interface CheckpointListViewProps {
    projectName: string;
    projectDir: string;
    onCreateCheckpoint: () => void;
    onSelectCheckpoint: (id: number) => void;
    onGitCommit: () => void;
}

export function CheckpointListView({ projectName, projectDir, onCreateCheckpoint, onSelectCheckpoint, onGitCommit }: CheckpointListViewProps) {
    const [checkpoints, setCheckpoints] = useState<CheckpointSummary[]>([]);
    const [currentChanges, setCurrentChanges] = useState<ChangedFile[]>([]);
    const [currentDiffs, setCurrentDiffs] = useState<FileDiff[]>([]);
    const [loading, setLoading] = useState(true);
    const [showDiffs, setShowDiffs] = useState(false);
    const [loadingDiffs, setLoadingDiffs] = useState(false);
    const [fetching, setFetching] = useState(false);
    const [fetchResult, setFetchResult] = useState<{ ok: boolean; message: string } | null>(null);
    const [deletingCheckpointId, setDeletingCheckpointId] = useState<number | null>(null);

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
            setDeletingCheckpointId(null);
        } catch {
            // ignore
        }
    };

    const handleDeleteClick = (e: React.MouseEvent, id: number) => {
        e.stopPropagation();
        setDeletingCheckpointId(id);
    };

    const handleCancelDelete = (e: React.MouseEvent) => {
        e.stopPropagation();
        setDeletingCheckpointId(null);
    };

    const handleGitFetch = async () => {
        setFetching(true);
        setFetchResult(null);
        try {
            // Find the project to get its SSH key ID
            const projects = await fetchProjects();
            const project = projects.find(p => p.name === projectName || p.dir === projectDir);
            
            let encryptedSshKey: string | undefined;
            
            if (project?.ssh_key_id) {
                // Load SSH keys from localStorage
                const sshKeys = loadSSHKeys();
                const key = sshKeys.find(k => k.id === project.ssh_key_id);
                
                if (key) {
                    try {
                        encryptedSshKey = await encryptWithServerKey(key.privateKey);
                    } catch (err) {
                        if (err instanceof EncryptionNotAvailableError) {
                            setFetchResult({ ok: false, message: 'Server encryption keys not configured. Ask the server admin to run: go run ./script/crypto/gen' });
                            setFetching(false);
                            return;
                        }
                        throw err;
                    }
                }
            }
            
            const result = await gitFetch(projectDir, encryptedSshKey);
            setFetchResult({ ok: true, message: result.output || 'Fetch complete' });
        } catch (err: any) {
            setFetchResult({ ok: false, message: err.message || 'Fetch failed' });
        } finally {
            setFetching(false);
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
                                            {deletingCheckpointId === cp.id ? (
                                                <div className="mcc-checkpoint-delete-confirm" onClick={e => e.stopPropagation()}>
                                                    <span className="mcc-checkpoint-delete-confirm-text">Delete?</span>
                                                    <button className="mcc-checkpoint-delete-confirm-btn mcc-checkpoint-delete-confirm-yes" onClick={e => { e.stopPropagation(); handleDelete(cp.id); }}>Yes</button>
                                                    <button className="mcc-checkpoint-delete-confirm-btn mcc-checkpoint-delete-confirm-no" onClick={handleCancelDelete}>No</button>
                                                </div>
                                            ) : (
                                                <button className="mcc-checkpoint-delete-btn" onClick={e => handleDeleteClick(e, cp.id)}>Delete</button>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </>
                    )}

                    {/* Git action buttons */}
                    <div className="mcc-git-actions">
                        <button
                            className="mcc-git-commit-nav-btn"
                            onClick={onGitCommit}
                        >
                            Git Commit
                        </button>
                        <button
                            className="mcc-git-commit-nav-btn mcc-git-fetch-btn"
                            onClick={handleGitFetch}
                            disabled={fetching}
                        >
                            {fetching ? 'Fetching...' : 'Git Fetch'}
                        </button>
                    </div>
                    {fetchResult && (
                        <div className={`mcc-git-fetch-result ${fetchResult.ok ? 'success' : 'error'}`}>
                            {fetchResult.message || 'Fetch completed successfully'}
                        </div>
                    )}
                </>
            )}
        </>
    );
}
