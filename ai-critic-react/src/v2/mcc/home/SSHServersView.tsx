import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchSSHServers, createSSHServer, updateSSHServer, deleteSSHServer } from '../../../api/sshservers';
import type { SSHServer } from '../../../api/sshservers';
import { loadSSHKeys, type SSHKey } from './settings/gitStorage';
import { EmbeddedTerminal } from './EmbeddedTerminal';
import { TerminalIcon } from '../../../pure-view/icons/TerminalIcon';
import { PlusIcon } from '../../../pure-view/icons/PlusIcon';
import { KeyIcon } from '../../../pure-view/icons/KeyIcon';
import './SSHServersView.css';

export function SSHServersView() {
    const navigate = useNavigate();
    const [servers, setServers] = useState<SSHServer[]>([]);
    const [sshKeys, setSSHKeys] = useState<SSHKey[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [editingServer, setEditingServer] = useState<SSHServer | null>(null);
    const [isCreating, setIsCreating] = useState(false);
    const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null);
    const [activeConnection, setActiveConnection] = useState<SSHServer | null>(null);

    // Form state
    const [formName, setFormName] = useState('');
    const [formHost, setFormHost] = useState('');
    const [formPort, setFormPort] = useState(22);
    const [formUsername, setFormUsername] = useState('');
    const [formSSHKeyId, setFormSSHKeyId] = useState('');

    useEffect(() => {
        loadData();
        setSSHKeys(loadSSHKeys());
    }, []);

    const loadData = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await fetchSSHServers();
            setServers(data);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to load SSH servers');
        } finally {
            setLoading(false);
        }
    };

    const resetForm = () => {
        setFormName('');
        setFormHost('');
        setFormPort(22);
        setFormUsername('');
        setFormSSHKeyId('');
        setEditingServer(null);
        setIsCreating(false);
    };

    const handleStartCreate = () => {
        if (sshKeys.length === 0) {
            alert('Please add an SSH key in Git Settings first.');
            return;
        }
        resetForm();
        setFormSSHKeyId(sshKeys[0]?.id || '');
        setIsCreating(true);
    };

    const handleStartEdit = (server: SSHServer) => {
        setEditingServer(server);
        setFormName(server.name);
        setFormHost(server.host);
        setFormPort(server.port);
        setFormUsername(server.username);
        setFormSSHKeyId(server.ssh_key_id);
        setIsCreating(false);
    };

    const handleSave = async () => {
        if (!formName.trim() || !formHost.trim() || !formUsername.trim()) {
            alert('Please fill in all required fields.');
            return;
        }

        setError('');
        const serverData = {
            name: formName.trim(),
            host: formHost.trim(),
            port: formPort,
            username: formUsername.trim(),
            ssh_key_id: formSSHKeyId,
        };

        try {
            if (editingServer) {
                await updateSSHServer(editingServer.id, serverData);
            } else {
                await createSSHServer(serverData);
            }
            resetForm();
            await loadData();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to save SSH server');
        }
    };

    const handleDelete = async (id: string) => {
        try {
            await deleteSSHServer(id);
            await loadData();
            setShowDeleteConfirm(null);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to delete SSH server');
        }
    };

    const handleConnect = (server: SSHServer) => {
        const sshKey = sshKeys.find(k => k.id === server.ssh_key_id);
        if (!sshKey) {
            alert('SSH key not found. Please configure SSH keys in Git Settings.');
            return;
        }
        
        // Show embedded terminal for this server
        setActiveConnection(server);
    };

    const handleCloseTerminal = () => {
        setActiveConnection(null);
    };

    const getSSHKeyName = (keyId: string) => {
        const key = sshKeys.find(k => k.id === keyId);
        return key?.name || 'Unknown Key';
    };

    const isEditing = isCreating || editingServer !== null;

    return (
        <div className="mcc-ssh-servers-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>← Back</button>
                <h2>SSH Servers</h2>
            </div>

            {!isEditing && (
                <button className="mcc-ssh-add-btn" onClick={handleStartCreate}>
                    <PlusIcon />
                    <span>Add SSH Server</span>
                </button>
            )}

            {sshKeys.length === 0 && !isEditing && (
                <div className="mcc-ssh-warning">
                    <KeyIcon />
                    <span>No SSH keys configured. </span>
                    <button onClick={() => navigate('../settings/git')}>
                        Add SSH Key
                    </button>
                </div>
            )}

            {error && <div className="mcc-ssh-error">{error}</div>}

            {/* Edit/Create Form */}
            {isEditing && (
                <div className="mcc-ssh-form">
                    <div className="mcc-ssh-form-row">
                        <label>Name</label>
                        <input
                            type="text"
                            value={formName}
                            onChange={(e) => setFormName(e.target.value)}
                            placeholder="e.g., Production Server"
                        />
                    </div>

                    <div className="mcc-ssh-form-row">
                        <label>Host / IP</label>
                        <input
                            type="text"
                            value={formHost}
                            onChange={(e) => setFormHost(e.target.value)}
                            placeholder="e.g., 192.168.1.100 or server.example.com"
                        />
                    </div>

                    <div className="mcc-ssh-form-row">
                        <label>Port</label>
                        <input
                            type="number"
                            value={formPort}
                            onChange={(e) => setFormPort(parseInt(e.target.value) || 22)}
                            placeholder="22"
                            min={1}
                            max={65535}
                        />
                    </div>

                    <div className="mcc-ssh-form-row">
                        <label>Username</label>
                        <input
                            type="text"
                            value={formUsername}
                            onChange={(e) => setFormUsername(e.target.value)}
                            placeholder="e.g., root or ubuntu"
                        />
                    </div>

                    <div className="mcc-ssh-form-row">
                        <label>SSH Key</label>
                        <select
                            value={formSSHKeyId}
                            onChange={(e) => setFormSSHKeyId(e.target.value)}
                        >
                            {sshKeys.map(key => (
                                <option key={key.id} value={key.id}>
                                    {key.name} ({key.host})
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className="mcc-ssh-form-buttons">
                        <button className="mcc-ssh-save-btn" onClick={handleSave}>
                            Save
                        </button>
                        <button className="mcc-ssh-cancel-btn" onClick={resetForm}>
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            {/* Servers List */}
            {!isEditing && (
                <div className="mcc-ssh-servers-list">
                    {loading ? (
                        <div className="mcc-ssh-empty">Loading...</div>
                    ) : servers.length === 0 ? (
                        <div className="mcc-ssh-empty">
                            <TerminalIcon />
                            <p>No SSH servers configured yet.</p>
                            <p>Click "Add SSH Server" to get started.</p>
                        </div>
                    )                     : (
                        servers.map(server => (
                            <div key={server.id} className="mcc-ssh-server-card">
                                <div className="mcc-ssh-server-row">
                                    <div className="mcc-ssh-server-info">
                                        <div className="mcc-ssh-server-header">
                                            <span className="mcc-ssh-server-name">{server.name}</span>
                                            <span className="mcc-ssh-server-key">
                                                <KeyIcon /> {getSSHKeyName(server.ssh_key_id)}
                                            </span>
                                        </div>
                                        <div className="mcc-ssh-server-details">
                                            <code>{server.username}@{server.host}:{server.port}</code>
                                        </div>
                                    </div>
                                    <div className="mcc-ssh-server-actions">
                                        <button
                                            className="mcc-ssh-connect-btn"
                                            onClick={() => handleConnect(server)}
                                            title="Connect via SSH"
                                            disabled={activeConnection?.id === server.id}
                                        >
                                            <TerminalIcon />
                                            {activeConnection?.id === server.id ? 'Connected' : 'Connect'}
                                        </button>
                                        <button
                                            className="mcc-ssh-edit-btn"
                                            onClick={() => handleStartEdit(server)}
                                            title="Edit"
                                        >
                                            Edit
                                        </button>
                                        {showDeleteConfirm === server.id ? (
                                            <div className="mcc-ssh-delete-confirm">
                                                <span>Delete?</span>
                                                <button onClick={() => handleDelete(server.id)}>Yes</button>
                                                <button onClick={() => setShowDeleteConfirm(null)}>No</button>
                                            </div>
                                        ) : (
                                            <button
                                                className="mcc-ssh-delete-btn"
                                                onClick={() => setShowDeleteConfirm(server.id)}
                                                title="Delete"
                                            >
                                                Delete
                                            </button>
                                        )}
                                    </div>
                                </div>
                                
                                {/* Inline Terminal for this server - appears below the row */}
                                {activeConnection?.id === server.id && (
                                    <EmbeddedTerminal
                                        host={server.host}
                                        port={server.port}
                                        username={server.username}
                                        sshKeyId={server.ssh_key_id}
                                        onClose={handleCloseTerminal}
                                    />
                                )}
                            </div>
                        ))
                    )}
                </div>
            )}
        </div>
    );
}
