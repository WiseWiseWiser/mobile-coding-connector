import { DomainProviders } from '../../../../api/domains';
import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { EditButton } from '../../../../pure-view/buttons/EditButton';
import { DomainStatusView } from './DomainStatusView';
import './DomainRowView.css';

export interface DomainRowViewProps {
    entry: DomainWithStatus;
    cfStatus: CloudflareStatus | null;
    displayTunnelName: string;
    starting: boolean;
    startingDomain: string | null;
    startLogs: LogLine[];
    startDone: boolean;
    startError: boolean;
    isLast: boolean;
    onStart: (domain: string) => void;
    onStop: (domain: string) => void;
    onEdit?: () => void;
}

export function DomainRowView({
    entry, cfStatus, displayTunnelName,
    starting, startingDomain, startLogs, startDone, startError, isLast,
    onStart, onStop, onEdit,
}: DomainRowViewProps) {
    const showLogs = startLogs.length > 0 && (
        startingDomain === entry.domain ||
        (!startingDomain && isLast && (startDone || startError))
    );

    return (
        <div className="domain-row">
            <div className="domain-row-header">
                <a
                    href={entry.domain.startsWith('http://') || entry.domain.startsWith('https://') ? entry.domain : `https://${entry.domain}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="domain-row-link"
                >
                    {entry.domain}
                </a>
                {onEdit && <EditButton onClick={onEdit} />}
            </div>
            {entry.provider === DomainProviders.Cloudflare && (
                <div className="domain-row-tunnel">
                    <span className="domain-row-badge">{entry.provider}</span>
                    <span className="domain-row-tunnel-value">tunnel: {displayTunnelName}</span>
                </div>
            )}
            {entry.provider !== DomainProviders.Cloudflare && (
                <div className="domain-row-provider">
                    <span className="domain-row-badge">{entry.provider}</span>
                </div>
            )}
            <DomainStatusView
                entry={entry}
                cfStatus={cfStatus}
                starting={starting}
                onStart={onStart}
                onStop={onStop}
            />
            {showLogs && (
                <div className="domain-row-logs">
                    <span className="domain-row-logs-label">
                        {starting ? 'Starting tunnel...' : startDone ? 'Tunnel started' : startError ? 'Tunnel start failed' : ''}
                    </span>
                    <LogViewer
                        lines={startLogs}
                        pending={starting}
                        pendingMessage="Starting tunnel..."
                        maxHeight={200}
                    />
                </div>
            )}
        </div>
    );
}
