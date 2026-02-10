import { useState, useEffect } from 'react';
import { fetchFiles } from '../../../api/checkpoints';
import type { FileEntry } from '../../../api/checkpoints';
import { runGitOpByDir, GitOps } from '../../../api/projects';
import { encryptProjectSSHKey, EncryptionNotAvailableError } from '../home/crypto';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import { SSHKeyRequiredHint } from '../components/SSHKeyRequiredHint';
import './FilesView.css';

export interface FileBrowserViewProps {
    projectDir: string;
    currentPath: string;
    sshKeyId?: string;
    onNavigate: (path: string) => void;
    onViewFile: (path: string) => void;
}

export function FileBrowserView({ projectDir, currentPath, sshKeyId, onNavigate, onViewFile }: FileBrowserViewProps) {
    const [entries, setEntries] = useState<FileEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [encryptionError, setEncryptionError] = useState<string | null>(null);

    useEffect(() => {
        setLoading(true);
        fetchFiles(projectDir, currentPath || undefined)
            .then(data => { setEntries(data); setLoading(false); })
            .catch(() => setLoading(false));
    }, [projectDir, currentPath]);

    const [gitState, gitControls] = useStreamingAction();

    const handleGitAction = (op: typeof GitOps.Pull | typeof GitOps.Push) => {
        gitControls.run(async () => {
            setEncryptionError(null);
            let encryptedKey: string | undefined;
            try {
                encryptedKey = await encryptProjectSSHKey(sshKeyId);
            } catch (err) {
                if (err instanceof EncryptionNotAvailableError) {
                    setEncryptionError('Server encryption keys not configured.');
                    throw err;
                }
                throw err;
            }
            return runGitOpByDir(op, projectDir, encryptedKey);
        });
    };

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

            {/* Git Actions - each on its own row */}
            <div className="mcc-git-actions-column">
                {!sshKeyId && (
                    <SSHKeyRequiredHint message="SSH key required for git operations. Configure in project settings." />
                )}
                <button
                    className="mcc-git-commit-nav-btn mcc-git-fetch-btn"
                    onClick={() => handleGitAction(GitOps.Pull)}
                    disabled={gitState.running || !sshKeyId}
                    style={{ opacity: !sshKeyId ? 0.5 : 1 }}
                >
                    {gitState.running ? 'Running...' : 'Git Pull'}
                </button>
                <button
                    className="mcc-git-commit-nav-btn"
                    onClick={() => handleGitAction(GitOps.Push)}
                    disabled={gitState.running || !sshKeyId}
                    style={{ opacity: !sshKeyId ? 0.5 : 1 }}
                >
                    {gitState.running ? 'Running...' : 'Git Push'}
                </button>
                <StreamingLogs
                    state={gitState}
                    pendingMessage="Running..."
                    maxHeight={200}
                />
            </div>
            {encryptionError && (
                <div className="mcc-git-fetch-result error">{encryptionError}</div>
            )}
        </>
    );
}
