import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useV2Context } from '../../V2Context';
import { updateProject } from '../../../api/projects';
import { loadSSHKeys } from './settings/gitStorage';
import type { SSHKey } from './settings/gitStorage';
import { KeyIcon } from '../../icons';

export function ProjectConfigView() {
    const navigate = useNavigate();
    const { projectId } = useParams<{ projectId: string }>();
    const { projectsList, fetchProjects } = useV2Context();

    const project = projectsList.find(p => p.id === projectId);
    const [sshKeys, setSshKeys] = useState<SSHKey[]>([]);
    const [selectedKeyId, setSelectedKeyId] = useState('');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

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
                    <button className="mcc-back-btn" onClick={() => navigate(-1)}>&larr;</button>
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

    const currentKey = sshKeys.find(k => k.id === project.ssh_key_id);

    return (
        <div className="mcc-workspace-list">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate(-1)}>&larr;</button>
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
                        <select
                            value={selectedKeyId}
                            onChange={e => setSelectedKeyId(e.target.value)}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                background: '#1e293b',
                                border: '1px solid #334155',
                                borderRadius: 8,
                                color: '#e2e8f0',
                                fontSize: '14px',
                                marginBottom: 12,
                                appearance: 'none',
                                WebkitAppearance: 'none',
                            }}
                        >
                            <option value="">-- No SSH key --</option>
                            {sshKeys.map(k => (
                                <option key={k.id} value={k.id}>
                                    {k.name} ({k.host})
                                </option>
                            ))}
                        </select>

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
