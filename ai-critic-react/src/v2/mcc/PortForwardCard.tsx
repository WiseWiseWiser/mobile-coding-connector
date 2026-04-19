import { useState, useEffect } from 'react';
import type { PortForward } from '../../hooks/usePortForwards';
import { PortStatuses } from '../../hooks/usePortForwards';
import { fetchPortLogs as apiFetchPortLogs, fetchDomainHealthLogs } from '../../api/ports';
import { LogViewer } from '../LogViewer';

interface PortForwardCardProps {
    port: PortForward;
    onRemove: () => void;
    onNavigateToView: (view: string) => void;
}

export function PortForwardCard({ port, onRemove, onNavigateToView }: PortForwardCardProps) {
    const [showLogs, setShowLogs] = useState(false);
    const [logs, setLogs] = useState<string[]>([]);
    const [copied, setCopied] = useState(false);

    const statusIcon = port.status === PortStatuses.Active ? '🟢' :
                       port.status === PortStatuses.Connecting ? '🟡' : '🔴';

    const handleCopy = () => {
        if (port.publicUrl) {
            navigator.clipboard.writeText(port.publicUrl);
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        }
    };

    useEffect(() => {
        if (!showLogs) return;

        const fetchLogs = async () => {
            try {
                let data: string[];
                if (port.bootstrap) {
                    data = await fetchDomainHealthLogs(port.label);
                } else {
                    data = await apiFetchPortLogs(port.localPort);
                }
                setLogs(data);
            } catch { /* ignore */ }
        };

        fetchLogs();
        const timer = setInterval(fetchLogs, 2000);
        return () => clearInterval(timer);
    }, [showLogs, port.localPort, port.bootstrap, port.label]);

    return (
        <div className="mcc-port-card">
            <div className="mcc-port-header">
                <span className="mcc-port-status">{statusIcon}</span>
                <span className="mcc-port-number">:{port.localPort}</span>
                <span className="mcc-port-arrow">→</span>
                <span className="mcc-port-label">{port.label}</span>
                <span className="mcc-port-provider-badge">{port.provider}</span>
                {port.bootstrap && (
                    <span className="mcc-port-bootstrap-badge" title="Started automatically during server bootstrap">
                        Bootstrap
                    </span>
                )}
                {port.type && (
                    <span className="mcc-port-type-badge" title={`Type: ${port.type}`}>
                        {port.type.replace('_', ' ')}
                    </span>
                )}
            </div>
            {port.publicUrl ? (
                <div className="mcc-port-url">
                    <a href={port.publicUrl} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                        {port.publicUrl}
                    </a>
                    <button className="mcc-port-copy-icon" onClick={handleCopy} title="Copy URL">
                        {copied ? '✓' : '📋'}
                    </button>
                </div>
            ) : port.status === PortStatuses.Connecting ? (
                <div className="mcc-port-url mcc-port-url-connecting">Establishing tunnel...</div>
            ) : port.error ? (
                <div className="mcc-port-url mcc-port-url-error">{port.error}</div>
            ) : null}
            <div className="mcc-port-actions">
                <button
                    className={`mcc-port-action-btn mcc-port-logs-btn ${showLogs ? 'active' : ''}`}
                    onClick={() => setShowLogs(!showLogs)}
                >
                    Logs
                </button>
                <button
                    className="mcc-port-action-btn"
                    onClick={() => onNavigateToView(`port-diagnose/${port.localPort}`)}
                >
                    Diagnose
                </button>
                <button className="mcc-port-action-btn mcc-port-stop" onClick={onRemove}>Stop</button>
            </div>
            {showLogs && (
                <LogViewer
                    lines={logs.map(text => ({ text }))}
                    className="mcc-port-logs-margin"
                />
            )}
        </div>
    );
}
