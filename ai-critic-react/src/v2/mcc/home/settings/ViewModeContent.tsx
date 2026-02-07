import { DomainProviders } from '../../../../api/domains';
import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import { LogViewer } from '../../../LogViewer';
import type { LogLine } from '../../../LogViewer';
import { DomainStatusView } from './DomainStatusView';

export interface ViewModeContentProps {
    domainsList: DomainWithStatus[];
    cfStatus: CloudflareStatus | null;
    displayTunnelName: string;
    error: string | null;
    startingDomain: string | null;
    startLogs: LogLine[];
    startDone: boolean;
    startError: boolean;
    isStarting: (domain: string) => boolean;
    onStart: (domain: string) => void;
    onStop: (domain: string) => void;
    onEdit: () => void;
}

export function ViewModeContent({
    domainsList, cfStatus, displayTunnelName, error,
    startingDomain, startLogs, startDone, startError,
    isStarting, onStart, onStop, onEdit,
}: ViewModeContentProps) {
    return (
        <div className="diagnose-webaccess-card">
            {domainsList.length === 0 ? (
                <div className="diagnose-webaccess-empty">
                    No domains configured.
                    <button className="diagnose-webaccess-link-btn" onClick={onEdit}>Add one</button>
                    to enable public access.
                </div>
            ) : (
                <div className="diagnose-webaccess-list">
                    {domainsList.map((entry, i) => (
                        <div key={entry.domain} className="diagnose-webaccess-row">
                            <div className="diagnose-webaccess-row-domain">
                                <a
                                    href={entry.domain.startsWith('http://') || entry.domain.startsWith('https://') ? entry.domain : `https://${entry.domain}`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="diagnose-webaccess-domain-link"
                                >
                                    {entry.domain}
                                </a>
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
                                starting={isStarting(entry.domain)}
                                onStart={onStart}
                                onStop={onStop}
                            />
                            {startLogs.length > 0 && (startingDomain === entry.domain || (!startingDomain && i === domainsList.length - 1 && (startDone || startError))) && (
                                <div className="diagnose-webaccess-tunnel-logs">
                                    <span className="diagnose-webaccess-tunnel-logs-label">
                                        {isStarting(entry.domain) ? 'Starting tunnel...' : startDone ? 'Tunnel started' : startError ? 'Tunnel start failed' : ''}
                                    </span>
                                    <LogViewer
                                        lines={startLogs}
                                        pending={isStarting(entry.domain)}
                                        pendingMessage="Starting tunnel..."
                                        maxHeight={200}
                                    />
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            )}

            {error && <div className="diagnose-security-error">{error}</div>}
        </div>
    );
}
