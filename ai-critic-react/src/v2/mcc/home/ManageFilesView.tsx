import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { LocalTerminal } from './LocalTerminal';
import { FileActionSheet } from './FileActionSheet';
import { FileViewer } from './FileViewer';
import { FileEditor } from './FileEditor';
import { fetchHomeDir, fetchServerFiles } from '../../../api/files';
import type { FileEntry } from '../../../api/files';
import { TerminalIcon, BackIcon } from '../../icons';
import './ManageFilesView.css';

interface FileInfo {
    name: string;
    path: string;
    size: number;
    isDir: boolean;
    modifiedTime?: string;
}

export function ManageFilesView() {
    const navigate = useNavigate();
    const [currentPath, setCurrentPath] = useState('');
    const [homeDir, setHomeDir] = useState('');
    const [currentBasePath, setCurrentBasePath] = useState('');
    const [entries, setEntries] = useState<FileEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [showTerminal, setShowTerminal] = useState(false);
    const [terminalCwd, setTerminalCwd] = useState('');
    
    // File action states
    const [selectedFile, setSelectedFile] = useState<FileInfo | null>(null);
    const [showViewer, setShowViewer] = useState(false);
    const [showEditor, setShowEditor] = useState(false);

    useEffect(() => {
        fetchHomeDir().then(dir => {
            setHomeDir(dir);
            setCurrentBasePath(dir);
            setTerminalCwd(dir);
            loadFiles(dir, '');
        });
    }, []);

    const loadFiles = async (base: string, path: string) => {
        setLoading(true);
        try {
            const data = await fetchServerFiles(base, path || undefined);
            setEntries(data);
        } catch (e) {
            console.error('Failed to load files:', e);
            setEntries([]);
        } finally {
            setLoading(false);
        }
    };

    const getFullPath = (entryPath: string): string => {
        if (currentPath === '..') {
            return entryPath ? `/${entryPath}` : '/';
        }
        if (currentPath.startsWith('../')) {
            const relativePath = currentPath.slice(3);
            return relativePath ? `/${relativePath}/${entryPath}` : `/${entryPath}`;
        }
        return currentPath ? `${currentPath}/${entryPath}` : entryPath;
    };

    const handleFileClick = (entry: FileEntry) => {
        console.log('File clicked:', entry);
        const fullPath = getFullPath(entry.path);
        const fileInfo: FileInfo = {
            name: entry.name,
            path: fullPath,
            size: entry.size || 0,
            isDir: !!entry.is_dir,
            modifiedTime: entry.modified_time,
        };
        console.log('Setting selected file:', fileInfo);
        setSelectedFile(fileInfo);
    };

    const handleNavigate = async (path: string) => {
        if (path === '..' && currentPath === '') {
            setCurrentPath('..');
            setCurrentBasePath('/');
            await loadFiles('/', '');
            setTerminalCwd('/');
            return;
        }

        if (currentPath === '..') {
            if (path === '') {
                setCurrentPath('');
                setCurrentBasePath(homeDir);
                await loadFiles(homeDir, '');
                setTerminalCwd(homeDir);
                return;
            }
            const newPath = path === '..' ? '' : path;
            setCurrentPath(newPath ? `../${newPath}` : '..');
            await loadFiles('/', newPath);
            setTerminalCwd(newPath ? `/${newPath}` : '/');
            return;
        }

        setCurrentPath(path);
        await loadFiles(homeDir, path);
        if (path) {
            setTerminalCwd(`${homeDir}/${path}`);
        } else {
            setTerminalCwd(homeDir);
        }
    };

    const toggleTerminal = () => {
        setShowTerminal(!showTerminal);
    };

    const handleView = () => {
        setShowViewer(true);
    };

    const handleEdit = () => {
        setShowEditor(true);
    };

    const getBreadcrumbPath = () => {
        if (currentPath === '..') {
            return [{ name: 'root', path: '' }];
        }
        if (currentPath === '') {
            return [];
        }
        if (currentPath.startsWith('../')) {
            const relativePath = currentPath.slice(3);
            const segments = relativePath.split('/').filter(Boolean);
            return [{ name: 'root', path: '' }, ...segments.map((seg) => ({
                name: seg,
                path: '../' + segments.slice(0, segments.indexOf(seg) + 1).join('/')
            }))];
        }
        const segments = currentPath.split('/').filter(Boolean);
        return segments.map((seg, i) => ({
            name: seg,
            path: segments.slice(0, i + 1).join('/')
        }));
    };

    const breadcrumbSegments = getBreadcrumbPath();

    return (
        <div className="mcc-manage-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate(-1)}>
                    <BackIcon />
                </button>
                <h2>Manage Server Files</h2>
            </div>

            <button 
                className={`mcc-manage-files-terminal-toggle ${showTerminal ? 'active' : ''}`}
                onClick={toggleTerminal}
            >
                <TerminalIcon />
                {showTerminal ? 'Hide Terminal' : 'Show Terminal'}
            </button>

            {showTerminal && (
                <div className="mcc-manage-files-terminal">
                    <LocalTerminal
                        cwd={terminalCwd}
                        onClose={() => setShowTerminal(false)}
                    />
                </div>
            )}

            <div className="mcc-manage-files-browser">
                <div className="mcc-filebrowser-breadcrumb">
                    <button className="mcc-filebrowser-crumb" onClick={() => handleNavigate('')}>
                        {currentPath === '..' ? 'root' : 'home'}
                    </button>
                    {breadcrumbSegments.map((seg) => (
                        <span key={seg.path}>
                            <span className="mcc-filebrowser-crumb-sep">/</span>
                            <button className="mcc-filebrowser-crumb" onClick={() => handleNavigate(seg.path)}>
                                {seg.name}
                            </button>
                        </span>
                    ))}
                </div>

                {loading ? (
                    <div className="mcc-files-empty">Loading...</div>
                ) : (
                    <div className="mcc-filebrowser-list">
                        {(currentPath !== '..' || breadcrumbSegments.length > 0) && (
                            <div className="mcc-filebrowser-entry" onClick={() => {
                                if (currentPath === '') {
                                    handleNavigate('..');
                                } else if (currentPath === '..') {
                                    return;
                                } else if (currentPath.startsWith('../')) {
                                    const relativePath = currentPath.slice(3);
                                    const parentPath = relativePath.includes('/') 
                                        ? '../' + relativePath.substring(0, relativePath.lastIndexOf('/'))
                                        : '..';
                                    handleNavigate(parentPath);
                                } else {
                                    const parentPath = currentPath.includes('/') 
                                        ? currentPath.substring(0, currentPath.lastIndexOf('/'))
                                        : '';
                                    handleNavigate(parentPath);
                                }
                            }}>
                                <span className="mcc-filebrowser-icon">📁</span>
                                <span className="mcc-filebrowser-name">..</span>
                            </div>
                        )}
                        {entries.length === 0 ? (
                            <div className="mcc-files-empty">Empty directory</div>
                        ) : (
                            entries.map(entry => (
                                <div 
                                    key={entry.path} 
                                    className="mcc-filebrowser-entry" 
                                    onClick={() => {
                                        if (entry.is_dir) {
                                            if (currentPath === '..') {
                                                handleNavigate('../' + entry.name);
                                            } else if (currentPath.startsWith('../')) {
                                                handleNavigate(currentPath + '/' + entry.name);
                                            } else if (currentPath) {
                                                handleNavigate(currentPath + '/' + entry.name);
                                            } else {
                                                handleNavigate(entry.name);
                                            }
                                        } else {
                                            handleFileClick(entry);
                                        }
                                    }}
                                >
                                    <span className="mcc-filebrowser-icon">{entry.is_dir ? '📁' : '📄'}</span>
                                    <span className="mcc-filebrowser-name">{entry.name}</span>
                                    {!entry.is_dir && entry.size !== undefined && (
                                        <span className="mcc-filebrowser-size">{formatSize(entry.size)}</span>
                                    )}
                                </div>
                            ))
                        )}
                    </div>
                )}
            </div>

            <FileActionSheet
                file={selectedFile}
                onClose={() => setSelectedFile(null)}
                onView={handleView}
                onEdit={handleEdit}
            />

            {showViewer && selectedFile && (
                <FileViewer
                    filePath={currentBasePath + '/' + selectedFile.path}
                    onClose={() => setShowViewer(false)}
                />
            )}

            {showEditor && selectedFile && (
                <FileEditor
                    filePath={selectedFile.path}
                    basePath={currentBasePath}
                    onClose={() => setShowEditor(false)}
                    onSave={() => {
                        setShowEditor(false);
                        setSelectedFile(null);
                    }}
                />
            )}
        </div>
    );
}

function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
