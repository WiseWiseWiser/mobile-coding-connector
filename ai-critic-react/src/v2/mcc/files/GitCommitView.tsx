import { useState, useEffect, useRef } from 'react';
import { getGitStatus, getDiff, stageFile, unstageFile, gitCommit, gitCheckout, gitRemove, listUntrackedDir, generateCommitMessage } from '../../../api/review';
import type { GitStatusFile } from '../../../api/review';
import type { DiffFile } from '../../../components/code-review/types';
import { DiffViewer } from '../../DiffViewer';
import type { FileDiff, DiffHunk, DiffLine } from '../../../api/checkpoints';
import { statusBadge, getFileIcon, formatFileSize, getFileSuffix } from './utils';
import { loadGitUserConfig } from '../home/settings/gitStorage';
import { GitPushSection } from './GitPushSection';
import { ConfirmModal } from '../ConfirmModal';
import { NoZoomingInput } from '../components/NoZoomingInput';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import './FilesView.css';
import './GitCommitView.css';

export interface GitCommitViewProps {
    projectDir: string;
    sshKeyId?: string;
    onBack: () => void;
}

// Convert a raw diff string for a single file into FileDiff format for the DiffViewer
function parseDiffToFileDiff(file: DiffFile): FileDiff {
    const hunks: DiffHunk[] = [];
    const lines = file.diff.split('\n');

    let currentHunk: DiffHunk | null = null;
    let oldNum = 0;
    let newNum = 0;

    for (const line of lines) {
        const hunkMatch = line.match(/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/);
        if (hunkMatch) {
            currentHunk = {
                old_start: parseInt(hunkMatch[1]),
                old_lines: parseInt(hunkMatch[2] || '1'),
                new_start: parseInt(hunkMatch[3]),
                new_lines: parseInt(hunkMatch[4] || '1'),
                lines: [],
            };
            hunks.push(currentHunk);
            oldNum = currentHunk.old_start;
            newNum = currentHunk.new_start;
            continue;
        }

        if (!currentHunk) continue;

        if (line.startsWith('+')) {
            const diffLine: DiffLine = { type: 'add', content: line.substring(1), new_num: newNum };
            currentHunk.lines.push(diffLine);
            newNum++;
        } else if (line.startsWith('-')) {
            const diffLine: DiffLine = { type: 'delete', content: line.substring(1), old_num: oldNum };
            currentHunk.lines.push(diffLine);
            oldNum++;
        } else if (line.startsWith(' ') || line === '') {
            // Only add context lines if we're inside a hunk (line starts with space)
            if (line.startsWith(' ')) {
                const diffLine: DiffLine = { type: 'context', content: line.substring(1), old_num: oldNum, new_num: newNum };
                currentHunk.lines.push(diffLine);
                oldNum++;
                newNum++;
            }
        }
    }

    return {
        path: file.path,
        status: file.status,
        hunks,
    };
}

// Helper function to render file tags (Git dir, Big File, Large File)
function renderFileTags(f: GitStatusFile): React.ReactNode {
    const tags: React.ReactNode[] = [];

    // Git directory tag - show different tag for worktrees
    if (f.isDir && f.isGitDir) {
        if (f.isGitWorktree) {
            tags.push(<span key="git" className="mcc-file-tag mcc-file-tag-git-worktree">Git Worktree</span>);
        } else {
            tags.push(<span key="git" className="mcc-file-tag mcc-file-tag-git">Git</span>);
        }
    }

    // Large file tags (only for files, not directories)
    if (!f.isDir && f.size) {
        if (f.size > 1000 * 1000) { // > 1MB
            tags.push(<span key="large" className="mcc-file-tag mcc-file-tag-large">Large File</span>);
        } else if (f.size > 100 * 1000) { // > 100KB
            tags.push(<span key="big" className="mcc-file-tag mcc-file-tag-big">Big File</span>);
        }
    }

    if (tags.length === 0) return null;
    return <span className="mcc-file-tags">{tags}</span>;
}

