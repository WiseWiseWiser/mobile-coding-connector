import { useState, useEffect, useCallback, useMemo } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import type { ProjectInfo, WorktreeConfig } from '../../../api/projects';
import { updateProject } from '../../../api/projects';
import { 
    listWorktrees, 
    createWorktree, 
    removeWorktree, 
    moveWorktree,
    getGitBranches,
    type Worktree,
    type GitBranch 
} from '../../../api/review';
import { ServerFileBrowser, SelectModes } from './ServerFileBrowser';
import { CustomSelect } from './CustomSelect';
import { PlusIcon } from '../../../pure-view/icons/PlusIcon';
import { TrashIcon } from '../../../pure-view/icons/TrashIcon';
import { RefreshIcon } from '../../../pure-view/icons/RefreshIcon';
import { FolderMoveIcon } from '../../../pure-view/icons/FolderMoveIcon';
import { CheckCircleIcon } from '../../../pure-view/icons/CheckCircleIcon';

interface WorktreesSectionProps {
    project: ProjectInfo;
}

function getWorktreeId(worktree: Worktree, project: ProjectInfo): number {
    // Main worktree always has ID 0
    if (worktree.isMain) return 0;
    
    // Look up ID from project worktree config
    const worktreeConfig = project.worktrees || {};
    for (const [id, config] of Object.entries(worktreeConfig)) {
        if (config.path === worktree.path) {
            return parseInt(id, 10);
        }
    }
    
    // Fallback: return 0 if no ID found (shouldn't happen after initial load)
    return 0;
}

