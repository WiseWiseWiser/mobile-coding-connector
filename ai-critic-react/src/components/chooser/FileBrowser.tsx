import { useState, useEffect } from 'react';
import { browseDirectory } from '../../api/filedownload';
import type { BrowseEntry } from '../../api/filedownload';
import { useCurrent } from '../../hooks/useCurrent';
import './FileBrowser.css';

function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function pathSegments(path: string): { name: string; path: string }[] {
    const parts = path.split('/').filter(Boolean);
    const segments: { name: string; path: string }[] = [{ name: '/', path: '/' }];
    let current = '';
    for (const part of parts) {
        current += '/' + part;
        segments.push({ name: part, path: current });
    }
    return segments;
}

export const SelectModes = {
    File: 'file',
    FileOrDir: 'file_or_dir',
    Dir: 'dir',
} as const;

export type SelectMode = typeof SelectModes[keyof typeof SelectModes];

interface FileBrowserProps {
    selectMode?: SelectMode;
    onSelect?: (path: string | null) => void;
    onDirectoryChange?: (path: string) => void;
    initialDir?: string;
}

export function FileBrowser({
    selectMode = SelectModes.File,
    onSelect,
    onDirectoryChange,
    initialDir = '/',
}: FileBrowserProps) {
    const [currentDir, setCurrentDir] = useState(initialDir);
    const [entries, setEntries] = useState<BrowseEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [selectedPath, setSelectedPath] = useState<string | null>(null);

    const currentDirRef = useCurrent(currentDir);
    const onSelectRef = useCurrent(onSelect);
    const onDirectoryChangeRef = useCurrent(onDirectoryChange);

    useEffect(() => {
        loadDirectory(currentDirRef.current);
    }, []); // eslint-disable-line react-hooks/exhaustive-deps

    const loadDirectory = async (dir: string) => {
        setLoading(true);
        setError(null);
        setSelectedPath(null);
        onSelectRef.current?.(null);
        try {
            const result = await browseDirectory(dir);
            setEntries(result.entries);
            setCurrentDir(result.path);
            onDirectoryChangeRef.current?.(result.path);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
            setEntries([]);
        }
        setLoading(false);
    };

    const handleEntryClick = (entry: BrowseEntry) => {
        if (entry.is_dir) {
            if (selectMode === SelectModes.FileOrDir) {
                const newSelected = selectedPath === entry.path ? null : entry.path;
                setSelectedPath(newSelected);
                onSelectRef.current?.(newSelected);
            } else {
                loadDirectory(entry.path);
            }
        } else {
            if (selectMode === SelectModes.Dir) return;
            const newSelected = selectedPath === entry.path ? null : entry.path;
            setSelectedPath(newSelected);
            onSelectRef.current?.(newSelected);
        }
    };

    const handleEntryDoubleClick = (entry: BrowseEntry) => {
        if (entry.is_dir) {
            loadDirectory(entry.path);
        }
    };

    const breadcrumbs = pathSegments(currentDir);

    return (
        <>
            <div className="fb-breadcrumb">
                {breadcrumbs.map((seg, i) => (
                    <span key={seg.path}>
                        {i > 0 && <span className="fb-breadcrumb-sep">/</span>}
                        {i < breadcrumbs.length - 1 ? (
                            <button className="fb-breadcrumb-item" onClick={() => loadDirectory(seg.path)}>
                                {seg.name}
                            </button>
                        ) : (
                            <span className="fb-breadcrumb-current">{seg.name}</span>
                        )}
                    </span>
                ))}
            </div>

            {error && <div className="fb-error">{error}</div>}

            {loading ? (
                <div className="fb-loading">Loading...</div>
            ) : entries.length === 0 && !error ? (
                <div className="fb-empty">Directory is empty</div>
            ) : (
                <div className="fb-file-list">
                    {entries.map(entry => (
                        <button
                            key={entry.path}
                            className={`fb-file-item${selectedPath === entry.path ? ' fb-file-item-selected' : ''}`}
                            onClick={() => handleEntryClick(entry)}
                            onDoubleClick={() => handleEntryDoubleClick(entry)}
                        >
                            <span className="fb-file-icon">{entry.is_dir ? '📁' : '📄'}</span>
                            <span className="fb-file-name">{entry.name}</span>
                            {entry.is_dir ? (
                                <span className="fb-file-dir-hint">→</span>
                            ) : (
                                <span className="fb-file-size">{formatFileSize(entry.size)}</span>
                            )}
                        </button>
                    ))}
                </div>
            )}
        </>
    );
}
