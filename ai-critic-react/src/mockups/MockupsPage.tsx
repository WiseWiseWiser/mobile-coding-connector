import { useState } from 'react';
import { Routes, Route, Link, useLocation } from 'react-router-dom';
import './MockupsPage.css';

interface Mockup {
    id: string;
    name: string;
    description: string;
    path: string;
}

const mockups: Mockup[] = [
    {
        id: 'file-view-menu',
        name: 'File View Menu',
        description: 'File action sheet that appears when clicking a file in Manage Files',
        path: '/mockup/file-view-menu',
    },
];

export function MockupsPage() {
    const location = useLocation();
    const isIndex = location.pathname === '/mockups' || location.pathname === '/mockups/';

    if (isIndex) {
        return (
            <div className="mockups-page">
                <div className="mockups-header">
                    <h1>Mockups</h1>
                    <p>Isolated design reviews - no server requests</p>
                </div>
                <div className="mockups-list">
                    {mockups.map(mockup => (
                        <Link key={mockup.id} to={mockup.path} className="mockup-card">
                            <div className="mockup-card-icon">📱</div>
                            <div className="mockup-card-content">
                                <h3>{mockup.name}</h3>
                                <p>{mockup.description}</p>
                            </div>
                            <div className="mockup-card-arrow">→</div>
                        </Link>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <Routes>
            <Route path="/file-view-menu" element={<FileViewMenuMockup />} />
        </Routes>
    );
}

function FileViewMenuMockup() {
    const [selectedFile, setSelectedFile] = useState<{
        name: string;
        path: string;
        size: number;
        modifiedTime?: string;
    } | null>(null);

    const mockFiles = [
        { name: 'server.go', path: '/project/main.go', size: 15234, modifiedTime: '2024-01-15T10:30:00Z' },
        { name: 'App.tsx', path: '/src/App.tsx', size: 4521, modifiedTime: '2024-01-14T15:45:00Z' },
        { name: 'utils.ts', path: '/src/utils.ts', size: 2341, modifiedTime: '2024-01-13T09:20:00Z' },
    ];

    const handleFileClick = (file: typeof mockFiles[0]) => {
        setSelectedFile(file);
    };

    const handleClose = () => {
        setSelectedFile(null);
    };

    const handleView = () => {
        alert('View clicked - mockup only');
    };

    const handleEdit = () => {
        alert('Edit clicked - mockup only');
    };

    return (
        <div className="file-view-mockup">
            <div className="mockup-nav">
                <Link to="/mockups" className="mockup-back">← Back to Mockups</Link>
                <span className="mockup-title">File View Menu</span>
            </div>

            <div className="mockup-content">
                <h3>Click a file to see the menu:</h3>
                <div className="mockup-file-list">
                    {mockFiles.map(file => (
                        <div
                            key={file.name}
                            className="mockup-file-item"
                            onClick={() => handleFileClick(file)}
                        >
                            <span className="mockup-file-icon">📄</span>
                            <span className="mockup-file-name">{file.name}</span>
                            <span className="mockup-file-size">{(file.size / 1024).toFixed(1)} KB</span>
                        </div>
                    ))}
                </div>

                <p className="mockup-hint">
                    This is a mockup - no server requests are made. 
                    All data is local mock data.
                </p>
            </div>

            {/* File Action Sheet - same as FileActionSheet */}
            {selectedFile && (
                <>
                    <div className="file-action-sheet-overlay" onClick={handleClose} />
                    <div className="file-action-sheet">
                        <div className="file-action-sheet-handle" />
                        <div className="file-action-sheet-content">
                            <div className="file-action-sheet-header">
                                <div className="file-action-sheet-icon">📄</div>
                                <div className="file-action-sheet-name">{selectedFile.name}</div>
                            </div>

                            <div className="file-action-sheet-info">
                                <div className="file-action-sheet-info-row">
                                    <span className="file-action-sheet-info-label">Size</span>
                                    <span className="file-action-sheet-info-value">
                                        {selectedFile.size < 1024 
                                            ? `${selectedFile.size} B` 
                                            : `${(selectedFile.size / 1024).toFixed(1)} KB`}
                                    </span>
                                </div>
                                <div className="file-action-sheet-info-row">
                                    <span className="file-action-sheet-info-label">Path</span>
                                    <span className="file-action-sheet-info-value">{selectedFile.path}</span>
                                </div>
                                {selectedFile.modifiedTime && (
                                    <div className="file-action-sheet-info-row">
                                        <span className="file-action-sheet-info-label">Modified</span>
                                        <span className="file-action-sheet-info-value">
                                            {new Date(selectedFile.modifiedTime).toLocaleString()}
                                        </span>
                                    </div>
                                )}
                            </div>

                            <div className="file-action-sheet-actions">
                                <button className="file-action-sheet-btn view" onClick={handleView}>
                                    👁 View
                                </button>
                                <button className="file-action-sheet-btn edit" onClick={handleEdit}>
                                    ✎ Edit
                                </button>
                                <button className="file-action-sheet-btn cancel" onClick={handleClose}>
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                </>
            )}
        </div>
    );
}
