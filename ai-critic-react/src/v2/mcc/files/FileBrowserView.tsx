import { useState, useEffect } from 'react';
import { fetchFiles } from '../../../api/checkpoints';
import type { FileEntry } from '../../../api/checkpoints';
import './FilesView.css';

export interface FileBrowserViewProps {
    projectDir: string;
    currentPath: string;
    onNavigate: (path: string) => void;
    onViewFile: (path: string) => void;
}

export function FileBrowserView({ projectDir, currentPath, onNavigate, onViewFile }: FileBrowserViewProps) {
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
