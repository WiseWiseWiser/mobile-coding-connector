import { useState } from 'react';
import { killProcess } from '../../api/ports';
import './KillProcessModal.css';

export interface LocalPortInfo {
    port: number;
    pid: number;
    ppid: number;
    command: string;
    cmdline: string;
}

export interface KillProcessModalProps {
    port: LocalPortInfo;
    protectedPorts: number[];
    onClose: () => void;
    onKilled: () => void;
}

export function KillProcessModal({ port, protectedPorts, onClose, onKilled }: KillProcessModalProps) {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const isPidOne = port.pid === 1;
    const isProtected = protectedPorts.includes(port.port);
    const canKill = !isPidOne && !isProtected;

    const command = `kill ${port.pid}`;

    const handleKill = async () => {
        if (!canKill) return;
        setLoading(true);
        setError(null);
        try {
            await killProcess(port.pid, port.port);
            onKilled();
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="mcc-modal-overlay" onClick={onClose}>
            <div className="mcc-modal" onClick={e => e.stopPropagation()}>
                <div className="mcc-modal-header">
                    <h3>Kill Process</h3>
                    <button className="mcc-modal-close" onClick={onClose} disabled={loading}>×</button>
                </div>
                <div className="mcc-modal-body">
                    {isPidOne ? (
                        <p className="mcc-modal-warning">Cannot kill init process (PID 1).</p>
                    ) : isProtected ? (
                        <p className="mcc-modal-warning">This port is protected and cannot be killed.</p>
                    ) : (
                        <p>Are you sure you want to kill this process?</p>
                    )}
                    <div className="mcc-modal-info">
                        <div className="mcc-modal-row">
                            <span className="mcc-modal-label">Port:</span>
                            <span className="mcc-modal-value">{port.port}</span>
                        </div>
                        <div className="mcc-modal-row">
                            <span className="mcc-modal-label">PID:</span>
                            <span className="mcc-modal-value">{port.pid}</span>
                        </div>
                        <div className="mcc-modal-row">
                            <span className="mcc-modal-label">Command:</span>
                            <span className="mcc-modal-value">{port.command}</span>
                        </div>
                    </div>
                    <div className="mcc-modal-command">
                        <span className="mcc-modal-label">Command to execute:</span>
                        <code>{command}</code>
                    </div>
                    {error && <div className="mcc-modal-error">{error}</div>}
                </div>
                <div className="mcc-modal-footer">
                    <button className="mcc-modal-btn mcc-modal-btn-cancel" onClick={onClose} disabled={loading}>
                        Cancel
                    </button>
                    <button 
                        className="mcc-modal-btn mcc-modal-btn-kill" 
                        onClick={handleKill} 
                        disabled={loading || !canKill}
                    >
                        {loading ? 'Killing...' : 'Kill Process'}
                    </button>
                </div>
            </div>
        </div>
    );
}
