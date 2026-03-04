export interface ACPChatConnectLogsProps {
    connectLogs: string[];
    debugLogs: string[];
    debugMode: boolean;
    showConnectLogs: boolean;
    onDismiss: () => void;
}

export function ACPChatConnectLogs({ connectLogs, debugLogs, debugMode, showConnectLogs, onDismiss }: ACPChatConnectLogsProps) {
    if (!showConnectLogs || connectLogs.length === 0) return null;

    return (
        <div className="acp-ui-connect-logs">
            <button
                className="acp-ui-connect-logs-dismiss"
                onClick={onDismiss}
                title="Dismiss logs"
            >
                <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
                    <path d="M6 6l12 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                    <path d="M18 6L6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                </svg>
            </button>
            {(debugMode ? debugLogs : connectLogs).map((log, i) => (
                <div key={i} className="acp-ui-connect-log-line">{log}</div>
            ))}
        </div>
    );
}