export function WorktreesSection({ project }: WorktreesSectionProps) {
    const navigate = useNavigate();
    const params = useParams<{ projectName?: string }>();
    const location = useLocation();
    
    // Parse current worktree ID from URL
    const currentWorktreeId = useMemo(() => {
        const fullProjectName = params.projectName || '';
        const separatorIndex = fullProjectName.lastIndexOf('~');
        if (separatorIndex === -1) {
            return 0; // Root worktree
        }
        const worktreeIdStr = fullProjectName.substring(separatorIndex + 1);
        const worktreeId = parseInt(worktreeIdStr, 10);
        return isNaN(worktreeId) ? 0 : worktreeId;
    }, [params.projectName]);
    
    const [worktrees, setWorktrees] = useState<Worktree[]>([]);
    const [branches, setBranches] = useState<GitBranch[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    
    // Add worktree form state
    const [showAddForm, setShowAddForm] = useState(false);
    const [selectedBranch, setSelectedBranch] = useState('');
    const [worktreePath, setWorktreePath] = useState('');
    const [showFileBrowser, setShowFileBrowser] = useState(false);
    const [creating, setCreating] = useState(false);
    
    // Move worktree state
    const [movingWorktree, setMovingWorktree] = useState<Worktree | null>(null);
    const [newMovePath, setNewMovePath] = useState('');
    const [showMoveFileBrowser, setShowMoveFileBrowser] = useState(false);
    const [moving, setMoving] = useState(false);

    // Auto-assign IDs to worktrees that don't have config entries
    const autoAssignWorktreeIds = useCallback(async (wtList: Worktree[]) => {
        const worktreeConfig: WorktreeConfig = project.worktrees || {};
        let needsUpdate = false;
        const newConfig = { ...worktreeConfig };
        
        // Find all existing IDs
        const existingIds = new Set([0]); // 0 is reserved for main worktree
        for (const id of Object.keys(worktreeConfig)) {
            existingIds.add(parseInt(id, 10));
        }
        
        // Find worktrees without IDs
        for (const wt of wtList) {
            if (wt.isMain) continue; // Main worktree doesn't need an ID
            
            // Check if this worktree already has an ID
            let hasId = false;
            for (const [, config] of Object.entries(worktreeConfig)) {
                if (config.path === wt.path) {
                    hasId = true;
                    break;
                }
            }
            
            if (!hasId) {
                // Assign new ID
                let newId = 1;
                while (existingIds.has(newId)) {
                    newId++;
                }
                existingIds.add(newId);
                newConfig[newId] = { path: wt.path, branch: wt.branch };
                needsUpdate = true;
            }
        }
        
        if (needsUpdate) {
            try {
                await updateProject(project.id, { worktrees: newConfig });
            } catch (err) {
                console.error('Failed to auto-assign worktree IDs:', err);
            }
        }
    }, [project.id, project.worktrees]);

    const loadWorktrees = useCallback(async () => {
        if (!project.dir || !project.dir_exists) return;
        setLoading(true);
        setError('');
        try {
            const [wtList, branchList] = await Promise.all([
                listWorktrees(project.dir),
                getGitBranches(project.dir),
            ]);
            
            // Auto-assign IDs to worktrees without config
            await autoAssignWorktreeIds(wtList);
            
            setWorktrees(wtList);
            setBranches(branchList);
            // Set default branch to current branch
            const currentBranch = branchList.find(b => b.isCurrent);
            if (currentBranch && !selectedBranch) {
                setSelectedBranch(currentBranch.name);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load worktrees');
        } finally {
            setLoading(false);
        }
    }, [project.dir, project.dir_exists, selectedBranch, autoAssignWorktreeIds]);

    useEffect(() => {
        loadWorktrees();
    }, [loadWorktrees]);

    const handleCreateWorktree = async () => {
        if (!selectedBranch || !worktreePath) {
            setError('Please select a branch and specify a path');
            return;
        }
        setCreating(true);
        setError('');
        try {
            await createWorktree(selectedBranch, worktreePath, project.dir);
            setShowAddForm(false);
            setWorktreePath('');
            loadWorktrees();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to create worktree');
        } finally {
            setCreating(false);
        }
    };

    const handleRemoveWorktree = async (worktree: Worktree, force: boolean = false) => {
        if (!confirm(`Remove worktree at "${worktree.path}"? ${force ? '(force)' : ''}`)) {
            return;
        }
        setError('');
        try {
            await removeWorktree(worktree.path, force, project.dir);
            loadWorktrees();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to remove worktree');
        }
    };

    const handleMoveWorktree = async () => {
        if (!movingWorktree || !newMovePath) return;
        setMoving(true);
        setError('');
        try {
            await moveWorktree(movingWorktree.path, newMovePath, project.dir);
            setMovingWorktree(null);
            setNewMovePath('');
            loadWorktrees();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to move worktree');
        } finally {
            setMoving(false);
        }
    };

    const branchOptions = [
        { value: '', label: '-- Select a branch --' },
        ...branches.map(b => ({
            value: b.name,
            label: b.name,
            sublabel: b.isCurrent ? '(current)' : '',
        })),
    ];

    if (!project.dir_exists) {
        return null;
    }

    return (
        <div style={{ padding: '16px', marginTop: 16, borderTop: '1px solid #334155' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                <div style={{ fontSize: '15px', fontWeight: 600, color: '#e2e8f0' }}>
                    Worktrees ({worktrees.length})
                </div>
                <button
                    onClick={loadWorktrees}
                    disabled={loading}
                    style={{ padding: '4px 8px', background: 'transparent', border: 'none', color: '#94a3b8', cursor: 'pointer', fontSize: '12px' }}
                >
                    <RefreshIcon />
                </button>
            </div>

            {error && (
                <div style={{ marginBottom: 12, padding: '10px 14px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 8, color: '#fca5a5', fontSize: '13px' }}>
                    {error}
                </div>
            )}

            {loading ? (
                <div style={{ fontSize: '13px', color: '#64748b', padding: '12px 0' }}>Loading worktrees...</div>
            ) : worktrees.length > 0 ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 16 }}>
                    {worktrees.map(wt => {
                        const worktreeId = getWorktreeId(wt, project);
                        const isSelected = worktreeId === currentWorktreeId;
                        return (
                        <div key={wt.path} style={{
                            padding: '12px 14px',
                            background: isSelected 
                                ? 'rgba(34, 197, 94, 0.15)' 
                                : wt.isMain 
                                    ? 'rgba(96, 165, 250, 0.08)' 
                                    : 'rgba(30, 41, 59, 0.5)',
                            border: `1px solid ${isSelected 
                                ? 'rgba(34, 197, 94, 0.4)' 
                                : wt.isMain 
                                    ? 'rgba(96, 165, 250, 0.2)' 
                                    : '#334155'}`,
                            borderRadius: 8,
                        }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
                                    <span style={{ fontSize: '13px', fontWeight: 500, color: '#e2e8f0', wordBreak: 'break-all' }}>
                                        {wt.path}
                                    </span>
                                    {wt.isMain && (
                                        <span style={{ padding: '2px 6px', background: 'rgba(96, 165, 250, 0.2)', color: '#93c5fd', borderRadius: 4, fontSize: '10px' }}>
                                            main
                                        </span>
                                    )}
                                </div>
                            </div>
                            <div style={{ fontSize: '12px', color: '#64748b', marginBottom: 8 }}>
                                Branch: {wt.branch}
                            </div>
                            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                                <button
                                    onClick={() => {
                                        const worktreeId = getWorktreeId(wt, project);
                                        const projectName = project.name;
                                        const fullProjectName = worktreeId === 0 
                                            ? projectName 
                                            : `${projectName}~${worktreeId}`;
                                        // Extract the current route path after the project name
                                        // e.g., from /project/opencode~1/chat -> /chat
                                        const currentPath = location.pathname;
                                        const projectPrefixMatch = currentPath.match(/\/project\/[^/]+/);
                                        let routePath = '/home';
                                        if (projectPrefixMatch) {
                                            routePath = currentPath.substring(projectPrefixMatch[0].length) || '/home';
                                        }
                                        navigate(`/project/${fullProjectName}${routePath}`);
                                    }}
                                    style={{ display: 'flex', alignItems: 'center', gap: 4, padding: '4px 8px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 6, fontSize: '11px', cursor: 'pointer' }}
                                >
                                    <CheckCircleIcon />
                                    Select
                                </button>
                                {!wt.isMain && (
                                <div style={{ display: 'flex', gap: 6 }}>
                                    <button
                                        onClick={() => setMovingWorktree(wt)}
                                        style={{ display: 'flex', alignItems: 'center', gap: 4, padding: '4px 8px', background: '#1e293b', color: '#94a3b8', border: '1px solid #334155', borderRadius: 6, fontSize: '11px', cursor: 'pointer' }}
                                    >
                                        <FolderMoveIcon />
                                        Move
                                    </button>
                                    <button
                                        onClick={() => handleRemoveWorktree(wt, false)}
                                        style={{ display: 'flex', alignItems: 'center', gap: 4, padding: '4px 8px', background: '#1e293b', color: '#f87171', border: '1px solid #334155', borderRadius: 6, fontSize: '11px', cursor: 'pointer' }}
                                    >
                                        <TrashIcon />
                                        Remove
                                    </button>
                                </div>
                            )}
                            </div>
                        </div>
                    );
                    })}
                </div>
            ) : (
                <div style={{ fontSize: '13px', color: '#64748b', marginBottom: 16, padding: '8px 0' }}>
                    No worktrees found. Create one to work on multiple branches simultaneously.
                </div>
            )}

            {/* Add Worktree Form */}
            {showAddForm ? (
                <div style={{
                    padding: '16px',
                    background: 'rgba(30, 41, 59, 0.8)',
                    border: '1px solid #334155',
                    borderRadius: 8,
                }}>
                    <div style={{ fontSize: '14px', fontWeight: 600, color: '#e2e8f0', marginBottom: 12 }}>
                        Add New Worktree
                    </div>
                    
                    <div style={{ marginBottom: 12 }}>
                        <label style={{ fontSize: '12px', color: '#94a3b8', display: 'block', marginBottom: 4 }}>
                            Branch *
                        </label>
                        <CustomSelect
                            value={selectedBranch}
                            onChange={setSelectedBranch}
                            placeholder="-- Select a branch --"
                            options={branchOptions}
                        />
                    </div>
                    
                    <div style={{ marginBottom: 12 }}>
                        <label style={{ fontSize: '12px', color: '#94a3b8', display: 'block', marginBottom: 4 }}>
                            Path *
                        </label>
                        <div style={{ display: 'flex', gap: 8 }}>
                            <input
                                type="text"
                                value={worktreePath}
                                onChange={e => setWorktreePath(e.target.value)}
                                placeholder="/path/to/worktree"
                                style={{ flex: 1, padding: '8px 10px', background: '#0f172a', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: '13px' }}
                            />
                            <button
                                onClick={() => setShowFileBrowser(true)}
                                style={{ padding: '8px 12px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 6, fontSize: '12px', cursor: 'pointer' }}
                            >
                                Browse...
                            </button>
                        </div>
                    </div>
                    
                    {showFileBrowser && (
                        <div style={{
                            marginBottom: 12,
                            padding: '12px',
                            background: '#0f172a',
                            border: '1px solid #334155',
                            borderRadius: 8,
                            maxHeight: '300px',
                            overflow: 'auto',
                        }}>
                            <ServerFileBrowser
                                selectMode={SelectModes.Dir}
                                onSelect={(path) => {
                                    if (path) {
                                        setWorktreePath(path);
                                        setShowFileBrowser(false);
                                    }
                                }}
                                onDirectoryChange={(path) => {
                                    setWorktreePath(path);
                                }}
                                initialDir={project.dir}
                            />
                        </div>
                    )}
                    
                    <div style={{ display: 'flex', gap: 8 }}>
                        <button
                            onClick={handleCreateWorktree}
                            disabled={creating || !selectedBranch || !worktreePath}
                            style={{ flex: 1, padding: '8px 12px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 6, fontSize: '13px', cursor: 'pointer', opacity: creating || !selectedBranch || !worktreePath ? 0.5 : 1 }}
                        >
                            {creating ? 'Creating...' : 'Create'}
                        </button>
                        <button
                            onClick={() => {
                                setShowAddForm(false);
                                setShowFileBrowser(false);
                                setWorktreePath('');
                                setSelectedBranch('');
                                setError('');
                            }}
                            style={{ padding: '8px 12px', background: '#1e293b', color: '#94a3b8', border: '1px solid #334155', borderRadius: 6, fontSize: '13px', cursor: 'pointer' }}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            ) : (
                <button
                    onClick={() => setShowAddForm(true)}
                    style={{ width: '100%', padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}
                >
                    <PlusIcon />
                    Add Worktree
                </button>
            )}

            {/* Move Worktree Modal */}
            {movingWorktree && (
                <div style={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    background: 'rgba(0, 0, 0, 0.7)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    zIndex: 1000,
                    padding: '16px',
                }}>
                    <div style={{
                        background: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: 12,
                        padding: '20px',
                        width: '100%',
                        maxWidth: '500px',
                    }}>
                        <h3 style={{ margin: '0 0 16px 0', fontSize: '16px', color: '#e2e8f0' }}>
                            Move Worktree
                        </h3>
                        <p style={{ fontSize: '13px', color: '#94a3b8', marginBottom: 12 }}>
                            Moving from: <code style={{ color: '#e2e8f0' }}>{movingWorktree.path}</code>
                        </p>
                        <div style={{ marginBottom: 12 }}>
                            <label style={{ fontSize: '12px', color: '#94a3b8', display: 'block', marginBottom: 4 }}>
                                New Path *
                            </label>
                            <div style={{ display: 'flex', gap: 8 }}>
                                <input
                                    type="text"
                                    value={newMovePath}
                                    onChange={e => setNewMovePath(e.target.value)}
                                    placeholder="/new/path/to/worktree"
                                    style={{ flex: 1, padding: '8px 10px', background: '#0f172a', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: '13px' }}
                                />
                                <button
                                    onClick={() => setShowMoveFileBrowser(true)}
                                    style={{ padding: '8px 12px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 6, fontSize: '12px', cursor: 'pointer' }}
                                >
                                    Browse...
                                </button>
                            </div>
                        </div>
                        {showMoveFileBrowser && (
                            <div style={{
                                marginBottom: 12,
                                padding: '12px',
                                background: '#0f172a',
                                border: '1px solid #334155',
                                borderRadius: 8,
                                maxHeight: '250px',
                                overflow: 'auto',
                            }}>
                                <ServerFileBrowser
                                    selectMode={SelectModes.Dir}
                                    onSelect={(path) => {
                                        if (path) {
                                            setNewMovePath(path);
                                            setShowMoveFileBrowser(false);
                                        }
                                    }}
                                    onDirectoryChange={(path) => {
                                        setNewMovePath(path);
                                    }}
                                    initialDir={project.dir}
                                />
                            </div>
                        )}
                        <div style={{ display: 'flex', gap: 8 }}>
                            <button
                                onClick={handleMoveWorktree}
                                disabled={moving || !newMovePath}
                                style={{ flex: 1, padding: '8px 12px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 6, fontSize: '13px', cursor: 'pointer', opacity: moving || !newMovePath ? 0.5 : 1 }}
                            >
                                {moving ? 'Moving...' : 'Move'}
                            </button>
                            <button
                                onClick={() => {
                                    setMovingWorktree(null);
                                    setNewMovePath('');
                                    setShowMoveFileBrowser(false);
                                }}
                                style={{ padding: '8px 12px', background: '#1e293b', color: '#94a3b8', border: '1px solid #334155', borderRadius: 6, fontSize: '13px', cursor: 'pointer' }}
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
