import { useNavigate } from 'react-router-dom';
import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import { DomainProviders } from '../../../../api/domains';
import { Button } from '../../../../pure-view/buttons';
import './DomainStatusView.css';

interface DomainStatusViewProps {
    entry: DomainWithStatus;
    cfStatus: CloudflareStatus | null;
    starting: boolean;
    onStart: (domain: string) => void;
    onStop: (domain: string) => void;
}

export function DomainStatusView({ entry, cfStatus, starting, onStart, onStop }: DomainStatusViewProps) {
    const navigate = useNavigate();

    if (entry.provider === DomainProviders.Cloudflare && cfStatus) {
        if (!cfStatus.installed) {
            return (
                <div className="domain-status domain-status--warning">
                    <span className="domain-status-icon">⚠️</span>
                    <span>cloudflared not installed. Go to System Diagnostics to install.</span>
                </div>
            );
        }
        if (!cfStatus.authenticated) {
            return (
                <div className="domain-status domain-status--warning">
                    <span className="domain-status-icon">⚠️</span>
                    <span>cloudflared not authenticated.</span>
                    <Button variant="link" onClick={() => navigate('cloudflare')}>
                        Go to Cloudflare Settings
                    </Button>
                </div>
            );
        }
    }

    if (entry.provider === DomainProviders.Ngrok) {
        return (
            <div className="domain-status domain-status--warning">
                <span className="domain-status-icon">⚠️</span>
                <span>ngrok is not supported yet</span>
            </div>
        );
    }

    switch (entry.status) {
        case 'active':
            return (
                <div className="domain-status domain-status--active">
                    <span className="domain-status-icon">✅</span>
                    <a
                        href={entry.tunnel_url || `https://${entry.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="domain-status-url"
                    >
                        {entry.tunnel_url || `https://${entry.domain}`}
                    </a>
                    <Button variant="stop" onClick={() => onStop(entry.domain)}>
                        Stop
                    </Button>
                </div>
            );
        case 'connecting':
            return (
                <div className="domain-status domain-status--connecting">
                    <span className="domain-status-icon domain-status-spinner" />
                    <span>Connecting...</span>
                </div>
            );
        case 'error':
            return (
                <div className="domain-status domain-status--error">
                    <span className="domain-status-icon">❌</span>
                    <span>{entry.error || 'Tunnel error'}</span>
                    <Button variant="start" onClick={() => onStart(entry.domain)} disabled={starting}>
                        {starting ? 'Starting...' : 'Retry'}
                    </Button>
                </div>
            );
        default: // stopped
            return (
                <div className="domain-status domain-status--stopped">
                    <span className="domain-status-icon">
                        {starting ? <span className="domain-status-spinner" /> : '⏸'}
                    </span>
                    <span>{starting ? 'Starting...' : 'Stopped'}</span>
                    <Button variant="start" onClick={() => onStart(entry.domain)} disabled={starting}>
                        {starting ? 'Starting...' : 'Start'}
                    </Button>
                </div>
            );
    }
}
