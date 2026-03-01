import { DomainProviders } from '../../../../api/domains';
import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { EditIcon } from '../../../../pure-view/icons/EditIcon';
import { DomainStatusView } from './DomainStatusView';

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
    onEdit: () => void;
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
        <div className="diagnose-webaccess-row">
            <div className="diagnose-webaccess-row-domain">
                <a
                    href={entry.domain.startsWith('http://') || entry.domain.startsWith('https://') ? entry.domain : `https://${entry.domain}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="diagnose-webaccess-domain-link"
                >
                    {entry.domain}
                </a>
                <button className="diagnose-webaccess-edit-btn" onClick={onEdit} title="Edit">
                    <EditIcon />
                </button>
            </div>
            {entry.provider === DomainProviders.Cloudflare && (
                <div className="diagnose-webaccess-tunnel-name-view">
                    <span className="diagnose-webaccess-provider-badge">{entry.provider}</span>
                    <span className="diagnose-webaccess-tunnel-name-value">tunnel: {displayTunnelName}</span>
                </div>
            )}
            {entry.provider !== DomainProviders.Cloudflare && (
                <div className="diagnose-webaccess-row-header">
                    <span className="diagnose-webaccess-provider-badge">{entry.provider}</span>
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
                <div className="diagnose-webaccess-tunnel-logs">
                    <span className="diagnose-webaccess-tunnel-logs-label">
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
