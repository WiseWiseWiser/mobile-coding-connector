import { useState, useEffect, useRef } from 'react';
import { getGitStatus, getDiff, stageFile, unstageFile, gitCommit } from '../../../api/review';
import type { GitStatusFile } from '../../../api/review';
import type { DiffFile } from '../../../components/code-review/types';
import { DiffViewer } from '../../DiffViewer';
import type { FileDiff, DiffHunk, DiffLine } from '../../../api/checkpoints';
import { statusBadge } from './utils';
import { loadGitUserConfig } from '../home/settings/gitStorage';
import { GitPushSection } from './GitPushSection';
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

    const messageRef = useRef<HTMLTextAreaElement>(null);

    // Load git user config on mount
    useEffect(() => {
        const config = loadGitUserConfig();
        setGitUserConfig(config);
    }, []);

    const refresh = async () => {
        setLoading(true);
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

    const handleStageAll = async () => {
        try {
            for (const f of unstagedFiles) {
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

    const handleFileClick = (path: string) => {
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

            {loading ? (
                <div className="mcc-files-empty">Loading...</div>
            ) : (
                <>
                    {/* Error message at top */}
                    {error && <div className="mcc-checkpoint-error">{error}</div>}

                    {/* Staged Files */}
                    <div className="mcc-checkpoint-section-header">
                        <span className="mcc-checkpoint-section-label">
                            Staged ({stagedFiles.length})
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
                                    <span className="mcc-changed-file-path">{f.path}</span>
                                    <button
                                        className="mcc-git-file-action"
                                        onClick={(e) => { e.stopPropagation(); handleUnstage(f.path); }}
                                    >
                                        −
                                    </button>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>
                            No staged changes
                        </div>
                    )}

                    {/* Unstaged Files */}
                    <div className="mcc-checkpoint-section-header">
                        <span className="mcc-checkpoint-section-label">
                            Unstaged ({unstagedFiles.length})
                        </span>
                        {unstagedFiles.length > 0 && (
                            <button className="mcc-git-action-btn" onClick={handleStageAll}>
                                Stage All
                            </button>
                        )}
                    </div>
                    {unstagedFiles.length > 0 ? (
                        <div className="mcc-changed-files-list">
                            {unstagedFiles.map(f => (
                                <div
                                    key={`unstaged-${f.path}`}
                                    className={`mcc-changed-file-item${selectedFile === f.path ? ' mcc-changed-file-item-selected' : ''}`}
                                    onClick={() => handleFileClick(f.path)}
                                >
                                    {statusBadge(f.status === 'untracked' ? 'added' : f.status)}
                                    <span className="mcc-changed-file-path">{f.path}</span>
                                    <button
                                        className="mcc-git-file-action"
                                        onClick={(e) => { e.stopPropagation(); handleStage(f.path); }}
                                    >
                                        +
                                    </button>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div className="mcc-files-empty" style={{ padding: '12px 16px' }}>
                            No unstaged changes
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
                            <textarea
                                ref={messageRef}
                                className="mcc-checkpoint-message-input"
                                placeholder="Enter commit message..."
                                value={commitMessage}
                                onChange={(e) => setCommitMessage(e.target.value)}
                                rows={3}
                            />
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
                </>
            )}
        </div>
    );
}
