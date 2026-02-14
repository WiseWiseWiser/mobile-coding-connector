import { useState, useEffect, useRef, useCallback } from 'react';
import {
    createCheckpoint,
    fetchCurrentChanges,
    fetchSingleFileDiff,
} from '../../../api/checkpoints';
import type { ChangedFile, FileDiff } from '../../../api/checkpoints';
import { DiffViewer } from '../../DiffViewer';
import { statusBadge } from './utils';
import './FilesView.css';

export interface CreateCheckpointViewProps {
    projectName: string;
    projectDir: string;
    onBack: () => void;
    onCreated: () => void;
}

export function CreateCheckpointView({ projectName, projectDir, onBack, onCreated }: CreateCheckpointViewProps) {
    const [changedFiles, setChangedFiles] = useState<ChangedFile[]>([]);
    const [diffs, setDiffs] = useState<Map<string, FileDiff>>(new Map());
    const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());
    const [loading, setLoading] = useState(true);
    const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
    const [checkpointName, setCheckpointName] = useState('');
    const [checkpointMessage, setCheckpointMessage] = useState('');
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        setLoading(true);
        fetchCurrentChanges(projectName, projectDir)
            .then(files => {
                const fileList = files || [];
                setChangedFiles(fileList);
                setSelectedPaths(new Set(fileList.map(f => f.path)));
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [projectName, projectDir]);

    const loadFileDiff = useCallback(async (path: string) => {
        if (diffs.has(path) || expandedFiles.has(path)) return;
        
        try {
            const diff = await fetchSingleFileDiff(projectDir, path);
            setDiffs(prev => new Map(prev).set(path, diff));
            setExpandedFiles(prev => new Set(prev).add(path));
        } catch (err) {
            console.error('Failed to load diff for', path, err);
        }
    }, [projectDir, diffs, expandedFiles]);

    useEffect(() => {
        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const path = entry.target.getAttribute('data-file-path');
                        if (path) {
                            loadFileDiff(path);
                        }
                    }
                });
            },
            { rootMargin: '100px' }
        );

        const container = containerRef.current;
        if (container) {
            const fileItems = container.querySelectorAll('[data-file-path]');
            fileItems.forEach(item => observer.observe(item));
        }

        return () => observer.disconnect();
    }, [changedFiles, loadFileDiff]);

    const toggleFile = (path: string) => {
        setSelectedPaths(prev => {
            const next = new Set(prev);
            if (next.has(path)) {
                next.delete(path);
            } else {
                next.add(path);
            }
            return next;
        });
    };

    const toggleExpand = (path: string) => {
        setExpandedFiles(prev => {
            const next = new Set(prev);
            if (next.has(path)) {
                next.delete(path);
            } else {
                next.add(path);
                loadFileDiff(path);
            }
            return next;
        });
    };

    const toggleAll = () => {
        if (selectedPaths.size === changedFiles.length) {
            setSelectedPaths(new Set());
        } else {
            setSelectedPaths(new Set(changedFiles.map(f => f.path)));
        }
    };

    const handleCreate = async () => {
        if (selectedPaths.size === 0) return;
        setCreating(true);
        setError(null);
        try {
            await createCheckpoint(projectName, {
                project_dir: projectDir,
                name: checkpointName || undefined,
                message: checkpointMessage || undefined,
                file_paths: Array.from(selectedPaths),
            });
            onCreated();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to create checkpoint');
        } finally {
            setCreating(false);
        }
    };

    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2>New Checkpoint</h2>
            </div>

            {loading ? (
                <div className="mcc-files-empty">Scanning for changes...</div>
            ) : changedFiles.length === 0 ? (
                <div className="mcc-files-empty">No changes detected in the working tree.</div>
            ) : (
                <>
                    <div className="mcc-files-select-all">
                        <label>
                            <input
                                type="checkbox"
                                checked={selectedPaths.size === changedFiles.length}
                                onChange={toggleAll}
                            />
                            <span>Select all ({changedFiles.length} file{changedFiles.length !== 1 ? 's' : ''})</span>
                        </label>
                    </div>
                    <div className="mcc-changed-files-list" ref={containerRef}>
                        {changedFiles.map(f => (
                            <div key={f.path} data-file-path={f.path} className="mcc-changed-file-wrapper">
                                <label className="mcc-changed-file-item">
                                    <input
                                        type="checkbox"
                                        checked={selectedPaths.has(f.path)}
                                        onChange={() => toggleFile(f.path)}
                                    />
                                    {statusBadge(f.status)}
                                    <span className="mcc-changed-file-path">{f.path}</span>
                                </label>
                                <button 
                                    className="mcc-changed-file-expand"
                                    onClick={() => toggleExpand(f.path)}
                                >
                                    {expandedFiles.has(f.path) ? '▼' : '▶'}
                                </button>
                                {expandedFiles.has(f.path) && diffs.get(f.path) && (
                                    <div className="mcc-changed-file-diff">
                                        <DiffViewer diffs={[diffs.get(f.path)!]} />
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                </>
            )}

            <div className="mcc-checkpoint-form">
                <input
                    className="mcc-checkpoint-name-input"
                    type="text"
                    placeholder="Checkpoint name (optional)"
                    value={checkpointName}
                    onChange={e => setCheckpointName(e.target.value)}
                />
                <textarea
                    className="mcc-checkpoint-message-input"
                    placeholder="Message (optional)"
                    value={checkpointMessage}
                    onChange={e => setCheckpointMessage(e.target.value)}
                    rows={3}
                />
            </div>

            {error && <div className="mcc-checkpoint-error">{error}</div>}

            <button
                className="mcc-create-checkpoint-btn"
                onClick={handleCreate}
                disabled={creating || selectedPaths.size === 0}
            >
                {creating ? 'Creating...' : `Create Checkpoint (${selectedPaths.size} files)`}
            </button>
        </div>
    );
}
