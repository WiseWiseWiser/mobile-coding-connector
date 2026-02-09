import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useV2Context } from '../../V2Context';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { NavTabs } from '../types';
import { updateProject, runGitOp, GitOps } from '../../../api/projects';
import type { GitOp } from '../../../api/projects';
import { loadSSHKeys } from './settings/gitStorage';
import type { SSHKey } from './settings/gitStorage';
import { encryptWithServerKey, EncryptionNotAvailableError } from './crypto';
import { LogViewer } from '../../LogViewer';
import { KeyIcon } from '../../icons';

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

    // Git operation state
    const [gitRunning, setGitRunning] = useState<GitOp | null>(null);
    const [gitLogs, setGitLogs] = useState<string[]>([]);
    const [gitResult, setGitResult] = useState<{ status: string; message?: string; error?: string } | null>(null);

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

    const handleGitOp = async (op: GitOp) => {
        setGitRunning(op);
        setGitLogs([]);
        setGitResult(null);

        // Prepare SSH key if the project uses one
        let sshKey: string | undefined;
        if (project.use_ssh && project.ssh_key_id) {
            const key = sshKeys.find(k => k.id === project.ssh_key_id);
            if (key) {
                try {
                    sshKey = await encryptWithServerKey(key.privateKey);
                } catch (err) {
                    if (err instanceof EncryptionNotAvailableError) {
                        setGitResult({ status: 'error', error: 'Server encryption keys not configured.' });
                    } else {
                        setGitResult({ status: 'error', error: String(err) });
                    }
                    setGitRunning(null);
                    return;
                }
            }
        }

        try {
            const resp = await runGitOp(op, {
                project_id: project.id,
                ssh_key: sshKey,
            });

            const contentType = resp.headers.get('Content-Type') || '';
            if (contentType.includes('text/event-stream')) {
                const reader = resp.body?.getReader();
                if (!reader) {
                    setGitResult({ status: 'error', error: 'Failed to read response stream' });
                    setGitRunning(null);
                    return;
                }

                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        if (!line.startsWith('data: ')) continue;
                        try {
                            const data = JSON.parse(line.slice(6));
                            if (data.type === 'log') {
                                setGitLogs(prev => [...prev, data.message]);
                            } else if (data.type === 'error') {
                                setGitLogs(prev => [...prev, `ERROR: ${data.message}`]);
                                setGitResult({ status: 'error', error: data.message });
                            } else if (data.type === 'done') {
                                setGitResult({ status: 'ok', message: data.message });
                            }
                        } catch {
                            // Skip malformed SSE data
                        }
                    }
                }
            } else {
                const data = await resp.json();
                if (data.error) {
                    setGitResult({ status: 'error', error: data.error });
                } else {
                    setGitResult({ status: 'ok', message: data.message });
                }
            }
        } catch (err) {
            setGitResult({ status: 'error', error: String(err) });
        }
        setGitRunning(null);
    };

    const currentKey = sshKeys.find(k => k.id === project.ssh_key_id);
    const gitOpLabel = (op: GitOp) => op.charAt(0).toUpperCase() + op.slice(1);

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
                <div style={{ display: 'flex', gap: 10, marginBottom: 12 }}>
                    <button
                        className="mcc-port-action-btn"
                        onClick={() => handleGitOp(GitOps.Fetch)}
                        disabled={!!gitRunning}
                        style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
                    >
                        {gitRunning === GitOps.Fetch ? 'Fetching...' : 'Git Fetch'}
                    </button>
                    <button
                        className="mcc-port-action-btn"
                        onClick={() => handleGitOp(GitOps.Pull)}
                        disabled={!!gitRunning}
                        style={{ flex: 1, padding: '10px 16px', background: '#1e293b', color: '#e2e8f0', border: '1px solid #334155', borderRadius: 8, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
                    >
                        {gitRunning === GitOps.Pull ? 'Pulling...' : 'Git Pull'}
                    </button>
                </div>

                {(gitLogs.length > 0 || !!gitRunning) && (
                    <LogViewer
                        lines={gitLogs.map(text => ({ text, error: text.startsWith('ERROR:') }))}
                        pending={!!gitRunning}
                        pendingMessage={`Git ${gitRunning ? gitOpLabel(gitRunning) : ''} in progress...`}
                    />
                )}

                {gitResult && (
                    <div style={{
                        marginTop: 8,
                        padding: '10px 14px',
                        borderRadius: 8,
                        fontSize: '13px',
                        background: gitResult.status === 'ok' ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                        border: gitResult.status === 'ok' ? '1px solid rgba(34, 197, 94, 0.3)' : '1px solid rgba(239, 68, 68, 0.3)',
                        color: gitResult.status === 'ok' ? '#86efac' : '#fca5a5',
                    }}>
                        {gitResult.status === 'ok' ? gitResult.message : `Error: ${gitResult.error}`}
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
