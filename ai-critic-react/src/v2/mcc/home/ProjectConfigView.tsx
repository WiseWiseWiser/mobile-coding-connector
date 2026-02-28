import { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useV2Context } from '../../V2Context';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { useSSHKeyValidation } from '../../../hooks/useSSHKeyValidation';
import { NavTabs } from '../types';
import { updateProject, runGitOp, GitOps, addProject } from '../../../api/projects';
import type { GitOp } from '../../../api/projects';
import { cloneRepo } from '../../../api/auth';
import { loadSSHKeys } from './settings/gitStorage';
import type { SSHKey } from './settings/gitStorage';
import { encryptWithServerKey } from './crypto';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import { KeyIcon, PlusIcon } from '../../icons';
import { CustomSelect } from './CustomSelect';
import { GitPushSection } from '../files/GitPushSection';
import { SSHKeyRequiredHint } from '../components/SSHKeyRequiredHint';
import { ProjectTodos } from './ProjectTodos';
import { WorktreesSection } from './WorktreesSection';
import { ProjectReadmeEditor } from './ProjectReadmeEditor';
import { ErrorBoundary } from '../../../components/ErrorBoundary';
import './ProjectConfigView.css';

export function ProjectConfigView() {
    const { projectName } = useParams<{ projectName: string }>();
    const navigate = useNavigate();
    const { projectsList, fetchProjects } = useV2Context();
    // When going back from project config, always go to /home (project list)
    const { goBack } = useTabHistory(NavTabs.Home, { defaultBackPath: '/home' });

    const project = projectsList.find(p => p.name === projectName);
    const sshValidation = useSSHKeyValidation(project);
    const [sshKeys, setSshKeys] = useState<SSHKey[]>([]);
    const [selectedKeyId, setSelectedKeyId] = useState('');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    // Git operation state - shared streaming state for fetch/pull/clone
    const [gitState, gitControls] = useStreamingAction((result) => {
        if (result.ok) {
            fetchProjects();
        }
    });

    const subProjects = useMemo(() =>
        projectsList.filter(p => p.parent_id === project?.id),
        [projectsList, project?.id]
    );
    const [showAddSubProject, setShowAddSubProject] = useState(false);
    const [newSubProjectName, setNewSubProjectName] = useState('');
    const [newSubProjectDir, setNewSubProjectDir] = useState('');
    const [addingSubProject, setAddingSubProject] = useState(false);
    const [subProjectError, setSubProjectError] = useState('');

    useEffect(() => {
        const keys = loadSSHKeys();
        setSshKeys(keys);
        if (project?.ssh_key_id) {
            setSelectedKeyId(project.ssh_key_id);
        }
    }, [project?.ssh_key_id]);

    useEffect(() => {
        if (project?.dir && !newSubProjectDir) {
            setNewSubProjectDir(project.dir);
        }
    }, [project?.dir, newSubProjectDir]);

    if (!project) {
        return (
            <div className="mcc-workspace-list">
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={goBack}>&larr;</button>
                    <h2>Project Not Found</h2>
                </div>
                <div className="mcc-ports-empty">Project not found.</div>
            </div>
        );
    }

    const handleSave = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateProject(project.id, {
                ssh_key_id: selectedKeyId || null,
                use_ssh: !!selectedKeyId,
            });
            fetchProjects();
            setSuccess('SSH key updated successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update project');
        } finally {
            setSaving(false);
        }
    };

    const handleUnset = async () => {
        setSaving(true);
        setError('');
        setSuccess('');
        try {
            await updateProject(project.id, {
                ssh_key_id: null,
                use_ssh: false,
            });
            setSelectedKeyId('');
            fetchProjects();
            setSuccess('SSH key removed');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update project');
        } finally {
            setSaving(false);
        }
    };

    const prepareSSHKey = async (): Promise<string | undefined> => {
        if (!project.use_ssh || !project.ssh_key_id) return undefined;
        const key = sshKeys.find(k => k.id === project.ssh_key_id);
        if (!key) return undefined;
        return encryptWithServerKey(key.privateKey);
    };

    const handleGitOp = (op: GitOp) => {
        gitControls.run(async () => {
            const sshKey = await prepareSSHKey();
            return runGitOp(op, {
                project_id: project.id,
                ssh_key: sshKey,
            });
        });
    };

    const handleClone = () => {
        if (!project.repo_url) return;
        gitControls.run(async () => {
            const body: Record<string, unknown> = {
                repo_url: project.repo_url,
                target_dir: project.dir,
            };

            if (project.use_ssh && project.ssh_key_id) {
                const sshKey = await prepareSSHKey();
                if (sshKey) {
                    body.ssh_key = sshKey;
                    body.use_ssh = true;
                    body.ssh_key_id = project.ssh_key_id;
                }
            }

            return cloneRepo(body);
        });
    };

    const currentKey = sshKeys.find(k => k.id === project.ssh_key_id);

    const isSubProject = !!project.parent_id;

    const handleAddSubProject = async () => {
        if (!newSubProjectDir.trim()) {
            setSubProjectError('Directory is required');
            return;
        }
        setAddingSubProject(true);
        setSubProjectError('');
        try {
            await addProject({
                name: newSubProjectName.trim() || undefined,
                dir: newSubProjectDir.trim(),
                parent_id: project.id,
            });
            fetchProjects();
            setShowAddSubProject(false);
            setNewSubProjectName('');
            setNewSubProjectDir(project.dir);
        } catch (err) {
            setSubProjectError(err instanceof Error ? err.message : 'Failed to add sub-project');
        } finally {
            setAddingSubProject(false);
        }
    };

    const handleRemoveFromParent = async (subProjectId: string) => {
        try {
            await updateProject(subProjectId, { parent_id: null });
            fetchProjects();
        } catch (err) {
            console.error('Failed to unlink sub-project:', err);
        }
    };

    return (
        <div className="mcc-workspace-list">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={goBack}>&larr;</button>
                <h2>Configure Project</h2>
            </div>

            {/* Project info */}
            <div style={{ padding: '0 16px' }}>
                <div style={{ fontSize: '16px', fontWeight: 600, color: '#e2e8f0', marginBottom: 4 }}>
                    {project.name}
                </div>
                <div style={{ fontSize: '13px', color: '#94a3b8', marginBottom: 2 }}>
                    {project.dir}
                </div>
                <div style={{ fontSize: '13px', color: '#64748b' }}>
                    {project.repo_url}
                </div>
            </div>

            {/* Git Operations Section */}
            <div style={{ padding: '16px', marginTop: 16 }}>
                <div style={{ fontSize: '15px', fontWeight: 600, color: '#e2e8f0', marginBottom: 12 }}>
                    Git Operations
                </div>

                {!project.dir_exists ? (
                    /* Directory doesn't exist - show Clone button */
                    <div style={{ marginBottom: 12 }}>
                        {project.repo_url ? (
                            <>
                                <div style={{ fontSize: '13px', color: '#f59e0b', marginBottom: 10, padding: '8px 12px', background: 'rgba(245, 158, 11, 0.1)', border: '1px solid rgba(245, 158, 11, 0.2)', borderRadius: 8 }}>
                                    Directory does not exist on filesystem. Clone the repository to create it.
                                </div>
                                {!sshValidation.hasSSHKey && (
                                    <SSHKeyRequiredHint message={sshValidation.message} />
                                )}
                                <button
                                    className="mcc-port-action-btn"
                                    onClick={handleClone}
                                    disabled={gitState.running || !sshValidation.hasSSHKey}
                                    style={{ width: '100%', padding: '10px 16px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer', opacity: !sshValidation.hasSSHKey ? 0.5 : 1 }}
                                >
                                    {gitState.running ? 'Cloning...' : 'Git Clone'}
                                </button>
                            </>
                        ) : (
                            <div style={{ fontSize: '13px', color: '#94a3b8', padding: '8px 12px', background: 'rgba(148, 163, 184, 0.05)', border: '1px solid #334155', borderRadius: 8 }}>
                                Directory does not exist and no repository URL is configured.
                            </div>
                        )}
                    </div>
                ) : (
                    /* Directory exists - show Fetch/Pull buttons on a row */
                    <>
                        {!sshValidation.hasSSHKey && (
                            <SSHKeyRequiredHint message={sshValidation.message} />
                        )}
                        <div style={{ display: 'flex', gap: 10, marginBottom: 12 }}>
                            <button
                                className="mcc-port-action-btn"
                                onClick={() => handleGitOp(GitOps.Fetch)}
                                disabled={gitState.running || !sshValidation.hasSSHKey}
                                style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer', opacity: !sshValidation.hasSSHKey ? 0.5 : 1 }}
                            >
                                {gitState.running ? '...' : 'Git Fetch'}
                            </button>
                            <button
                                className="mcc-port-action-btn"
                                onClick={() => handleGitOp(GitOps.Pull)}
                                disabled={gitState.running || !sshValidation.hasSSHKey}
                                style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer', opacity: !sshValidation.hasSSHKey ? 0.5 : 1 }}
                            >
                                {gitState.running ? '...' : 'Git Pull'}
                            </button>
                        </div>
                    </>
                )}

                {/* Shared streaming logs area */}
                <StreamingLogs
                    state={gitState}
                    pendingMessage="Running..."
                    maxHeight={200}
                />

                {/* Git Push section - only when directory exists */}
                {project.dir_exists && (
                    <div style={{ marginTop: 16 }}>
                        <GitPushSection projectDir={project.dir} sshKeyId={project.ssh_key_id} />
                    </div>
                )}
            </div>

            {/* SSH Key Section */}
            <div style={{ padding: '16px', marginTop: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
                    <KeyIcon />
                    <span style={{ fontSize: '15px', fontWeight: 600, color: '#e2e8f0' }}>SSH Key for Git Operations</span>
                </div>
                <div style={{ fontSize: '13px', color: '#94a3b8', marginBottom: 12 }}>
                    Select an SSH private key to use for fetch and push operations. Keys are managed in Git Settings.
                </div>

                {/* Current key display */}
                {currentKey ? (
                    <div style={{
                        padding: '10px 14px',
                        background: 'rgba(96, 165, 250, 0.08)',
                        border: '1px solid rgba(96, 165, 250, 0.2)',
                        borderRadius: 8,
                        marginBottom: 12,
                        fontSize: '13px',
                        color: '#93c5fd',
                    }}>
                        Current: <strong>{currentKey.name}</strong> ({currentKey.host})
                    </div>
                ) : (
                    <div style={{
                        padding: '10px 14px',
                        background: 'rgba(148, 163, 184, 0.05)',
                        border: '1px solid #334155',
                        borderRadius: 8,
                        marginBottom: 12,
                        fontSize: '13px',
                        color: '#64748b',
                    }}>
                        No SSH key configured
                    </div>
                )}

                {sshKeys.length === 0 ? (
                    <div style={{ fontSize: '13px', color: '#94a3b8' }}>
                        No SSH keys available.{' '}
                        <button
                            style={{ background: 'none', border: 'none', color: '#60a5fa', cursor: 'pointer', fontSize: '13px', textDecoration: 'underline', padding: 0 }}
                            onClick={() => navigate('/home/settings/git')}
                        >
                            Add one in Git Settings
                        </button>
                    </div>
                ) : (
                    <>
                        <CustomSelect
                            value={selectedKeyId}
                            onChange={setSelectedKeyId}
                            placeholder="-- No SSH key --"
                            options={[
                                { value: '', label: '-- No SSH key --' },
                                ...sshKeys.map(k => ({
                                    value: k.id,
                                    label: k.name,
                                    sublabel: k.host,
                                })),
                            ]}
                        />

                        <div style={{ display: 'flex', gap: 10 }}>
                            <button
                                className="mcc-port-action-btn"
                                onClick={handleSave}
                                disabled={saving}
                                style={{ flex: 1, padding: '10px 16px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
                            >
                                {saving ? 'Saving...' : 'Save'}
                            </button>
                            {project.ssh_key_id && (
                                <button
                                    className="mcc-port-action-btn"
                                    onClick={handleUnset}
                                    disabled={saving}
                                    style={{ padding: '10px 16px', background: '#1e293b', color: '#f87171', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', cursor: 'pointer' }}
                                >
                                    Unset Key
                                </button>
                            )}
                        </div>
                    </>
                )}

                {error && (
                    <div style={{ marginTop: 12, padding: '10px 14px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 8, color: '#fca5a5', fontSize: '13px' }}>
                        {error}
                    </div>
                )}
                {success && (
                    <div style={{ marginTop: 12, padding: '10px 14px', background: 'rgba(34, 197, 94, 0.1)', border: '1px solid rgba(34, 197, 94, 0.3)', borderRadius: 8, color: '#86efac', fontSize: '13px' }}>
                        {success}
                    </div>
                )}
            </div>

            {/* Sub Projects Section - only for root projects */}
            {!isSubProject && (
                <div style={{ padding: '16px', marginTop: 16 }}>
                    <div style={{ fontSize: '15px', fontWeight: 600, color: '#e2e8f0', marginBottom: 12 }}>
                        Sub Projects ({subProjects.length})
                    </div>
                    
                    {subProjects.length > 0 ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
                            {subProjects.map(sp => (
                                <div key={sp.id} style={{
                                    padding: '10px 14px',
                                    background: 'rgba(30, 41, 59, 0.5)',
                                    border: '1px solid #334155',
                                    borderRadius: 8,
                                    display: 'flex',
                                    flexDirection: 'column',
                                    gap: 4,
                                }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                        <span style={{ fontSize: '14px', fontWeight: 500, color: '#e2e8f0' }}>{sp.name}</span>
                                        <div style={{ display: 'flex', gap: 8 }}>
                                            <button
                                                onClick={() => navigate(`/project/${encodeURIComponent(sp.name)}`)}
                                                style={{ padding: '4px 10px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 6, fontSize: '12px', cursor: 'pointer' }}
                                            >
                                                Open
                                            </button>
                                            <button
                                                onClick={() => handleRemoveFromParent(sp.id)}
                                                style={{ padding: '4px 10px', background: '#1e293b', color: '#f87171', border: '1px solid #334155', borderRadius: 6, fontSize: '12px', cursor: 'pointer' }}
                                            >
                                                Unlink
                                            </button>
                                        </div>
                                    </div>
                                    <div style={{ fontSize: '12px', color: '#64748b' }}>{sp.dir}</div>
                                    {!sp.dir_exists && <span style={{ fontSize: '11px', color: '#f59e0b' }}>Not cloned</span>}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div style={{ fontSize: '13px', color: '#64748b', marginBottom: 12 }}>
                            No sub-projects yet.
                        </div>
                    )}

                    {showAddSubProject ? (
                        <div style={{
                            padding: '12px',
                            background: 'rgba(30, 41, 59, 0.5)',
                            border: '1px solid #334155',
                            borderRadius: 8,
                        }}>
                            <div style={{ marginBottom: 10 }}>
                                <label style={{ fontSize: '12px', color: '#94a3b8', display: 'block', marginBottom: 4 }}>Name (optional)</label>
                                <input
                                    type="text"
                                    value={newSubProjectName}
                                    onChange={e => setNewSubProjectName(e.target.value)}
                                    placeholder="Uses directory name if empty"
                                    style={{ width: '100%', padding: '8px 10px', background: '#0f172a', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: '13px' }}
                                />
                            </div>
                            <div style={{ marginBottom: 10 }}>
                                <label style={{ fontSize: '12px', color: '#94a3b8', display: 'block', marginBottom: 4 }}>Directory *</label>
                                <input
                                    type="text"
                                    value={newSubProjectDir}
                                    onChange={e => setNewSubProjectDir(e.target.value)}
                                    placeholder="/path/to/project"
                                    style={{ width: '100%', padding: '8px 10px', background: '#0f172a', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: '13px' }}
                                />
                            </div>
                            {subProjectError && (
                                <div style={{ marginBottom: 10, fontSize: '12px', color: '#f87171' }}>{subProjectError}</div>
                            )}
                            <div style={{ display: 'flex', gap: 8 }}>
                                <button
                                    onClick={handleAddSubProject}
                                    disabled={addingSubProject}
                                    style={{ flex: 1, padding: '8px 12px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 6, fontSize: '13px', cursor: 'pointer' }}
                                >
                                    {addingSubProject ? 'Adding...' : 'Add'}
                                </button>
                                <button
                                    onClick={() => { setShowAddSubProject(false); setSubProjectError(''); }}
                                    style={{ padding: '8px 12px', background: '#1e293b', color: '#94a3b8', border: '1px solid #334155', borderRadius: 6, fontSize: '13px', cursor: 'pointer' }}
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    ) : (
                        <button
                            onClick={() => setShowAddSubProject(true)}
                            style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '8px 12px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '13px', cursor: 'pointer' }}
                        >
                            <PlusIcon />
                            <span>Add Sub Project</span>
                        </button>
                    )}
                </div>
            )}

            <ErrorBoundary>
                <WorktreesSection project={project} />
            </ErrorBoundary>

            <ErrorBoundary>
                <ProjectReadmeEditor projectId={project.id} />
            </ErrorBoundary>

            <ErrorBoundary>
                <ProjectTodos projectId={project.id} />
            </ErrorBoundary>
        </div>
    );
}
