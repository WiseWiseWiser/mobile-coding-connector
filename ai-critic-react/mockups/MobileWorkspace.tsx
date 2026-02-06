import { useState } from 'react';
import './MobileWorkspace.css';

// Mobile-first workspace manager mockup - like VS Code for mobile

const FileTreeModes = {
    Collapsed: 'collapsed',
    Expanded: 'expanded',
} as const;

type FileTreeMode = typeof FileTreeModes[keyof typeof FileTreeModes];

interface FileNode {
    name: string;
    type: 'file' | 'folder';
    children?: FileNode[];
}

const mockFileTree: FileNode[] = [
    {
        name: 'src',
        type: 'folder',
        children: [
            { name: 'App.tsx', type: 'file' },
            { name: 'main.tsx', type: 'file' },
            {
                name: 'components',
                type: 'folder',
                children: [
                    { name: 'Header.tsx', type: 'file' },
                    { name: 'Sidebar.tsx', type: 'file' },
                    { name: 'Editor.tsx', type: 'file' },
                ],
            },
        ],
    },
    { name: 'package.json', type: 'file' },
    { name: 'README.md', type: 'file' },
];

export function MobileWorkspace() {
    const [activeTab, setActiveTab] = useState<'files' | 'search' | 'terminal' | 'settings'>('files');
    const [fileTreeMode, setFileTreeMode] = useState<FileTreeMode>(FileTreeModes.Collapsed);
    const [selectedFile, setSelectedFile] = useState<string | null>('App.tsx');
    const [openFiles, setOpenFiles] = useState<string[]>(['App.tsx', 'main.tsx']);

    const handleFileSelect = (fileName: string) => {
        setSelectedFile(fileName);
        if (!openFiles.includes(fileName)) {
            setOpenFiles([...openFiles, fileName]);
        }
        setFileTreeMode(FileTreeModes.Collapsed);
    };

    const handleCloseFile = (fileName: string) => {
        const newOpenFiles = openFiles.filter(f => f !== fileName);
        setOpenFiles(newOpenFiles);
        if (selectedFile === fileName) {
            setSelectedFile(newOpenFiles[0] || null);
        }
    };

    return (
        <div className="mobile-workspace">
            {/* Top Bar */}
            <div className="mw-topbar">
                <button 
                    className="mw-menu-btn"
                    onClick={() => setFileTreeMode(fileTreeMode === FileTreeModes.Expanded ? FileTreeModes.Collapsed : FileTreeModes.Expanded)}
                >
                    <MenuIcon />
                </button>
                <div className="mw-project-name">my-project</div>
                <button className="mw-action-btn">
                    <PlayIcon />
                </button>
            </div>

            {/* File Tree Overlay (Mobile) */}
            {fileTreeMode === FileTreeModes.Expanded && (
                <div className="mw-filetree-overlay" onClick={() => setFileTreeMode(FileTreeModes.Collapsed)}>
                    <div className="mw-filetree" onClick={e => e.stopPropagation()}>
                        <div className="mw-filetree-header">
                            <span>Explorer</span>
                            <button onClick={() => setFileTreeMode(FileTreeModes.Collapsed)}>×</button>
                        </div>
                        <div className="mw-filetree-content">
                            <FileTree nodes={mockFileTree} onSelect={handleFileSelect} selectedFile={selectedFile} />
                        </div>
                    </div>
                </div>
            )}

            {/* Open Files Tabs */}
            <div className="mw-tabs">
                {openFiles.map(file => (
                    <div 
                        key={file}
                        className={`mw-tab ${selectedFile === file ? 'active' : ''}`}
                        onClick={() => setSelectedFile(file)}
                    >
                        <span>{file}</span>
                        <button 
                            className="mw-tab-close"
                            onClick={(e) => { e.stopPropagation(); handleCloseFile(file); }}
                        >
                            ×
                        </button>
                    </div>
                ))}
            </div>

            {/* Editor Area */}
            <div className="mw-editor">
                {selectedFile ? (
                    <div className="mw-editor-content">
                        <div className="mw-line"><span className="mw-line-num">1</span><span className="mw-keyword">import</span> {'{'} useState {'}'} <span className="mw-keyword">from</span> <span className="mw-string">'react'</span>;</div>
                        <div className="mw-line"><span className="mw-line-num">2</span></div>
                        <div className="mw-line"><span className="mw-line-num">3</span><span className="mw-keyword">function</span> <span className="mw-function">App</span>() {'{'}</div>
                        <div className="mw-line"><span className="mw-line-num">4</span>  <span className="mw-keyword">const</span> [count, setCount] = <span className="mw-function">useState</span>(0);</div>
                        <div className="mw-line"><span className="mw-line-num">5</span></div>
                        <div className="mw-line"><span className="mw-line-num">6</span>  <span className="mw-keyword">return</span> (</div>
                        <div className="mw-line"><span className="mw-line-num">7</span>    {'<'}<span className="mw-tag">div</span>{'>'}</div>
                        <div className="mw-line"><span className="mw-line-num">8</span>      {'<'}<span className="mw-tag">h1</span>{'>'}Hello World{'</'}<span className="mw-tag">h1</span>{'>'}</div>
                        <div className="mw-line"><span className="mw-line-num">9</span>      {'<'}<span className="mw-tag">button</span> <span className="mw-attr">onClick</span>={'{'}() ={'>'} setCount(c ={'>'} c + 1){'}'}{'>'}Count: {'{'}count{'}'}{'</'}<span className="mw-tag">button</span>{'>'}</div>
                        <div className="mw-line"><span className="mw-line-num">10</span>    {'</'}<span className="mw-tag">div</span>{'>'}</div>
                        <div className="mw-line"><span className="mw-line-num">11</span>  );</div>
                        <div className="mw-line"><span className="mw-line-num">12</span>{'}'}</div>
                        <div className="mw-line"><span className="mw-line-num">13</span></div>
                        <div className="mw-line"><span className="mw-line-num">14</span><span className="mw-keyword">export default</span> App;</div>
                    </div>
                ) : (
                    <div className="mw-editor-empty">
                        <p>No file selected</p>
                        <p>Tap the menu to open a file</p>
                    </div>
                )}
            </div>

            {/* Bottom Navigation */}
            <div className="mw-bottomnav">
                <button 
                    className={`mw-nav-btn ${activeTab === 'files' ? 'active' : ''}`}
                    onClick={() => { setActiveTab('files'); setFileTreeMode(FileTreeModes.Expanded); }}
                >
                    <FilesIcon />
                    <span>Files</span>
                </button>
                <button 
                    className={`mw-nav-btn ${activeTab === 'search' ? 'active' : ''}`}
                    onClick={() => setActiveTab('search')}
                >
                    <SearchIcon />
                    <span>Search</span>
                </button>
                <button 
                    className={`mw-nav-btn ${activeTab === 'terminal' ? 'active' : ''}`}
                    onClick={() => setActiveTab('terminal')}
                >
                    <TerminalIcon />
                    <span>Terminal</span>
                </button>
                <button 
                    className={`mw-nav-btn ${activeTab === 'settings' ? 'active' : ''}`}
                    onClick={() => setActiveTab('settings')}
                >
                    <SettingsIcon />
                    <span>Settings</span>
                </button>
            </div>
        </div>
    );
}

