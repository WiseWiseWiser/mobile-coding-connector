import { useState } from 'react';
import './KillProcessModal.css';

export interface LocalPortInfo {
    port: number;
    pid: number;
    ppid: number;
    command: string;
    cmdline: string;
}

export interface ConfirmModalProps {
    title: string;
    message: string;
    info?: Record<string, string>;
    command: string;
    confirmLabel: string;
    confirmVariant?: 'danger' | 'default';
    onConfirm: () => Promise<void>;
    onClose: () => void;
    warning?: string;
}

export function ConfirmModal({
    title,
    message,
    info,
    command,
    confirmLabel,
    confirmVariant = 'danger',
    onConfirm,
    onClose,
    warning,
}: ConfirmModalProps) {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleConfirm = async () => {
        setLoading(true);
        setError(null);
        try {
            await onConfirm();
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
                    <h3>{title}</h3>
                    <button className="mcc-modal-close" onClick={onClose} disabled={loading}>×</button>
                </div>
                <div className="mcc-modal-body">
                    {warning ? (
                        <p className="mcc-modal-warning">{warning}</p>
                    ) : (
                        <p>{message}</p>
                    )}
                    {info && (
                        <div className="mcc-modal-info">
                            {Object.entries(info).map(([key, value]) => (
                                <div key={key} className="mcc-modal-row">
                                    <span className="mcc-modal-label">{key}:</span>
                                    <span className="mcc-modal-value">{value}</span>
                                </div>
                            ))}
                        </div>
                    )}
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
                        className={`mcc-modal-btn ${confirmVariant === 'danger' ? 'mcc-modal-btn-kill' : ''}`}
                        onClick={handleConfirm} 
                        disabled={loading}
                    >
                        {loading ? 'Processing...' : confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    );
}