export function GitCommitView({ projectDir, sshKeyId, onBack }: GitCommitViewProps) {
    const [stagedFiles, setStagedFiles] = useState<GitStatusFile[]>([]);
    const [unstagedFiles, setUnstagedFiles] = useState<GitStatusFile[]>([]);
    const [branch, setBranch] = useState('');
    const [loading, setLoading] = useState(true);
    const [commitMessage, setCommitMessage] = useState('');
    const [committing, setCommitting] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');
    const [showDiffs, setShowDiffs] = useState(false);
    const [diffs, setDiffs] = useState<FileDiff[]>([]);
    const [loadingDiffs, setLoadingDiffs] = useState(false);
    const [selectedFile, setSelectedFile] = useState<string | null>(null);
    const [gitUserConfig, setGitUserConfig] = useState<{ name: string; email: string }>({ name: '', email: '' });
    const [modalState, setModalState] = useState<{
        type: 'discard' | 'remove';
        file: GitStatusFile;
    } | null>(null);
    const [browsePath, setBrowsePath] = useState<string | null>(null);
    const [browsedFiles, setBrowsedFiles] = useState<GitStatusFile[]>([]);
    const [loadingBrowsed, setLoadingBrowsed] = useState(false);

    const [generateState, generateControls] = useStreamingAction((result) => {
        if (result.ok && result.message) {
            setCommitMessage(result.message);
        }
    });

    const messageRef = useRef<HTMLTextAreaElement>(null);

    // Load git user config on mount
    useEffect(() => {
        const config = loadGitUserConfig();
        setGitUserConfig(config);
    }, []);

    const refresh = async () => {
        if (stagedFiles.length === 0 && unstagedFiles.length === 0) {
            setLoading(true);
        }
        setError('');
        try {
            const status = await getGitStatus(projectDir);
            setBranch(status.branch);
            setStagedFiles(status.files.filter(f => f.isStaged));
            setUnstagedFiles(status.files.filter(f => !f.isStaged));
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to get git status');
        } finally {
            setLoading(false);
        }
    };

    const navigateToDir = async (path: string) => {
        const normalizedPath = path.replace(/\/+$/, '');
        setLoadingBrowsed(true);
        setError('');
        try {
            const result = await listUntrackedDir(normalizedPath, projectDir);
            setBrowsedFiles(result.files);
            setBrowsePath(normalizedPath);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to list directory');
        } finally {
            setLoadingBrowsed(false);
        }
    };

    const navigateBack = () => {
        setBrowsePath(null);
        setBrowsedFiles([]);
    };

    const navigateUp = () => {
        if (!browsePath) return;
        const parts = browsePath.split('/');
        parts.pop();
        if (parts.length === 0) {
            navigateBack();
        } else {
            navigateToDir(parts.join('/'));
        }
    };

    useEffect(() => {
        refresh();
    }, [projectDir]);

    // Load diffs when toggle is enabled
    useEffect(() => {
        if (!showDiffs) return;
        setLoadingDiffs(true);
        getDiff(projectDir)
            .then(result => {
                const fileDiffs = result.files.map(parseDiffToFileDiff);
                setDiffs(fileDiffs);
                setLoadingDiffs(false);
            })
            .catch(() => setLoadingDiffs(false));
    }, [showDiffs, projectDir]);

    const handleStage = async (path: string) => {
        try {
            await stageFile(path, projectDir);
            await refresh();
            // Refresh diffs if showing
            if (showDiffs) {
                setShowDiffs(false);
                setTimeout(() => setShowDiffs(true), 100);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to stage file');
        }
    };

    const handleUnstage = async (path: string) => {
        try {
            await unstageFile(path, projectDir);
            await refresh();
            if (showDiffs) {
                setShowDiffs(false);
                setTimeout(() => setShowDiffs(true), 100);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to unstage file');
        }
    };

    const handleDiscard = async () => {
        if (!modalState || modalState.type !== 'discard') return;
        const file = modalState.file;
        try {
            await gitCheckout(file.path, projectDir);
            setModalState(null);
            await refresh();
            if (showDiffs) {
                setShowDiffs(false);
                setTimeout(() => setShowDiffs(true), 100);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to discard changes');
        }
    };

    const handleRemove = async () => {
        if (!modalState || modalState.type !== 'remove') return;
        const file = modalState.file;
        try {
            await gitRemove(file.path, projectDir);
            setModalState(null);
            await refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to remove file');
        }
    };

    const handleStageAll = async () => {
        try {
            for (const f of unstagedFiles) {
                // Skip directories that are git repositories or git worktrees
                if (f.isDir && f.isGitDir) {
                    continue;
                }
                await stageFile(f.path, projectDir);
            }
            await refresh();
            if (showDiffs) {
                setShowDiffs(false);
                setTimeout(() => setShowDiffs(true), 100);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to stage files');
        }
    };

    const handleUnstageAll = async () => {
        try {
            for (const f of stagedFiles) {
                await unstageFile(f.path, projectDir);
            }
            await refresh();
            if (showDiffs) {
                setShowDiffs(false);
                setTimeout(() => setShowDiffs(true), 100);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to unstage files');
        }
    };

    const handleCommit = async () => {
        if (!commitMessage.trim()) {
            setError('Commit message is required');
            return;
        }
        if (!gitUserConfig.name.trim()) {
            setError('Git user name is not configured. Please configure it in Settings > Git Settings > Git Config.');
            return;
        }
        if (!gitUserConfig.email.trim()) {
            setError('Git user email is not configured. Please configure it in Settings > Git Settings > Git Config.');
            return;
        }
        setCommitting(true);
        setError('');
        setSuccess('');
        try {
            const result = await gitCommit(commitMessage.trim(), projectDir, {
                name: gitUserConfig.name,
                email: gitUserConfig.email,
            });
            setSuccess(result.output || 'Committed successfully');
            setCommitMessage('');
            await refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to commit');
        } finally {
            setCommitting(false);
        }
    };

    const handleGenerateCommitMessage = () => {
        if (stagedFiles.length === 0) {
            setError('No staged changes to generate commit message for');
            return;
        }
        generateControls.run(() => generateCommitMessage(projectDir));
    };

    const handleFileClick = (path: string, isDir?: boolean) => {
        if (isDir) {
            navigateToDir(path);
            return;
        }
        setSelectedFile(selectedFile === path ? null : path);
        if (!showDiffs) {
            setShowDiffs(true);
        }
    };

    const handleBrowsedFileClick = (path: string, isDir?: boolean) => {
        if (isDir) {
            navigateToDir(path);
            return;
        }
        setSelectedFile(selectedFile === path ? null : path);
        if (!showDiffs) {
            setShowDiffs(true);
        }
    };

    const selectedDiff = selectedFile ? diffs.filter(d => d.path === selectedFile) : [];

    return (
        <div className="mcc-git-commit">
            {/* Header */}
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2>Git Commit</h2>
                {branch && <span className="mcc-git-branch">{branch}</span>}
            </div>

            {/* Error message at top */}
            {error && <div className="mcc-checkpoint-error">{error}</div>}

            {/* Staged Files */}
            <div className="mcc-checkpoint-section-header">
                <span className="mcc-checkpoint-section-label">
                    Staged ({stagedFiles.length}){loading && stagedFiles.length > 0 && <span className="mcc-loading-indicator">...</span>}
                </span>
                {stagedFiles.length > 0 && (
                    <button className="mcc-git-action-btn" onClick={handleUnstageAll}>
                        Unstage All
                    </button>
                )}
            </div>
            {stagedFiles.length > 0 ? (
                <div className="mcc-changed-files-list">
                    {stagedFiles.map(f => (
                        <div
                            key={`staged-${f.path}`}
                            className={`mcc-changed-file-item${selectedFile === f.path ? ' mcc-changed-file-item-selected' : ''}`}
                            onClick={() => handleFileClick(f.path)}
                        >
                            {statusBadge(f.status)}
                            <div className="mcc-changed-file-info">
                                <span className="mcc-changed-file-path">{f.path}</span>
                                <span className="mcc-changed-file-size">{formatFileSize(f.size ?? 0)}</span>
                                {renderFileTags(f)}
                                <span className="mcc-changed-file-suffix">{getFileIcon(f.path)}{getFileSuffix(f.path)}</span>
                            </div>
                            <span className="mcc-git-file-actions">
                                <button
                                    className="mcc-git-file-action"
                                    onClick={(e) => { e.stopPropagation(); handleUnstage(f.path); }}
                                >
                                    −
                                </button>
                            </span>
                        </div>
                    ))}
                </div>
            ) : (
                <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>
                    {loading ? 'Loading...' : 'No staged changes'}
                </div>
            )}

            {/* Unstaged Files */}
            <div className="mcc-checkpoint-section-header">
                <span className="mcc-checkpoint-section-label">
                    Unstaged ({browsePath ? browsedFiles.length : unstagedFiles.length}){loading && unstagedFiles.length > 0 && <span className="mcc-loading-indicator">...</span>}
                </span>
                {browsePath ? (
                    <button className="mcc-git-action-btn" onClick={navigateUp}>
                        Up
                    </button>
                ) : unstagedFiles.length > 0 ? (
                    <button className="mcc-git-action-btn" onClick={handleStageAll}>
                        Stage All
                    </button>
                ) : null}
            </div>
            {browsePath ? (
                <div className="mcc-git-browse-path">
                    <button className="mcc-back-btn mcc-back-btn-small" onClick={navigateBack}>×</button>
                    <span className="mcc-git-breadcrumb">{browsePath.endsWith('/') ? browsePath : browsePath + '/'}</span>
                </div>
            ) : null}
            {loadingBrowsed ? (
                <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>Loading...</div>
            ) : browsePath ? (
                browsedFiles.length > 0 ? (
                    <div className="mcc-changed-files-list">
                        {browsedFiles.map(f => (
                            <div
                                key={`browsed-${f.path}`}
                                className={`mcc-changed-file-item${selectedFile === f.path ? ' mcc-changed-file-item-selected' : ''}${f.isDir ? ' mcc-changed-file-item-dir' : ''}`}
                                onClick={() => handleBrowsedFileClick(f.path, f.isDir)}
                            >
                                {statusBadge(f.isDir ? 'dir' : 'added')}
                                <div className="mcc-changed-file-info">
                                    <span className="mcc-changed-file-path">{(() => { const name = f.path.split('/').pop() || ''; return f.isDir && !name.endsWith('/') ? name + '/' : name; })()}</span>
                                    <span className="mcc-changed-file-size">{f.isDir ? 'dir' : formatFileSize(f.size ?? 0)}</span>
                                    {renderFileTags(f)}
                                </div>
                                <span className="mcc-git-file-actions">
                                    {!f.isDir && (
                                        <>
                                            <button
                                                className="mcc-git-file-action mcc-git-file-action-remove"
                                                title={`Remove ${f.path}`}
                                                onClick={(e) => { e.stopPropagation(); setModalState({ type: 'remove', file: f }); }}
                                            >
                                                ×
                                            </button>
                                            <button
                                                className="mcc-git-file-action"
                                                onClick={(e) => { e.stopPropagation(); handleStage(f.path); }}
                                            >
                                                +
                                            </button>
                                        </>
                                    )}
                                </span>
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>Empty directory</div>
                )
            ) : unstagedFiles.length > 0 ? (
                <div className="mcc-changed-files-list">
                    {unstagedFiles.map(f => (
                        <div
                            key={`unstaged-${f.path}`}
                            className={`mcc-changed-file-item${selectedFile === f.path ? ' mcc-changed-file-item-selected' : ''}${f.isDir ? ' mcc-changed-file-item-dir' : ''}`}
                            onClick={() => handleFileClick(f.path, f.isDir)}
                        >
                            {statusBadge(f.isDir ? 'dir' : (f.status === 'untracked' ? 'added' : f.status))}
                            <div className="mcc-changed-file-info">
                                <span className="mcc-changed-file-path">{f.isDir && !f.path.endsWith('/') ? f.path + '/' : f.path}</span>
                                <span className="mcc-changed-file-size">{f.isDir ? 'dir' : formatFileSize(f.size ?? 0)}</span>
                                {renderFileTags(f)}
                                {!f.isDir && <span className="mcc-changed-file-suffix">{getFileIcon(f.path)}{getFileSuffix(f.path)}</span>}
                            </div>
                            <span className="mcc-git-file-actions">
                                {f.isDir ? (
                                    <button
                                        className="mcc-git-file-action"
                                        title={`Enter ${f.path}`}
                                    >
                                        →
                                    </button>
                                ) : f.status === 'untracked' ? (
                                    <>
                                        <button
                                            className="mcc-git-file-action mcc-git-file-action-remove"
                                            title={`Remove ${f.path}`}
                                            onClick={(e) => { e.stopPropagation(); setModalState({ type: 'remove', file: f }); }}
                                        >
                                            ×
                                        </button>
                                        <button
                                            className="mcc-git-file-action"
                                            onClick={(e) => { e.stopPropagation(); handleStage(f.path); }}
                                        >
                                            +
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <button
                                            className="mcc-git-file-action mcc-git-file-action-discard"
                                            title={`Discard changes to ${f.path}`}
                                            onClick={(e) => { e.stopPropagation(); setModalState({ type: 'discard', file: f }); }}
                                        >
                                            ↩
                                        </button>
                                        <button
                                            className="mcc-git-file-action"
                                            onClick={(e) => { e.stopPropagation(); handleStage(f.path); }}
                                        >
                                            +
                                        </button>
                                    </>
                                )}
                            </span>
                        </div>
                    ))}
                </div>
            ) : (
                <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>
                    {loading ? 'Loading...' : 'No unstaged changes'}
                </div>
            )}

            {/* Diff View */}
            {selectedFile && (
                <div className="mcc-git-diff-section">
                    <div className="mcc-checkpoint-section-label">Diff: {selectedFile}</div>
                    <div className="mcc-current-diffs">
                        {loadingDiffs ? (
                            <div className="mcc-files-empty">Loading diffs...</div>
                        ) : selectedDiff.length > 0 ? (
                            <DiffViewer diffs={selectedDiff} />
                        ) : (
                            <div className="mcc-files-empty">No diff available</div>
                        )}
                    </div>
                </div>
            )}

            {/* Show All Diffs Toggle */}
            {!selectedFile && (stagedFiles.length > 0 || unstagedFiles.length > 0) && (
                <div className="mcc-git-diff-toggle-section">
                    <button
                        className={`mcc-diff-toggle-btn ${showDiffs ? 'active' : ''}`}
                        onClick={() => setShowDiffs(!showDiffs)}
                    >
                        {showDiffs ? 'Hide All Diffs' : 'Show All Diffs'}
                    </button>
                    {showDiffs && (
                        <div className="mcc-current-diffs" style={{ marginTop: 8 }}>
                            {loadingDiffs ? (
                                <div className="mcc-files-empty">Loading diffs...</div>
                            ) : diffs.length > 0 ? (
                                <DiffViewer diffs={diffs} />
                            ) : (
                                <div className="mcc-files-empty">No diffs available</div>
                            )}
                        </div>
                    )}
                </div>
            )}

            {/* Commit Section */}
            <div className="mcc-git-commit-section">
                <div className="mcc-checkpoint-section-label">Commit Message</div>
                <div className="mcc-git-commit-form">
                    <NoZoomingInput>
                        <textarea
                            ref={messageRef}
                            className="mcc-checkpoint-message-input"
                            placeholder="Enter commit message..."
                            value={commitMessage}
                            onChange={(e) => setCommitMessage(e.target.value)}
                            rows={3}
                        />
                    </NoZoomingInput>
                    {/* Generate button - next to textarea */}
                    <button
                        className="mcc-git-commit-btn"
                        onClick={handleGenerateCommitMessage}
                        disabled={generateState.running || stagedFiles.length === 0}
                        style={{ marginTop: 8, marginBottom: 8 }}
                    >
                        {generateState.running ? 'Generating...' : 'Generate'}
                    </button>
                    {/* Streaming logs */}
                    <StreamingLogs state={generateState} pendingMessage="Running agent..." maxHeight={200} />
                    {/* Commit button - standalone row */}
                    <button
                        className="mcc-git-commit-btn"
                        onClick={handleCommit}
                        disabled={committing || stagedFiles.length === 0 || !commitMessage.trim()}
                    >
                        {committing ? 'Committing...' : 'Commit'}
                    </button>
                    {/* Success message below commit button */}
                    {success && <div className="mcc-git-success">{success}</div>}

                    {/* Push section */}
                    <div style={{ marginTop: 12 }}>
                        <GitPushSection projectDir={projectDir} sshKeyId={sshKeyId} />
                    </div>
                </div>
            </div>

            {/* Confirm Modal for Discard/Remove */}
            {modalState && (
                modalState.type === 'discard' ? (
                    <ConfirmModal
                        title="Discard Changes"
                        message={`Are you sure you want to discard changes to "${modalState.file.path}"?`}
                        info={{
                            File: modalState.file.path,
                            Status: modalState.file.status,
                        }}
                        command={`git checkout -- "${modalState.file.path}"`}
                        confirmLabel="Discard Changes"
                        confirmVariant="danger"
                        onConfirm={handleDiscard}
                        onClose={() => setModalState(null)}
                    />
                ) : (
                    <ConfirmModal
                        title="Remove File"
                        message={`Are you sure you want to remove "${modalState.file.path}"?`}
                        info={{
                            File: modalState.file.path,
                            Status: modalState.file.status,
                        }}
                        command={`rm -f "${modalState.file.path}"`}
                        confirmLabel="Remove File"
                        confirmVariant="danger"
                        onConfirm={handleRemove}
                        onClose={() => setModalState(null)}
                    />
                )
            )}
        </div>
    );
}
