import { useState, useEffect } from 'react';
import { browseDirectory } from '../../../api/filedownload';
import type { BrowseEntry } from '../../../api/filedownload';
import { useCurrent } from '../../../hooks/useCurrent';
import './DownloadFileView.css'; // Reuses download-* CSS classes

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

const SelectModes = {
    File: 'file',
    FileOrDir: 'file_or_dir',
} as const;

type SelectMode = typeof SelectModes[keyof typeof SelectModes];

interface ServerFileBrowserProps {
    /** Selection mode: 'file' only selects files, 'file_or_dir' allows selecting files and directories */
    selectMode?: SelectMode;
    /** Called when a file (or directory in file_or_dir mode) is selected/deselected */
    onSelect?: (path: string | null) => void;
    /** Called when current directory changes */
    onDirectoryChange?: (path: string) => void;
    /** Initial directory to browse */
    initialDir?: string;
}

export function ServerFileBrowser({
    selectMode = SelectModes.File,
    onSelect,
    onDirectoryChange,
    initialDir = '/',
}: ServerFileBrowserProps) {
    const [currentDir, setCurrentDir] = useState(initialDir);
    const [pathInput, setPathInput] = useState(initialDir);
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
            setPathInput(result.path);
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
                // In file_or_dir mode, single click on dir selects it, double-click navigates
                const newSelected = selectedPath === entry.path ? null : entry.path;
                setSelectedPath(newSelected);
                onSelectRef.current?.(newSelected);
            } else {
                loadDirectory(entry.path);
            }
        } else {
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

    const handlePathGo = () => {
        const trimmed = pathInput.trim();
        if (!trimmed) return;
        loadDirectory(trimmed);
    };

    const handlePathKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handlePathGo();
        }
    };

    const breadcrumbs = pathSegments(currentDir);

    return (
        <>
            {/* Path Input */}
            <div className="download-section">
                <label className="download-label">Server Path</label>
                <div className="download-path-row">
                    <input
                        type="text"
                        className="download-path-input"
                        placeholder="/path/to/directory"
                        value={pathInput}
                        onChange={e => setPathInput(e.target.value)}
                        onKeyDown={handlePathKeyDown}
                    />
                    <button
                        className="download-go-btn"
                        onClick={handlePathGo}
                        disabled={loading || !pathInput.trim()}
                    >
                        Go
                    </button>
                </div>
            </div>

            {/* Breadcrumb */}
            <div className="download-breadcrumb">
                {breadcrumbs.map((seg, i) => (
                    <span key={seg.path}>
                        {i > 0 && <span className="download-breadcrumb-sep">/</span>}
                        {i < breadcrumbs.length - 1 ? (
                            <button className="download-breadcrumb-item" onClick={() => loadDirectory(seg.path)}>
                                {seg.name}
                            </button>
                        ) : (
                            <span className="download-breadcrumb-current">{seg.name}</span>
                        )}
                    </span>
                ))}
            </div>

            {/* Error */}
            {error && <div className="download-error">{error}</div>}

            {/* File Browser */}
            {loading ? (
                <div className="download-loading">Loading...</div>
            ) : entries.length === 0 && !error ? (
                <div className="download-empty">Directory is empty</div>
            ) : (
                <div className="download-file-list">
                    {entries.map(entry => (
                        <button
                            key={entry.path}
                            className={`download-file-item${selectedPath === entry.path ? ' download-file-item-selected' : ''}`}
                            onClick={() => handleEntryClick(entry)}
                            onDoubleClick={() => handleEntryDoubleClick(entry)}
                        >
                            <span className="download-file-icon">{entry.is_dir ? '📁' : '📄'}</span>
                            <span className="download-file-name">{entry.name}</span>
                            {entry.is_dir ? (
                                <span className="download-file-dir-hint">→</span>
                            ) : (
                                <span className="download-file-size">{formatFileSize(entry.size)}</span>
                            )}
                        </button>
                    ))}
                </div>
            )}
        </>
    );
}

export { SelectModes };
export type { SelectMode };
