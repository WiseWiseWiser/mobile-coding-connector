import type { DomainWithStatus, CloudflareStatus } from '../../../../api/domains';
import type { LogLine } from '../../../LogViewer';
import { DomainRowView } from './DomainRowView';
import { InlineError } from '../../../../pure-view/InlineError';
import { Button } from '../../../../pure-view/buttons';
import './ViewModeContent.css';

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
        <>
            {domainsList.length === 0 ? (
                <div className="view-mode-empty">
                    No domains configured.
                    <Button variant="link" onClick={onEdit}>Add one</Button>
                    to enable public access.
                </div>
            ) : (
                domainsList.map((entry, i) => (
                    <DomainRowView
                        key={entry.domain}
                        entry={entry}
                        cfStatus={cfStatus}
                        displayTunnelName={displayTunnelName}
                        starting={isStarting(entry.domain)}
                        startingDomain={startingDomain}
                        startLogs={startLogs}
                        startDone={startDone}
                        startError={startError}
                        isLast={i === domainsList.length - 1}
                        onStart={onStart}
                        onStop={onStop}
                    />
                ))
            )}

            {error && <InlineError>{error}</InlineError>}
        </>
    );
}
