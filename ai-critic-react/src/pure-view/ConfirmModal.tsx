import { useState } from 'react';
import './ConfirmModal.css';

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
    loading?: boolean;
    error?: string | null;
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
    loading: externalLoading,
    error: externalError,
}: ConfirmModalProps) {
    const [internalLoading, setInternalLoading] = useState(false);
    const [internalError, setInternalError] = useState<string | null>(null);

    const loading = externalLoading ?? internalLoading;
    const error = externalError ?? internalError;

    const handleConfirm = async () => {
        if (externalLoading !== undefined) {
            await onConfirm();
            return;
        }
        setInternalLoading(true);
        setInternalError(null);
        try {
            await onConfirm();
        } catch (err) {
            setInternalError(String(err));
        } finally {
            setInternalLoading(false);
        }
    };

    return (
        <div className="pure-confirm-modal-overlay" onClick={onClose}>
            <div className="pure-confirm-modal" onClick={e => e.stopPropagation()}>
                <div className="pure-confirm-modal-header">
                    <h3>{title}</h3>
                    <button className="pure-confirm-modal-close" onClick={onClose} disabled={loading}>×</button>
                </div>
                <div className="pure-confirm-modal-body">
                    {warning ? (
                        <p className="pure-confirm-modal-warning">{warning}</p>
                    ) : (
                        <p>{message}</p>
                    )}
                    {info && (
                        <div className="pure-confirm-modal-info">
                            {Object.entries(info).map(([key, value]) => (
                                <div key={key} className="pure-confirm-modal-row">
                                    <span className="pure-confirm-modal-label">{key}:</span>
                                    <span className="pure-confirm-modal-value">{value}</span>
                                </div>
                            ))}
                        </div>
                    )}
                    <div className="pure-confirm-modal-command">
                        <span className="pure-confirm-modal-label">Command to execute:</span>
                        <code>{command}</code>
                    </div>
                    {error && <div className="pure-confirm-modal-error">{error}</div>}
                </div>
                <div className="pure-confirm-modal-footer">
                    <button className="pure-confirm-modal-btn pure-confirm-modal-btn-cancel" onClick={onClose} disabled={loading}>
                        Cancel
                    </button>
                    <button 
                        className={`pure-confirm-modal-btn ${confirmVariant === 'danger' ? 'pure-confirm-modal-btn-kill' : ''}`}
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
