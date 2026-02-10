import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { pingKeepAlive, getKeepAliveStatus, restartServer, uploadBinary, getUploadTarget, getBuildableProjects } from '../../../api/keepalive';
import type { KeepAliveStatus, UploadTarget, BuildableProject } from '../../../api/keepalive';
import { consumeSSEStream } from '../../../api/sse';
import { BackIcon, UploadIcon, DownloadIcon } from '../../icons';
import { LogViewer } from '../../LogViewer';
import type { LogLine } from '../../LogViewer';
import { useTabHistory } from '../../../hooks/useTabHistory';
import { NavTabs } from '../types';
import { TransferProgress } from './TransferProgress';
import type { TransferProgressData } from './TransferProgress';
import { StreamingActionButton } from '../../StreamingActionButton';
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
    const [restarting, setRestarting] = useState(false);

    // Build states
    const [buildableProjects, setBuildableProjects] = useState<BuildableProject[]>([]);
    const [nextBinaryPath, setNextBinaryPath] = useState<string | null>(null);

    // Log states
    const [logLines, setLogLines] = useState<LogLine[]>([]);
    const [logStreaming, setLogStreaming] = useState(false);

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

    // Log streaming via fetch + consumeSSEStream
    useEffect(() => {
        if (!daemonRunning) {
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
                const resp = await fetch('/api/keep-alive/logs?lines=100', {
                    headers: { 'Accept': 'text/event-stream' },
                    signal: controller.signal,
                });
                if (!resp.ok) {
                    setLogStreaming(false);
                    return;
                }

                await consumeSSEStream(resp, {
                    onLog: (line) => {
                        setLogLines(prev => {
                            const next = [...prev, line];
                            return next.length > 500 ? next.slice(next.length - 500) : next;
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
    }, [daemonRunning]);

    const handleRestart = async () => {
        if (!confirm('Are you sure you want to restart the server?')) return;
        setRestarting(true);
        setActionMessage(null);
        try {
            const result = await restartServer();
            setActionMessage(`Restart: ${result.status}`);
            setTimeout(fetchStatus, 5000);
        } catch (err: any) {
            setActionMessage(`Restart failed: ${err.message}`);
        } finally {
            setRestarting(false);
        }
    };

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
                        {status.next_binary && (
                            <InfoRow label="Next Binary" value={status.next_binary} mono highlight />
                        )}
                        {nextCheckCountdown && (
                            <InfoRow label="Next Check" value={nextCheckCountdown} />
                        )}
                        {status.uptime && <InfoRow label="Uptime" value={status.uptime} />}
                        {status.started_at && <InfoRow label="Started At" value={new Date(status.started_at).toLocaleString()} />}
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
                    <button
                        className="manage-server-btn manage-server-btn--restart"
                        onClick={handleRestart}
                        disabled={restarting}
                    >
                        {restarting ? 'Restarting...' : 'Restart Server'}
                    </button>

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

            {/* Server Logs */}
            {daemonRunning && (
                <div className="manage-server-card" style={{ marginTop: 12 }}>
                    <div className="manage-server-card-header">
                        <span className="manage-server-card-title">Server Logs</span>
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
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
