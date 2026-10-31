interface BrowserOfflineProps {
    busy: boolean;
    stage?: string;
    error?: string;
    onStart: () => void;
}

/** Honest empty state: a disconnected extension does not prove Chrome quit. */
export function BrowserOffline({ busy, stage, error, onStart }: BrowserOfflineProps) {
    const pending = busy || stage === 'starting' || stage === 'waiting_extension';
    const label = stage === 'waiting_extension' ? 'Waiting for extension…' : 'Starting Chrome…';
    return (
        <div className="bw-overlay bw-offline" role="status">
            <span>No connected Chrome window for this session</span>
            <small>The session page may be closed, or the Browser Agent extension may be disconnected.</small>
            <button type="button" className="bw-offline-start" onClick={onStart} disabled={pending} aria-busy={pending}>
                {pending ? label : error || stage === 'failed' ? 'Retry' : 'Start Chrome'}
            </button>
            {error && <small className="bw-offline-error" role="alert">{error}</small>}
            <small>Opens Chrome on the Mac using your existing profile.</small>
        </div>
    );
}