interface FileTreeProps {
    nodes: FileNode[];
    onSelect: (name: string) => void;
    selectedFile: string | null;
    depth?: number;
}

function FileTree({ nodes, onSelect, selectedFile, depth = 0 }: FileTreeProps) {
    const [expanded, setExpanded] = useState<Record<string, boolean>>({});

    return (
        <div className="mw-tree">
            {nodes.map(node => (
                <div key={node.name}>
                    <div 
                        className={`mw-tree-item ${node.type === 'file' && selectedFile === node.name ? 'selected' : ''}`}
                        style={{ paddingLeft: `${depth * 16 + 8}px` }}
                        onClick={() => {
                            if (node.type === 'folder') {
                                setExpanded(prev => ({ ...prev, [node.name]: !prev[node.name] }));
                            } else {
                                onSelect(node.name);
                            }
                        }}
                    >
                        {node.type === 'folder' ? (
                            <span className="mw-tree-icon">{expanded[node.name] ? '📂' : '📁'}</span>
                        ) : (
                            <span className="mw-tree-icon">📄</span>
                        )}
                        <span>{node.name}</span>
                    </div>
                    {node.type === 'folder' && expanded[node.name] && node.children && (
                        <FileTree 
                            nodes={node.children} 
                            onSelect={onSelect} 
                            selectedFile={selectedFile}
                            depth={depth + 1}
                        />
                    )}
                </div>
            ))}
        </div>
    );
}

// Icons
function MenuIcon() {
    return (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
        </svg>
    );
}

function PlayIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <polygon points="5 3 19 12 5 21 5 3"></polygon>
        </svg>
    );
}

function FilesIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
    );
}

function SearchIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
    );
}

function TerminalIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="4 17 10 11 4 5"></polyline>
            <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
    );
}

function SettingsIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
    );
}
