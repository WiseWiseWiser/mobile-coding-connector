import { useState } from 'react';
import { MockupPageContainer } from './MockupPageContainer';
import './ServerFiles.css';

interface FileItem {
    name: string;
    isDirectory: boolean;
    size?: number;
}

const mockFiles: FileItem[] = [
    { name: '..', isDirectory: true },
    { name: 'src', isDirectory: true },
    { name: 'config', isDirectory: true },
    { name: 'go.mod', isDirectory: false, size: 89 },
    { name: 'go.sum', isDirectory: false, size: 2345 },
    { name: 'README.md', isDirectory: false, size: 567 },
    { name: '.gitignore', isDirectory: false, size: 123 },
    { name: 'Makefile', isDirectory: false, size: 456 },
    { name: 'docker-compose.yml', isDirectory: false, size: 789 },
    { name: '.env.example', isDirectory: false, size: 234 },
];

export function ServerFiles() {
    const [currentPath, setCurrentPath] = useState('/project/mobile-agent');
    const [copied, setCopied] = useState(false);
    const [isEditing, setIsEditing] = useState(false);
    const [editPath, setEditPath] = useState(currentPath);

    const handleCopy = () => {
        navigator.clipboard.writeText(currentPath);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const handleEditClick = () => {
        setEditPath(currentPath);
        setIsEditing(true);
    };

    const handleEditSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        setCurrentPath(editPath);
        setIsEditing(false);
    };

    const handleItemClick = (item: FileItem) => {
        if (item.name === '..') {
            const parts = currentPath.split('/').filter(Boolean);
            parts.pop();
            setCurrentPath('/' + parts.join('/') || '/');
        } else if (item.isDirectory) {
            setCurrentPath(currentPath + '/' + item.name);
        }
    };

    return (
        <MockupPageContainer 
            title="Server Files"
            description="Redesigned mobile-first file browser with tree view, file actions, and navigation"
        >
            <div className="server-files-header">
                <div className="server-files-path-bar">
                    <div className="server-files-path-label">Path:</div>
                    {isEditing ? (
                        <form onSubmit={handleEditSubmit} className="server-files-path-edit">
                            <input
                                type="text"
                                value={editPath}
                                onChange={(e) => setEditPath(e.target.value)}
                                autoFocus
                            />
                            <button type="submit">Go</button>
                            <button type="button" onClick={() => setIsEditing(false)}>Cancel</button>
                        </form>
                    ) : (
                        <div className="server-files-path-value">
                            <span className="server-files-path-text">{currentPath}</span>
                            <button className="server-files-path-copy" onClick={handleCopy}>
                                {copied ? 'Copied!' : 'Copy'}
                            </button>
                            <button className="server-files-path-change" onClick={handleEditClick}>
                                Change
                            </button>
                        </div>
                    )}
                </div>
                <div className="server-files-actions">
                    <button className="server-files-btn" onClick={() => window.location.reload()}>
                        Refresh
                    </button>
                </div>
            </div>
            <div className="server-files-list">
                {mockFiles.map((item, idx) => (
                    <div 
                        key={idx} 
                        className="server-files-item"
                        onClick={() => handleItemClick(item)}
                    >
                        <span className="server-files-icon">
                            {item.name === '..' ? '⬆️' : item.isDirectory ? '📁' : '📄'}
                        </span>
                        <span className="server-files-name">{item.name}</span>
                        {!item.isDirectory && item.size !== undefined && (
                            <span className="server-files-size">{(item.size / 1024).toFixed(1)} KB</span>
                        )}
                    </div>
                ))}
            </div>
        </MockupPageContainer>
    );
}
