import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { pingKeepAlive, getKeepAliveStatus, uploadBinary, getUploadTarget, getBuildableProjects, restartServerExecStreaming } from '../../../api/keepalive';
import type { KeepAliveStatus, UploadTarget, BuildableProject } from '../../../api/keepalive';
import { consumeSSEStream } from '../../../api/sse';
import { fetchLogFiles, addLogFile, removeLogFile, streamLogFile, type LogFile } from '../../../api/logs';
import { BackIcon, UploadIcon, DownloadIcon, FolderIcon, PlusIcon } from '../../icons';
import { LogViewer } from '../../LogViewer';
import type { LogLine } from '../../LogViewer';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { NavTabs } from '../types';
import { restartDaemonStreaming } from '../../../api/keepalive';
import { TransferProgress } from './TransferProgress';
import type { TransferProgressData } from './TransferProgress';
import { StreamingActionButton } from '../../StreamingActionButton';
import { getServerStatus, type ServerStatus } from '../../../api/serverStatus';
import './ManageServerView.css';

export function ManageServerView() {
    const navigate = useNavigate();
    const { goBack } = useTabHistory(NavTabs.Home, { defaultBackPath: '/home' });

    const [daemonRunning, setDaemonRunning] = useState<boolean | null>(null);
    const [startCommand, setStartCommand] = useState<string | null>(null);
    const [status, setStatus] = useState<KeepAliveStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [actionMessage, setActionMessage] = useState<string | null>(null);

    // Keep-alive check countdown
    const [nextCheckCountdown, setNextCheckCountdown] = useState<string | null>(null);

    // Upload states
    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [uploadTarget, setUploadTarget] = useState<UploadTarget | null>(null);
    const [uploading, setUploading] = useState(false);
    const [uploadProgress, setUploadProgress] = useState<TransferProgressData | null>(null);

    // Build states
    const [buildableProjects, setBuildableProjects] = useState<BuildableProject[]>([]);
    const [nextBinaryPath, setNextBinaryPath] = useState<string | null>(null);

    // Log states
    const [logLines, setLogLines] = useState<LogLine[]>([]);
    const [logStreaming, setLogStreaming] = useState(false);

    // Log files management states
    const [logFiles, setLogFiles] = useState<LogFile[]>([]);
    const [selectedLogFile, setSelectedLogFile] = useState<string>('');
    const [newLogFileName, setNewLogFileName] = useState('');
    const [newLogFilePath, setNewLogFilePath] = useState('');
    const [addingLogFile, setAddingLogFile] = useState(false);

    // Server status states
    const [serverStatus, setServerStatus] = useState<ServerStatus | null>(null);
    const [serverStatusLoading, setServerStatusLoading] = useState(true);

    const fileInputRef = useRef<HTMLInputElement>(null);
    const abortControllerRef = useRef<AbortController | null>(null);

    const fetchStatus = useCallback(async () => {
        try {
            setError(null);
            const ping = await pingKeepAlive();
            setDaemonRunning(ping.running);
            setStartCommand(ping.start_command ?? null);

            if (ping.running) {
                const st = await getKeepAliveStatus();
                setStatus(st);
            } else {
                setStatus(null);
            }
        } catch (err: any) {
            setError(err.message || 'Failed to fetch status');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchStatus();
        const interval = setInterval(fetchStatus, 10000);
        return () => clearInterval(interval);
    }, [fetchStatus]);

    const fetchServerStatus = useCallback(async () => {
        try {
            const st = await getServerStatus();
            setServerStatus(st);
        } catch {
            setServerStatus(null);
        } finally {
            setServerStatusLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchServerStatus();
        const interval = setInterval(fetchServerStatus, 10000);
        return () => clearInterval(interval);
    }, [fetchServerStatus]);

    // Countdown timer for next health check
    useEffect(() => {
        if (!status?.next_health_check_time) {
            setNextCheckCountdown(null);
            return;
        }

        const updateCountdown = () => {
            if (!status.next_health_check_time) return;
            
            const nextCheck = new Date(status.next_health_check_time).getTime();
            const now = Date.now();
            const diff = Math.max(0, Math.ceil((nextCheck - now) / 1000));
            
            setNextCheckCountdown(`${diff}s`);
        };

        updateCountdown();
        const interval = setInterval(updateCountdown, 1000);
        return () => clearInterval(interval);
    }, [status?.next_health_check_time]);

    // Fetch buildable projects and calculate next binary path
    useEffect(() => {
        const loadBuildInfo = async () => {
            try {
                const projects = await getBuildableProjects();
                setBuildableProjects(projects);
                
                // Calculate next binary path from current binary
                if (status?.binary_path) {
                    const currentPath = status.binary_path;
                    const versionMatch = currentPath.match(/-v(\d+)$/);
                    const currentVersion = versionMatch ? parseInt(versionMatch[1], 10) : 0;
                    const nextVersion = currentVersion + 1;
                    const nextPath = versionMatch 
                        ? currentPath.replace(/-v\d+$/, `-v${nextVersion}`)
                        : `${currentPath}-v1`;
                    setNextBinaryPath(nextPath);
                }
            } catch {
                setBuildableProjects([]);
            }
        };
        loadBuildInfo();
    }, [status?.binary_path]);

    // Fetch log files on mount
    useEffect(() => {
        const loadLogFiles = async () => {
            try {
                const files = await fetchLogFiles();
                setLogFiles(files);
                if (files.length > 0 && !selectedLogFile) {
                    setSelectedLogFile(files[0].name);
                }
            } catch (err) {
                console.error('Failed to load log files:', err);
            }
        };
        loadLogFiles();
    }, []);

    // Log streaming via fetch + consumeSSEStream
    useEffect(() => {
        if (!daemonRunning || !selectedLogFile) {
            if (abortControllerRef.current) {
                abortControllerRef.current.abort();
                abortControllerRef.current = null;
            }
            setLogStreaming(false);
            return;
        }

        const controller = new AbortController();
        abortControllerRef.current = controller;

        setLogLines([]);
        setLogStreaming(true);

        (async () => {
            try {
                const resp = await streamLogFile({ file: selectedLogFile, lines: 1000 });
                if (!resp.ok) {
                    setLogStreaming(false);
                    return;
                }

                await consumeSSEStream(resp, {
                    onLog: (line) => {
                        setLogLines(prev => {
                            const next = [...prev, line];
                            return next.length > 1000 ? next.slice(next.length - 1000) : next;
                        });
                    },
                    onError: (line) => {
                        setLogLines(prev => [...prev, line]);
                    },
                    onDone: () => {
                        setLogStreaming(false);
                    },
                });
            } catch {
                // Aborted or network error
            } finally {
                setLogStreaming(false);
            }
        })();

        return () => {
            controller.abort();
            abortControllerRef.current = null;
        };
    }, [daemonRunning, selectedLogFile]);

    // Step 1: User selects a file -> fetch upload target and show confirm
    const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;
        setSelectedFile(file);
        setActionMessage(null);
        try {
            const target = await getUploadTarget();
            setUploadTarget(target);
        } catch (err: any) {
            setError(`Failed to get upload target: ${err.message}`);
            setSelectedFile(null);
        }
    };

    // Step 2: User confirms upload
    const handleConfirmUpload = async () => {
        if (!selectedFile) return;
        setUploading(true);
        setUploadProgress(null);
        setActionMessage(null);
        try {
            const result = await uploadBinary(selectedFile, (progress) => {
                setUploadProgress(progress);
            });
            setActionMessage(`Uploaded: ${result.target.binary_name} (${formatBytes(result.size)}) — version ${result.target.next_version}`);
            setUploadProgress(null);
            setSelectedFile(null);
            setUploadTarget(null);
            fetchStatus();
        } catch (err: any) {
            setActionMessage(`Upload failed: ${err.message}`);
            setUploadProgress(null);
        } finally {
            setUploading(false);
            if (fileInputRef.current) fileInputRef.current.value = '';
        }
    };

    // Cancel file selection
    const handleCancelUpload = () => {
        setSelectedFile(null);
        setUploadTarget(null);
        if (fileInputRef.current) fileInputRef.current.value = '';
    };

    // Build action that returns SSE Response
    // Uses main server's build API (not keep-alive daemon) for proper environment setup
    const handleBuildAction = async (): Promise<Response> => {
        return fetch('/api/build/build-next', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ project_id: buildableProjects[0]?.id }),
        });
    };

    // Add new log file
    const handleAddLogFile = async () => {
        if (!newLogFileName.trim() || !newLogFilePath.trim()) return;
        setAddingLogFile(true);
        try {
            await addLogFile(newLogFileName.trim(), newLogFilePath.trim());
            const files = await fetchLogFiles();
            setLogFiles(files);
            setSelectedLogFile(newLogFileName.trim());
            setNewLogFileName('');
            setNewLogFilePath('');
            setActionMessage('Log file added successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to add log file');
        } finally {
            setAddingLogFile(false);
        }
    };

    // Remove log file
    const handleRemoveLogFile = async (name: string) => {
        try {
            await removeLogFile(name);
            const files = await fetchLogFiles();
            setLogFiles(files);
            if (selectedLogFile === name) {
                setSelectedLogFile(files.length > 0 ? files[0].name : '');
            }
            setActionMessage('Log file removed successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to remove log file');
        }
    };

    // Change selected log file
    const handleLogFileChange = (name: string) => {
        setSelectedLogFile(name);
    };

    return (
        <div className="mcc-manage-server">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button className="mcc-back-btn" onClick={goBack} style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}>
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0 }}>Manage Server</h2>
                <button
                    className="mcc-port-action-btn"
                    onClick={fetchStatus}
                    style={{ marginLeft: 'auto' }}
                    disabled={loading}
                >
                    Refresh
                </button>
            </div>

            {error && (
                <div className="manage-server-error">{error}</div>
            )}

            {actionMessage && (
                <div className="manage-server-success">{actionMessage}</div>
            )}

            {/* Daemon Status Card */}
            <div className="manage-server-card">
                <div className="manage-server-card-header">
                    <span className="manage-server-card-title">Keep-Alive Daemon</span>
                    {daemonRunning === null ? (
                        <span className="manage-server-badge manage-server-badge--checking">Checking...</span>
                    ) : daemonRunning ? (
                        <span className="manage-server-badge manage-server-badge--running">Running</span>
                    ) : (
                        <span className="manage-server-badge manage-server-badge--stopped">Not Running</span>
                    )}
                </div>

                {status && (
                    <div className="manage-server-info">
                        <InfoRow label="Server PID" value={status.server_pid ? String(status.server_pid) : 'N/A'} />
                        <InfoRow label="Server Port" value={String(status.server_port)} />
                        <InfoRow label="Keep-Alive PID" value={String(status.keep_alive_pid)} />
                        <InfoRow label="Keep-Alive Port" value={String(status.keep_alive_port)} />
                        <InfoRow label="Binary" value={status.binary_path} mono />
                        <InfoRow label="Daemon Binary" value={status.daemon_binary_path} mono />
                        <InfoRow label="Restart Count" value={String(status.restart_count)} />
                        {status.next_binary && (
                            <InfoRow label="Next Binary" value={status.next_binary} mono highlight />
                        )}
                        {nextCheckCountdown && (
                            <InfoRow label="Next Check" value={nextCheckCountdown} />
                        )}
                        {status.uptime && <InfoRow label="Uptime" value={status.uptime} />}
                        {status.started_at && <InfoRow label="Started At" value={(() => {
                            const d = new Date(status.started_at);
                            const date = d.toLocaleDateString('en-CA');
                            const time = d.toLocaleTimeString('en-US', { hour12: true });
                            return `${date} ${time}`;
                        })()} />}
                    </div>
                )}

                {!daemonRunning && !loading && (
                    <div className="manage-server-hint">
                        <p>The keep-alive daemon is not running. Start it with:</p>
                        {startCommand && (
                            <pre className="manage-server-command">{startCommand}</pre>
                        )}
                    </div>
                )}
            </div>

            {/* Actions */}
            {daemonRunning && (
                <div className="manage-server-actions">
                    <StreamingActionButton
                        label="Restart Server"
                        runningLabel="Restarting Server..."
                        action={restartServerExecStreaming}
                        className="manage-server-btn manage-server-btn--restart"
                        logMaxHeight={200}
                        onComplete={(result) => {
                            if (result.ok) {
                                // Server will restart via exec, give it time to come back
                                setTimeout(fetchStatus, 3000);
                            }
                        }}
                    />

                    <StreamingActionButton
                        label="Restart Daemon"
                        runningLabel="Restarting Daemon..."
                        action={restartDaemonStreaming}
                        className="manage-server-btn manage-server-btn--restart"
                        logMaxHeight={200}
                        onComplete={(result) => {
                            if (result.ok) {
                                // Daemon will restart, give it time to come back
                                setTimeout(fetchStatus, 3000);
                            }
                        }}
                    />

                    <div>
                        {/* File selection / confirm step */}
                        {!selectedFile ? (
                            <label className="manage-server-btn manage-server-btn--upload" style={{ cursor: uploading ? 'not-allowed' : 'pointer' }}>
                                Upload New Binary
                                <input
                                    ref={fileInputRef}
                                    type="file"
                                    style={{ display: 'none' }}
                                    onChange={handleFileSelect}
                                    disabled={uploading}
                                />
                            </label>
                        ) : (
                            <div className="manage-server-confirm">
                                <div className="manage-server-confirm-info">
                                    <div className="manage-server-confirm-row">
                                        <span className="manage-server-info-label">File</span>
                                        <span className="manage-server-info-value">{selectedFile.name} ({formatBytes(selectedFile.size)})</span>
                                    </div>
                                    {uploadTarget && (
                                        <>
                                            <div className="manage-server-confirm-row">
                                                <span className="manage-server-info-label">Target Path</span>
                                                <span className="manage-server-info-value manage-server-info-value--mono">{uploadTarget.path}</span>
                                            </div>
                                            <div className="manage-server-confirm-row">
                                                <span className="manage-server-info-label">Version</span>
                                                <span className="manage-server-info-value">v{uploadTarget.current_version} → v{uploadTarget.next_version}</span>
                                            </div>
                                        </>
                                    )}
                                </div>
                                <div className="manage-server-confirm-actions">
                                    <button
                                        className="manage-server-btn manage-server-btn--confirm"
                                        onClick={handleConfirmUpload}
                                        disabled={uploading}
                                    >
                                        {uploading ? 'Uploading...' : 'Confirm Upload'}
                                    </button>
                                    <button
                                        className="manage-server-btn manage-server-btn--cancel"
                                        onClick={handleCancelUpload}
                                        disabled={uploading}
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </div>
                        )}

                        {uploading && <TransferProgress progress={uploadProgress} label="Upload" />}

                        <p className="manage-server-upload-hint">
                            Upload a new binary. It will be placed with the next version number and used on next restart.
                        </p>

                        {/* Build Next button */}
                        <div style={{ marginTop: 16, borderTop: '1px solid #e5e7eb', paddingTop: 16 }}>
                            <StreamingActionButton
                                label="Build Next"
                                runningLabel="Building..."
                                action={handleBuildAction}
                                className="manage-server-btn manage-server-btn--upload"
                                logMaxHeight={200}
                                disabled={buildableProjects.length === 0}
                                onComplete={(result) => {
                                    if (result.ok) {
                                        fetchStatus();
                                    }
                                }}
                            />

                            {nextBinaryPath && (
                                <div className="manage-server-confirm-row" style={{ marginTop: 8, fontSize: 12 }}>
                                    <span className="manage-server-info-label">Output:</span>
                                    <span className="manage-server-info-value manage-server-info-value--mono" style={{ fontSize: 11 }}>
                                        {nextBinaryPath}
                                    </span>
                                </div>
                            )}

                            {buildableProjects.length === 0 ? (
                                <p className="manage-server-upload-hint" style={{ color: '#dc2626' }}>
                                    No buildable projects found. Ensure the project has a script/server/build directory.
                                </p>
                            ) : (
                                <p className="manage-server-upload-hint">
                                    Build the next binary from source using script/server/build/for-linux-amd64.
                                    {buildableProjects[0] && (
                                        <> Project: <strong>{buildableProjects[0].name}</strong></>
                                    )}
                                </p>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Server Status */}
            <div className="manage-server-card" style={{ marginTop: 12 }}>
                <div className="manage-server-card-header">
                    <span className="manage-server-card-title">Server Status</span>
                </div>
                {serverStatusLoading ? (
                    <div style={{ padding: 16, color: '#6b7280' }}>Loading...</div>
                ) : serverStatus ? (
                    <div className="manage-server-info">
                        <InfoRow label="OS" value={serverStatus.os_info.os} />
                        <InfoRow label="Arch" value={serverStatus.os_info.arch} />
                        <InfoRow label="Kernel" value={serverStatus.os_info.kernel} />
                        <InfoRow label="CPU Cores" value={String(serverStatus.cpu.num_cpu)} />
                        <InfoRow label="CPU Usage" value={`${serverStatus.cpu.used_percent.toFixed(1)}%`} />
                        <InfoRow label="Total Memory" value={formatBytes(serverStatus.memory.total)} />
                        <InfoRow label="Used Memory" value={`${formatBytes(serverStatus.memory.used)} (${serverStatus.memory.used_percent.toFixed(1)}%)`} />
                        {serverStatus.disk.map((d, i) => (
                            <InfoRow key={i} label={`Disk ${d.mount_point}`} value={`${formatBytes(d.used)} / ${formatBytes(d.size)} (${d.use_percent.toFixed(1)}%)`} />
                        ))}
                        <div style={{ marginTop: 12, fontWeight: 600, fontSize: 13, color: '#374151' }}>Top CPU Processes</div>
                        {serverStatus.top_cpu.map((p, i) => (
                            <InfoRow key={i} label={`${p.name} (PID: ${p.pid})`} value={`CPU: ${p.cpu} | Mem: ${p.mem}`} mono />
                        ))}
                        <div style={{ marginTop: 12, fontWeight: 600, fontSize: 13, color: '#374151' }}>Top Memory Processes</div>
                        {serverStatus.top_mem.map((p, i) => (
                            <InfoRow key={i} label={`${p.name} (PID: ${p.pid})`} value={`CPU: ${p.cpu} | Mem: ${p.mem}`} mono />
                        ))}
                    </div>
                ) : (
                    <div style={{ padding: 16, color: '#6b7280' }}>Unable to load server status</div>
                )}
            </div>

            {/* Server Logs */}
            {daemonRunning && (
                <div className="manage-server-card" style={{ marginTop: 12 }}>
                    <div className="manage-server-card-header">
                        <span className="manage-server-card-title">Server Logs</span>
                        <select
                            value={selectedLogFile}
                            onChange={(e) => handleLogFileChange(e.target.value)}
                            style={{
                                padding: '4px 8px',
                                borderRadius: 4,
                                border: '1px solid #d1d5db',
                                background: '#fff',
                                fontSize: 12,
                                color: '#374151',
                            }}
                        >
                            {logFiles.map((f) => (
                                <option key={f.name} value={f.name}>
                                    {f.name} ({f.path})
                                </option>
                            ))}
                        </select>
                        <span className={`manage-server-badge ${logStreaming ? 'manage-server-badge--running' : 'manage-server-badge--stopped'}`}>
                            {logStreaming ? 'Live' : 'Disconnected'}
                        </span>
                    </div>
                    <LogViewer
                        lines={logLines}
                        pending={logStreaming}
                        pendingMessage="Streaming..."
                        emptyMessage="No log output yet..."
                        maxHeight={300}
                    />
                </div>
            )}

            {/* Log Files Management */}
            <div className="manage-server-card" style={{ marginTop: 12 }}>
                <div className="manage-server-card-header">
                    <span className="manage-server-card-title">Log Files</span>
                </div>
                <div style={{ padding: 12 }}>
                    {/* Add new log file */}
                    <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                        <input
                            type="text"
                            placeholder="Name (e.g., server)"
                            value={newLogFileName}
                            onChange={(e) => setNewLogFileName(e.target.value)}
                            style={{
                                flex: 1,
                                padding: '8px 10px',
                                borderRadius: 4,
                                border: '1px solid #d1d5db',
                                fontSize: 13,
                            }}
                        />
                        <input
                            type="text"
                            placeholder="Path (e.g., /var/log/app.log)"
                            value={newLogFilePath}
                            onChange={(e) => setNewLogFilePath(e.target.value)}
                            style={{
                                flex: 2,
                                padding: '8px 10px',
                                borderRadius: 4,
                                border: '1px solid #d1d5db',
                                fontSize: 13,
                            }}
                        />
                        <button
                            onClick={handleAddLogFile}
                            disabled={addingLogFile || !newLogFileName.trim() || !newLogFilePath.trim()}
                            style={{
                                padding: '8px 12px',
                                borderRadius: 4,
                                border: 'none',
                                background: '#3b82f6',
                                color: '#fff',
                                fontSize: 13,
                                cursor: 'pointer',
                                opacity: addingLogFile || !newLogFileName.trim() || !newLogFilePath.trim() ? 0.6 : 1,
                            }}
                        >
                            <PlusIcon />
                        </button>
                    </div>

                    {/* List of configured log files */}
                    {logFiles.length === 0 ? (
                        <div style={{ color: '#6b7280', fontSize: 13, textAlign: 'center', padding: 12 }}>
                            No log files configured
                        </div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                            {logFiles.map((f) => (
                                <div
                                    key={f.name}
                                    style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'space-between',
                                        padding: '8px 12px',
                                        background: selectedLogFile === f.name ? '#eff6ff' : '#f9fafb',
                                        borderRadius: 4,
                                        border: '1px solid #e5e7eb',
                                    }}
                                >
                                    <div>
                                        <div style={{ fontWeight: 500, fontSize: 13 }}>{f.name}</div>
                                        <div style={{ color: '#6b7280', fontSize: 12 }}>{f.path}</div>
                                    </div>
                                    <button
                                        onClick={() => handleRemoveLogFile(f.name)}
                                        style={{
                                            padding: '4px 8px',
                                            borderRadius: 4,
                                            border: 'none',
                                            background: 'transparent',
                                            color: '#ef4444',
                                            cursor: 'pointer',
                                            fontSize: 16,
                                        }}
                                    >
                                        ×
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            {/* Manage Files */}
            <div className="manage-server-card" style={{ marginTop: 12 }}>
                <div className="manage-server-card-header">
                    <span className="manage-server-card-title">File Manager</span>
                </div>
                <button 
                    className="manage-server-btn manage-server-btn--file-transfer" 
                    onClick={() => navigate('../manage-files')}
                    style={{ width: '100%', marginBottom: 0 }}
                >
                    <FolderIcon />
                    <span>Manage Files</span>
                </button>
            </div>

            {/* File Transfer */}
            <div className="manage-server-card" style={{ marginTop: 12 }}>
                <div className="manage-server-card-header">
                    <span className="manage-server-card-title">File Transfer</span>
                </div>
                <div className="manage-server-file-transfer">
                    <button className="manage-server-btn manage-server-btn--file-transfer" onClick={() => navigate('../upload-file')}>
                        <UploadIcon />
                        <span>Upload File</span>
                    </button>
                    <button className="manage-server-btn manage-server-btn--file-transfer" onClick={() => navigate('../download-file')}>
                        <DownloadIcon />
                        <span>Download File</span>
                    </button>
                </div>
            </div>
        </div>
    );
}

function InfoRow({ label, value, mono, highlight }: { label: string; value: string; mono?: boolean; highlight?: boolean }) {
    return (
        <div className="manage-server-info-row">
            <span className="manage-server-info-label">{label}</span>
            <span className={`manage-server-info-value${mono ? ' manage-server-info-value--mono' : ''}${highlight ? ' manage-server-info-value--highlight' : ''}`}>{value}</span>
        </div>
    );
}

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    if (bytes < 1024 * 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    return `${(bytes / (1024 * 1024 * 1024 * 1024)).toFixed(1)} TB`;
}
