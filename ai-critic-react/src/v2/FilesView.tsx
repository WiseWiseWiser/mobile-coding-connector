import { useState, useEffect, useRef } from 'react';
import {
    fetchCheckpoints,
    createCheckpoint,
    deleteCheckpoint,
    fetchCurrentChanges,
    fetchCheckpointDetail,
    fetchCheckpointDiff,
    fetchCurrentDiff,
    fetchFiles,
    fetchFileContent,
} from '../api/checkpoints';
import type { CheckpointSummary, ChangedFile, CheckpointDetail, FileDiff, FileEntry } from '../api/checkpoints';
import { DiffViewer } from './DiffViewer';
import './FilesView.css';

// Sub-tabs within Files tab
const FilesSubTabs = {
    Checkpoints: 'checkpoints',
    Browse: 'browse',
} as const;

type FilesSubTab = typeof FilesSubTabs[keyof typeof FilesSubTabs];

interface FilesViewProps {
    projectName: string;
    projectDir: string;
    view: string;
    onNavigateToView: (view: string) => void;
}

export function FilesView({ projectName, projectDir, view, onNavigateToView }: FilesViewProps) {
    // Determine sub-tab from view
    const isBrowseView = view.startsWith('browse') || view.startsWith('file:');
    // Parse browse path
    const browsePath = view.startsWith('browse') ? view.replace(/^browse\/?/, '') : '';

    // Remember the last browse view to restore when switching back
    const lastBrowseViewRef = useRef('browse');
    if (view.startsWith('browse')) {
        lastBrowseViewRef.current = view;
    }

    // If we're in a specific checkpoint sub-view, render it directly
    if (view === 'create-checkpoint') {
        return (
            <CreateCheckpointView
                projectName={projectName}
                projectDir={projectDir}
                onBack={() => onNavigateToView('')}
                onCreated={() => onNavigateToView('')}
            />
        );
    }

    if (view.startsWith('checkpoint-')) {
        const idStr = view.replace('checkpoint-', '');
        const id = parseInt(idStr, 10);
        if (!isNaN(id)) {
            return (
                <CheckpointDetailView
                    projectName={projectName}
                    checkpointId={id}
                    onBack={() => onNavigateToView('')}
                />
            );
        }
    }

    // File content view
    if (view.startsWith('file:')) {
        const filePath = view.slice('file:'.length);
        return (
            <FileContentView
                projectDir={projectDir}
                filePath={filePath}
                onBack={() => {
                    // Go back to the parent directory in browse view
                    const parentDir = filePath.includes('/') ? 'browse/' + filePath.substring(0, filePath.lastIndexOf('/')) : 'browse';
                    onNavigateToView(parentDir);
                }}
            />
        );
    }

    const activeSubTab: FilesSubTab = isBrowseView ? FilesSubTabs.Browse : FilesSubTabs.Checkpoints;

    return (
        <div className="mcc-files">
            {/* Sub-tab bar */}
            <div className="mcc-files-subtabs">
                <button
                    className={`mcc-files-subtab${activeSubTab === FilesSubTabs.Checkpoints ? ' mcc-files-subtab-active' : ''}`}
                    onClick={() => onNavigateToView('')}
                >
                    Checkpoints
                </button>
                <button
                    className={`mcc-files-subtab${activeSubTab === FilesSubTabs.Browse ? ' mcc-files-subtab-active' : ''}`}
                    onClick={() => onNavigateToView(lastBrowseViewRef.current)}
                >
                    Browse Files
                </button>
            </div>

            {activeSubTab === FilesSubTabs.Checkpoints ? (
                <CheckpointListView
                    projectName={projectName}
                    projectDir={projectDir}
                    onCreateCheckpoint={() => onNavigateToView('create-checkpoint')}
                    onSelectCheckpoint={(id) => onNavigateToView(`checkpoint-${id}`)}
                />
            ) : (
                <FileBrowserView
                    projectDir={projectDir}
                    currentPath={browsePath}
                    onNavigate={(path) => onNavigateToView(path ? `browse/${path}` : 'browse')}
                    onViewFile={(path) => onNavigateToView(`file:${path}`)}
                />
            )}
        </div>
    );
}

// --- Checkpoint List View ---

interface CheckpointListViewProps {
    projectName: string;
    projectDir: string;
    onCreateCheckpoint: () => void;
    onSelectCheckpoint: (id: number) => void;
}

