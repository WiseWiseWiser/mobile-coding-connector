import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useV2Context } from '../../V2Context';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { NavTabs } from '../types';
import { updateProject, runGitOp, GitOps } from '../../../api/projects';
import type { GitOp } from '../../../api/projects';
import { cloneRepo } from '../../../api/auth';
import { loadSSHKeys } from './settings/gitStorage';
import type { SSHKey } from './settings/gitStorage';
import { encryptWithServerKey } from './crypto';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import { KeyIcon } from '../../icons';
import { CustomSelect } from './CustomSelect';
import './ProjectConfigView.css';

export function ProjectConfigView() {
    const { projectName } = useParams<{ projectName: string }>();
    const navigate = useNavigate();
    const { projectsList, fetchProjects } = useV2Context();
    // When going back from project config, always go to /home (project list)
    const { goBack } = useTabHistory(NavTabs.Home, { defaultBackPath: '/home' });

    const project = projectsList.find(p => p.name === projectName);
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

    useEffect(() => {
        const keys = loadSSHKeys();
        setSshKeys(keys);
        if (project?.ssh_key_id) {
            setSelectedKeyId(project.ssh_key_id);
        }
    }, [project?.ssh_key_id]);

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
                                <button
                                    className="mcc-port-action-btn"
                                    onClick={handleClone}
                                    disabled={gitState.running}
                                    style={{ width: '100%', padding: '10px 16px', background: '#3b82f6', color: '#fff', border: 'none', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
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
                    <div style={{ display: 'flex', gap: 10, marginBottom: 12 }}>
                        <button
                            className="mcc-port-action-btn"
                            onClick={() => handleGitOp(GitOps.Fetch)}
                            disabled={gitState.running}
                            style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
                        >
                            {gitState.running ? '...' : 'Git Fetch'}
                        </button>
                        <button
                            className="mcc-port-action-btn"
                            onClick={() => handleGitOp(GitOps.Pull)}
                            disabled={gitState.running}
                            style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
                        >
                            {gitState.running ? '...' : 'Git Pull'}
                        </button>
                    </div>
                )}

                {/* Shared streaming logs area */}
                <StreamingLogs
                    state={gitState}
                    pendingMessage="Running..."
                    maxHeight={200}
                />
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
        </div>
    );
}
