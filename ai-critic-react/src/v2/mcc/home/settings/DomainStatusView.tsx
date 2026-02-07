import { useNavigate } from 'react-router-dom';
import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import { DomainProviders } from '../../../../api/domains';

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
                <div className="diagnose-webaccess-status diagnose-webaccess-status--warning">
                    <span className="diagnose-webaccess-status-icon">⚠️</span>
                    <span>cloudflared not installed. Go to System Diagnostics to install.</span>
                </div>
            );
        }
        if (!cfStatus.authenticated) {
            return (
                <div className="diagnose-webaccess-status diagnose-webaccess-status--warning">
                    <span className="diagnose-webaccess-status-icon">⚠️</span>
                    <span>cloudflared not authenticated.</span>
                    <button
                        className="diagnose-webaccess-link-btn"
                        onClick={() => navigate('cloudflare')}
                    >
                        Go to Cloudflare Settings
                    </button>
                </div>
            );
        }
    }

    if (entry.provider === DomainProviders.Ngrok) {
        return (
            <div className="diagnose-webaccess-status diagnose-webaccess-status--warning">
                <span className="diagnose-webaccess-status-icon">⚠️</span>
                <span>ngrok is not supported yet</span>
            </div>
        );
    }

    switch (entry.status) {
        case 'active':
            return (
                <div className="diagnose-webaccess-status diagnose-webaccess-status--active">
                    <span className="diagnose-webaccess-status-icon">✅</span>
                    <a
                        href={entry.tunnel_url || `https://${entry.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="diagnose-webaccess-tunnel-url"
                    >
                        {entry.tunnel_url || `https://${entry.domain}`}
                    </a>
                    <button
                        className="diagnose-webaccess-stop-btn"
                        onClick={() => onStop(entry.domain)}
                    >
                        Stop
                    </button>
                </div>
            );
        case 'connecting':
            return (
                <div className="diagnose-webaccess-status diagnose-webaccess-status--connecting">
                    <span className="diagnose-webaccess-status-icon diagnose-webaccess-spinner" />
                    <span>Connecting...</span>
                </div>
            );
        case 'error':
            return (
                <div className="diagnose-webaccess-status diagnose-webaccess-status--error">
                    <span className="diagnose-webaccess-status-icon">❌</span>
                    <span>{entry.error || 'Tunnel error'}</span>
                    <button
                        className="diagnose-webaccess-start-btn"
                        onClick={() => onStart(entry.domain)}
                        disabled={starting}
                    >
                        {starting ? 'Starting...' : 'Retry'}
                    </button>
                </div>
            );
        default: // stopped
            return (
                <div className="diagnose-webaccess-status diagnose-webaccess-status--stopped">
                    <span className="diagnose-webaccess-status-icon">
                        {starting ? <span className="diagnose-webaccess-spinner" /> : '⏸'}
                    </span>
                    <span>{starting ? 'Starting...' : 'Stopped'}</span>
                    <button
                        className="diagnose-webaccess-start-btn"
                        onClick={() => onStart(entry.domain)}
                        disabled={starting}
                    >
                        {starting ? 'Starting...' : 'Start'}
                    </button>
                </div>
            );
    }
}