function CheckpointListView({ projectName, projectDir, onCreateCheckpoint, onSelectCheckpoint }: CheckpointListViewProps) {
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

    const statusBadge = (status: string) => {
        const cls = `mcc-file-status mcc-file-status-${status}`;
        const label = status === 'added' ? 'A' : status === 'deleted' ? 'D' : 'M';
        return <span className={cls}>{label}</span>;
    };

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

// --- Create Checkpoint View ---

interface CreateCheckpointViewProps {
    projectName: string;
    projectDir: string;
    onBack: () => void;
    onCreated: () => void;
}

function CreateCheckpointView({ projectName, projectDir, onBack, onCreated }: CreateCheckpointViewProps) {
    const [changedFiles, setChangedFiles] = useState<ChangedFile[]>([]);
    const [diffs, setDiffs] = useState<FileDiff[]>([]);
    const [loading, setLoading] = useState(true);
    const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
    const [checkpointName, setCheckpointName] = useState('');
    const [checkpointMessage, setCheckpointMessage] = useState('');
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchCurrentChanges(projectName, projectDir),
            fetchCurrentDiff(projectName, projectDir),
        ])
            .then(([files, diffData]) => {
                const fileList = files || [];
                setChangedFiles(fileList);
                setDiffs(diffData || []);
                // Select all by default
                setSelectedPaths(new Set(fileList.map(f => f.path)));
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [projectName, projectDir]);

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

    const statusBadge = (status: string) => {
        const cls = `mcc-file-status mcc-file-status-${status}`;
        const label = status === 'added' ? 'A' : status === 'deleted' ? 'D' : 'M';
        return <span className={cls}>{label}</span>;
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
                    <div className="mcc-changed-files-list">
                        {changedFiles.map(f => (
                            <label key={f.path} className="mcc-changed-file-item">
                                <input
                                    type="checkbox"
                                    checked={selectedPaths.has(f.path)}
                                    onChange={() => toggleFile(f.path)}
                                />
                                {statusBadge(f.status)}
                                <span className="mcc-changed-file-path">{f.path}</span>
                            </label>
                        ))}
                    </div>

                    {/* Show diffs for selected files */}
                    {diffs.length > 0 && (
                        <div className="mcc-create-checkpoint-diffs">
                            <div className="mcc-checkpoint-section-label">File Diffs</div>
                            <DiffViewer diffs={diffs.filter(d => selectedPaths.has(d.path))} />
                        </div>
                    )}
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

// --- Checkpoint Detail View ---

interface CheckpointDetailViewProps {
    projectName: string;
    checkpointId: number;
    onBack: () => void;
}

function CheckpointDetailView({ projectName, checkpointId, onBack }: CheckpointDetailViewProps) {
    const [detail, setDetail] = useState<CheckpointDetail | null>(null);
    const [diffs, setDiffs] = useState<FileDiff[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchCheckpointDetail(projectName, checkpointId),
            fetchCheckpointDiff(projectName, checkpointId),
        ])
            .then(([detailData, diffData]) => {
                setDetail(detailData);
                setDiffs(diffData || []);
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [projectName, checkpointId]);

    const statusBadge = (status: string) => {
        const cls = `mcc-file-status mcc-file-status-${status}`;
        const label = status === 'added' ? 'A' : status === 'deleted' ? 'D' : 'M';
        return <span className={cls}>{label}</span>;
    };

    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2>{detail?.name || `Checkpoint #${checkpointId}`}</h2>
            </div>
            {loading ? (
                <div className="mcc-files-empty">Loading...</div>
            ) : !detail ? (
                <div className="mcc-files-empty">Checkpoint not found.</div>
            ) : (
                <>
                    <div className="mcc-checkpoint-detail-meta">
                        <span>{new Date(detail.timestamp).toLocaleString()}</span>
                        <span>{detail.files.length} file{detail.files.length !== 1 ? 's' : ''}</span>
                    </div>
                    <div className="mcc-changed-files-list">
                        {detail.files.map(f => (
                            <div key={f.path} className="mcc-changed-file-item mcc-changed-file-item-readonly">
                                {statusBadge(f.status)}
                                <span className="mcc-changed-file-path">{f.path}</span>
                            </div>
                        ))}
                    </div>

                    {/* File Diffs */}
                    <div className="mcc-checkpoint-section-label">File Diffs</div>
                    <DiffViewer diffs={diffs} />
                </>
            )}
        </div>
    );
}

// --- File Browser View ---

interface FileBrowserViewProps {
    projectDir: string;
    currentPath: string;
    onNavigate: (path: string) => void;
    onViewFile: (path: string) => void;
}

function FileBrowserView({ projectDir, currentPath, onNavigate, onViewFile }: FileBrowserViewProps) {
    const [entries, setEntries] = useState<FileEntry[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        fetchFiles(projectDir, currentPath || undefined)
            .then(data => { setEntries(data); setLoading(false); })
            .catch(() => setLoading(false));
    }, [projectDir, currentPath]);

    const handleEntryClick = (entry: FileEntry) => {
        if (entry.is_dir) {
            onNavigate(entry.path);
        } else {
            onViewFile(entry.path);
        }
    };

    const formatSize = (bytes: number): string => {
        if (bytes < 1024) return `${bytes} B`;
        if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    };

    // Breadcrumb path segments
    const segments = currentPath ? currentPath.split('/') : [];

    return (
        <>
            {/* Breadcrumb navigation */}
            <div className="mcc-filebrowser-breadcrumb">
                <button className="mcc-filebrowser-crumb" onClick={() => onNavigate('')}>
                    /
                </button>
                {segments.map((seg, i) => {
                    const segPath = segments.slice(0, i + 1).join('/');
                    return (
                        <span key={segPath}>
                            <span className="mcc-filebrowser-crumb-sep">/</span>
                            <button className="mcc-filebrowser-crumb" onClick={() => onNavigate(segPath)}>
                                {seg}
                            </button>
                        </span>
                    );
                })}
            </div>

            {loading ? (
                <div className="mcc-files-empty">Loading...</div>
            ) : entries.length === 0 ? (
                <div className="mcc-files-empty">Empty directory</div>
            ) : (
                <div className="mcc-filebrowser-list">
                    {/* Parent directory link */}
                    {currentPath && (
                        <div className="mcc-filebrowser-entry" onClick={() => {
                            const parentPath = currentPath.includes('/') ? currentPath.substring(0, currentPath.lastIndexOf('/')) : '';
                            onNavigate(parentPath);
                        }}>
                            <span className="mcc-filebrowser-icon">📁</span>
                            <span className="mcc-filebrowser-name">..</span>
                        </div>
                    )}
                    {entries.map(entry => (
                        <div key={entry.path} className="mcc-filebrowser-entry" onClick={() => handleEntryClick(entry)}>
                            <span className="mcc-filebrowser-icon">{entry.is_dir ? '📁' : '📄'}</span>
                            <span className="mcc-filebrowser-name">{entry.name}</span>
                            {!entry.is_dir && entry.size !== undefined && (
                                <span className="mcc-filebrowser-size">{formatSize(entry.size)}</span>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </>
    );
}

// --- File Content View ---

interface FileContentViewProps {
    projectDir: string;
    filePath: string;
    onBack: () => void;
}

function FileContentView({ projectDir, filePath, onBack }: FileContentViewProps) {
    const [content, setContent] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [wordWrap, setWordWrap] = useState(true);

    useEffect(() => {
        setLoading(true);
        setError(null);
        fetchFileContent(projectDir, filePath)
            .then(data => { setContent(data); setLoading(false); })
            .catch(err => { setError(err instanceof Error ? err.message : 'Failed to load file'); setLoading(false); });
    }, [projectDir, filePath]);

    const fileName = filePath.includes('/') ? filePath.substring(filePath.lastIndexOf('/') + 1) : filePath;

    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2 className="mcc-file-viewer-title">{fileName}</h2>
            </div>
            <div className="mcc-file-viewer-toolbar">
                <span className="mcc-file-viewer-path-inline">{filePath}</span>
                <label className="mcc-file-viewer-wrap-toggle">
                    <input type="checkbox" checked={wordWrap} onChange={e => setWordWrap(e.target.checked)} />
                    <span>Wrap</span>
                </label>
            </div>
            {loading ? (
                <div className="mcc-files-empty">Loading file...</div>
            ) : error ? (
                <div className="mcc-checkpoint-error">{error}</div>
            ) : (
                <div className="mcc-file-viewer-content">
                    <pre className={`mcc-file-viewer-code${wordWrap ? ' mcc-file-viewer-code-wrap' : ''}`}>{content}</pre>
                </div>
            )}
        </div>
    );
}
