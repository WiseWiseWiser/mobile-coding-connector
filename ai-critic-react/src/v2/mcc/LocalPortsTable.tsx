import { useState, useEffect } from 'react';
import type { LocalPortInfo } from '../../api/ports';
import { fetchProtectedPorts, addProtectedPort, removeProtectedPort } from '../../api/ports';
import { PlusIcon } from '../../pure-view/icons/PlusIcon';
import { installTool } from '../../api/tools';
import { consumeSSEStream } from '../../api/sse';
import { LogViewer } from '../LogViewer';
import type { LogLine } from '../LogViewer';
import { KillProcessModal } from './KillProcessModal';
import './LocalPortsTable.css';

const SortFields = {
    Port: 'port',
    Pid: 'pid',
    Command: 'command',
} as const;

type SortField = typeof SortFields[keyof typeof SortFields];

const SortDirections = {
    Asc: 'asc',
    Desc: 'desc',
} as const;

type SortDirection = typeof SortDirections[keyof typeof SortDirections];

export interface LocalPortsTableProps {
    ports: LocalPortInfo[];
    loading: boolean;
    error: string | null;
    forwardedPorts: Set<number>;
    onForwardPort: (port: number) => void;
    onLsofInstalled?: () => void;
}

export function LocalPortsTable({
    ports,
    loading,
    error,
    forwardedPorts,
    onForwardPort,
    onLsofInstalled,
}: LocalPortsTableProps) {
    const [sortField, setSortField] = useState<SortField>(SortFields.Port);
    const [sortDirection, setSortDirection] = useState<SortDirection>(SortDirections.Asc);
    const [installing, setInstalling] = useState(false);
    const [installLogs, setInstallLogs] = useState<LogLine[]>([]);
    const [showInstallLogs, setShowInstallLogs] = useState(false);
    const [killModalPort, setKillModalPort] = useState<LocalPortInfo | null>(null);
    const [protectedPorts, setProtectedPorts] = useState<number[]>([]);
    const [protectingPorts, setProtectingPorts] = useState<Set<number>>(new Set());

    useEffect(() => {
        fetchProtectedPorts()
            .then(setProtectedPorts)
            .catch(() => {});
    }, []);

    const handleProtect = async (port: number) => {
        if (protectingPorts.has(port)) return;
        setProtectingPorts(prev => new Set(prev).add(port));
        try {
            await addProtectedPort(port);
            setProtectedPorts(prev => [...prev, port]);
        } catch (err) {
            console.error('Failed to protect port:', err);
        } finally {
            setProtectingPorts(prev => {
                const next = new Set(prev);
                next.delete(port);
                return next;
            });
        }
    };

    const handleUnprotect = async (port: number) => {
        if (protectingPorts.has(port)) return;
        setProtectingPorts(prev => new Set(prev).add(port));
        try {
            await removeProtectedPort(port);
            setProtectedPorts(prev => prev.filter(p => p !== port));
        } catch (err) {
            console.error('Failed to unprotect port:', err);
        } finally {
            setProtectingPorts(prev => {
                const next = new Set(prev);
                next.delete(port);
                return next;
            });
        }
    };

    const isLsofError = error?.toLowerCase().includes('lsof not installed') || false;

    const handleInstallLsof = async () => {
        setInstalling(true);
        setInstallLogs([]);
        setShowInstallLogs(true);
        try {
            const resp = await installTool('lsof');
            await consumeSSEStream(resp, {
                onLog: (line) => setInstallLogs(prev => [...prev, line]),
                onError: (line) => setInstallLogs(prev => [...prev, line]),
                onDone: (message) => {
                    setInstallLogs(prev => [...prev, { text: message }]);
                    onLsofInstalled?.();
                    // Reload the page after a short delay to re-establish SSE connection
                    setTimeout(() => {
                        window.location.reload();
                    }, 1500);
                },
            });
        } catch (err) {
            setInstallLogs(prev => [...prev, { text: String(err), error: true }]);
        } finally {
            setInstalling(false);
        }
    };

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDirection(sortDirection === SortDirections.Asc ? SortDirections.Desc : SortDirections.Asc);
        } else {
            setSortField(field);
            setSortDirection(SortDirections.Asc);
        }
    };

    const sortedPorts = [...ports].sort((a, b) => {
        let comparison = 0;
        switch (sortField) {
            case SortFields.Port:
                comparison = a.port - b.port;
                break;
            case SortFields.Pid:
                comparison = a.pid - b.pid;
                break;
            case SortFields.Command:
                comparison = a.command.localeCompare(b.command);
                break;
        }
        return sortDirection === SortDirections.Asc ? comparison : -comparison;
    });

    const getSortIndicator = (field: SortField) => {
        if (sortField !== field) return '⇅';
        return sortDirection === SortDirections.Asc ? '↑' : '↓';
    };

    if (loading && ports.length === 0) {
        return (
            <div className="mcc-lp-loading">
                <div className="mcc-loading-spinner" />
                <span>Loading local ports...</span>
            </div>
        );
    }

    if (error && ports.length === 0) {
        return (
            <div className="mcc-lp-error">
                <div className="mcc-lp-error-content">
                    <span className="mcc-lp-error-message">{error}</span>
                    {isLsofError && (
                        <div className="mcc-lp-error-actions">
                            {!showInstallLogs ? (
                                <button
                                    className="mcc-lp-install-btn"
                                    onClick={handleInstallLsof}
                                    disabled={installing}
                                >
                                    {installing ? 'Installing...' : 'Install lsof'}
                                </button>
                            ) : (
                                <button
                                    className="mcc-lp-install-btn"
                                    onClick={() => setShowInstallLogs(false)}
                                    disabled={installing}
                                >
                                    Hide Logs
                                </button>
                            )}
                        </div>
                    )}
                </div>
                {showInstallLogs && installLogs.length > 0 && (
                    <div className="mcc-lp-install-logs">
                        <LogViewer
                            lines={installLogs}
                            pending={installing}
                            pendingMessage="Installing lsof..."
                            maxHeight={200}
                        />
                    </div>
                )}
            </div>
        );
    }

    return (
        <div className="mcc-lp-section">
            <div className="mcc-lp-header">
                <h3 className="mcc-lp-title">Local Listening Ports</h3>
                <span className="mcc-lp-count">{ports.length}</span>
            </div>
            
            {ports.length === 0 ? (
                <div className="mcc-lp-empty">
                    No listening ports found on this machine.
                </div>
            ) : (
                <>
                    <div className="mcc-lp-sort-bar">
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Port ? 'active' : ''}`} onClick={() => handleSort(SortFields.Port)}>
                            Port {getSortIndicator(SortFields.Port)}
                        </button>
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Pid ? 'active' : ''}`} onClick={() => handleSort(SortFields.Pid)}>
                            PID {getSortIndicator(SortFields.Pid)}
                        </button>
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Command ? 'active' : ''}`} onClick={() => handleSort(SortFields.Command)}>
                            Process {getSortIndicator(SortFields.Command)}
                        </button>
                    </div>
                    <div className="mcc-lp-list">
                        {sortedPorts.map((port) => {
                            const isForwarded = forwardedPorts.has(port.port);
                            const isProtected = protectedPorts.includes(port.port);
                            const isPidOne = port.pid === 1;
                            const isLoading = protectingPorts.has(port.port);
                            return (
                                <div key={`${port.port}-${port.pid}`} className="mcc-lp-row">
                                    <div className="mcc-lp-row-main">
                                        <code className="mcc-lp-port-num">{port.port}</code>
                                        <span className="mcc-lp-command">{port.command}</span>
                                        <button 
                                            className={`mcc-lp-kill-btn ${isPidOne || isProtected ? 'mcc-lp-kill-btn-disabled' : ''}`}
                                            onClick={() => setKillModalPort(port)}
                                            title={isPidOne ? 'Cannot kill init process' : isProtected ? 'Port is protected' : `Kill process ${port.pid}`}
                                            disabled={isPidOne || isProtected}
                                        >
                                            ✕
                                        </button>
                                        {isProtected ? (
                                            <button 
                                                className="mcc-lp-protect-btn mcc-lp-protect-btn-active"
                                                onClick={() => handleUnprotect(port.port)}
                                                title={`Unprotect port ${port.port}`}
                                                disabled={isLoading}
                                            >
                                                {isLoading ? '...' : '🛡'}
                                            </button>
                                        ) : (
                                            <button 
                                                className="mcc-lp-protect-btn"
                                                onClick={() => handleProtect(port.port)}
                                                title={`Protect port ${port.port}`}
                                                disabled={isLoading}
                                            >
                                                {isLoading ? '...' : <span style={{ opacity: 0.5 }}>🛡</span>}
                                            </button>
                                        )}
                                        {isForwarded ? (
                                            <span className="mcc-lp-forwarded-badge">Forwarded</span>
                                        ) : (
                                            <button 
                                                className="mcc-lp-forward-btn"
                                                onClick={() => onForwardPort(port.port)}
                                                title={`Forward port ${port.port}`}
                                            >
                                                <PlusIcon />
                                            </button>
                                        )}
                                    </div>
                                    <div className="mcc-lp-row-meta">
                                        <span className="mcc-lp-pid">PID {port.pid}</span>
                                        <span className="mcc-lp-ppid">PPID {port.ppid}</span>
                                    </div>
                                    {port.cmdline && (
                                        <div className="mcc-lp-cmdline">{port.cmdline}</div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                    {killModalPort && (
                        <KillProcessModal
                            port={killModalPort}
                            protectedPorts={protectedPorts}
                            onClose={() => setKillModalPort(null)}
                            onKilled={() => setKillModalPort(null)}
                        />
                    )}
                </>
            )}
        </div>
    );
}
