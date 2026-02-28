import { WebServiceUI } from './WebServiceUI';

export function CodexWebUI() {
    return (
        <WebServiceUI
            port={3000}
            title="Codex Web"
            statusEndpoint="/api/codex-web/status"
            startEndpoint="/api/codex-web/start"
            stopEndpoint="/api/codex-web/stop"
            backPath="../experimental"
        />
    );
}
